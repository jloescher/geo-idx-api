package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/mark3labs/mcp-go/server"

	"github.com/quantyralabs/idx-api/internal/config"
	"github.com/quantyralabs/idx-api/internal/mcp/monitor"
	"github.com/quantyralabs/idx-api/internal/repository"
	"github.com/quantyralabs/idx-api/internal/service/comps"
	"github.com/quantyralabs/idx-api/internal/service/dashboard"
)

// buildResourceMetadataURL returns the absolute URL for RFC 9728 Protected Resource Metadata.
// It prefers MCP_PUBLIC_URL (set in Coolify for the mcp app), then falls back using
// X-Forwarded-* headers (important behind Cloudflare/Traefik) and finally r.Host.
func buildResourceMetadataURL(r *http.Request) string {
	if u := os.Getenv("MCP_PUBLIC_URL"); u != "" {
		// Strip any path after the host and append the well-known endpoint
		if idx := strings.Index(u, "/mcp"); idx != -1 {
			return u[:idx] + "/.well-known/oauth-protected-resource"
		}
		if strings.HasSuffix(u, "/") {
			return u + ".well-known/oauth-protected-resource"
		}
		if !strings.HasSuffix(u, "/.well-known/oauth-protected-resource") {
			return u + "/.well-known/oauth-protected-resource"
		}
		return u
	}

	// Proxy-aware fallback
	scheme := r.Header.Get("X-Forwarded-Proto")
	if scheme == "" {
		scheme = "https"
		if r.TLS == nil && os.Getenv("APP_ENV") != "production" {
			scheme = "http"
		}
	}
	host := r.Header.Get("X-Forwarded-Host")
	if host == "" {
		host = r.Host
	}

	// Final safety net for production (prevents the "https://.well-known/..." bug
	// when running behind Traefik/Cloudflare that sometimes doesn't forward the host headers
	// in the way we expect for this specific service).
	if host == "" {
		return "https://mcp.quantyralabs.cc/.well-known/oauth-protected-resource"
	}

	return scheme + "://" + host + "/.well-known/oauth-protected-resource"
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	db, err := repository.New(context.Background(), cfg.DB)
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	mcpKeyRepo := repository.NewMCPKeyRepo(db)
	oauthRepo := repository.NewOAuthRepo(db)
	monitoringService := dashboard.NewMonitoringService(cfg, db)
	monitoringRepo := repository.NewMonitoringRepo(db)
	compsEngine := comps.NewEngine(cfg, db)

	if os.Getenv("MCP_KEY") != "" {
		logger.Info("MCP_KEY detected in environment")
	}

	// Create the core monitoring + comps MCP server (shared by both transports)
	monitorServer := monitor.NewServer(mcpKeyRepo, monitoringService, monitoringRepo, compsEngine)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Graceful shutdown on SIGINT/SIGTERM
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Decide transport mode:
	// - If MCP_HTTP_ADDR, PORT, or MCP_HTTP_PORT is set (or MCP_HTTP_ENABLED=true),
	//   run the Streamable HTTP + SSE server (suitable for Coolify / remote agents).
	// - Otherwise fall back to stdio (ideal for local Claude Desktop, Cursor, Grok connectors, etc.).
	httpAddr := os.Getenv("MCP_HTTP_ADDR")
	if httpAddr == "" {
		if p := os.Getenv("PORT"); p != "" {
			httpAddr = ":" + p
		} else if p := os.Getenv("MCP_HTTP_PORT"); p != "" {
			httpAddr = ":" + p
		}
	}
	runHTTP := httpAddr != "" || os.Getenv("MCP_HTTP_ENABLED") == "true" || os.Getenv("MCP_HTTP_PORT") != ""

	if runHTTP {
		if httpAddr == "" {
			httpAddr = ":3000" // default matches historical Coolify app config for this service
		}

		// Build a small mux with the MCP handler + unauthenticated health endpoints.
		// Health endpoints are critical for Coolify / container orchestrators.
		mux := http.NewServeMux()

		mcpHandler := monitorServer.HTTPHandler()

		// Dual auth middleware (functional version):
		// - If Authorization: Bearer mcp_... → use existing direct mcp_key path (unchanged, highest priority)
		// - Else if Authorization: Bearer <oauth_access_token> → validate against oauth_access_tokens table
		// - Otherwise → 401 with resource_metadata pointing to OAuth flow
		authChallengeHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// HTTPS enforcement (must happen before any auth decision)
			if r.Header.Get("X-Forwarded-Proto") != "https" && os.Getenv("APP_ENV") == "production" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"error":"https_required","error_description":"This endpoint must be accessed over HTTPS"}`))
				return
			}

			auth := r.Header.Get("Authorization")
			hasSession := r.Header.Get("Mcp-Session-Id") != "" || r.Header.Get("mcp-session-id") != ""

			// If this is a continuation of an established Streamable HTTP session, forward
			// unconditionally. The inner streamable server owns session lifecycle and auth.
			// This preserves pre-OAuth behavior for raw mcp_ key clients that rely on
			// Mcp-Session-Id for follow-up messages (initialize + subsequent tool calls).
			if hasSession {
				mcpHandler.ServeHTTP(w, r)
				return
			}

			// Path 1: Direct mcp_ key (existing behavior, completely unchanged and highest priority)
			if strings.HasPrefix(auth, "Bearer mcp_") {
				mcpHandler.ServeHTTP(w, r)
				return
			}

			// Path 2: Any other Bearer token (OAuth access token path)
			// Forward to the inner handler so it can validate or return a properly-typed
			// MCP error response. Only completely unauthenticated requests get the RFC 9728 challenge.
			if strings.HasPrefix(auth, "Bearer ") {
				token := strings.TrimPrefix(auth, "Bearer ")
				tokenHash := sha256.Sum256([]byte(token))
				tokenHashStr := hex.EncodeToString(tokenHash[:])

				accessToken, err := oauthRepo.FindAccessTokenByHash(r.Context(), tokenHashStr)
				if err == nil && accessToken != nil && time.Now().Before(accessToken.ExpiresAt) {
					// Valid OAuth access token → enrich context for the existing mcpKey / scope logic
					ctx := r.Context()
					ctx = context.WithValue(ctx, monitor.OAuthAccessTokenContextKey, accessToken)

					if len(accessToken.GrantedMCPKeyIDs) > 0 {
						key, err := mcpKeyRepo.FindByID(r.Context(), accessToken.GrantedMCPKeyIDs[0])
						if err == nil && key != nil {
							ctx = context.WithValue(ctx, monitor.MCPKeyContextKey, key)
						}
					}

					r = r.WithContext(ctx)
					mcpHandler.ServeHTTP(w, r)
					return
				}
				// Invalid OAuth token: fall through to forward the request so the streamable
				// handler (or inner httpContextFunc) can produce a correct JSON-RPC error
				// with proper Content-Type instead of a raw 401 challenge body.
			}

			// No Authorization header at all, and no established session → RFC 9728 challenge.
			// This is the only path that should ever return the WWW-Authenticate challenge.
			// Clients doing raw mcp_ key (pre-OAuth style) or holding a session will never hit this.
			resourceMetaURL := buildResourceMetadataURL(r)

			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("WWW-Authenticate", `Bearer resource_metadata="`+resourceMetaURL+`"`)
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"unauthorized","error_description":"MCP key or OAuth access token required"}`))
		})

		mux.Handle("/mcp", authChallengeHandler)
		mux.Handle("/mcp/", authChallengeHandler)

		mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ok","service":"idx-api-mcp"}`))
		})
		mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		})

		// Tiny unauthenticated debug endpoint for OAuth / RFC 9728 discovery troubleshooting.
		// Hit this directly (no auth) to see exactly what the process sees for the critical vars
		// and what buildResourceMetadataURL would return right now.
		mux.HandleFunc("/debug/oauth-config", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")

			mcpURL := os.Getenv("MCP_PUBLIC_URL")
			oauthServer := os.Getenv("OAUTH_AUTH_SERVER")

			// Simulate a realistic request that would come through Traefik/Cloudflare
			simReq := &http.Request{
				Header: http.Header{
					"X-Forwarded-Proto": []string{r.Header.Get("X-Forwarded-Proto")},
					"X-Forwarded-Host":  []string{r.Header.Get("X-Forwarded-Host")},
				},
			}
			if simReq.Header.Get("X-Forwarded-Proto") == "" {
				simReq.Header.Set("X-Forwarded-Proto", "https")
			}
			if simReq.Header.Get("X-Forwarded-Host") == "" {
				simReq.Header.Set("X-Forwarded-Host", r.Host)
			}

			producedURL := buildResourceMetadataURL(simReq)

			resp := map[string]any{
				"process_mcp_public_url":        mcpURL,
				"process_oauth_auth_server":     oauthServer,
				"produced_resource_metadata_url": producedURL,
				"note":                          "If produced_resource_metadata_url is wrong while process_mcp_public_url is set, the helper is hitting the fallback (header/env visibility issue in this environment).",
			}
			_ = json.NewEncoder(w).Encode(resp)
		})

		// Protected Resource Metadata (RFC 9728) - required for proper OAuth discovery by clients like Grok
		mux.HandleFunc("/.well-known/oauth-protected-resource", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)

			resource := os.Getenv("MCP_PUBLIC_URL")
			if resource == "" {
				// Fallback (should be set in Coolify for the mcp app)
				resource = "https://" + r.Host + "/mcp"
			}
			authServer := os.Getenv("OAUTH_AUTH_SERVER")
			if authServer == "" {
				// Fallback to same host (works only in single-binary dev setups)
				authServer = "https://" + r.Host
			}

			_, _ = w.Write([]byte(fmt.Sprintf(`{
  "resource": "%s",
  "authorization_servers": ["%s"],
  "scopes_supported": ["monitor", "comps", "content"]
}`, resource, authServer)))
		})

		srv := &http.Server{
			Addr:         httpAddr,
			Handler:      mux,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 60 * time.Second, // allow for long-running tool calls that stream
			IdleTimeout:  120 * time.Second,
		}

		go func() {
			<-sigChan
			logger.Info("shutting down MCP monitor (HTTP)")
			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer shutdownCancel()
			_ = srv.Shutdown(shutdownCtx)
			cancel()
		}()

		// Diagnostic logging for OAuth / RFC 9728 discovery (helps debug proxy header + env injection issues)
		mcpPublicURL := os.Getenv("MCP_PUBLIC_URL")
		oauthAuthServer := os.Getenv("OAUTH_AUTH_SERVER")
		exampleResourceMeta := buildResourceMetadataURL(&http.Request{})
		logger.Info("MCP OAuth config at startup (for Grok Web / RFC 9728)",
			"MCP_PUBLIC_URL", mcpPublicURL,
			"OAUTH_AUTH_SERVER", oauthAuthServer,
			"example_resource_metadata_url", exampleResourceMeta,
		)

		logger.Info("starting idx-api MCP monitor (HTTP + SSE streamable transport)",
			"addr", httpAddr,
			"mcp_endpoint", "/mcp",
			"health", "/healthz",
			"version", "0.2.0",
		)

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP server error", "error", err)
			os.Exit(1)
		}
		return
	}

	// --- Stdio transport (original local / connector mode) ---
	stdioServer := server.NewStdioServer(monitorServer.GetMCPServer())

	go func() {
		<-sigChan
		logger.Info("shutting down MCP monitor (stdio)")
		cancel()
	}()

	logger.Info("starting idx-api MCP monitor (stdio)", "version", "0.2.0")

	if err := stdioServer.Listen(ctx, os.Stdin, os.Stdout); err != nil {
		logger.Error("MCP server error", "error", err)
		os.Exit(1)
	}
}

package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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
			// HTTPS enforcement
			if r.Header.Get("X-Forwarded-Proto") != "https" && os.Getenv("APP_ENV") == "production" {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"error":"https_required","error_description":"This endpoint must be accessed over HTTPS"}`))
				return
			}

			auth := r.Header.Get("Authorization")

			// Path 1: Direct mcp_ key (existing behavior, unchanged)
			if strings.HasPrefix(auth, "Bearer mcp_") {
				mcpHandler.ServeHTTP(w, r)
				return
			}

			// Path 2: OAuth access token (new flow)
			if strings.HasPrefix(auth, "Bearer ") {
				token := strings.TrimPrefix(auth, "Bearer ")
				tokenHash := sha256.Sum256([]byte(token))
				tokenHashStr := hex.EncodeToString(tokenHash[:])

				accessToken, err := oauthRepo.FindAccessTokenByHash(r.Context(), tokenHashStr)
				if err == nil && accessToken != nil && time.Now().Before(accessToken.ExpiresAt) {
					// Valid OAuth access token.
					// For a functional implementation, we map it to the granted mcp keys.
					// If the token recorded specific mcp_key IDs, we can inject the first one
					// so the existing `authenticated()` + `requireScope()` logic continues to work.

					ctx := r.Context()
					ctx = context.WithValue(ctx, monitor.OAuthAccessTokenContextKey, accessToken)

					// If this token was issued with specific granted mcp keys, inject them
					// into the context so the existing authenticated() + requireScope() logic works.
					if len(accessToken.GrantedMCPKeyIDs) > 0 {
						// For functional v1, inject the first granted key.
						// A more advanced version could combine scopes from all granted keys.
						key, err := mcpKeyRepo.FindByID(r.Context(), accessToken.GrantedMCPKeyIDs[0])
						if err == nil && key != nil {
							ctx = context.WithValue(ctx, monitor.MCPKeyContextKey, key)
						}
					}

					r = r.WithContext(ctx)
					mcpHandler.ServeHTTP(w, r)
					return
				}
			}

			// No valid credential → tell client to start OAuth flow
			// Build the resource_metadata URL dynamically (never hardcode a placeholder)
			resourceMetaURL := os.Getenv("MCP_PUBLIC_URL")
			if resourceMetaURL == "" {
				// Fallback using the incoming request host (works for most Coolify setups)
				scheme := "https"
				if r.TLS == nil && os.Getenv("APP_ENV") != "production" {
					scheme = "http"
				}
				resourceMetaURL = scheme + "://" + r.Host + "/.well-known/oauth-protected-resource"
			} else {
				// Convert e.g. https://mcp.quantyralabs.cc/mcp  →  https://mcp.quantyralabs.cc/.well-known/oauth-protected-resource
				if idx := strings.Index(resourceMetaURL, "/mcp"); idx != -1 {
					resourceMetaURL = resourceMetaURL[:idx] + "/.well-known/oauth-protected-resource"
				} else if !strings.HasSuffix(resourceMetaURL, "/.well-known/oauth-protected-resource") {
					if strings.HasSuffix(resourceMetaURL, "/") {
						resourceMetaURL += ".well-known/oauth-protected-resource"
					} else {
						resourceMetaURL += "/.well-known/oauth-protected-resource"
					}
				}
			}

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

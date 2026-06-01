package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/mark3labs/mcp-go/server"

	"github.com/quantyralabs/idx-api/internal/config"
	"github.com/quantyralabs/idx-api/internal/mcp/apiclient"
	"github.com/quantyralabs/idx-api/internal/mcp/auth"
	"github.com/quantyralabs/idx-api/internal/mcp/idx"
	"github.com/quantyralabs/idx-api/internal/mcp/monitor"
	"github.com/quantyralabs/idx-api/internal/mcp/ratelimit"
	"github.com/quantyralabs/idx-api/internal/repository"
	"github.com/quantyralabs/idx-api/internal/service/comps"
	"github.com/quantyralabs/idx-api/internal/service/dashboard"
	"github.com/quantyralabs/idx-api/internal/service/search"
)

// buildResourceMetadataURL returns the absolute URL for RFC 9728 Protected Resource Metadata.
func buildResourceMetadataURL(r *http.Request) string {
	if raw := os.Getenv("MCP_PUBLIC_URL"); raw != "" {
		if parsed, err := url.Parse(raw); err == nil && parsed.Host != "" {
			origin := parsed.Scheme + "://" + parsed.Host
			return origin + "/.well-known/oauth-protected-resource"
		}
	}

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
	mcpUsageRepo := repository.NewMCPUsageRepo(db)
	monitoringService := dashboard.NewMonitoringService(cfg, db)
	monitoringRepo := repository.NewMonitoringRepo(db)
	compsEngine := comps.NewEngine(cfg, db)
	postgisSearch := search.NewPostgisSearch(db)

	apiClientCfg := apiclient.LoadConfigFromEnv()
	apiClient := apiclient.New(apiClientCfg)
	domainSlug := apiClientCfg.DomainSlug

	authInjector := &auth.Injector{
		KeyRepo:   mcpKeyRepo,
		OAuthRepo: oauthRepo,
	}
	rateLimiter := ratelimit.NewLimiter(mcpUsageRepo)

	monitorServer := monitor.NewServer(
		mcpKeyRepo,
		monitoringService,
		monitoringRepo,
		compsEngine,
		cfg,
		postgisSearch,
		domainSlug,
		authInjector,
		rateLimiter,
	)

	idxServer := idx.NewServer(cfg, mcpKeyRepo, db, apiClient, domainSlug, rateLimiter)
	idxServer.RegisterTools(monitorServer.GetMCPServer())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

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
			httpAddr = ":3000"
		}

		mux := http.NewServeMux()
		mcpHandler := monitorServer.HTTPHandler()

		authChallengeHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("X-Forwarded-Proto") != "https" && os.Getenv("APP_ENV") == "production" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"error":"https_required","error_description":"This endpoint must be accessed over HTTPS"}`))
				return
			}

			authHeader := r.Header.Get("Authorization")
			hasSession := r.Header.Get("Mcp-Session-Id") != "" || r.Header.Get("mcp-session-id") != ""

			// Always inject auth context (OAuth or mcp_ key) before forwarding.
			ctx := authInjector.InjectFromHTTP(r.Context(), r)
			r = r.WithContext(ctx)

			if hasSession || strings.HasPrefix(authHeader, "Bearer ") {
				mcpHandler.ServeHTTP(w, r)
				return
			}

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

		mux.HandleFunc("/debug/oauth-config", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			mcpURL := os.Getenv("MCP_PUBLIC_URL")
			oauthServer := os.Getenv("OAUTH_AUTH_SERVER")
			simReq := &http.Request{Header: http.Header{
				"X-Forwarded-Proto": []string{r.Header.Get("X-Forwarded-Proto")},
				"X-Forwarded-Host":  []string{r.Header.Get("X-Forwarded-Host")},
			}}
			if simReq.Header.Get("X-Forwarded-Proto") == "" {
				simReq.Header.Set("X-Forwarded-Proto", "https")
			}
			if simReq.Header.Get("X-Forwarded-Host") == "" {
				simReq.Header.Set("X-Forwarded-Host", r.Host)
			}
			resp := map[string]any{
				"process_mcp_public_url":         mcpURL,
				"process_oauth_auth_server":      oauthServer,
				"produced_resource_metadata_url": buildResourceMetadataURL(simReq),
			}
			_ = json.NewEncoder(w).Encode(resp)
		})

		mux.HandleFunc("/.well-known/oauth-protected-resource", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)

			resource := os.Getenv("MCP_PUBLIC_URL")
			if resource == "" {
				resource = "https://" + r.Host + "/mcp"
			}
			authServer := os.Getenv("OAUTH_AUTH_SERVER")
			if authServer == "" {
				authServer = "https://" + r.Host
			}

			_, _ = w.Write([]byte(fmt.Sprintf(`{
  "resource": "%s",
  "authorization_servers": ["%s"],
  "scopes_supported": ["monitor", "comps", "content", "api"]
}`, resource, authServer)))
		})

		srv := &http.Server{
			Addr:         httpAddr,
			Handler:      mux,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 60 * time.Second,
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

		logger.Info("starting idx-api MCP monitor (HTTP + SSE)",
			"addr", httpAddr,
			"mcp_endpoint", "/mcp",
			"api_client_enabled", apiClient.Enabled(),
			"domain_slug", domainSlug,
			"version", "0.3.0",
		)

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP server error", "error", err)
			os.Exit(1)
		}
		return
	}

	stdioServer := server.NewStdioServer(monitorServer.GetMCPServer())

	go func() {
		<-sigChan
		logger.Info("shutting down MCP monitor (stdio)")
		cancel()
	}()

	logger.Info("starting idx-api MCP monitor (stdio)", "version", "0.3.0")

	if err := stdioServer.Listen(ctx, os.Stdin, os.Stdout); err != nil {
		logger.Error("MCP server error", "error", err)
		os.Exit(1)
	}
}

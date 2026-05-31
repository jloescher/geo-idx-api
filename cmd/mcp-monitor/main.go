package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
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
		mux.Handle("/mcp", mcpHandler)
		mux.Handle("/mcp/", mcpHandler) // allow trailing slash variants

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

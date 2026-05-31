package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

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

	// For Grok Connectors / easy stdio usage, you can set MCP_KEY env var.
	// The tools still accept mcp_key as a parameter for maximum flexibility,
	// but future versions can default to the env var when not provided per-call.
	if os.Getenv("MCP_KEY") != "" {
		logger.Info("MCP_KEY detected in environment (recommended for connector usage)")
	}

	// Create our clean monitoring + comps MCP server
	monitorServer := monitor.NewServer(mcpKeyRepo, monitoringService, monitoringRepo, compsEngine)

	// Wrap it for stdio transport
	stdioServer := server.NewStdioServer(monitorServer.GetMCPServer())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		logger.Info("shutting down MCP monitor")
		cancel()
	}()

	logger.Info("starting idx-api MCP monitor (stdio)", "version", "0.1.0")

	if err := stdioServer.Listen(ctx, os.Stdin, os.Stdout); err != nil {
		logger.Error("MCP server error", "error", err)
		os.Exit(1)
	}
}

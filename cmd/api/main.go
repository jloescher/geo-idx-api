package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/joho/godotenv"
	"github.com/quantyralabs/idx-api/internal/api"
	"github.com/quantyralabs/idx-api/internal/config"
	"github.com/quantyralabs/idx-api/internal/db"
	"github.com/quantyralabs/idx-api/internal/repository"
)

func main() {
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config", "error", err)
		os.Exit(1)
	}

	logger := newLogger(cfg)
	slog.SetDefault(logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dbConn, err := repository.New(ctx, cfg.DB)
	if err != nil {
		logger.Error("database", "error", err)
		os.Exit(1)
	}

	var selector *db.ReplicaSelector
	if cfg.DB.ReadOnlyBaseDSN != "" {
		selector, err = db.NewReplicaSelector(ctx, cfg.DB.PatroniHosts, cfg.DB.ReadOnlyBaseDSN, dbConn.Pool, logger)
		if err != nil {
			logger.Error("replica selector", "error", err)
			dbConn.Close()
			os.Exit(1)
		}
		dbConn.Selector = selector
		logger.Info("read replica selector enabled", "patroni_hosts", cfg.DB.PatroniHosts)
	}

	defer func() {
		if selector != nil {
			selector.Close()
		}
		dbConn.Close()
	}()

	app := fiber.New(fiber.Config{
		AppName:      cfg.App.Name,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  120 * time.Second,
	})
	app.Use(recover.New())

	api.RegisterRoutes(app, cfg, dbConn, logger)

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		logger.Info("shutdown signal received")
		cancel()
		_ = app.ShutdownWithTimeout(30 * time.Second)
	}()

	addr := ":" + cfg.App.Port
	logger.Info("api listening", "addr", addr)
	if err := app.Listen(addr); err != nil {
		logger.Error("listen", "error", err)
		os.Exit(1)
	}
}

func newLogger(cfg config.Config) *slog.Logger {
	opts := &slog.HandlerOptions{Level: slog.LevelInfo}
	if cfg.App.Debug {
		opts.Level = slog.LevelDebug
	}
	var h slog.Handler = slog.NewTextHandler(os.Stdout, opts)
	if cfg.App.LogJSON {
		h = slog.NewJSONHandler(os.Stdout, opts)
	}
	return slog.New(h)
}

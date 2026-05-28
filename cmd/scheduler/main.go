package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/quantyralabs/idx-api/internal/config"
	"github.com/quantyralabs/idx-api/internal/queue"
	"github.com/quantyralabs/idx-api/internal/repository"
	"github.com/quantyralabs/idx-api/internal/scheduler"
)

func main() {
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config", "error", err)
		os.Exit(1)
	}

	logger := slog.Default()
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	db, err := repository.NewFromDSN(ctx, cfg.DB.RWDSNWithApplicationName("idx-scheduler"))
	if err != nil {
		logger.Error("database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	q := queue.NewClient(db.Pool, cfg.Queue.Table, cfg.Queue.NotifyChannel, cfg.Queue.RetryAfter, cfg.Queue.ReservationTimeout)
	sched := scheduler.New(cfg, q, db, logger)

	logger.Info("scheduler started")
	if err := sched.Run(ctx); err != nil && ctx.Err() == nil {
		logger.Error("scheduler", "error", err)
		os.Exit(1)
	}
}

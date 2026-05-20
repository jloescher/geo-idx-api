package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/quantyralabs/idx-api/internal/config"
	"github.com/quantyralabs/idx-api/internal/job"
	"github.com/quantyralabs/idx-api/internal/queue"
	"github.com/quantyralabs/idx-api/internal/repository"
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

	db, err := repository.New(ctx, cfg.DB)
	if err != nil {
		logger.Error("database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	q := queue.NewClient(db.Pool, cfg.Queue.Table, cfg.Queue.NotifyChannel, cfg.Queue.RetryAfter)
	registry := job.NewRegistry(cfg, db, logger)
	registry.InitServices(q)

	worker := queue.NewWorker(q, cfg.Queue.WorkerQueues, cfg.Queue.PollInterval, logger)
	registry.RegisterAll(worker)

	logger.Info("worker started", "queues", cfg.Queue.WorkerQueues)
	if err := worker.Run(ctx); err != nil && ctx.Err() == nil {
		logger.Error("worker", "error", err)
		os.Exit(1)
	}
}

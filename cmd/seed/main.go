package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/joho/godotenv"
	"github.com/quantyralabs/idx-api/internal/config"
	"github.com/quantyralabs/idx-api/internal/repository"
	"github.com/quantyralabs/idx-api/internal/seed"
)

func main() {
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config", "error", err)
		os.Exit(1)
	}

	ctx := context.Background()
	db, err := repository.New(ctx, cfg.DB)
	if err != nil {
		slog.Error("database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	id, err := seed.Admin(ctx, db.Pool, cfg.Auth.AdminSeedEmail, cfg.Auth.AdminSeedPass, cfg.Auth.AdminSeedName)
	if err != nil {
		slog.Error("seed admin", "error", err)
		os.Exit(1)
	}

	slog.Info("admin user ready", "id", id, "email", cfg.Auth.AdminSeedEmail)
}

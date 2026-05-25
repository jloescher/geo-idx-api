package main

import (
	"context"
	"flag"
	"log/slog"
	"os"

	"github.com/joho/godotenv"
	"github.com/quantyralabs/idx-api/internal/config"
	"github.com/quantyralabs/idx-api/internal/queue"
	"github.com/quantyralabs/idx-api/internal/repository"
	gisrepo "github.com/quantyralabs/idx-api/internal/repository/gis"
	"github.com/quantyralabs/idx-api/internal/service/gis"
)

func main() {
	job := flag.String("job", "", "Job to enqueue: parcels or zips")
	county := flag.String("county", "", "Optional county slug for single-county parcel sync (e.g. hillsborough)")
	flag.Parse()

	switch *job {
	case "parcels", "zips":
	default:
		slog.Error("usage: gis-enqueue -job parcels|zips [-county slug]")
		os.Exit(1)
	}

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

	q := queue.NewClient(db.Pool, cfg.Queue.Table, cfg.Queue.NotifyChannel, cfg.Queue.RetryAfter)
	queueName := cfg.GIS.SyncQueue
	if queueName == "" {
		queueName = "default"
	}

	if *job == "parcels" && *county != "" {
		gisRepo := gisrepo.New(db)
		svc := gis.NewParcelSyncService(cfg, gisRepo, q, slog.Default())
		if err := svc.KickoffCounty(ctx, *county); err != nil {
			slog.Error("county parcel kickoff failed", "county", *county, "error", err)
			os.Exit(1)
		}
		slog.Info("gis county parcel kickoff enqueued", "county", *county, "queue", queueName)
		return
	}

	var jobType string
	switch *job {
	case "parcels":
		jobType = queue.TypeGISMonthlyParcelRefresh
	case "zips":
		jobType = queue.TypeGISZipSync
	}

	id, err := q.Enqueue(ctx, queueName, jobType, nil, 0)
	if err != nil {
		slog.Error("enqueue failed", "job", *job, "type", jobType, "queue", queueName, "error", err)
		os.Exit(1)
	}

	slog.Info("gis job enqueued", "job", *job, "type", jobType, "queue", queueName, "job_id", id)
}

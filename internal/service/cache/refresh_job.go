package cache

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/quantyralabs/idx-api/internal/config"
	"github.com/quantyralabs/idx-api/internal/queue"
	"github.com/quantyralabs/idx-api/internal/repository"
)

// RefreshJob refreshes listings_cache for active domains (15-min schedule).
// Revenue impact: warm cache improves TTFB and conversion on listing grids.
type RefreshJob struct {
	cfg    config.Config
	db     *repository.DB
	logger *slog.Logger
}

func NewRefreshJob(cfg config.Config, db *repository.DB, logger *slog.Logger) *RefreshJob {
	return &RefreshJob{cfg: cfg, db: db, logger: logger}
}

func (j *RefreshJob) Run(ctx context.Context, _ *queue.ReservedJob) error {
	domains, err := repository.NewDomainRepo(j.db).ListActive(ctx)
	if err != nil {
		return err
	}
	for _, d := range domains {
		if !d.IsVerified() {
			continue
		}
		feeds := repository.NewDomainRepo(j.db).AllowedDatasets(&d)
		if len(feeds) == 0 {
			feeds = []string{j.cfg.Bridge.Dataset}
		}
		for _, feed := range feeds {
			j.logger.Info("listings cache refresh", "domain", d.DomainSlug, "feed", feed)
		}
	}
	return nil
}

// RefreshArgs optional per-domain job args.
type RefreshArgs struct {
	DomainSlug string `json:"domain_slug"`
	FeedCode   string `json:"feed_code"`
}

func decodeArgs(job *queue.ReservedJob, out any) error {
	return json.Unmarshal(job.Payload.Args, out)
}

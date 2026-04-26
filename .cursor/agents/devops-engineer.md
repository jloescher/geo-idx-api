---
  Docker (FrankenPHP + Nginx), Docker Compose dev/prod stacks, Dokploy deployment, queue workers, and Cloudflare tunnel config
tools: Read, Edit, Write, Bash, Glob, Grep
skills: php, laravel, postgresql, livewire, tailwind, frontend-design, stripe, docker, scoping-feature-work, prioritizing-roadmap-bets, mapping-user-journeys, designing-onboarding-paths, improving-activation-flow, crafting-empty-states, orchestrating-feature-adoption, designing-inapp-guidance, instrumenting-product-metrics, running-product-experiments, triaging-user-feedback, writing-release-notes, clarifying-market-fit, structuring-offer-ladders, framing-release-stories, generating-growth-hypotheses, embedding-decision-cues, crafting-page-messaging, tightening-brand-voice, designing-lifecycle-messages, planning-editorial-arcs, orchestrating-social-rhythm, tuning-landing-journeys, streamlining-signup-steps, accelerating-first-run, reducing-form-falloff, refining-prompt-surfaces, strengthening-upgrade-moments, mapping-conversion-events, designing-variation-tests, calibrating-paid-campaigns, building-acquisition-tools, engineering-referral-loops, inspecting-search-coverage, scaling-template-pages, adding-structured-signals, building-compare-hubs
name: devops-engineer
model: inherit
description: |
---

# Run queue worker (for job processing)
php artisan queue:work

# Or use Laravel Horizon if configured
```

**Scheduled tasks** (require `schedule:run` via cron or `schedule:work` in dev):
- `ghl:refresh-tokens` — hourly GHL OAuth token refresh
- `bridge-listings-cache-refresh` — every 15 minutes per active domain
- `gis-source-metadata-probe` — weekly Monday 06:30 ArcGIS layer fingerprinting

## Environment Variables

**Critical for Docker:**
| Variable | Service | Purpose |
|----------|---------|---------|
| `APP_KEY` | idx-api | Laravel encryption key |
| `APP_URL` | idx-api | Base URL for routes |
| `DB_CONNECTION` | idx-api | `pgsql` (prod) / `sqlite` (dev) |
| `DB_HOST` | idx-api | Database host |
| `DB_DATABASE` | idx-api | Database name |
| `DB_USERNAME` | idx-api | Database user |
| `DB_PASSWORD` | idx-api | Database password |
| `IMAGE_CACHE_PATH` | idx-api | `/var/cache/geoidx/images` |
| `IDX_IMAGES_PUBLIC_URL` | idx-api | Public image CDN URL |
| `BRIDGE_API_KEY` | idx-api | Bridge MLS API key |
| `GHL_CLIENT_ID` | idx-api | GHL OAuth client ID |
| `GHL_CLIENT_SECRET` | idx-api | GHL OAuth secret |
| `STRIPE_WEBHOOK_SECRET` | idx-api | Stripe webhook signing |
| `CLOUDFLARED_TOKEN` | cloudflared | Cloudflare tunnel auth |

## Security & Compliance

- **Never commit secrets**: `.env` contains all sensitive values
- **Build context**: Always build from project root: `docker build -f Dockerfile.idx-api -t quantyra/idx-api:latest .`
- **Least privilege**: idx-images service only needs to proxy `/images/*`
- **Image cache**: `IMAGE_CACHE_PATH` must be writable by `www-data` (uid 82 in Alpine)
- **Audit logs**: `storage/logs/ghl_audit.log` and `storage/logs/` should be on persistent volume

## Deployment Commands

```bash
# Build production images
docker build -f Dockerfile.idx-api -t quantyra/idx-api:latest .
docker build -f Dockerfile.idx-images -t quantyra/idx-images:latest .

# Dev environment
./scripts/docker-dev.sh up-watch   # Start with file watching
./scripts/docker-dev.sh down       # Stop all services

# Stripe webhook forwarding (dev)
./scripts/stripe-dev.sh listen
./scripts/stripe-dev.sh trigger-checkout-completed

# Queue worker (required in production)
docker compose exec idx-api php artisan queue:work --sleep=3 --tries=3

# Scheduler (for cron jobs)
docker compose exec idx-api php artisan schedule:work
```

## Database Operations

```bash
# Run migrations
docker compose exec idx-api php artisan migrate

# Seed GHL configuration
docker compose exec idx-api php artisan db:seed --class=GhlConfigSeeder

# GIS cache management
docker compose exec idx-api php artisan gis:clear-cache --all
docker compose exec idx-api php artisan gis:probe-sources
```

## Troubleshooting

| Symptom | Check |
|---------|-------|
| 502 on `/images/*` | idx-images → idx-api connectivity, nginx config syntax |
| Queue jobs not processing | Worker running, `QUEUE_CONNECTION` set correctly |
| Scheduled tasks not firing | `schedule:run` or `schedule:work` active |
| Stripe webhooks failing | `STRIPE_WEBHOOK_SECRET` matches CLI/dashboard endpoint |
| Cache permission denied | `www-data` owns `storage/` and `IMAGE_CACHE_PATH` |
| Cloudflare tunnel down | `CLOUDFLARED_TOKEN` validity, tunnel status in dashboard |

## CRITICAL for This Project

1. **Build context is repository root** — Dockerfiles reference `idx-api/` subdirectory
2. **idx-images is NOT standalone** — it proxies to idx-api:8000 for auth enforcement
3. **Queue workers required** — Without them, GHL token refresh, lead sync, and cache jobs won't run
4. **Cloudflare tunnel for HTTPS** — Dev environment needs valid `CLOUDFLARED_TOKEN` for GHL OAuth callbacks
5. **PostgreSQL in production** — SQLite only for local dev and tests; migrations tested on both
6. **Image cache TTL separate from CDN** — `IMAGE_CACHE_TTL` (origin refresh) vs `Cache-Control` headers (CDN)
7. **Environment files** — `.env` at repo root for Docker Compose; `idx-api/.env` for Laravel directly
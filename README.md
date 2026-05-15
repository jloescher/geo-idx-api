# Quantyra IDX API

Laravel 13 + Octane service for Quantyra's Bridge MLS proxy and secured image proxy delivery.

## What This Project Contains

- Bridge Data Output proxy APIs under `/api/v1/*`
- **Multi-MLS catalog:** internal feed keys are `bridge_{dataset}` (e.g. `bridge_stellar`) for cache rows, queues, and `mls.feed_code`. Public API clients still pass **`?dataset=`** using the **unprefixed** Bridge dataset slug (`stellar`, `miami`, …) as configured in `BRIDGE_DATASET` / `BRIDGE_DATASETS`.
- Domain/token authorization middleware and proxy audit logging
- Image proxy flow (`/images/*`) with cache path support and Nginx edge support
- Listings row cache (`listings_cache`) with `php artisan mls:refresh-cache` (scheduled every 15 minutes; requires `QUEUE_CONNECTION=database` and a queue worker)
- Supporting migrations, seeders, and automated tests
- **Invite-only dashboard signup:** administrators send email invitations; `ADMIN_SEED_*` env vars seed a bootstrap admin (see `.env.example`)

## Octane (FrankenPHP)

Production and local high-concurrency mode:

```bash
php artisan octane:install --server=frankenphp
php artisan octane:start --server=frankenphp
```

Run a queue worker in parallel (`php artisan queue:work`) so MLS listings cache refresh jobs execute.

## Project Structure

```text
idx-api/
├── app/
├── bootstrap/
├── config/
├── database/
├── docs/
├── public/
├── resources/
├── routes/
├── tests/
├── Dockerfile.production   # Production API (Coolify targets: octane, queue-worker, scheduler; aliases idx-api-worker, idx-api-scheduler)
├── Dockerfile.staging      # Staging API (Xdebug); same targets as production
├── Dockerfile.idx-images   # Nginx edge for /images/* (staging + production; see file header)
├── nginx.idx-images.conf
└── README.md
```

## Local Setup

PostgreSQL is used for local development, staging (shared), and production (dedicated). Configure `DB_*` in `.env` (see `.env.example`).

```bash
cp .env.example .env
composer install
php artisan key:generate
php artisan migrate
php artisan serve
```

If you are using frontend assets locally:

```bash
npm install
npm run build
```

## Docker Images

Build from this project root. **Production API:** [`Dockerfile.production`](Dockerfile.production). **Staging API:** [`Dockerfile.staging`](Dockerfile.staging). **Image edge (all envs):** [`Dockerfile.idx-images`](Dockerfile.idx-images).

```bash
docker build -f Dockerfile.production --target octane -t quantyra/idx-api:latest .
docker build -f Dockerfile.production --target queue-worker -t quantyra/idx-api-worker:latest .
docker build -f Dockerfile.production --target scheduler -t quantyra/idx-api-scheduler:latest .
docker build -f Dockerfile.idx-images -t quantyra/idx-images:latest .
```

**Coolify:** use the Dockerfile build pack, set **port 8000**, and deploy **three** API services (build targets `octane`, `queue-worker`, `scheduler`) with `QUEUE_CONNECTION=database` and shared `DB_*`. Add **idx-images** on port **8080**. Full steps: **[docs/coolify-deployment.md](docs/coolify-deployment.md)**. See [`AGENTS.md`](AGENTS.md) (Docker Deployment) for vCPU/RAM and PHP memory guidance. The production image does not bake **`route:cache`** when multiple platform hosts reuse the same route names; use post-deploy `php artisan route:cache` only if your environment uses a single host or unique names (see [deployment-operations.md](docs/deployment-operations.md)).

## Docs

Start at `docs/INDEX.md` for implementation and operations guides. **Coolify (production & staging):** [`docs/coolify-deployment.md`](docs/coolify-deployment.md).

## Testing

Tests use **PostgreSQL** (PostGIS-backed features are not exercised on SQLite). Database connection settings come from **`.env`** (loaded by [`tests/bootstrap.php`](tests/bootstrap.php) before PHPUnit merges other variables). `phpunit.xml` no longer hardcodes `DB_*`.

- **Recommended:** create a local empty database named **`idx_api_testing`** (or `testing`) and point `DB_*` at it for `php artisan test`. `tests/TestCase.php` allows these names without extra flags.
- **Staging / shared Postgres:** point `DB_*` at your instance only if you accept the risk from **`RefreshDatabase`** in many feature tests (migrations rerun; data loss). Set **`ALLOW_DESTRUCTIVE_TEST_DB=true`** in the environment (or `.env`) when `DB_DATABASE` is not `testing` or `idx_api_testing`.

```bash
createdb idx_api_testing
# Set DB_DATABASE=idx_api_testing (and host/user/password) in .env, then:
php artisan test --compact
```

For a non-whitelisted database name (e.g. staging):

```bash
ALLOW_DESTRUCTIVE_TEST_DB=true php artisan test --compact
```

## License

Proprietary - Quantyra Labs LLC

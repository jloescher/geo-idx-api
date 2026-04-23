# Quantyra IDX API

Laravel 13 + Octane service for Quantyra's Bridge MLS proxy, GHL Marketplace integration, lead ingestion, and secured image proxy delivery.

## What This Project Contains

- Bridge Data Output proxy APIs under `/api/v1/*`
- Domain/token authorization middleware and proxy audit logging
- GoHighLevel OAuth install flow, token refresh, webhooks, widgets, and sync jobs
- Image proxy flow (`/images/*`) with cache path support and Nginx edge support
- Supporting migrations, seeders, and automated tests

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
├── Dockerfile.idx-api
├── Dockerfile.idx-images
├── nginx.idx-images.conf
└── README.md
```

## Local Setup

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

Build images from this project root:

```bash
docker build -f Dockerfile.idx-api -t quantyra/idx-api:latest .
docker build -f Dockerfile.idx-images -t quantyra/idx-images:latest .
```

## Docs

Start at `docs/INDEX.md` for implementation and operations guides.

## Testing

```bash
php artisan test --compact
```

## License

Proprietary - Quantyra Labs LLC

# Quantyra GeoIDX

**Quantyra Labs Full-Stack IDX Platform for Stellar MLS**

A high-conversion, lead-generation SaaS platform powering 58 hyper-local real estate domains in secondary and tertiary markets (Pinellas, Tampa Bay, Clearwater, St. Pete focus). Live MLS data via BridgeDataOutput API, AI-powered programmatic SEO, aggressive lead gating, and interactive mapping.

## Product Overview

- **Public Experience**: Instant teaser value → hard gate after 3 listings → OTP registration → full access + lead forms
- **Agent/Lender**: Paywall triggers after teaser lead stats
- **Revenue Focus**: 85%+ margins, maximize lead capture and subscription conversion

## Stellar MLS Compliance

This project operates under the signed Stellar MLS Agreement (effective 4/17/2026):
- **Consultant**: Quantyra Labs LLC
- **Firm**: Realty Of America, LLC
- **IDX**: Approved
- **Approved Domain**: searchtampabayhouses.com (Exhibit A)
- **Mobile App**: Demo/teaser mode only until Exhibit A amendment

All MLS calls are proxied through `idx-api` with audit logging. Domain restrictions and caching rules are strictly enforced.

## Monorepo Structure

```
quantyra-geoidx/
├── docs/                  # All reference docs — start at docs/INDEX.md
├── docker/                # Production Dockerfiles (root build context) — see docker/README.md
├── idx-api/               # Laravel 13 + Octane + Swoole (MLS proxy + image proxy)
├── geo-web/               # Laravel 13 + Octane + Swoole (multi-domain frontend)
├── mobile/                # Flutter/Dart native mobile app
├── scripts/               # Build/deploy scripts
├── docker-compose.yml     # Dokploy-compatible compose
├── .env.example           # Environment template
└── README.md
```

## Documentation

Canonical documentation lives under **`docs/`**. Start at **[docs/INDEX.md](docs/INDEX.md)** for the full table of contents (Bridge API, GoHighLevel Marketplace integration, deployment, environment variables, database schema, and internal design links).

## Tech Stack

### idx-api & geo-web
- Laravel 13 + Octane + Swoole
- Livewire 3 + Blade + Alpine.js
- PostgreSQL (Patroni)
- NVMe filesystem cache
- Laravel Cashier + Stripe
- Sanctum (API auth)
- Queued AI jobs (Groq/Claude/OpenAI)

### mobile
- Flutter 3.24+
- Riverpod 2.0+ (state management)
- flutter_map + OpenStreetMap tiles
- Hive/Isar (local cache)
- flutter_stripe

### Shared Infrastructure
- PostgreSQL: listings_cache, seo_content, domains table
- Image proxy: idx-images.quantyralabs.cc

## Quick Start

### Prerequisites
- PHP 8.5+
- Composer 2.9+
- Node.js 20+
- Flutter 3.24+
- Docker + Dokploy

### Local Development

```bash
# Clone repository
git clone https://github.com/quantyralabs/quantyra-geoidx.git
cd quantyra-geoidx

# Copy environment file
cp .env.example .env

# Start services with Docker
docker-compose up -d

# Install idx-api dependencies
cd idx-api && composer install

# Install geo-web dependencies
cd geo-web && composer install && npm install && npm run build

# Run migrations
php artisan migrate --path=database/migrations/domains

# Seed domains (58 domains)
php artisan db:seed --class=DomainsSeeder
```

### Deployment (Dokploy)

1. Configure environment variables in Dokploy dashboard
2. Deploy services:
   - `quantyra-idx-api` → idx-api.quantyralabs.cc
   - `quantyra-geo-web` → Multi-domain routing
   - `quantyra-idx-images` → idx-images.quantyralabs.cc
3. Configure Traefik routing for all 58 domains

## API Endpoints

### idx-api (Stellar MLS Proxy)

All endpoints use `stellar` dataset:

```
GET  /api/v1/stellar/listings
GET  /api/v1/stellar/listings/{listingId}
GET  /api/v1/stellar/listings/search
GET  /api/v1/stellar/listings/nearby
GET  /api/v1/stellar/properties
GET  /api/v1/stellar/agents
GET  /api/v1/stellar/offices
POST /api/v1/leads
POST /api/v1/auth/request-otp
POST /api/v1/auth/verify-otp
```

### Image Proxy

```
GET https://idx-images.quantyralabs.cc/proxy?url=<encoded_url>
```

## Domain Configuration

58 hyper-local domains for Pinellas/Tampa Bay markets:

**Pilot (8 domains)**:
1. searchtampabayhouses.com (Approved in Exhibit A)
2. clearwaterhouses.com
3. stpetehouses.com
4. pinellashouses.com
5. tampabayhouses.com
6. searchclearwaterhouses.com
7. searchstpetehouses.com
8. searchpinellashouses.com

**Full list**: See `database/seeders/DomainsSeeder.php`

## Lead Capture Flow

```
Visitor → Homepage → Map/Listings → View 3 listings → Hard Gate → OTP → Full Access → Lead Forms
```

- **Gate Limit**: 3 listings before registration required
- **OTP**: TextGrid phone verification
- **Subscription**: Stripe for Agent/Lender paywall

## Revenue Impact

Every component is designed for maximum lead capture:
- Instant teaser value on entry
- Hard gate after 3 interactions
- Phone-first registration (high conversion)
- Lead forms on every listing
- Agent/Lender subscription paywall

## Security & Compliance

- All Bridge API credentials server-side only
- MLS audit logging enabled
- Domain whitelist enforcement
- Image proxy for MLS photos
- Mobile app demo mode until Exhibit A amendment

## License

Proprietary - Quantyra Labs LLC

## Contact

Quantyra Labs LLC - https://quantyralabs.cc
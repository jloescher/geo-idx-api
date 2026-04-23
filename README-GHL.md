# GoHighLevel (GHL) — idx-api

**All operator and technical documentation for the GHL Marketplace integration is in the monorepo `docs/` folder.**

Start here: **[../docs/INDEX.md](../docs/INDEX.md)**

Quick links:

- [GHL marketplace integration](../docs/ghl-marketplace-integration.md)
- [API & routes reference](../docs/ghl-api-routes-reference.md)
- [Deployment & operations](../docs/ghl-deployment-and-operations.md)
- [Environment variables](../docs/ghl-environment-variables.md)
- [Database schema](../docs/ghl-database-schema.md)

Local one-liner after clone:

```bash
cd idx-api && cp .env.example .env && php artisan key:generate && php artisan migrate && php artisan db:seed --class=GhlConfigSeeder
```

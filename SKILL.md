# Quantyra GeoIDX — workspace skill index

This file lists **in-repo** Cursor skills under [`.cursor/skills/`](.cursor/skills/). Growth-only marketing skills were removed; use the entries below for this codebase.

## Invoke when

| Area | Skill folder |
|------|----------------|
| Laravel routing, ORM, queues, providers | `laravel` |
| Laravel conventions / security / testing rules | `laravel-best-practices` |
| PostgreSQL migrations and queries | `postgresql` |
| PHP 8.5 style | `php` |
| Docker / FrankenPHP / Compose | `docker` |
| Nginx (idx-images) | `nginx` |
| Vite + Tailwind build | `vite`, `tailwind` |
| Livewire / Volt / Tailwind-class-heavy UI | `livewire`, `volt-development`, `livewire-development`, `tailwindcss-development` |
| Dashboard / Blade UX polish | `frontend-design` |
| Bridge MLS + GIS search surfaces (filters, SEO) | `inspecting-search-coverage` |
| Empty states and onboarding affordances | `crafting-empty-states` |
| Tooltips, tours, in-app hints | `designing-inapp-guidance` |
| Fortify auth | `fortify-development` |
| Pulse | `pulse-development` |

## Product surfaces (current)

- **Marketing home**: [`resources/views/marketing/home.blade.php`](resources/views/marketing/home.blade.php) via [`app/Http/Controllers/Marketing/SalesPageController.php`](app/Http/Controllers/Marketing/SalesPageController.php) (`marketing.sales`).
- **User dashboard**: [`app/Http/Controllers/DashboardController.php`](app/Http/Controllers/DashboardController.php), Filament [`app/Filament/Pages/UserDashboard.php`](app/Filament/Pages/UserDashboard.php) — domains, DNS verification, MLS datasets per domain, Sanctum API tokens.
- **MLS / GIS API**: [`routes/api.php`](routes/api.php) — Bridge proxy and GIS; auth via [`app/Http/Middleware/DomainOrTokenAuth.php`](app/Http/Middleware/DomainOrTokenAuth.php) and [`app/Http/Middleware/CheckMlsAccess.php`](app/Http/Middleware/CheckMlsAccess.php).

See [AGENTS.md](AGENTS.md) for full architecture and env vars.

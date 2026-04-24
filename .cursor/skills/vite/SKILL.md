---
name: vite
description: Configures Vite 8 build pipeline with Tailwind plugin and HMR
allowed-tools: [Read, Edit, Write, Glob, Grep, Bash]
---

```

# Vite Skill

Configures and maintains the Vite 8 build pipeline for Laravel 13 + Livewire applications, integrating Tailwind CSS 4 with the official Vite plugin and HMR support for Cloudflare tunnel development.

## Quick Start

```bash
npm install
npm run build
```

Development with HMR:
```bash
npm run dev
```

Or use the composer meta-command for full stack:
```bash
composer dev  # Starts server + queue + logs + Vite in parallel
```

## Key Concepts

- **Vite 8**: Modern build tool with esbuild-powered dev server and Rollup production builds
- **@tailwindcss/vite**: Official Tailwind 4 plugin for Vite with zero-config CSS processing
- **laravel-vite-plugin**: Laravel integration providing entry point resolution and manifest handling
- **HMR**: Hot Module Replacement preserves state during development; configured for Cloudflare tunnel hosts

## Common Patterns

**Vite Configuration** — `vite.config.js` at project root:
- Uses `laravel-vite-plugin` with entry points `resources/css/app.css` and `resources/js/app.js`
- Includes `@tailwindcss/vite` plugin for Tailwind 4 integration
- HMR server configuration allows external hosts (Cloudflare tunnels)

**Tailwind Entry** — `resources/css/app.css`:
- Uses `@import "tailwindcss"` for Tailwind 4 (no tailwind.config.js required)
- Contains custom layer directives and component styles

**Blade Integration** — Use `@vite()` directive in templates:
- `@vite(['resources/css/app.css', 'resources/js/app.js'])` in layout head

**Build Commands**:
- `npm run dev` — Development server with HMR
- `npm run build` — Production build with asset manifest generation

**Build Output** — Assets compiled to `public/build/` with hashed filenames for cache-busting.
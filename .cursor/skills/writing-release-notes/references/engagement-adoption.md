# Engagement & Adoption Reference

How release notes drive adoption of features across the Quantyra IDX API surface.

## Feature Adoption Through Release Notes

Quantyra ships features across MLS proxy, search, GIS, comps, and dashboard. Release notes are the primary channel for driving adoption of new capabilities.

### Feature-to-Audience Mapping

| Feature | Route | Primary Audience | Adoption Signal |
|---------|-------|-----------------|-----------------|
| Hybrid search | `POST /api/v1/search` | API consumers | Requests hitting PostGIS mirror vs live Bridge |
| Comps / BPO | `POST /api/v1/comps/run` | API consumers | Mode distribution across A-E, investor, bpo, home_value |
| GIS parcels | `GET /api/v1/gis` | API consumers | GeoJSON requests per source |
| Pricing enrichment | `GET /api/v1/listings` | API consumers | Listings responses with `pricing` object |
| Dashboard tokens | `/dashboard` | Dashboard users | Token creation events |
| Multi-DC deploy | `Dockerfile` targets | operators | Scheduler leader lock acquisition logs |

### DO: Highlight capability expansion with concrete API changes

```markdown
## v2.3.0

### Features
- **Comps**: Investor mode `rent_hold_cashflow` now includes renovation credit derivation from market data (no longer static presets)
- **Search**: `low_risk_floodzone` and `min_monthly_fees`/`max_monthly_fees` filters now query indexed columns on the PostGIS mirror for faster response
```

### DON'T: Vaguely describe improvements without API impact

```markdown
<!-- BAD — no actionable information -->
### Features
- Improved comps analysis
- Better search performance
```

### Adoption-Driving Patterns

When writing notes for features that need adoption push:

1. **Show the endpoint** — always include the HTTP method and path
2. **Show the new field or param** — what changed in request/response
3. **Show the benefit** — faster, more accurate, new capability
4. **Link to docs** — `docs/comps-api.md`, `docs/gis-api.md`, etc.

### Release Notes for Operator Adoption

Operator-facing changes (queue config, env vars, deployment) should specify:

- Exact env var name and default (e.g., `SCHEDULER_LEADER_LOCK_ID=913374211`)
- Dockerfile target (e.g., `scheduler` target)
- Migration requirement (`goose -dir migrations up`)
- See the **deploy-coolify** skill for deployment note templates

### Engagement Note Checklist

Copy this checklist when writing adoption-focused release notes:
- [ ] Every feature entry includes the HTTP method + path
- [ ] New params or response fields are named explicitly
- [ ] Breaking changes are in their own section at the top
- [ ] Operator changes specify env vars, migrations, or queue changes
- [ ] Links to relevant `docs/*.md` files

## Related References

- See the **geospatial** skill for GIS feature language
- See the **queue-postgresql** skill for operator-facing queue changes
- See `docs/api.md` for the full API surface
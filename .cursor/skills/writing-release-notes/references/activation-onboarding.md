# Activation & Onboarding Reference

How release notes connect to the activation journey for Quantyra IDX API customers.

## Activation Flow and Release Notes

Customers activate by verifying a domain (TXT record) and receiving an auto-issued `idx:full` PAT. Release notes targeting new users should reference these surfaces.

### Product Surfaces for Activation Notes

| Surface | Route / Location | Relevance to New Users |
|---------|------------------|----------------------|
| Domain verification | Dashboard setup panel | First step after signup |
| Auto-issued PAT | `POST /api/auth/token` | Required for all `/api/v1` calls |
| Staging key | Dashboard API Keys panel | One-click staging access |
| `/api/v1/search` | `POST /api/v1/search` | First API integration point |
| `/api/v1/gis` | `GET /api/v1/gis` | Map overlay setup |

### DO: Frame notes around user milestones

```markdown
## v2.1.0

### For New Integrations
- **Search**: `POST /api/v1/search` now returns `pricing` enrichment on every listing — no extra request needed
- **GIS**: Parcel overlay supports county-level bounding box without dataset param
```

### DON'T: Lead with internal architecture

```markdown
<!-- BAD — new users don't care about replication internals -->
### For New Integrations
- Replication pipeline now uses fair reservation across `bridge-sync-fetch` and `spark-sync-fetch` queues
- `replica_pages` gzip staging improved chunk persist throughput
```

New users need to know what endpoints changed and how to use them, not how the sausage is made. Queue names and staging tables are operator concerns.

### When Onboarding Changes Ship

If a release modifies activation flow (domain verification, token issuance, dashboard setup):

1. Note the change under a **Breaking Changes** or **Dashboard** section
2. Link to updated `docs/idx-api-bridge-proxy.md` dashboard API keys section
3. Flag if existing users need to re-issue tokens (see `docs/go-cutover.md`)

### Activation-Focused Release Checklist

Copy this checklist when a release touches onboarding:
- [ ] Does this change how domains are verified?
- [ ] Does this change the token format or scopes (`idx:access`, `idx:full`)?
- [ ] Does this add or remove dashboard panels?
- [ ] Does this change the `/api/v1/search` request or response shape?
- [ ] Is the staging key flow affected?
- [ ] Update `docs/api.md` auth section if token behavior changed

## Related References

- See the **auth-api-token** skill for token scope changes
- See `docs/go-cutover.md` for the Laravel migration context (SHA-256 token hashes)
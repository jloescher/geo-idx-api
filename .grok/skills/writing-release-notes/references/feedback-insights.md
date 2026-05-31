# Feedback & Insights Reference

How release notes close the loop on customer feedback and operational insights for Quantyra IDX API.

## Closing Feedback Loops in Release Notes

Release notes are the primary mechanism for communicating that reported issues have been addressed. Quantyra tracks API access via `audit_logs` and receives operator feedback through log patterns.

### DO: Attribute fixes to user-facing symptoms

```markdown
## v2.1.1

### Fixes
- **Bridge**: Stellar listings now correctly hydrate `Rooms`, `UnitTypes`, and `OpenHouses` after replication — previously these expanded collections were empty on mirror-backed search results
- **Spark**: Association fees for Beaches listings are now normalized to monthly at persist — `estimated_total_monthly_fees` was previously incorrect for annual/quarterly fee schedules
```

### DON'T: Describe fixes in terms of internal code

```markdown
<!-- BAD — what symptom did the user see? -->
### Fixes
- Fixed `BuildListingRecord` to map Bridge upstream keys `Rooms` → `room` JSONB column
- Fixed `listing_payload.go` normalize function for association fee frequency
```

### Feedback Signals That Should Produce Release Notes

| Signal | Source | Example Fix to Document |
|--------|--------|------------------------|
| Empty expanded collections | API consumer reports | Rooms/UnitTypes missing after replication |
| Incorrect derived fields | `audit_logs` query patterns | Monthly fees calculation wrong |
| Replication stalls | Worker logs, `replica_pages` state | Kickoff skipping when pages remain |
| Cache staleness | `GET /api/v1/bridge/stats` | `replication_in_progress` stuck true |
| Auth failures | Dashboard support tickets | Legacy Sanctum tokens rejected |

### Operator Feedback in Release Notes

Operators provide feedback through deployment friction. Document fixes that reduce operational burden:

- New env vars that replace hardcoded values
- Queue changes that reduce manual intervention
- Health endpoint improvements (`/healthz`, `/readyz`)
- Scheduler leader lock behavior changes

See the **deploy-coolify** skill and **queue-postgresql** skill for operator-facing patterns.

### Breaking Changes and Migration Notes

Breaking changes are the highest-stakes feedback to communicate. Always include:

1. What changed (API shape, auth, env vars)
2. Who is affected (API consumers, operators, dashboard users)
3. Required action (re-issue tokens, update env, run migration)
4. Deadline or version where old behavior stops

```markdown
### Breaking Changes
- **Auth**: API tokens now use SHA-256 hashes. Legacy `id|secret` Sanctum tokens are rejected. Re-issue all keys from `/dashboard` API Keys panel. See `docs/go-cutover.md`.
```

### WARNING: Undocumented Fixes

A fix that ships without a release note is a fix that will be reported again. Every merged fix that touches user-facing behavior (API response shape, error messages, auth flow) must appear in release notes — even if "minor."

### Feedback Release Checklist

Copy this checklist when closing feedback loops:
- [ ] Fix describes the user-visible symptom, not the code change
- [ ] Affected endpoint or surface is named
- [ ] Breaking changes include required action and deadline
- [ ] Operator fixes specify env vars or migration steps
- [ ] No fix that changes API behavior is undocumented

## Related References

- See the **auth-api-token** skill for auth change documentation
- See `docs/go-cutover.md` for the canonical breaking change example (Laravel→Go token migration)
- See `docs/deployment-operations.md` for operational troubleshooting context
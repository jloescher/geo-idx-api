# Growth Engineering Reference

Growth loops and advocacy mechanisms for Quantyra IDX API's invite-only platform.

## Contents
- Growth Model
- Advocacy Surfaces
- Onboarding as Growth
- Anti-Patterns

## Growth Model

Quantyra IDX API is **invite-only**. Growth is driven by:

1. **Admin invitation** (`internal/service/auth/invitation.go`) — the only signup path
2. **Domain verification** — each new domain expands API usage scope
3. **Token creation** — production tokens unlock real API calls

### Growth metrics available

| Metric | Query | Growth signal |
|--------|-------|---------------|
| New users | `SELECT COUNT(*) FROM users WHERE created_at > NOW() - interval '7 days'` | Invitation conversion |
| Domains added | `SELECT COUNT(*) FROM domains WHERE created_at > NOW() - interval '7 days'` | Integration breadth |
| Verified domains | `SELECT COUNT(*) FROM domains WHERE verification_status = 'verified'` | Activation |
| Active tokens | `SELECT COUNT(*) FROM personal_access_tokens WHERE last_used_at > NOW() - interval '7 days'` | Retention |
| Staging token usage | Filter by `name = 'Staging'` and `last_used_at` | Trial before production |

## Advocacy Surfaces

### WARNING: No referral or sharing mechanism

**The Problem:** The invitation system is admin-only. There is no way for satisfied users to invite peers, share API examples, or refer new domains.

**Why This Breaks:** Invite-only without advocacy loops means growth depends entirely on admin outreach. Each new user requires manual effort.

**The Fix:** For release stories, frame features as shareable wins: "Show your team the new polygon search" with a pre-built curl command. Add `utm_source` params to docs links when shared externally.

### Existing shareable artifacts

| Artifact | How to share | Release story use |
|----------|-------------|-------------------|
| OpenAPI spec | `GET /openapi.json` or `GET /swagger` | "Import our updated spec" |
| API docs | `docs/*.md` | Link in release announcements |
| curl examples | Copy-paste from docs | Include in every feature story |
| Dashboard invite link | `/invite/:token` (admin-generated) | Pre-seed new users for major releases |

## Onboarding as Growth

The onboarding funnel is the primary growth lever:

```
Invite → Register → Login → Add domain → Verify DNS → Get token → First API call
```

Each step in `internal/handler/dashboard/handler.go` is an opportunity to communicate value:

| Step | Current copy | Growth-optimized copy |
|------|-------------|----------------------|
| Hero | "MLS proxy, image delivery, and developer setup" | "One API for every MLS feed" |
| Domain add | "Add domain" form | "Connect your first site" |
| Token reveal | "Save this token now" | "Your key to production data" |

## Anti-Patterns

### WARNING: Releasing features without updating onboarding

**The Problem:** Shipping GIS parcel proxy (`/api/v1/gis`) or comps BPO (`/api/v1/comps/run`) without reflecting them in the dashboard or onboarding.

**Why This Breaks:** New users onboarding after a release never discover features that aren't surfaced in their first session. Feature adoption stalls.

**The Fix:** For each feature release, check if the dashboard needs a new card, link, or setup step. Add feature discovery to the release checklist.

### WARNING: Treating staging tokens as a growth dead-end

**The Problem:** Staging tokens are created via a separate button and return plain text (`"Staging token: ..."`). No guidance on what to do next.

**Why This Breaks:** Users create staging tokens but never progress to domain verification and production tokens. The staging → production conversion is invisible.

**The Fix:** After staging token creation, show a "Next step: Add your domain" prompt with a direct link to the domain setup form.

See the **auth-api-token** skill for token lifecycle patterns.
See the **writing-release-notes** skill for feature announcement integration.
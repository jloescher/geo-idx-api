# Conversion Optimization Reference

Optimize conversion paths for Quantyra IDX API — from landing page visit to first API call.

## Contents
- Conversion Surfaces
- Dashboard Onboarding Funnel
- Anti-Patterns

## Conversion Surfaces

| Surface | Location | Conversion goal |
|---------|----------|-----------------|
| Marketing home | `internal/handler/marketing/handler.go` | Click "Open dashboard" |
| Login form | `internal/handler/dashboard/handler.go` LoginForm | Successful sign-in |
| Domain setup | Dashboard → "Add domain" | TXT verification passed |
| Token creation | Dashboard → "Create token" | First API call with token |
| Domain verified | `VerifyTXT` handler | Save production token |

## Dashboard Onboarding Funnel

The critical path in this invite-only product:

1. User receives invite link (`/invite/:token`)
2. Creates account (`AcceptInvitation`)
3. Logs in (`/login`)
4. Adds domain (`StoreDomain`)
5. Verifies DNS TXT (`VerifyTXT`)
6. Receives production token (shown once)

### WARNING: Token shown once is a conversion cliff

**The Problem:** In `VerifyTXT`, the production token is rendered in HTML one time. If the user navigates away without copying it, there is no recovery path — they must revoke and create a new token.

**Why This Breaks:** Users verifying DNS on mobile or in a hurry frequently lose tokens. No email delivery, no copy confirmation. This creates support burden and frustration at the moment of highest engagement.

**The Fix:** Add a "Token saved" confirmation step before redirecting back to dashboard. Consider optional email delivery of token (one-time, encrypted).

### WARNING: No onboarding progress indicator

**The Problem:** The dashboard renders all cards simultaneously with no indication of what to do next. New users see "Setup", "API keys", "Add domain", and "Invite user" with equal visual weight.

**Why This Breaks:** Users skip domain verification and create staging tokens instead, leading to API calls that fail domain auth checks.

**The Fix:** Sequence the dashboard cards by completion state. Hide "API keys" until at least one domain is verified. Show a progress indicator: "Step 1: Add domain → Step 2: Verify DNS → Step 3: Get your token".

## Anti-Patterns

### WARNING: Marketing page with no value proposition

The marketing handler (`internal/handler/marketing/handler.go`) renders a single hero section:

```go
// Current — generic, no differentiation
body := `<section class="hero">
<h1>Quantyra IDX</h1>
<p>MLS proxy, image delivery, and developer setup for your IDX sites.</p>
```

This describes what the product does, not why a developer should choose it over direct MLS API access. The value prop should lead with: "One API for multiple MLS feeds. No RESO OData integration needed. PostGIS-backed search with sub-second responses."

## DO/DON'T for release-driven conversion

| DO | DON'T |
|----|-------|
| Link release announcements to docs with working code examples | Bury new features in a changelog no one reads |
| Frame breaking changes as upgrade paths with deadlines | Drop auth changes without migration instructions |
| Show `Before → After` for API response changes | List internal refactor commits as "improvements" |
| Use `/dashboard` messaging for token re-issuance | Assume users read commit messages |

## Integration with release stories

When framing a release story that affects conversion:

1. Check `internal/handler/marketing/handler.go` for hero copy that needs updating
2. Check `internal/handler/dashboard/handler.go` for dashboard flow changes
3. Map new API features to endpoint docs in `docs/`
4. Verify `IDX_API_PUBLIC_URL` is correct for code examples in announcements

See the **writing-release-notes** skill for the commit-to-changelog pipeline.
See the **auth-api-token** skill for auth change messaging.
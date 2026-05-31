# Journey Map Reference

## Contents
- Dashboard Authentication Journey
- API Token Creation Journey
- MLS Proxy Request Journey
- GIS Teaser Journey
- Journey Anti-Patterns

## Dashboard Authentication Journey

Users authenticate to `/dashboard` via admin credentials seeded with `make seed-admin`.

| Step | User intent | Precondition | Success state | Failure state |
|------|------------|--------------|---------------|---------------|
| 1 | Access dashboard | Know URL | Login form rendered | 404 if route missing |
| 2 | Submit credentials | Email + password in `.env` | Session set, redirect to dashboard | "Invalid credentials" message |
| 3 | Manage domains | Authenticated session | Domain list with CRUD | Session expired → back to login |
| 4 | Manage tokens | Domain selected | Token list with scopes | No domain → empty state |

## API Token Creation Journey

| Step | User intent | Precondition | Success state | Failure state |
|------|------------|--------------|---------------|---------------|
| 1 | Create token | Domain exists, user on tokens page | Form with scope checkboxes | No domains → prompt to create one first |
| 2 | Submit | Scope selected, name filled | Token shown ONCE | Validation error per field |
| 3 | Copy token | Token visible | Copied to clipboard | Token never shown again after nav |

### WARNING: Premature Success Messaging

**The Problem:**

```html
<!-- BAD — token shown before server confirms -->
<div class="success">Your token has been created!</div>
```

**Why This Breaks:** If the server returns 409 (duplicate name) or 500, the user saw a success message that was a lie. Trust is broken and they may navigate away thinking the token exists.

**The Fix:**

```html
<!-- GOOD — show success only after response -->
<div data-state="pending" hidden>Creating token…</div>
<div data-state="success" hidden>Token created. Copy it now — it won't be shown again.</div>
<div data-state="error" hidden></div>
```

**When You Might Be Tempted:** Optimistic UI feels faster. For read operations this is acceptable; for mutations (create, delete, update) it is not.

## MLS Proxy Request Journey

API consumers (not dashboard users) make requests through the proxy.

| Step | Consumer intent | Precondition | Success state | Failure state |
|------|----------------|--------------|---------------|---------------|
| 1 | Search listings | Valid domain + token in header | RESO JSON response | 401/403 with error code |
| 2 | Page through results | `@odata.nextLink` present | Next page of results | Cursor expired → restart |
| 3 | Get images | Listing key known | Image URLs via `/images/*` | 404 if listing removed |

See the **auth-api-token** and **auth-domain** skills for auth implementation details.

## GIS Teaser Journey

| Step | User intent | Precondition | Success state | Failure state |
|------|------------|--------------|---------------|---------------|
| 1 | Query parcel | Coordinates or address | Full geometry (if `idx:access` scope) | Teaser: partial data + upsell message |
| 2 | View teaser | No `idx:access` scope | Partial fields returned | 401 if no token at all |

The teaser tier is defined in `internal/handler/gis/` — authenticated users with `idx:access` scope get full parcel data; others get a redacted subset.

## Journey Anti-Patterns

1. **Dead-end states** — A flow that reaches an error with no recovery path. Always provide a "try again" or "contact support" action.
2. **Missing preconditions** — Token creation page with no domains, but no prompt to create one first.
3. **Orphaned pending states** — Spinner shown forever because the fetch failed silently. Always wire error handlers.
4. **Undocumented side effects** — Token shown once and never again, but no warning before the user navigates away.
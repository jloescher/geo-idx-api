# In-App Guidance Reference

## Contents
- Dashboard HTML Surfaces
- Error Message Inventory
- Form Validation
- Navigation Structure
- State Visibility

---

## Dashboard HTML Surfaces

The dashboard is server-rendered HTML generated in `internal/handler/dashboard/handler.go` using layout helpers from `internal/web/layout.go`.

### Layout Structure

```
web.Page(title, body)       → Standard header + nav + main content
web.LoginPage(form)          → Centered card with "Sign in" heading
```

Header navigation (`layout.go`):
- Brand link: `Quantyra IDX` → `/`
- `Dashboard` → `/dashboard` (visible when logged in)
- `Login` → `/login` (visible when logged out)

No role-based nav differentiation. Admin-only features (invitations) are conditionally rendered in the dashboard body.

### WARNING: No Client-Side Validation

**The Problem:** Dashboard forms use HTML5 `required` attributes but no JavaScript validation. All error handling is server-side.

**Why This Breaks:** Users submit invalid forms, wait for a full round-trip, then see an error. No inline feedback, no field-level error display.

**The Fix:** When mapping guidance improvements, note that error feedback is currently page-level only.

---

## Error Message Inventory

### Authentication Errors

| Context | Message | Status |
|---------|---------|--------|
| Login failure | `"Invalid credentials"` | 401 |
| Missing Bearer token | `"Unauthenticated."` | 401 |
| Bad token hash | `"Invalid API token."` | 403 |
| Token missing abilities | `"Token is missing required IDX abilities."` | 403 |

### Domain Errors

| Context | Message | Status |
|---------|---------|--------|
| No domain header | `"Missing domain identification. Send X-Domain-Slug (or ?domain=) matching a verified domain on your account."` | 400 |
| Domain not found (token) | `"Domain is not registered, inactive, or not owned by this token."` | 403 |
| Domain not TXT-verified | `"Domain must be TXT-verified before API token access is allowed."` | 403 |
| Domain not found (header) | `"Domain is not registered or inactive."` | 403 |

### Verification Errors

| Context | Message | Status |
|---------|---------|--------|
| TXT not found | `"TXT record not found. Publish the verification record at your DNS host, then try again."` | 422 |
| DNS lookup failure | `"DNS lookup failed"` | 502 |
| Domain not found | `"domain not found"` | 404 |

### Token Errors

| Context | Message | Status |
|---------|---------|--------|
| Staging token exists | `"Staging token already exists"` | 409 |

### API Errors

| Context | Message | Status |
|---------|---------|--------|
| Bad search body | `"invalid search body"` | 400 |
| Bad comps body | `"invalid comps request"` | 400 |
| Upstream failure | Error from upstream | 502 |

---

## Form Validation

### Login Form

```html
<input type="email" name="email" required autocomplete="email">
<input type="password" name="password" required autocomplete="current-password">
```

No password requirements displayed. No "forgot password" flow.

### Domain Registration Form

```html
<input type="text" name="domain_slug" required placeholder="www.example.com">
<input type="text" name="mls_dataset" value="stellar">
```

No validation that `domain_slug` looks like a hostname. No preview of the TXT record the user will need to create.

### Invitation Acceptance Form

```html
<input type="text" name="name" required>
<input type="password" name="password" required minlength="8">
```

Only `minlength="8"` constraint. No strength indicator.

---

## Navigation Structure

The dashboard has no sidebar navigation. All actions are on a single page:

1. **Domains section**: List registered domains + add form
2. **API Tokens section**: List tokens + create form + revoke buttons
3. **Admin section** (conditional): Invitation creation form

No breadcrumbs, no step indicators, no progress tracking toward full activation.

---

## State Visibility

### What Users Can See

| State | Visibility |
|-------|------------|
| Domain list | Shown in dashboard |
| Token list | Token names shown (not values) |
| Domain verification status | Implicit (verify button present = unverified) |
| Token last used | Not shown |

### What Users Cannot See

| State | Impact |
|-------|--------|
| Replication status | No dashboard UI — must hit `GET /api/v1/bridge/stats` |
| Cache hit rates | Audit log only |
| MLS dataset health | Stats endpoint only |
| Upstream latency | Not surfaced |
| Queue depth | Not surfaced |

### WARNING: No Activation Progress Indicator

**The Problem:** New users see no indication of how many steps remain to become fully activated.

**Why This Breaks:** Users add a domain but don't realize they need TXT verification. They create a token but don't know they need to use it with `X-Domain-Slug`.

**The Fix:** When improving guidance, add a checklist or progress bar showing: domain registered → TXT verified → token created → first API call.

See the **ux** skill for UI pattern guidance.
See the **fiber** skill for HTML rendering patterns.
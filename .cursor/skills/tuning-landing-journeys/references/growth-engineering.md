# Growth Engineering Reference

## Contents
- Platform growth model
- Invitation loop mechanics
- API key lifecycle
- Activation blockers
- Anti-patterns

## Platform Growth Model

Quantyra IDX is invite-only, B2B, API-first. Growth is linear:

```
Admin (seeded) → invites User → User adds Domain → Domain verified → Token generated → API calls begin
```

There is no self-serve signup, no pricing page, no public docs that drive acquisition. Growth depends entirely on admin-initiated invitations.

## Invitation Loop Mechanics

The invite flow in `dashboard/handler.go:262-271`:

1. Admin enters email in "Invite user" card
2. `invitations.Create()` generates hashed token, stores in `user_invitations`
3. Link displayed once: `/invite/{plain_token}`
4. Admin shares link manually
5. Invitee opens link, sees name + password form
6. `invitations.Accept()` creates user account, marks invitation accepted
7. Redirect to `/login`

### Growth blocker: No email delivery

The invitation system has no outbound email. Admins must manually copy and send invite links. This limits velocity — each invite requires manual coordination.

**Quick win:** Add a transactional email service integration. The invitation handler already has the email address and the link. Wire a `notifications.SendInvite()` call that emails the link to the invitee.

### Growth blocker: Single-admin bottleneck

Only `is_admin = true` users see the invite card (`dashboard/handler.go:170`). If the seed admin is unavailable, no new users can be invited. Consider:

- Multi-admin support (allow inviting other admins)
- Role-based invite permissions (any verified user can invite)

## API Key Lifecycle

```
Create token → Use in API calls → Revoke when compromised
```

Token creation happens at two points:

| Trigger | Handler | Token type |
|---------|---------|------------|
| Domain verified (automatic) | `VerifyTXT` | Production (`idx:full`) |
| Manual creation | `CreateToken` | Named (`idx:full`) |
| Staging quick-create | `CreateStagingToken` | Staging (`idx:full`) |

### Growth lever: Staging token as trial

The staging token is created without domain verification. It lets users test the API before going through DNS verification. This is an activation accelerator.

**Problem:** The staging token is a plain text response (`c.SendString("Staging token: " + plain)`), not rendered in the dashboard layout. Users may miss it.

```go
// BAD — plain text, no layout
return c.SendString("Staging token: " + plain)

// GOOD — render in page layout with copy button, matching VerifyTXT pattern
body := `<div class="card"><h1>Staging token created</h1>
<p>Use this token for testing. Production tokens require domain verification.</p>
<div class="token-box" id="token">` + web.Esc(plain) + `</div>
<button data-copy="#token" class="btn btn-sm btn-secondary">Copy</button>
<p><a class="btn btn-primary" href="/dashboard">Back to dashboard</a></p></div>`
return c.Type("html").SendString(web.Page("Staging Token", body))
```

## Activation Blockers

Identified friction points in the setup flow:

### 1. DNS TXT verification is a multi-step manual process

User must:
1. Add domain in dashboard
2. Copy the TXT record value (not shown clearly — it's auto-generated as `_quantyra-verify.{slug}`)
3. Go to their DNS provider and add a TXT record
4. Return to dashboard and click "Verify TXT"
5. If DNS hasn't propagated, get a 422 error and must retry

**Quick win:** Show the TXT record details clearly after domain creation:

```go
// new code to add — after StoreDomain success
body := `<div class="card"><h1>Domain added</h1>
<p>Add this TXT record to your DNS:</p>
<div class="token-box">
<strong>Host:</strong> _quantyra-verify.` + web.Esc(slug) + `<br>
<strong>Value:</strong> ` + web.Esc(val) + `
</div>
<p>DNS propagation may take up to 24 hours.</p>
<a class="btn btn-primary" href="/dashboard">Back to dashboard</a></div>`
return c.Type("html").SendString(web.Page("Domain Added", body))
```

### 2. No password reset

Users who forget their password have no recovery path. The system has no "forgot password" flow, no email on file for recovery (email is only used for login). This is a support burden at scale.

### 3. No onboarding guidance

After first login, the dashboard shows four cards with no explanation of the recommended order. A step indicator would reduce time-to-activation.

## Anti-Patterns

### WARNING: Invite link is one-time, unrecoverable

If the admin navigates away from the "Invitation created" page, the plain token is lost. The hash is stored in `user_invitations` but the plain token cannot be recovered. Add a "regenerate invite link" action for accepted invitations.

### WARNING: Staging token limit is one-per-user

`CreateStagingToken` enforces a single staging token per user. Revoking the staging token requires using the revoke flow, but the dashboard doesn't distinguish staging from production tokens visually. Users may accidentally revoke production tokens.

### WARNING: No rate limiting on invitation creation

Any admin can create unlimited invitations. There is no rate limit, no confirmation, and no listing of pending invitations. This is acceptable for small teams but becomes a management problem at scale.
# Conversion Optimization Reference

## Contents
- Lifecycle funnel stages
- Conversion drop-off points
- Dashboard copy optimization
- Anti-patterns

## Lifecycle Funnel Stages

The Quantyra IDX platform has five measurable conversion stages:

| Stage | Entry | Success signal | Current copy |
|---|---|---|---|
| Invitation | Admin creates invite | Invitee opens `/invite/:token` | "Invitation created — Share this link (shown once)" |
| Registration | Invitee visits invite link | Account created, redirected to `/login` | "Create account" button |
| Domain verification | User adds domain in dashboard | DNS TXT record confirmed | "TXT record not found. Publish the verification record..." |
| Token creation | Domain verified or manual | Token displayed once | "Save this production token now — it will not be shown again." |
| API activation | Token in hand | First successful API call | No guidance currently |

### Drop-off points

1. **Invitation → Registration:** Admin must manually share the invite link (no email). The link is a raw hex token URL — no context about what the invitee is joining.

2. **Registration → Domain verification:** After login, users land on the dashboard with minimal guidance: "Register domains, verify DNS, and manage API keys." No step indicators or progress tracking.

3. **Domain verification → Token creation:** TXT verification requires DNS knowledge. The error message is technical but accurate. Consider linking to DNS setup docs for non-developer users.

4. **Token creation → API activation:** No onboarding for first API call. Users receive a token string with no example request.

## Dashboard Copy Optimization

### DO: Show one-time values with clear warnings

```go
// internal/handler/dashboard/handler.go:225 — GOOD pattern
body := `<div class="card"><h1>Domain verified</h1><p>Save this production token now — it will not be shown again.</p>
<div class="token-box" id="token">` + web.Esc(plain) + `</div>...`
```

The "will not be shown again" copy is effective — it creates urgency and sets expectations. Apply this pattern to any new one-time-value display.

### DON'T: Show raw token in a plain text response

```go
// internal/handler/dashboard/handler.go:243 — suboptimal
return c.SendString("Staging token: " + plain)
```

This returns plain text without the card layout, token-box styling, or copy-to-clipboard affordance. The staging token flow should use the same card + token-box pattern as the production token for consistency.

### DO: Use status badges for verification state

```go
// internal/handler/dashboard/handler.go:129-131 — existing badge pattern
badge := "badge-pending"
if status == "verified" || status == "verified_ghl" {
    badge = "badge-verified"
}
```

Badge classes (`badge-pending`, `badge-verified`) are defined in `app.css`. Use these for any new status indicators to maintain visual consistency. See the **frontend-design** skill for badge styling details.

## Anti-patterns

### WARNING: Sending email synchronously in HTTP handlers

**The Problem:**

```go
// BAD — blocks the HTTP response on SMTP latency
func (h *Handler) CreateInvitation(c *fiber.Ctx) error {
    // ... create invitation ...
    smtp.SendMail(addr, auth, from, []string{email}, msg) // blocks
    return c.Redirect("/dashboard")
}
```

**Why This Breaks:**
1. SMTP latency (1-30s) blocks the Fiber response
2. SMTP timeout becomes a 502 to the user
3. No retry — transient failures silently lose the email

**The Fix:**

```go
// new code to add — enqueue email job, return immediately
// After invitation creation:
// h.queue.Enqueue(ctx, "default", queue.Job{
//     Type: "email.send",
//     Payload: map[string]any{"to": email, "template": "invitation", "invite_link": link},
// })
```

### WARNING: Hardcoded copy in handler string literals

All dashboard copy lives as inline Go string literals in `internal/handler/dashboard/handler.go`. This makes A/B testing or localization impossible without redeployment. For copy that changes frequently (landing page hero, onboarding guidance), extract to a template file or config map before running experiments. See the **measurement-testing** reference for experiment setup patterns.

## Workflow: Optimize a dashboard conversion step

Copy this checklist and track progress:
- [ ] Identify the stage with highest drop-off (check `audit_logs` or analytics)
- [ ] Read the current inline copy at the handler route
- [ ] Write updated copy in the handler string literal
- [ ] Test locally: `make run-api`, walk the flow
- [ ] Deploy and measure conversion delta over 7 days
- [ ] Iterate if delta is not significant
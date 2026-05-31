# Microcopy Reference

## Contents
- Principles
- API Error Messages
- Dashboard Microcopy
- Token Lifecycle Copy
- Microcopy Anti-Patterns

## Principles

1. **Specific over generic** — "Domain 'example.com' is already registered" beats "An error occurred".
2. **Action-oriented** — "Copy token" beats "Token details".
3. **State-aware** — Copy changes with state: "Create" → "Creating…" → "Created".
4. **No blame** — "No domains found. Create one to get started." beats "You haven't added any domains."

## API Error Messages

| Error code | HTTP | Message | Why this phrasing |
|-----------|------|---------|-------------------|
| `UNAUTHENTICATED` | 401 | "Provide a valid API token in the Authorization header." | Tells the consumer exactly what's missing. |
| `FORBIDDEN` | 403 | "Your token does not have the required scope for this endpoint." | Distinguishes from auth failure. |
| `INVALID_DATASET` | 400 | "Dataset must be 'stellar' or 'beaches'." | Enumerates valid values. |
| `UPSTREAM_TIMEOUT` | 504 | "MLS provider did not respond in time. Try again." | Blames upstream, not the consumer. Suggests action. |
| `RATE_LIMITED` | 429 | "Too many requests. Try again later." | Simple, no jargon. |

### WARNING: Technical Jargon in User-Facing Messages

**The Problem:**

```json
{"error": "pgx: no rows in result set"}
```

**Why This Breaks:** Exposes implementation details. Confuses API consumers. May reveal architecture to attackers.

**The Fix:**

```json
{"error": {"code": "NOT_FOUND", "message": "No listing found with that key."}}
```

**When You Might Be Tempted:** Wrapping every error takes discipline. Use a consistent error helper so the right shape is automatic.

## Dashboard Microcopy

| Context | Text | Why |
|---------|------|-----|
| Empty domains | "No domains registered. Add your first domain to get started." | Action + next step |
| Empty tokens | "No API tokens. Create one to authenticate requests." | Explains purpose |
| Token created | "Token created. Copy it now — it won't be shown again." | Urgency + consequence |
| Token revoked | "Token revoked. Any requests using it will be denied." | Consequence clear |
| Replication running | "Replication in progress. Check back in a few minutes." | Sets expectation |

## Token Lifecycle Copy

Tokens follow a strict lifecycle with specific copy at each stage:

| Stage | Copy | Tone |
|-------|------|------|
| Before create | "Choose a name and scopes for this token." | Guiding |
| At creation | "Creating token…" | Pending |
| After create | "Token: `abc123...`. Copy it now — this is the only time it will be shown." | Urgent |
| After copy | "Token copied to clipboard." | Confirming |
| After nav away | (Token is gone forever — no "view token" option) | — |
| Revoke confirm | "Revoke this token? Active requests using it will start failing immediately." | Warning |
| After revoke | "Token revoked." | Final |

### WARNING: "Are you sure?" Without Consequence

**The Problem:** "Are you sure you want to revoke this token?" — vague, easy to click through without reading.

**The Fix:** Include the consequence: "Revoke this token? Active requests using it will start failing immediately." The user now has real information to make the decision.

**When You Might Be Tempted:** Short confirmations feel cleaner. They also get clicked automatically. Make the consequence visible.

## Microcopy Anti-Patterns

1. **"Success"** — What succeeded? Be specific: "Domain added", "Token created", "Settings saved".
2. **"Error"** — What went wrong? Be specific: "Domain already exists", "Token name is required".
3. **Humorous copy in error states** — Users are frustrated. A joke about "whoopsie" makes it worse. Be clear and helpful.
4. **Changing terminology** — "Token" in one place, "API key" in another, "credential" in a third. Pick one term. This project uses "token".
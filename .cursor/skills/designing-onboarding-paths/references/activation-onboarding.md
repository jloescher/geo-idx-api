# Activation Onboarding

When to use: Designing first-run flows that require users to complete critical setup steps before accessing core product value. Use for OAuth integrations, domain registration, and compliance checkpoints where linear progression matters.

## Patterns

**Session-Gated OAuth Flow with Step Validation**
GHL Marketplace integration uses session storage (`ghl_pending_oauth_token_id`) to maintain state between OAuth redirect and callback. The URL registration step validates this session key exists and redirects to `/leadconnector/install` if missing. Each step in `routes/ghl-web.php` enforces prerequisites: callback requires state match, registration requires pending token, completion requires registered URLs.

**Compliance Checkpoint Pattern**
MLS domain registration includes `mls_agreement_acknowledged` boolean stored on `ghl_registered_urls` before widget API key generation. The form validates this checkbox server-side and prevents progression until explicit acknowledgment. Store acknowledgment timestamp for audit trails in `ghl_audit_logs`.

**Completion Artifact Reveal**
The installation complete page generates and displays the widget API key (`qh_` prefix) with copy-to-clipboard functionality, embed code snippets for GHL websites, and direct links to IDX platform dashboard. Include immediate "Test embed" CTA pointing to `/widget/loader.js` documentation.

## Warning

Do not store intermediate OAuth state in the database—use session only. Database rows create orphan records when users abandon flows. Clear session keys immediately after successful token persistence or registration completion to prevent replay attacks on shared computers.
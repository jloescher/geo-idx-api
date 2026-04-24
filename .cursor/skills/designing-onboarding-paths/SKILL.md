---
name: designing-onboarding-paths
description: Designs onboarding paths, checklists, and first-run UI
allowed-tools: Read, Edit, Write, Glob, Grep, Bash
---

# Designing Onboarding Paths Skill

This skill designs multi-step onboarding flows, progress checklists, and first-run UI experiences for Laravel + Livewire applications. It favors session-based state persistence for OAuth-style flows, explicit step validation, and completion pages with actionable next steps.

## Quick Start

1. **Map the flow** - Identify distinct phases (e.g., OAuth redirect → callback → data collection → completion)
2. **Create session-gated steps** - Store intermediate state in session (e.g., `ghl_pending_oauth_token_id`) and validate in middleware/controller
3. **Build checklists** - Use Blade components with conditional styling for completed/current/pending states
4. **Design completion pages** - Include embed codes, API keys, and clear CTAs for the user's next action

## Key Concepts

**Session-Based Flow State**: Store pending onboarding state in session rather than database. Validate at each step gate and clear on completion or abandonment. Example: OAuth tokens pending URL registration.

**Step Validation Gates**: Each step checks prerequisites (session keys, previous completion) and redirects to the appropriate step if validation fails. Prevents URL hopping and ensures linear progression.

**Compliance Checkpoints**: Explicit acknowledgment steps for legal/compliance requirements (MLS agreements, terms acceptance) stored as flags before proceeding.

**Completion Artifacts**: Generate and display API keys, embed snippets, or configuration values on the final step with copy-to-clipboard functionality.

## Common Patterns

**Multi-Step OAuth + Registration Flow**:
- Installation landing (`/install`) → Authorization redirect → Callback handling → Session token storage → URL registration form (gated by session) → Installation complete with embed snippets

**Progress Checklist Component**:
- Blade partial rendering steps array with icons for completed (check), current (spinner/arrow), pending (circle)
- Steps: Connect Account → Register Domains → Configure Widget → Complete

**First-Run Dashboard State**:
- Conditional dashboard sections based on `onboarding_completed` flag
- Inline checklist for remaining setup tasks (create API token, configure billing, verify domain)

**Widget/API Key Reveal Pattern**:
- Masked key display with reveal toggle
- Immediate copy button with visual feedback
- Regenerate capability with confirmation modal
- Usage example code blocks (JS embed, cURL)
# Accessibility Reference

## Contents
- Dashboard Accessibility
- API Response Accessibility
- Keyboard Navigation
- Accessible Error Handling
- Accessibility Anti-Patterns

## Dashboard Accessibility

The dashboard is server-rendered HTML. No SPA framework. Accessibility comes from native semantics.

### Form Labels

```html
<!-- GOOD — explicit label associated with input -->
<label for="tokenName">Token name</label>
<input id="tokenName" name="name" type="text" required>

<!-- BAD — placeholder as label -->
<input placeholder="Token name" name="name">
```

Placeholders disappear on input. Screen readers may not announce them. Use visible `<label>` elements.

### Focus Management

After form submission, move focus to the result:

```html
<!-- new code to add — focus the success/error message -->
<div id="result" tabindex="-1" role="alert">
  Token created successfully.
</div>
```

`role="alert"` makes screen readers announce the message. `tabindex="-1"` allows programmatic focus.

## API Response Accessibility

API JSON responses are consumed by code, not screen readers directly. Accessibility applies to:

1. **Error messages** — Must be human-readable, not just codes. `"UPSTREAM_TIMEOUT"` is not helpful; "MLS provider did not respond in time" is.
2. **Documentation** — OpenAPI spec at `/openapi.json` and `/swagger` must describe error shapes.

See the **fiber** skill for routing and middleware patterns.

## Keyboard Navigation

| Element | Key | Action |
|---------|-----|--------|
| Tab | Focus next interactive element | Standard browser behavior |
| Enter/Space | Submit focused button | Standard for `<button>` |
| Escape | Close modal/dialog | Must be implemented |

### WARNING: Custom Controls Without Keyboard Support

**The Problem:** A `<div onclick="...">` acting as a button. No keyboard activation. No focus ring. Screen readers ignore it.

**The Fix:**

```html
<!-- GOOD — native button with keyboard support built in -->
<button type="button" class="token-action">Revoke</button>
```

**When You Might Be Tempted:** Styling a `<div>` feels easier than overriding button styles. Use native elements and style them.

## Accessible Error Handling

1. **Inline errors** — Place error messages next to the field they describe, not in a banner at the top.
2. **Error summary** — For multi-field forms, add a summary at the top linking to each invalid field.
3. **Color is not enough** — Don't rely solely on red text. Add an icon or text prefix like "Error:".

## Accessibility Anti-Patterns

1. **Hover-only content** — Critical instructions visible only on hover. Touch users and keyboard users never see them.
2. **Missing focus indicators** — Remove default focus outlines without providing a custom one. Users who navigate by keyboard cannot see where they are.
3. **Low contrast** — Gray text on gray background. Minimum 4.5:1 contrast ratio for normal text.
4. **Auto-advancing forms** — Select an option and immediately submit. Users who navigate by keyboard may trigger this accidentally.
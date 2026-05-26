// Platform UI helpers (Quantyra IDX)
document.addEventListener('DOMContentLoaded', () => {
  document.querySelectorAll('[data-copy]').forEach((el) => {
    el.addEventListener('click', () => {
      const target = document.querySelector(el.getAttribute('data-copy'));
      if (!target) return;
      const secret = target.getAttribute('data-token-secret');
      const text = secret || target.textContent.trim();
      if (text) navigator.clipboard.writeText(text);
    });
  });

  document.querySelectorAll('[data-token-toggle]').forEach((btn) => {
    btn.addEventListener('click', () => {
      const target = document.querySelector(btn.getAttribute('data-token-toggle'));
      if (!target) return;
      const secret = target.getAttribute('data-token-secret');
      if (!secret) return;
      const masked = target.getAttribute('data-masked') || 'idx_•••••••••••••••••••••••••••••••••';
      const shown = target.getAttribute('data-revealed') === 'true';
      if (shown) {
        target.textContent = masked;
        target.setAttribute('data-revealed', 'false');
        btn.textContent = 'Show';
        btn.setAttribute('aria-pressed', 'false');
      } else {
        target.textContent = secret;
        target.setAttribute('data-revealed', 'true');
        btn.textContent = 'Hide';
        btn.setAttribute('aria-pressed', 'true');
      }
    });
  });
});

// Platform UI helpers (Quantyra IDX)
document.addEventListener('DOMContentLoaded', () => {
  document.querySelectorAll('[data-copy]').forEach((el) => {
    el.addEventListener('click', async () => {
      const target = document.querySelector(el.getAttribute('data-copy'));
      if (!target) return;
      const secret = target.getAttribute('data-token-secret');
      const text = secret || target.textContent.trim();
      if (!text) return;
      try {
        await navigator.clipboard.writeText(text);
        const original = el.textContent;
        el.textContent = 'Copied!';
        setTimeout(() => { el.textContent = original; }, 1600);
      } catch (e) {
        // Fallback: at least select the text for manual copy
        try {
          const range = document.createRange();
          range.selectNodeContents(target);
          const sel = window.getSelection();
          sel.removeAllRanges();
          sel.addRange(range);
        } catch (_) {}
        // Gentle feedback (consistent with monitoring pages)
        const original = el.textContent;
        el.textContent = 'Copy failed';
        setTimeout(() => { el.textContent = original; }, 1600);
      }
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

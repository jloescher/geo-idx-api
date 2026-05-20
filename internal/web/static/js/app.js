// Platform UI helpers (Quantyra IDX)
document.addEventListener('DOMContentLoaded', () => {
  document.querySelectorAll('[data-copy]').forEach((el) => {
    el.addEventListener('click', () => {
      const target = document.querySelector(el.getAttribute('data-copy'));
      if (target) {
        navigator.clipboard.writeText(target.textContent.trim());
      }
    });
  });
});

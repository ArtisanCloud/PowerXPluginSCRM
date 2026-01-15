import { defineNuxtPlugin } from "#imports";

export default defineNuxtPlugin(() => {
  if (process.server) {
    return;
  }

  const root = document.getElementById("__nuxt") || document.body;
  if (!root || typeof MutationObserver === "undefined") {
    return;
  }

  const blurIfHidden = (target: HTMLElement) => {
    const active = document.activeElement;
    if (
      active instanceof HTMLElement &&
      target.contains(active) &&
      target.getAttribute("aria-hidden") === "true"
    ) {
      requestAnimationFrame(() => {
        if (active === document.activeElement) {
          active.blur();
        }
      });
    }
  };

  const observer = new MutationObserver((records) => {
    for (const record of records) {
      const target = record.target as HTMLElement;
      if (record.type === "attributes" && target) {
        blurIfHidden(target);
      }
    }
  });

  observer.observe(root, {
    attributes: true,
    subtree: true,
    attributeFilter: ["aria-hidden", "data-aria-hidden"],
  });

  window.addEventListener("beforeunload", () => observer.disconnect(), {
    once: true,
  });
});

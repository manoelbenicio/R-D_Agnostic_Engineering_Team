import type { Page } from '@playwright/test';

/**
 * CSS that neutralises CSS animations and transitions so Playwright's
 * actionability checks never see an "unstable" element.
 *
 * Why we need this: AgentVerse's voice 'listening' state renders a 🛑 stop
 * button with an infinite `pulse` keyframe animation. Playwright's normal
 * stability check waits for the bounding box to settle for two consecutive
 * animation frames before clicking; an infinite scale+box-shadow pulse never
 * settles, so the click would otherwise time out (or have to be force-clicked
 * with the actionability bypass flag, which masks real interaction bugs).
 */
const ANIMATION_KILL_CSS = `
  *, *::before, *::after {
    animation-duration: 0s !important;
    animation-delay: 0s !important;
    animation-iteration-count: 1 !important;
    transition-duration: 0s !important;
    transition-delay: 0s !important;
    scroll-behavior: auto !important;
  }
`;

/**
 * Disable CSS animations and transitions for the lifetime of the page,
 * including across navigations.
 *
 * Implementation notes:
 *   - Uses `addInitScript` so the override survives every `page.goto` in the
 *     test (the smoke spec navigates several times).
 *   - Honours `prefers-reduced-motion: reduce` as a belt-and-braces signal for
 *     any CSS that gates animations on the media query.
 *   - Idempotent: a second call won't inject duplicate <style> nodes.
 *
 * Call this in `beforeEach` *before* the first `page.goto`.
 */
export async function disableAnimations(page: Page): Promise<void> {
  await page.emulateMedia({ reducedMotion: 'reduce' });

  await page.addInitScript((css: string) => {
    const STYLE_ID = '__playwright_disable_animations__';

    const inject = (): void => {
      if (document.getElementById(STYLE_ID)) return;
      const target = document.head ?? document.documentElement;
      if (!target) return;
      const style = document.createElement('style');
      style.id = STYLE_ID;
      style.textContent = css;
      target.appendChild(style);
    };

    if (document.readyState !== 'loading') {
      inject();
    }

    // Re-attempt once <head> is parsed, in case the init script ran before it
    // existed.
    document.addEventListener('DOMContentLoaded', inject, { once: true });

    // And once more on full load, to win against late-mounted style resets.
    window.addEventListener('load', inject, { once: true });
  }, ANIMATION_KILL_CSS);
}

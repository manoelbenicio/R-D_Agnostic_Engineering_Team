import { useEffect } from 'react';

/**
 * DSS motion infrastructure — toggles `.animate-in` on every `[data-animate]`
 * element when it scrolls into the viewport.
 *
 * Pair with the CSS rules in `tokens.css`:
 *
 *   [data-animate]            { opacity: 0; transform: translateY(24px); transition: ...; }
 *   [data-animate].animate-in { opacity: 1; transform: translateY(0); }
 *
 * Mount this once at the layout level (e.g. `AppLayout.tsx`). It rescans the
 * DOM on each render so newly-mounted children pick up entrance animations
 * without having to opt in individually.
 *
 * Honors `prefers-reduced-motion`: when the user opts out, every element is
 * marked `.animate-in` immediately so nothing remains invisible.
 */
export function useDataAnimateObserver(rootMargin = '0px 0px -10% 0px'): void {
  useEffect(() => {
    if (typeof window === 'undefined' || typeof document === 'undefined') return;

    // Honor reduced-motion: skip observer entirely, reveal everything.
    const reduced =
      typeof window.matchMedia === 'function' &&
      window.matchMedia('(prefers-reduced-motion: reduce)').matches;
    if (reduced) {
      document.querySelectorAll<HTMLElement>('[data-animate]').forEach((el) => {
        el.classList.add('animate-in');
      });
      return;
    }

    if (typeof IntersectionObserver === 'undefined') {
      // Older browsers / SSR — fail open: reveal everything.
      document.querySelectorAll<HTMLElement>('[data-animate]').forEach((el) => {
        el.classList.add('animate-in');
      });
      return;
    }

    const observer = new IntersectionObserver(
      (entries) => {
        for (const entry of entries) {
          if (entry.isIntersecting && entry.target instanceof HTMLElement) {
            entry.target.classList.add('animate-in');
            // Once revealed, no need to keep observing (DSS pattern: one-shot).
            observer.unobserve(entry.target);
          }
        }
      },
      {
        rootMargin,
        threshold: 0.1,
      },
    );

    const targets = document.querySelectorAll<HTMLElement>('[data-animate]');
    targets.forEach((el) => observer.observe(el));

    return () => {
      observer.disconnect();
    };
  });
}

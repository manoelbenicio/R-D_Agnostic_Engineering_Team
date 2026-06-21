import '@testing-library/jest-dom/vitest';
import 'fake-indexeddb/auto';
import { afterAll, afterEach, beforeAll, beforeEach } from 'vitest';

// MSW lifecycle. Capability-owned handlers are aggregated in
// src/api/__tests__/msw/handlers.ts. Tests that need to override behaviour
// can do `server.use(...)` inline.
import { server } from '@/api/__tests__/msw/server';

beforeAll(() => server.listen({ onUnhandledRequest: 'error' }));
afterEach(() => server.resetHandlers());
afterAll(() => server.close());

// Stable ResizeObserver/IntersectionObserver shims for jsdom.
class ResizeObserverShim {
  observe(): void {
    /* no-op */
  }
  unobserve(): void {
    /* no-op */
  }
  disconnect(): void {
    /* no-op */
  }
}

if (typeof globalThis.ResizeObserver === 'undefined') {
  globalThis.ResizeObserver = ResizeObserverShim;
}

// matchMedia shim — the design system queries `prefers-reduced-motion`.
if (typeof window !== 'undefined' && typeof window.matchMedia !== 'function') {
  window.matchMedia = (query: string) =>
    ({
      matches: false,
      media: query,
      onchange: null,
      addEventListener: () => {},
      removeEventListener: () => {},
      addListener: () => {},
      removeListener: () => {},
      dispatchEvent: () => false,
    }) as unknown as MediaQueryList;
}

// Limpar Zustand stores entre testes para evitar poluição de estado.
beforeEach(async () => {
  // Reset session store
  try {
    const { useSessionStore } = await import('@/api/session-store');
    useSessionStore.setState({ sessions: [], loading: false, error: null, lastRefreshed: null });
  } catch {
    /* store pode nao existir em todos os contexts */
  }
});


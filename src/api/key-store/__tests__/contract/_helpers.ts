/**
 * Helpers for the keystore live contract suite.
 *
 * The contract tests in this directory hit real provider endpoints and are
 * gated behind `KEYSTORE_LIVE=1`. They complement (do not replace) the
 * deterministic MSW unit tests in `src/api/key-store/__tests__/validators/`.
 *
 * See `./README.md` and `docs/keystore-contract-tests.md` for the full env-var
 * matrix and triage guide.
 */
import { afterAll, beforeAll, it } from 'vitest';
import { server } from '@/api/__tests__/msw/server';

/**
 * Returns the value of `name` from `process.env` or `null` when absent.
 *
 * Emits a single `console.warn` on miss so that nightly CI logs surface every
 * provider that was implicitly skipped because its secret was unavailable.
 */
export function requireEnv(name: string): string | null {
  const value = process.env[name];
  if (!value || value.length === 0) {
    // eslint-disable-next-line no-console
    console.warn(
      `[keystore-contract] env "${name}" is not set; skipping live check for this provider.`,
    );
    return null;
  }
  return value;
}

/** True when the live gate is enabled via `KEYSTORE_LIVE=1`. */
export function isLive(): boolean {
  return process.env.KEYSTORE_LIVE === '1';
}

/**
 * Returns vitest's `it` when the live gate is on AND every supplied env value
 * is present, otherwise returns `it.skip` so the test is reported as skipped
 * (zero failures) when secrets are missing.
 *
 * Usage:
 *   liveOrSkip(apiKey)('lists models', async () => { ... });
 *   liveOrSkip(accessKey, secretKey)('GetCallerIdentity ok', async () => { ... });
 *   liveOrSkip()('always skipped unless KEYSTORE_LIVE=1', async () => { ... });
 */
export function liveOrSkip(...envValues: Array<string | null | undefined>): typeof it {
  const allPresent =
    envValues.length === 0 || envValues.every((v) => typeof v === 'string' && v.length > 0);
  return (isLive() && allPresent ? it : it.skip) as typeof it;
}

/**
 * Re-configures the shared MSW server so unhandled requests pass through to
 * the real network for the duration of this test file. The default test
 * setup (`src/__tests__/setup.ts`) starts MSW with
 * `onUnhandledRequest: 'error'`, which would otherwise reject every live
 * provider call.
 *
 * No-op when `KEYSTORE_LIVE !== '1'` so default `npm test` keeps the strict
 * MSW boundary.
 *
 * Must be invoked at the top of a `describe` block so the lifecycle hooks
 * register against vitest's runner.
 */
export function passthroughMsw(): void {
  if (!isLive()) {
    return;
  }
  beforeAll(() => {
    // `listen` can be called repeatedly; the new options replace the old.
    server.listen({ onUnhandledRequest: 'bypass' });
  });
  afterAll(() => {
    server.close();
  });
}

/** Default vitest test timeout for live provider calls (30 s). */
export const LIVE_TEST_TIMEOUT_MS = 30_000;

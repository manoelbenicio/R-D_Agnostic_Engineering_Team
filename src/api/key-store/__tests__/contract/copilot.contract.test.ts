/**
 * GitHub Copilot live contract test.
 *
 * Hits `https://api.github.com/copilot_internal/v2/token` with a real GitHub
 * token that has Copilot access. Skipped unless `KEYSTORE_LIVE=1` and
 * `GITHUB_COPILOT_TOKEN` are both set.
 *
 * The validator returns a synthetic models list on success, so we assert ok
 * + non-empty models — which proves the auth contract still holds.
 */
import { describe, expect } from 'vitest';
import { validateCopilot } from '../../validators/copilot';
import { LIVE_TEST_TIMEOUT_MS, liveOrSkip, passthroughMsw, requireEnv } from './_helpers';

const apiKey = requireEnv('GITHUB_COPILOT_TOKEN');

describe('GitHub Copilot live contract', () => {
  passthroughMsw();

  liveOrSkip(apiKey)(
    'returns ok=true from the Copilot token endpoint',
    async () => {
      const res = await validateCopilot(apiKey!);
      expect(res.ok).toBe(true);
      expect(res.error).toBeUndefined();
      expect(Array.isArray(res.models)).toBe(true);
      expect(res.models!.length).toBeGreaterThan(0);
    },
    LIVE_TEST_TIMEOUT_MS,
  );
});

/**
 * Anthropic live contract test.
 *
 * Hits `https://api.anthropic.com/v1/models` with a real key.
 * Skipped unless `KEYSTORE_LIVE=1` and `ANTHROPIC_API_KEY` are both set.
 */
import { describe, expect } from 'vitest';
import { validateAnthropic } from '../../validators/anthropic';
import { LIVE_TEST_TIMEOUT_MS, liveOrSkip, passthroughMsw, requireEnv } from './_helpers';

const apiKey = requireEnv('ANTHROPIC_API_KEY');

describe('Anthropic live contract', () => {
  passthroughMsw();

  liveOrSkip(apiKey)(
    'returns ok=true and a non-empty models list',
    async () => {
      const res = await validateAnthropic(apiKey!);
      expect(res.ok).toBe(true);
      expect(res.error).toBeUndefined();
      expect(Array.isArray(res.models)).toBe(true);
      expect(res.models!.length).toBeGreaterThan(0);
    },
    LIVE_TEST_TIMEOUT_MS,
  );
});

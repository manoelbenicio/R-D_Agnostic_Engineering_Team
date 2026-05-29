/**
 * Google Gemini live contract test.
 *
 * Hits `https://generativelanguage.googleapis.com/v1beta/models` with a real
 * key. Skipped unless `KEYSTORE_LIVE=1` and `GOOGLE_API_KEY` are both set.
 */
import { describe, expect } from 'vitest';
import { validateGoogle } from '../../validators/google';
import { LIVE_TEST_TIMEOUT_MS, liveOrSkip, passthroughMsw, requireEnv } from './_helpers';

const apiKey = requireEnv('GOOGLE_API_KEY');

describe('Google Gemini live contract', () => {
  passthroughMsw();

  liveOrSkip(apiKey)(
    'returns ok=true and a non-empty models list',
    async () => {
      const res = await validateGoogle(apiKey!);
      expect(res.ok).toBe(true);
      expect(res.error).toBeUndefined();
      expect(Array.isArray(res.models)).toBe(true);
      expect(res.models!.length).toBeGreaterThan(0);
    },
    LIVE_TEST_TIMEOUT_MS,
  );
});

/**
 * OpenCode CLI live contract test.
 *
 * Hits `${OPENCODE_ENDPOINT}/models` with a real key. Skipped unless
 * `KEYSTORE_LIVE=1` AND both `OPENCODE_ENDPOINT` and `OPENCODE_API_KEY` are
 * set.
 */
import { describe, expect } from 'vitest';
import { validateOpenCode } from '../../validators/opencode';
import { LIVE_TEST_TIMEOUT_MS, liveOrSkip, passthroughMsw, requireEnv } from './_helpers';

const endpoint = requireEnv('OPENCODE_ENDPOINT');
const apiKey = requireEnv('OPENCODE_API_KEY');

describe('OpenCode CLI live contract', () => {
  passthroughMsw();

  liveOrSkip(endpoint, apiKey)(
    'returns ok=true and a non-empty models list',
    async () => {
      const res = await validateOpenCode(endpoint!, apiKey!);
      expect(res.ok).toBe(true);
      expect(res.error).toBeUndefined();
      expect(Array.isArray(res.models)).toBe(true);
      expect(res.models!.length).toBeGreaterThan(0);
    },
    LIVE_TEST_TIMEOUT_MS,
  );
});

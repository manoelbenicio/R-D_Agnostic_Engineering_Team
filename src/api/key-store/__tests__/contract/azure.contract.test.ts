/**
 * Azure OpenAI live contract test.
 *
 * Hits `${AZURE_OPENAI_ENDPOINT}/openai/models?api-version=2024-02-01` with a
 * real `api-key` header. Skipped unless `KEYSTORE_LIVE=1` AND both
 * `AZURE_OPENAI_ENDPOINT` and `AZURE_OPENAI_API_KEY` are set.
 */
import { describe, expect } from 'vitest';
import { validateAzure } from '../../validators/azure';
import { LIVE_TEST_TIMEOUT_MS, liveOrSkip, passthroughMsw, requireEnv } from './_helpers';

const endpoint = requireEnv('AZURE_OPENAI_ENDPOINT');
const apiKey = requireEnv('AZURE_OPENAI_API_KEY');

describe('Azure OpenAI live contract', () => {
  passthroughMsw();

  liveOrSkip(endpoint, apiKey)(
    'returns ok=true and a non-empty models list',
    async () => {
      const res = await validateAzure(endpoint!, apiKey!);
      expect(res.ok).toBe(true);
      expect(res.error).toBeUndefined();
      expect(Array.isArray(res.models)).toBe(true);
      expect(res.models!.length).toBeGreaterThan(0);
    },
    LIVE_TEST_TIMEOUT_MS,
  );
});

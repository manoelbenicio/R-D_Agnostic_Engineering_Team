/**
 * AWS live contract test.
 *
 * Calls AWS STS `GetCallerIdentity` (sigv4-signed) using real IAM credentials.
 * Skipped unless `KEYSTORE_LIVE=1` AND both `AWS_ACCESS_KEY_ID` and
 * `AWS_SECRET_ACCESS_KEY` are set.
 *
 * Note: the validator returns a synthetic models list (`q-developer`,
 * `kiro-agent-v1`) when STS accepts the signed request, so the assertions
 * mirror that contract — what we are really proving here is that the sigv4
 * signing logic still produces a 200 from the live STS endpoint.
 */
import { describe, expect } from 'vitest';
import { validateAWS } from '../../validators/aws';
import { LIVE_TEST_TIMEOUT_MS, liveOrSkip, passthroughMsw, requireEnv } from './_helpers';

const accessKeyId = requireEnv('AWS_ACCESS_KEY_ID');
const secretAccessKey = requireEnv('AWS_SECRET_ACCESS_KEY');

describe('AWS live contract', () => {
  passthroughMsw();

  liveOrSkip(accessKeyId, secretAccessKey)(
    'returns ok=true from STS GetCallerIdentity',
    async () => {
      const res = await validateAWS(accessKeyId!, secretAccessKey!);
      expect(res.ok).toBe(true);
      expect(res.error).toBeUndefined();
      expect(Array.isArray(res.models)).toBe(true);
      expect(res.models!.length).toBeGreaterThan(0);
    },
    LIVE_TEST_TIMEOUT_MS,
  );
});

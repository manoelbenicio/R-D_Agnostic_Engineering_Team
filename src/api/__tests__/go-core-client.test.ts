import { describe, expect, it } from 'vitest';
import { GoCoreClient } from '@/api/go-core-client';
import { GoCoreApiError } from '@/api/errors';

const client = new GoCoreClient('http://127.0.0.1:8080');

describe('GoCoreClient', () => {
  it('reads representative GO Core resources through MSW', async () => {
    await expect(client.getHealth()).resolves.toEqual({ status: 'ok' });
    await expect(client.listProfiles()).resolves.toEqual(
      expect.arrayContaining([expect.objectContaining({ name: 'supervisor' })])
    );
    await expect(client.getTerminalOutput('term-supervisor', 'tail')).resolves.toContain('tail output');
    await expect(client.getAgentDirs()).resolves.toEqual({ dirs: expect.arrayContaining([expect.any(String)]) });
  });

  it('serializes HTTP failures as GoCoreApiError', async () => {
    await expect(client.getProfile('missing-profile')).rejects.toMatchObject({
      name: 'GoCoreApiError',
      status: 404,
      endpoint: '/agents/profiles/missing-profile',
    });

    try {
      await client.getProfile('missing-profile');
    } catch (error) {
      expect(error).toBeInstanceOf(GoCoreApiError);
    }
  });
});

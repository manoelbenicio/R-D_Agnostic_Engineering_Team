import { describe, expect, it } from 'vitest';
import { CaoClient } from '@/api/cao-client';
import { CaoApiError } from '@/api/errors';

const client = new CaoClient('http://127.0.0.1:9889');

describe('CaoClient', () => {
  it('reads representative CAO resources through MSW', async () => {
    await expect(client.getHealth()).resolves.toEqual({ status: 'ok' });
    await expect(client.listProfiles()).resolves.toEqual(
      expect.arrayContaining([expect.objectContaining({ name: 'supervisor' })])
    );
    await expect(client.getTerminalOutput('term-supervisor', 'tail')).resolves.toContain('tail output');
    await expect(client.getAgentDirs()).resolves.toEqual({ dirs: expect.arrayContaining([expect.any(String)]) });
  });

  it('serializes HTTP failures as CaoApiError', async () => {
    await expect(client.getProfile('missing-profile')).rejects.toMatchObject({
      name: 'CaoApiError',
      status: 404,
      endpoint: '/agents/profiles/missing-profile',
    });

    try {
      await client.getProfile('missing-profile');
    } catch (error) {
      expect(error).toBeInstanceOf(CaoApiError);
    }
  });
});

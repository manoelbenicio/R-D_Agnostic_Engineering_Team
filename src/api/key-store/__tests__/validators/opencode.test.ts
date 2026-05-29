import { describe, it, expect } from 'vitest';
import { http, HttpResponse } from 'msw';
import { server } from '@/api/__tests__/msw/server';
import { validateOpenCode } from '../../validators/opencode';

describe('validateOpenCode', () => {
  it('returns models for a valid OpenCode endpoint and API key', async () => {
    server.use(
      http.get('https://opencode.test/models', ({ request }) => {
        expect(request.headers.get('api-key')).toBe('valid-opencode-key');
        expect(request.headers.get('Authorization')).toBe('Bearer valid-opencode-key');

        return HttpResponse.json({
          data: [{ id: 'opencode-coder' }, { name: 'opencode-reviewer' }],
        });
      }),
    );

    const res = await validateOpenCode('https://opencode.test', 'valid-opencode-key');

    expect(res.ok).toBe(true);
    expect(res.models).toEqual(['opencode-coder', 'opencode-reviewer']);
  });

  it('returns a validation error without leaking the API key on 401', async () => {
    server.use(
      http.get('https://opencode.test/models', () =>
        HttpResponse.json({ error: { message: 'Unauthorized' } }, { status: 401 }),
      ),
    );

    const res = await validateOpenCode('https://opencode.test', 'invalid-opencode-key');

    expect(res.ok).toBe(false);
    expect(res.error).toContain('OpenCode validation failed (401)');
    expect(res.error).not.toContain('invalid-opencode-key');
  });

  it('returns a connection error without leaking the API key on network failure', async () => {
    server.use(http.get('https://opencode.test/models', () => HttpResponse.error()));

    const res = await validateOpenCode('https://opencode.test', 'network-opencode-key');

    expect(res.ok).toBe(false);
    expect(res.error).toContain('OpenCode connection failed');
    expect(res.error).not.toContain('network-opencode-key');
  });

  it('strips a trailing slash from the endpoint before requesting models', async () => {
    server.use(
      http.get('https://opencode.test/models', () =>
        HttpResponse.json({ models: [{ name: 'local-model' }] }),
      ),
    );

    const res = await validateOpenCode('https://opencode.test/', 'valid-opencode-key');

    expect(res.ok).toBe(true);
    expect(res.models).toEqual(['local-model']);
  });
});

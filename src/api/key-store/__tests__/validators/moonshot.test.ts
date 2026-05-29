import { describe, it, expect } from 'vitest';
import { http, HttpResponse } from 'msw';
import { server } from '@/api/__tests__/msw/server';
import { validateMoonshot } from '../../validators/moonshot';

describe('validateMoonshot', () => {
  it('returns models for a valid Moonshot API key', async () => {
    server.use(
      http.get('https://api.moonshot.cn/v1/models', ({ request }) => {
        if (request.headers.get('Authorization') === 'Bearer valid-moonshot-key') {
          return HttpResponse.json({
            data: [{ id: 'moonshot-v1-8k' }, { id: 'moonshot-v1-32k' }],
          });
        }

        return HttpResponse.json({ error: { message: 'Unauthorized' } }, { status: 401 });
      }),
    );

    const res = await validateMoonshot('valid-moonshot-key');

    expect(res.ok).toBe(true);
    expect(res.models).toEqual(['moonshot-v1-8k', 'moonshot-v1-32k']);
  });

  it('returns a validation error without leaking the API key on 401', async () => {
    server.use(
      http.get('https://api.moonshot.cn/v1/models', () =>
        HttpResponse.json({ error: { message: 'Unauthorized' } }, { status: 401 }),
      ),
    );

    const res = await validateMoonshot('invalid-moonshot-key');

    expect(res.ok).toBe(false);
    expect(res.error).toContain('Moonshot validation failed (401)');
    expect(res.error).not.toContain('invalid-moonshot-key');
  });

  it('returns a connection error without leaking the API key on network failure', async () => {
    server.use(http.get('https://api.moonshot.cn/v1/models', () => HttpResponse.error()));

    const res = await validateMoonshot('network-moonshot-key');

    expect(res.ok).toBe(false);
    expect(res.error).toContain('Moonshot connection failed');
    expect(res.error).not.toContain('network-moonshot-key');
  });
});

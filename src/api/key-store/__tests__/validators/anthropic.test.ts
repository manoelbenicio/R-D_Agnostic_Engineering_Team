import { describe, it, expect } from 'vitest';
import { http, HttpResponse } from 'msw';
import { server } from '@/api/__tests__/msw/server';
import { validateAnthropic } from '../../validators/anthropic';

describe('validateAnthropic', () => {
  it('succeeds with a valid API key and returns list of models', async () => {
    server.use(
      http.get('https://api.anthropic.com/v1/models', ({ request }) => {
        const apiKey = request.headers.get('x-api-key');
        const version = request.headers.get('anthropic-version');
        if (apiKey === 'valid-anthropic-key' && version === '2023-06-01') {
          return HttpResponse.json({
            data: [
              { id: 'claude-3-5-sonnet-latest' },
              { id: 'claude-3-haiku-20240307' },
            ],
          });
        }
        return HttpResponse.json({ error: 'Invalid key' }, { status: 401 });
      })
    );

    const res = await validateAnthropic('valid-anthropic-key');
    expect(res.ok).toBe(true);
    expect(res.models).toEqual(['claude-3-5-sonnet-latest', 'claude-3-haiku-20240307']);
  });

  it('fails with a 401 error on invalid key', async () => {
    server.use(
      http.get('https://api.anthropic.com/v1/models', () => {
        return HttpResponse.json({ error: { message: 'Invalid API key' } }, { status: 401 });
      })
    );

    const res = await validateAnthropic('invalid-anthropic-key');
    expect(res.ok).toBe(false);
    expect(res.error).toContain('Anthropic validation failed (401)');
    expect(res.error).not.toContain('invalid-anthropic-key');
  });

  it('fails with connection error on network failure', async () => {
    server.use(
      http.get('https://api.anthropic.com/v1/models', () => {
        return HttpResponse.error();
      })
    );

    const res = await validateAnthropic('anthropic-network-key');
    expect(res.ok).toBe(false);
    expect(res.error).toContain('Anthropic connection failed');
    expect(res.error).not.toContain('anthropic-network-key');
  });
});

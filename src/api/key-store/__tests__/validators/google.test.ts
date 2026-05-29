import { describe, it, expect } from 'vitest';
import { http, HttpResponse } from 'msw';
import { server } from '@/api/__tests__/msw/server';
import { validateGoogle } from '../../validators/google';

describe('validateGoogle', () => {
  it('returns models for a valid Google API key', async () => {
    server.use(
      http.get('https://generativelanguage.googleapis.com/v1beta/models', ({ request }) => {
        const url = new URL(request.url);

        if (url.searchParams.get('key') === 'valid-google-key') {
          return HttpResponse.json({
            models: [
              { name: 'models/gemini-1.5-pro-latest' },
              { name: 'models/gemini-1.5-flash-latest' },
            ],
          });
        }

        return HttpResponse.json({ error: { message: 'Invalid key' } }, { status: 401 });
      }),
    );

    const res = await validateGoogle('valid-google-key');

    expect(res.ok).toBe(true);
    expect(res.models).toEqual(['gemini-1.5-pro-latest', 'gemini-1.5-flash-latest']);
  });

  it('returns a validation error without leaking the API key on 401', async () => {
    server.use(
      http.get('https://generativelanguage.googleapis.com/v1beta/models', () =>
        HttpResponse.json({ error: { message: 'API key not valid' } }, { status: 401 }),
      ),
    );

    const res = await validateGoogle('invalid-google-key');

    expect(res.ok).toBe(false);
    expect(res.error).toContain('Google validation failed (401)');
    expect(res.error).not.toContain('invalid-google-key');
  });

  it('returns a connection error without leaking the API key on network failure', async () => {
    server.use(
      http.get('https://generativelanguage.googleapis.com/v1beta/models', () => HttpResponse.error()),
    );

    const res = await validateGoogle('network-google-key');

    expect(res.ok).toBe(false);
    expect(res.error).toContain('Google connection failed');
    expect(res.error).not.toContain('network-google-key');
  });
});

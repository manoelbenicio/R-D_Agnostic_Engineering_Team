import { describe, it, expect } from 'vitest';
import { http, HttpResponse } from 'msw';
import { server } from '@/api/__tests__/msw/server';
import { validateOpenAI } from '../../validators/openai';

describe('validateOpenAI', () => {
  it('succeeds with a valid API key and returns list of models', async () => {
    server.use(
      http.get('https://api.openai.com/v1/models', ({ request }) => {
        const auth = request.headers.get('Authorization');
        if (auth === 'Bearer valid-key') {
          return HttpResponse.json({
            data: [
              { id: 'gpt-4o' },
              { id: 'gpt-4o-mini' },
            ],
          });
        }
        return HttpResponse.json({ error: 'Invalid key' }, { status: 401 });
      })
    );

    const res = await validateOpenAI('valid-key');
    expect(res.ok).toBe(true);
    expect(res.models).toEqual(['gpt-4o', 'gpt-4o-mini']);
  });

  it('fails with a 401 error on invalid key', async () => {
    server.use(
      http.get('https://api.openai.com/v1/models', () => {
        return HttpResponse.json({ error: { message: 'Incorrect API key provided' } }, { status: 401 });
      })
    );

    const res = await validateOpenAI('invalid-key');
    expect(res.ok).toBe(false);
    expect(res.error).toContain('OpenAI validation failed (401)');
    expect(res.error).not.toContain('invalid-key'); // Must not leak the key!
  });

  it('fails with connection error on network failure', async () => {
    server.use(
      http.get('https://api.openai.com/v1/models', () => {
        return HttpResponse.error();
      })
    );

    const res = await validateOpenAI('any-key');
    expect(res.ok).toBe(false);
    expect(res.error).toContain('OpenAI connection failed');
    expect(res.error).not.toContain('any-key'); // Must not leak the key!
  });
});

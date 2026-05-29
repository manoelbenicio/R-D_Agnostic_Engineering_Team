import { describe, it, expect } from 'vitest';
import { http, HttpResponse } from 'msw';
import { server } from '@/api/__tests__/msw/server';
import { validateAzure } from '../../validators/azure';

const endpoint = 'https://test.openai.azure.com';
const modelsUrl = `${endpoint}/openai/models`;
const expectedModelsUrl = `${modelsUrl}?api-version=2024-02-01`;

describe('validateAzure', () => {
  it('succeeds with a valid API key and returns list of models', async () => {
    server.use(
      http.get(modelsUrl, ({ request }) => {
        const apiKey = request.headers.get('api-key');
        expect(request.url).toBe(expectedModelsUrl);
        if (apiKey === 'valid-azure-key') {
          return HttpResponse.json({
            data: [
              { id: 'gpt-4o-azure' },
              { id: 'gpt-4o-mini-azure' },
            ],
          });
        }
        return HttpResponse.json({ error: 'Invalid key' }, { status: 401 });
      })
    );

    const res = await validateAzure(endpoint, 'valid-azure-key');
    expect(res.ok).toBe(true);
    expect(res.models).toEqual(['gpt-4o-azure', 'gpt-4o-mini-azure']);
  });

  it('strips a trailing slash from the endpoint before requesting models', async () => {
    server.use(
      http.get(modelsUrl, ({ request }) => {
        const apiKey = request.headers.get('api-key');
        expect(request.url).toBe(expectedModelsUrl);
        if (apiKey === 'valid-azure-key') {
          return HttpResponse.json({
            data: [
              { id: 'gpt-4o-azure' },
              { id: 'gpt-4o-mini-azure' },
            ],
          });
        }
        return HttpResponse.json({ error: 'Invalid key' }, { status: 401 });
      })
    );

    const res = await validateAzure(`${endpoint}/`, 'valid-azure-key');
    expect(res.ok).toBe(true);
    expect(res.models).toEqual(['gpt-4o-azure', 'gpt-4o-mini-azure']);
  });

  it('fails with a 401 error on invalid key', async () => {
    server.use(
      http.get(modelsUrl, ({ request }) => {
        expect(request.url).toBe(expectedModelsUrl);
        return HttpResponse.json({ error: { message: 'Invalid API key' } }, { status: 401 });
      })
    );

    const res = await validateAzure(endpoint, 'invalid-azure-key');
    expect(res.ok).toBe(false);
    expect(res.error).toContain('Azure validation failed (401)');
    expect(res.error).not.toContain('invalid-azure-key');
  });

  it('fails with connection error on network failure', async () => {
    server.use(
      http.get(modelsUrl, ({ request }) => {
        expect(request.url).toBe(expectedModelsUrl);
        return HttpResponse.error();
      })
    );

    const res = await validateAzure(endpoint, 'azure-network-key');
    expect(res.ok).toBe(false);
    expect(res.error).toContain('Azure connection failed');
    expect(res.error).not.toContain('azure-network-key');
  });
});

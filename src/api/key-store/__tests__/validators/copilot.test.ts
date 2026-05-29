import { describe, it, expect } from 'vitest';
import { http, HttpResponse } from 'msw';
import { server } from '@/api/__tests__/msw/server';
import { validateCopilot } from '../../validators/copilot';

describe('validateCopilot', () => {
  it('succeeds with a valid API key and returns list of models', async () => {
    server.use(
      http.get('https://api.github.com/copilot_internal/v2/token', ({ request }) => {
        const auth = request.headers.get('Authorization');
        const editorVersion = request.headers.get('editor-version');
        if (auth === 'token valid-copilot-key' && editorVersion === 'Neovim/0.9.5') {
          return HttpResponse.json({ token: 'mock-copilot-token' });
        }
        return HttpResponse.json({ message: 'Bad credentials' }, { status: 401 });
      })
    );

    const res = await validateCopilot('valid-copilot-key');
    expect(res.ok).toBe(true);
    expect(res.models).toEqual(['copilot-chat', 'copilot-codex']);
  });

  it('fails with a 401 error on invalid key', async () => {
    server.use(
      http.get('https://api.github.com/copilot_internal/v2/token', () => {
        return HttpResponse.json({ message: 'Bad credentials' }, { status: 401 });
      })
    );

    const res = await validateCopilot('invalid-copilot-key');
    expect(res.ok).toBe(false);
    expect(res.error).toContain('Copilot validation failed (401)');
    expect(res.error).not.toContain('invalid-copilot-key');
  });

  it('fails with connection error on network failure', async () => {
    server.use(
      http.get('https://api.github.com/copilot_internal/v2/token', () => {
        return HttpResponse.error();
      })
    );

    const res = await validateCopilot('copilot-network-key');
    expect(res.ok).toBe(false);
    expect(res.error).toContain('Copilot connection failed');
    expect(res.error).not.toContain('copilot-network-key');
  });
});

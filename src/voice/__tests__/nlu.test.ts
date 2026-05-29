import { beforeEach, describe, expect, it, vi } from 'vitest';
import { extractIntent } from '../nlu';
import { useKeyStore } from '@/api/key-store/store';
import { KeyStore } from '@/api/key-store';
import { appFetch } from '@/shell/app-fetch';

vi.mock('@/api/key-store', () => ({
  KeyStore: {
    get: vi.fn(),
  },
}));

vi.mock('@/shell/app-fetch', () => ({
  appFetch: vi.fn(),
}));

const intent = {
  name: 'Code Review Pipeline',
  nodes: [{ display_name: 'Lead Supervisor', role: 'supervisor', provider: 'kiro_cli' }],
  edges: [],
  confidence: 0.91,
};

const jsonResponse = (body: unknown, ok = true, status = 200) =>
  ({
    ok,
    status,
    json: vi.fn().mockResolvedValue(body),
    text: vi.fn().mockResolvedValue(typeof body === 'string' ? body : JSON.stringify(body)),
  }) as unknown as Response;

describe('extractIntent', () => {
  beforeEach(() => {
    vi.useRealTimers();
    vi.resetAllMocks();
    useKeyStore.setState({
      validated: [],
      cachedModels: {
        google: ['gemini-1.5-flash-latest'],
        openai: ['gpt-4o-mini'],
        anthropic: ['claude-3-5-haiku-20241022'],
      } as any,
    });
  });

  it('uses Google first and parses JSON wrapped in a markdown fence', async () => {
    useKeyStore.setState({ validated: ['google'] as any });
    vi.mocked(KeyStore.get).mockResolvedValue({
      provider: 'google',
      keys: { apiKey: 'google-key' },
      models: ['gemini-1.5-flash-latest'],
      updatedAt: '2026-05-28T00:00:00.000Z',
    } as any);
    vi.mocked(appFetch).mockResolvedValue(
      jsonResponse({
        candidates: [{ content: { parts: [{ text: `\`\`\`json\n${JSON.stringify(intent)}\n\`\`\`` }] } }],
      })
    );

    await expect(extractIntent('cria um supervisor')).resolves.toEqual(intent);

    const [url, options] = vi.mocked(appFetch).mock.calls[0]!;
    expect(String(url)).toContain('generativelanguage.googleapis.com');
    expect(String(url)).toContain('models/gemini-1.5-flash-latest:generateContent?key=google-key');
    expect(options?.method).toBe('POST');
  });

  it('falls back to OpenAI when Google is not validated', async () => {
    useKeyStore.setState({ validated: ['openai'] as any });
    vi.mocked(KeyStore.get).mockResolvedValue({
      provider: 'openai',
      keys: { apiKey: 'openai-key' },
      models: ['gpt-4o-mini'],
      updatedAt: '2026-05-28T00:00:00.000Z',
    } as any);
    vi.mocked(appFetch).mockResolvedValue(
      jsonResponse({
        choices: [{ message: { content: JSON.stringify(intent) } }],
      })
    );

    await expect(extractIntent('build a review team')).resolves.toEqual(intent);

    const [url, options] = vi.mocked(appFetch).mock.calls[0]!;
    expect(url).toBe('https://api.openai.com/v1/chat/completions');
    expect((options?.headers as Record<string, string>).Authorization).toBe('Bearer openai-key');
    expect(JSON.parse(String(options?.body)).model).toBe('gpt-4o-mini');
  });

  it('returns the Anthropic create_canvas tool input directly', async () => {
    useKeyStore.setState({ validated: ['anthropic'] as any });
    vi.mocked(KeyStore.get).mockResolvedValue({
      provider: 'anthropic',
      keys: { apiKey: 'anthropic-key' },
      models: ['claude-3-5-haiku-20241022'],
      updatedAt: '2026-05-28T00:00:00.000Z',
    } as any);
    vi.mocked(appFetch).mockResolvedValue(
      jsonResponse({
        content: [{ type: 'tool_use', name: 'create_canvas', input: intent }],
      })
    );

    await expect(extractIntent('crie um pipeline')).resolves.toEqual(intent);

    const [url, options] = vi.mocked(appFetch).mock.calls[0]!;
    expect(url).toBe('https://api.anthropic.com/v1/messages');
    expect((options?.headers as Record<string, string>)['x-api-key']).toBe('anthropic-key');
    expect(JSON.parse(String(options?.body)).tool_choice).toEqual({ type: 'tool', name: 'create_canvas' });
  });

  it('rejects when no validated LLM provider is available', async () => {
    await expect(extractIntent('anything')).rejects.toThrow('No validated LLM provider available');
    expect(appFetch).not.toHaveBeenCalled();
  });

  it('rejects when the selected provider key is missing', async () => {
    useKeyStore.setState({ validated: ['google'] as any });
    vi.mocked(KeyStore.get).mockResolvedValue({
      provider: 'google',
      keys: {},
      models: [],
      updatedAt: '2026-05-28T00:00:00.000Z',
    } as any);

    await expect(extractIntent('anything')).rejects.toThrow('API key for google is missing');
    expect(appFetch).not.toHaveBeenCalled();
  });

  it('surfaces LLM HTTP failures with status and response text', async () => {
    useKeyStore.setState({ validated: ['openai'] as any });
    vi.mocked(KeyStore.get).mockResolvedValue({
      provider: 'openai',
      keys: { apiKey: 'openai-key' },
      models: [],
      updatedAt: '2026-05-28T00:00:00.000Z',
    } as any);
    vi.mocked(appFetch).mockResolvedValue(jsonResponse('bad request', false, 400));

    await expect(extractIntent('anything')).rejects.toThrow('LLM extraction failed (400): bad request');
  });

  it('maps aborted provider calls to the NLU timeout error', async () => {
    useKeyStore.setState({ validated: ['openai'] as any });
    vi.mocked(KeyStore.get).mockResolvedValue({
      provider: 'openai',
      keys: { apiKey: 'openai-key' },
      models: [],
      updatedAt: '2026-05-28T00:00:00.000Z',
    } as any);
    vi.mocked(appFetch).mockRejectedValue(Object.assign(new Error('aborted'), { name: 'AbortError' }));

    await expect(extractIntent('anything')).rejects.toThrow('NLU_TIMEOUT');
  });
});

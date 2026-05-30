import { beforeEach, describe, expect, it, vi } from 'vitest';

const repoMock = vi.hoisted(() => ({
  recordUsage: vi.fn(),
}));

vi.mock('../usage-repository', () => ({
  recordUsage: repoMock.recordUsage,
}));

import { captureUsageFromPayload } from '../usage-capture';

beforeEach(() => {
  repoMock.recordUsage.mockReset();
  repoMock.recordUsage.mockImplementation(async (usage, context) => ({
    ...usage,
    ...context,
    id: 'evt-1',
    timestampMs: 0,
  }));
});

describe('captureUsageFromPayload', () => {
  it('parses + records when the payload carries a usage block', async () => {
    const event = await captureUsageFromPayload(
      'openai',
      { model: 'gpt-4o', usage: { prompt_tokens: 100, completion_tokens: 50, total_tokens: 150 } },
      { sessionName: 's1', terminalId: 't1', canvasId: 'c1' },
    );

    expect(repoMock.recordUsage).toHaveBeenCalledTimes(1);
    expect(repoMock.recordUsage).toHaveBeenCalledWith(
      { provider: 'openai', model: 'gpt-4o', inputTokens: 100, outputTokens: 50, totalTokens: 150 },
      { sessionName: 's1', terminalId: 't1', canvasId: 'c1' },
    );
    expect(event?.canvasId).toBe('c1');
  });

  it('passes modelId through for AWS payloads without an echoed model', async () => {
    await captureUsageFromPayload(
      'aws',
      { usage: { inputTokens: 10, outputTokens: 5, totalTokens: 15 } },
      {},
      'nova-pro',
    );

    expect(repoMock.recordUsage).toHaveBeenCalledWith(
      { provider: 'aws', model: 'nova-pro', inputTokens: 10, outputTokens: 5, totalTokens: 15 },
      {},
    );
  });

  it('is a no-op (no record) when the payload has no usage block', async () => {
    const event = await captureUsageFromPayload('anthropic', { model: 'claude-3-5-sonnet' });

    expect(event).toBeNull();
    expect(repoMock.recordUsage).not.toHaveBeenCalled();
  });

  it('is a no-op for an empty payload', async () => {
    const event = await captureUsageFromPayload('google', {});

    expect(event).toBeNull();
    expect(repoMock.recordUsage).not.toHaveBeenCalled();
  });
});

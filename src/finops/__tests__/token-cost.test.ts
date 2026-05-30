import { describe, it, expect } from 'vitest';
import {
  parseUsage,
  parseOpenAIUsage,
  parseAnthropicUsage,
  parseGoogleUsage,
  parseAwsUsage,
} from '../token-usage';
import {
  computeTokenCost,
  aggregateTokenCost,
  resolveTokenPrice,
} from '../token-cost';
import type { UsageEvent } from '../token-usage';

describe('token-usage parsers', () => {
  it('parses OpenAI usage', () => {
    const u = parseOpenAIUsage({
      model: 'gpt-4o-2024-08-06',
      usage: { prompt_tokens: 100, completion_tokens: 50, total_tokens: 150 },
    });
    expect(u).toEqual({ provider: 'openai', model: 'gpt-4o-2024-08-06', inputTokens: 100, outputTokens: 50, totalTokens: 150 });
  });

  it('parses Anthropic usage and derives total', () => {
    const u = parseAnthropicUsage({ model: 'claude-3-5-sonnet-latest', usage: { input_tokens: 200, output_tokens: 80 } });
    expect(u?.totalTokens).toBe(280);
    expect(u?.provider).toBe('anthropic');
  });

  it('parses Google usageMetadata', () => {
    const u = parseGoogleUsage({
      modelVersion: 'gemini-1.5-pro',
      usageMetadata: { promptTokenCount: 10, candidatesTokenCount: 5, totalTokenCount: 15 },
    });
    expect(u?.inputTokens).toBe(10);
    expect(u?.outputTokens).toBe(5);
  });

  it('parses AWS Converse and InvokeModel metrics', () => {
    const converse = parseAwsUsage({ usage: { inputTokens: 12, outputTokens: 8, totalTokens: 20 } }, 'nova-pro');
    expect(converse?.model).toBe('nova-pro');
    const invoke = parseAwsUsage({ 'amazon-bedrock-invocationMetrics': { inputTokenCount: 3, outputTokenCount: 2 } }, 'nova-lite');
    expect(invoke?.totalTokens).toBe(5);
  });

  it('returns null when no usage present', () => {
    expect(parseOpenAIUsage({})).toBeNull();
    expect(parseUsage('openai', { nothing: true })).toBeNull();
  });

  it('dispatches by provider', () => {
    expect(parseUsage('anthropic', { usage: { input_tokens: 1, output_tokens: 1 } })?.provider).toBe('anthropic');
  });
});

describe('token-cost', () => {
  it('resolves versioned model ids to a price family', () => {
    expect(resolveTokenPrice('gpt-4o-mini-2024-07-18')?.key).toBe('gpt-4o-mini');
    expect(resolveTokenPrice('claude-3-5-sonnet-20241022')?.key).toBe('claude-3-5-sonnet');
  });

  it('computes measured cost from token counts', () => {
    const res = computeTokenCost({ provider: 'openai', model: 'gpt-4o', inputTokens: 1_000_000, outputTokens: 1_000_000, totalTokens: 2_000_000 });
    expect(res.cost).toBeCloseTo(12.5, 4); // 2.5 + 10
    expect(res.confidence).toBe('measured');
  });

  it('marks unpriced models as estimated with zero cost', () => {
    const res = computeTokenCost({ provider: 'openai', model: 'mystery-model', inputTokens: 100, outputTokens: 100, totalTokens: 200 });
    expect(res.cost).toBe(0);
    expect(res.confidence).toBe('estimated');
  });

  it('aggregates events with measured confidence', () => {
    const agg = aggregateTokenCost([
      event('e1', 'openai', 'gpt-4o', 1_000_000, 0, 'canvas-a'),
      event('e2', 'anthropic', 'claude-3-5-sonnet', 0, 1_000_000, 'canvas-b'),
    ]);
    expect(agg.confidence).toBe('measured');
    expect(agg.byProvider.openai).toBeCloseTo(2.5, 4);
    expect(agg.byProvider.anthropic).toBeCloseTo(15, 4);
    expect(agg.byCanvas['canvas-a']).toBeCloseTo(2.5, 4);
    expect(agg.totalTokens).toBe(2_000_000);
  });

  it('reports mixed confidence when some events are unpriced', () => {
    const agg = aggregateTokenCost([
      event('e1', 'openai', 'gpt-4o', 1_000_000, 0, 'canvas-a'),
      event('e2', 'openai', 'unknown-model', 500, 500, 'canvas-a'),
    ]);
    expect(agg.confidence).toBe('mixed');
    expect(agg.unpricedEvents).toBe(1);
  });

  it('reports estimated confidence for an empty set', () => {
    expect(aggregateTokenCost([]).confidence).toBe('estimated');
  });
});

function event(id: string, provider: UsageEvent['provider'], model: string, input: number, output: number, canvasId: string): UsageEvent {
  return {
    id,
    provider,
    model,
    inputTokens: input,
    outputTokens: output,
    totalTokens: input + output,
    timestampMs: Date.now(),
    canvasId,
  };
}

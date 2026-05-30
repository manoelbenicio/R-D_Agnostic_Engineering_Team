/**
 * src/finops/token-usage.ts
 *
 * FinOps Tier 2 — parse real token usage from provider payloads.
 *
 * v1 Tier 1 (`cost-estimate.ts`) only multiplies terminal wall-clock by a
 * per-hour rate. Tier 2 reads the actual input/output token counts each
 * provider reports so cost reflects real billing units. Providers expose
 * usage under different field names; this module normalises them all into a
 * single `TokenUsage` shape. Pure + dependency-free so it is trivially
 * unit-testable against recorded payloads.
 */

export type UsageProvider = 'openai' | 'anthropic' | 'google' | 'aws';

/** Normalised token usage extracted from a single provider response. */
export interface TokenUsage {
  provider: UsageProvider;
  model: string;
  inputTokens: number;
  outputTokens: number;
  totalTokens: number;
}

/** A persisted usage event tied to a session/terminal/canvas. */
export interface UsageEvent extends TokenUsage {
  id: string;
  timestampMs: number;
  sessionName?: string;
  terminalId?: string;
  canvasId?: string;
}

function num(value: unknown): number {
  return typeof value === 'number' && Number.isFinite(value) ? value : 0;
}

function record(value: unknown): Record<string, unknown> | undefined {
  return value && typeof value === 'object' ? (value as Record<string, unknown>) : undefined;
}

function finalize(
  provider: UsageProvider,
  model: string,
  input: number,
  output: number,
  total?: number,
): TokenUsage | null {
  if (input === 0 && output === 0 && !total) return null;
  return {
    provider,
    model: model || 'unknown',
    inputTokens: input,
    outputTokens: output,
    totalTokens: total && total > 0 ? total : input + output,
  };
}

/** OpenAI: `{ model, usage: { prompt_tokens, completion_tokens, total_tokens } }`. */
export function parseOpenAIUsage(payload: unknown): TokenUsage | null {
  const root = record(payload);
  const usage = record(root?.usage);
  if (!usage) return null;
  return finalize(
    'openai',
    String(root?.model ?? ''),
    num(usage.prompt_tokens),
    num(usage.completion_tokens),
    num(usage.total_tokens),
  );
}

/** Anthropic: `{ model, usage: { input_tokens, output_tokens } }`. */
export function parseAnthropicUsage(payload: unknown): TokenUsage | null {
  const root = record(payload);
  const usage = record(root?.usage);
  if (!usage) return null;
  return finalize(
    'anthropic',
    String(root?.model ?? ''),
    num(usage.input_tokens),
    num(usage.output_tokens),
  );
}

/** Google Gemini: `{ modelVersion, usageMetadata: { promptTokenCount, candidatesTokenCount, totalTokenCount } }`. */
export function parseGoogleUsage(payload: unknown): TokenUsage | null {
  const root = record(payload);
  const usage = record(root?.usageMetadata);
  if (!usage) return null;
  return finalize(
    'google',
    String(root?.modelVersion ?? root?.model ?? ''),
    num(usage.promptTokenCount),
    num(usage.candidatesTokenCount),
    num(usage.totalTokenCount),
  );
}

/**
 * AWS Bedrock: Converse API `{ usage: { inputTokens, outputTokens, totalTokens } }`
 * or InvokeModel metrics `{ "amazon-bedrock-invocationMetrics": { inputTokenCount, outputTokenCount } }`.
 * `modelId` is passed separately because Bedrock responses don't echo it.
 */
export function parseAwsUsage(payload: unknown, modelId = ''): TokenUsage | null {
  const root = record(payload);
  const converse = record(root?.usage);
  if (converse) {
    return finalize(
      'aws',
      modelId,
      num(converse.inputTokens),
      num(converse.outputTokens),
      num(converse.totalTokens),
    );
  }
  const metrics = record(root?.['amazon-bedrock-invocationMetrics']);
  if (metrics) {
    return finalize('aws', modelId, num(metrics.inputTokenCount), num(metrics.outputTokenCount));
  }
  return null;
}

/** Dispatch to the right parser by provider. */
export function parseUsage(
  provider: UsageProvider,
  payload: unknown,
  modelId?: string,
): TokenUsage | null {
  switch (provider) {
    case 'openai':
      return parseOpenAIUsage(payload);
    case 'anthropic':
      return parseAnthropicUsage(payload);
    case 'google':
      return parseGoogleUsage(payload);
    case 'aws':
      return parseAwsUsage(payload, modelId);
    default:
      return null;
  }
}

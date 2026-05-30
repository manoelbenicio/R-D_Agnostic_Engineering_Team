/**
 * src/finops/token-cost.ts
 *
 * FinOps Tier 2 — convert normalised `TokenUsage` / `UsageEvent`s into real
 * cost using per-model input/output token prices, and report a confidence
 * level so the UI can distinguish measured cost from the Tier 1 wall-clock
 * estimate.
 */

import type { TokenUsage, UsageEvent } from './token-usage';

/** USD price per 1M tokens, split input/output. */
export interface TokenPrice {
  inputPerM: number;
  outputPerM: number;
}

/**
 * Per-model price table (USD / 1M tokens). Keys are normalised, lowercased
 * model substrings matched longest-first so versioned ids
 * (`gpt-4o-2024-08-06`) resolve to their family entry. Figures are list
 * prices as of authoring; revisit alongside provider price changes.
 */
export const TOKEN_PRICES: Record<string, TokenPrice> = {
  'gpt-4o-mini': { inputPerM: 0.15, outputPerM: 0.6 },
  'gpt-4o': { inputPerM: 2.5, outputPerM: 10 },
  'gpt-4-turbo': { inputPerM: 10, outputPerM: 30 },
  'o1-mini': { inputPerM: 3, outputPerM: 12 },
  'o1': { inputPerM: 15, outputPerM: 60 },
  'claude-3-5-haiku': { inputPerM: 0.8, outputPerM: 4 },
  'claude-3-5-sonnet': { inputPerM: 3, outputPerM: 15 },
  'claude-3-opus': { inputPerM: 15, outputPerM: 75 },
  'gemini-1.5-flash': { inputPerM: 0.075, outputPerM: 0.3 },
  'gemini-1.5-pro': { inputPerM: 1.25, outputPerM: 5 },
  'gemini-2.0-flash': { inputPerM: 0.1, outputPerM: 0.4 },
  'nova-micro': { inputPerM: 0.035, outputPerM: 0.14 },
  'nova-lite': { inputPerM: 0.06, outputPerM: 0.24 },
  'nova-pro': { inputPerM: 0.8, outputPerM: 3.2 },
  // New model families — placeholder list prices (no public pricing yet);
  // revisit when providers publish official rates. Keyed as substrings so
  // versioned ids (opus-4.6/4.7/4.8) resolve to the family entry.
  'codex-5.5': { inputPerM: 2.5, outputPerM: 10 },
  'gemini-3.5-flash': { inputPerM: 0.1, outputPerM: 0.4 },
  'opus-4': { inputPerM: 15, outputPerM: 75 },
};

/** `measured` = from token usage; `estimated` = Tier 1 fallback; `mixed` = both. */
export type CostConfidence = 'measured' | 'estimated' | 'mixed';

export interface TokenCostResult {
  cost: number;
  confidence: CostConfidence;
  /** Model key matched in `TOKEN_PRICES`, or undefined when unpriced. */
  matchedModel?: string;
}

/** Resolve a (possibly versioned) model id to a price, longest key first. */
export function resolveTokenPrice(model: string): { key: string; price: TokenPrice } | undefined {
  const normalized = model.toLowerCase();
  const keys = Object.keys(TOKEN_PRICES).sort((a, b) => b.length - a.length);
  for (const key of keys) {
    if (normalized.includes(key)) {
      return { key, price: TOKEN_PRICES[key] as TokenPrice };
    }
  }
  return undefined;
}

function round(value: number): number {
  return Math.round(value * 10000) / 10000;
}

/** Cost of a single normalised usage record. `measured` when the model is priced. */
export function computeTokenCost(usage: TokenUsage): TokenCostResult {
  const resolved = resolveTokenPrice(usage.model);
  if (!resolved) {
    return { cost: 0, confidence: 'estimated' };
  }
  const cost =
    (usage.inputTokens / 1_000_000) * resolved.price.inputPerM +
    (usage.outputTokens / 1_000_000) * resolved.price.outputPerM;
  return { cost: round(cost), confidence: 'measured', matchedModel: resolved.key };
}

export interface AggregatedTokenCost {
  total: number;
  byProvider: Record<string, number>;
  byCanvas: Record<string, number>;
  byModel: Record<string, number>;
  confidence: CostConfidence;
  /** input + output tokens summed across all events. */
  totalTokens: number;
  /** number of events whose model could not be priced. */
  unpricedEvents: number;
}

/**
 * Aggregate persisted usage events into Tier 2 cost broken down by provider,
 * canvas and model. Confidence is `measured` when every event was priced,
 * `estimated` when none were, `mixed` otherwise.
 */
export function aggregateTokenCost(events: UsageEvent[]): AggregatedTokenCost {
  const result: AggregatedTokenCost = {
    total: 0,
    byProvider: {},
    byCanvas: {},
    byModel: {},
    confidence: 'estimated',
    totalTokens: 0,
    unpricedEvents: 0,
  };

  let priced = 0;
  for (const event of events) {
    result.totalTokens += event.totalTokens;
    const { cost, matchedModel } = computeTokenCost(event);
    if (!matchedModel) {
      result.unpricedEvents += 1;
      continue;
    }
    priced += 1;
    result.total += cost;
    result.byProvider[event.provider] = (result.byProvider[event.provider] ?? 0) + cost;
    const canvas = event.canvasId ?? 'unassigned';
    result.byCanvas[canvas] = (result.byCanvas[canvas] ?? 0) + cost;
    result.byModel[matchedModel] = (result.byModel[matchedModel] ?? 0) + cost;
  }

  result.total = round(result.total);
  result.byProvider = roundRecord(result.byProvider);
  result.byCanvas = roundRecord(result.byCanvas);
  result.byModel = roundRecord(result.byModel);

  if (events.length === 0 || priced === 0) {
    result.confidence = 'estimated';
  } else if (result.unpricedEvents === 0) {
    result.confidence = 'measured';
  } else {
    result.confidence = 'mixed';
  }

  return result;
}

function roundRecord(record: Record<string, number>): Record<string, number> {
  return Object.fromEntries(Object.entries(record).map(([k, v]) => [k, round(v)]));
}

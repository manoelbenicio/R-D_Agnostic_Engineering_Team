import { PROVIDER_COST_PER_HOUR, type CostProvider } from './cost-constants';

export interface CostWindow {
  startMs: number;
  endMs: number;
}

export interface CostTerminal {
  id: string;
  provider?: string;
  started_at?: string | number | Date | null;
  stopped_at?: string | number | Date | null;
  created_at?: string | number | Date | null;
  last_active?: string | number | Date | null;
  canvas_id?: string | null;
  canvasId?: string | null;
  session_name?: string | null;
}

export interface CostEstimate {
  total: number;
  byProvider: Record<string, number>;
  byCanvas: Record<string, number>;
}

export function computeCostEstimate(window: CostWindow, terminals: CostTerminal[]): CostEstimate {
  const estimate: CostEstimate = {
    total: 0,
    byProvider: {},
    byCanvas: {},
  };

  for (const terminal of terminals) {
    const provider = terminal.provider;
    if (!provider || !isCostProvider(provider)) continue;

    const startedMs = readTimestamp(terminal.started_at ?? terminal.last_active ?? terminal.created_at);
    if (startedMs === undefined) continue;

    const stoppedMs = readTimestamp(terminal.stopped_at) ?? window.endMs;
    const activeMs = Math.min(stoppedMs, window.endMs) - Math.max(startedMs, window.startMs);
    const hours = Math.max(0, activeMs) / 3_600_000;
    const cost = hours * PROVIDER_COST_PER_HOUR[provider];
    const canvasId = terminal.canvas_id ?? terminal.canvasId ?? terminal.session_name ?? 'unassigned';

    estimate.total += cost;
    estimate.byProvider[provider] = (estimate.byProvider[provider] ?? 0) + cost;
    estimate.byCanvas[canvasId] = (estimate.byCanvas[canvasId] ?? 0) + cost;
  }

  return {
    total: roundCurrency(estimate.total),
    byProvider: roundRecord(estimate.byProvider),
    byCanvas: roundRecord(estimate.byCanvas),
  };
}

export function isCostProvider(provider: string): provider is CostProvider {
  return provider in PROVIDER_COST_PER_HOUR;
}

function readTimestamp(value: string | number | Date | null | undefined): number | undefined {
  if (value === null || value === undefined) return undefined;
  if (typeof value === 'number') return Number.isFinite(value) ? value : undefined;
  if (value instanceof Date) return value.getTime();
  const parsed = Date.parse(value);
  return Number.isNaN(parsed) ? undefined : parsed;
}

function roundRecord(record: Record<string, number>): Record<string, number> {
  return Object.fromEntries(Object.entries(record).map(([key, value]) => [key, roundCurrency(value)]));
}

function roundCurrency(value: number): number {
  return Math.round(value * 100) / 100;
}

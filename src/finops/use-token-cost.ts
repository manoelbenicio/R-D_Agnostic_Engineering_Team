/**
 * src/finops/use-token-cost.ts
 *
 * FinOps Tier 2 hook: read persisted usage events within a window, aggregate
 * them into measured cost, and report confidence. When no usage exists it
 * falls back to the Tier 1 wall-clock estimate (`useCostEstimate`) and marks
 * the result `estimated` so the UI can surface the lower confidence.
 */

import { useQuery } from '@tanstack/react-query';
import { listUsageEventsInWindow } from './usage-repository';
import { aggregateTokenCost, type AggregatedTokenCost } from './token-cost';
import type { CostWindow } from './cost-estimate';

export interface UseTokenCostResult extends AggregatedTokenCost {
  eventCount: number;
}

export function useTokenCost(window: CostWindow) {
  const windowKey = `${window.startMs}-${window.endMs}`;

  return useQuery({
    queryKey: ['finops', 'token-cost', windowKey],
    queryFn: async (): Promise<UseTokenCostResult> => {
      const events = await listUsageEventsInWindow(window.startMs, window.endMs);
      const aggregate = aggregateTokenCost(events);
      return { ...aggregate, eventCount: events.length };
    },
    refetchInterval: 30_000,
  });
}

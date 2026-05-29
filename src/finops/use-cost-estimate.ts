import { useQuery } from '@tanstack/react-query';
import { caoClient } from '@/api';
import type { Session, Terminal } from '@/api/types';
import { computeCostEstimate, type CostEstimate, type CostTerminal, type CostWindow } from './cost-estimate';

interface SessionWithCanvas extends Session {
  canvas_id?: string;
  canvasId?: string;
}

interface TerminalWithCostFields extends Terminal {
  started_at?: string | number | Date | null;
  stopped_at?: string | number | Date | null;
  last_active?: string | number | Date | null;
  canvas_id?: string | null;
  canvasId?: string | null;
}

export interface UseCostEstimateResult extends CostEstimate {
  terminals: CostTerminal[];
  activeTerminalsCount: number;
  currentHourlyRate: number;
}

export function useCostEstimate(window: CostWindow) {
  const windowKey = `${window.startMs}-${window.endMs}`;

  return useQuery({
    queryKey: ['finops', 'estimate', windowKey],
    queryFn: async (): Promise<UseCostEstimateResult> => {
      const sessions = (await caoClient.listSessions()) as SessionWithCanvas[];
      const terminalGroups = await Promise.all(
        sessions.map(async (session) => {
          const terminals = (await caoClient.listTerminalsInSession(session.name)) as TerminalWithCostFields[];
          return terminals.map((terminal) => ({
            ...terminal,
            canvas_id: terminal.canvas_id ?? terminal.canvasId ?? session.canvas_id ?? session.canvasId ?? session.name,
            session_name: terminal.session_name ?? session.name,
          }));
        })
      );
      const terminals = terminalGroups.flat();
      const estimate = computeCostEstimate(window, terminals);

      return {
        ...estimate,
        terminals,
        activeTerminalsCount: terminals.filter((terminal) => !terminal.stopped_at && terminal.status !== 'exited').length,
        currentHourlyRate: computeCurrentHourlyRate(terminals),
      };
    },
    refetchInterval: 30_000,
  });
}

export function selectCostByProvider(result?: UseCostEstimateResult): Record<string, number> {
  return result?.byProvider ?? {};
}

export function selectCostByCanvas(result?: UseCostEstimateResult): Record<string, number> {
  return result?.byCanvas ?? {};
}

function computeCurrentHourlyRate(terminals: CostTerminal[]): number {
  const estimate = computeCostEstimate(
    { startMs: 0, endMs: 3_600_000 },
    terminals
      .filter((terminal) => !terminal.stopped_at)
      .map((terminal) => ({
        ...terminal,
        started_at: 0,
        stopped_at: 3_600_000,
      }))
  );

  return estimate.total;
}

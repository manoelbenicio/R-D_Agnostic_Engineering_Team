import { describe, expect, it } from 'vitest';
import { computeCostEstimate, type CostTerminal } from '../cost-estimate';

describe('computeCostEstimate', () => {
  it('sums a mixed-provider window with exact provider keys', () => {
    const startMs = Date.UTC(2026, 4, 1, 0, 0, 0);
    const terminals: CostTerminal[] = [
      terminal('claude', 'claude_code', startMs, startMs + 2 * 3_600_000, 'canvas-a'),
      terminal('codex', 'codex', startMs, startMs + 1 * 3_600_000, 'canvas-a'),
      terminal('gemini', 'gemini_cli', startMs, startMs + 0.5 * 3_600_000, 'canvas-b'),
    ];

    const estimate = computeCostEstimate({ startMs, endMs: startMs + 3 * 3_600_000 }, terminals);

    expect(estimate.total).toBe(35.25);
    expect(estimate.byProvider).toEqual({
      claude_code: 30,
      codex: 5,
      gemini_cli: 0.25,
    });
    expect(estimate.byCanvas).toEqual({
      'canvas-a': 35,
      'canvas-b': 0.25,
    });
  });

  it('counts only the partial overlap inside the window', () => {
    const startMs = Date.UTC(2026, 4, 1, 10, 0, 0);
    const estimate = computeCostEstimate(
      { startMs, endMs: startMs + 3_600_000 },
      [terminal('partial', 'codex', startMs - 30 * 60_000, startMs + 30 * 60_000, 'canvas-partial')]
    );

    expect(estimate.total).toBe(2.5);
    expect(estimate.byProvider).toEqual({ codex: 2.5 });
  });

  it('caps a still-active terminal at endMs when stopped_at is absent', () => {
    const startMs = Date.UTC(2026, 4, 1, 12, 0, 0);
    const estimate = computeCostEstimate(
      { startMs, endMs: startMs + 2 * 3_600_000 },
      [
        {
          id: 'active',
          provider: 'gemini_cli',
          started_at: startMs,
          canvas_id: 'canvas-active',
        },
      ]
    );

    expect(estimate.total).toBe(1);
    expect(estimate.byProvider).toEqual({ gemini_cli: 1 });
  });
});

function terminal(
  id: string,
  provider: string,
  startedAt: number,
  stoppedAt: number,
  canvasId: string
): CostTerminal {
  return {
    id,
    provider,
    started_at: startedAt,
    stopped_at: stoppedAt,
    canvas_id: canvasId,
  };
}

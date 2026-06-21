/**
 * src/finops/usage-capture.ts
 *
 * FinOps Tier 2 — capture wiring (finops-tier2-token-parsing §6.2).
 *
 * `captureUsageFromPayload` is the single, finops-owned entry point that turns
 * a completed provider turn into a persisted Tier 2 usage event. It is
 * deliberately conditional: it parses the payload and only records when a real
 * usage block is present. When the payload carries no usage (or the provider
 * is unknown) it is a no-op and resolves to `null`, so the Tier 1 wall-clock
 * estimate (`cost-estimate.ts`) stays the fallback. No usage figures are ever
 * synthesised here.
 *
 * BLOCKED (GO Core side): there is currently no call site in the SPA. Terminal
 * output arrives as raw binary xterm frames over the WebSocket
 * (`src/api/connect-terminal-socket.ts` → `src/terminal/use-terminal-stream.ts`);
 * the runtime never surfaces a structured per-response `usage` block. Wiring
 * the call requires GO Core to expose provider usage per terminal turn (response
 * body or a dedicated usage event) — see tasks.md §6.1. Once it does, the
 * cross-capability seam in `@/shared/usage-capture-bus` (registered by the
 * shell adapter) should delegate to this function from the terminal pipeline.
 */

import { parseUsage, type UsageEvent, type UsageProvider } from './token-usage';
import { recordUsage } from './usage-repository';

export interface UsageCaptureContext {
  sessionName?: string;
  terminalId?: string;
  canvasId?: string;
}

/**
 * Parse a provider response payload and, only when it carries token usage,
 * persist a Tier 2 usage event. Returns the stored event, or `null` when there
 * is no usage to record (Tier 1 fallback).
 */
export async function captureUsageFromPayload(
  provider: UsageProvider,
  payload: unknown,
  context: UsageCaptureContext = {},
  modelId?: string,
): Promise<UsageEvent | null> {
  const usage = parseUsage(provider, payload, modelId);
  if (!usage) return null; // no usage block → Tier 1 fallback, no synthetic data.
  return recordUsage(usage, context);
}

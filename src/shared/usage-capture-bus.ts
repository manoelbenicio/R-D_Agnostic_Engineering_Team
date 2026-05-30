/**
 * src/shared/usage-capture-bus.ts
 *
 * FinOps Tier 2 capture seam (D9). Decouples the terminal-stream pipeline
 * (`src/terminal/`, `src/api/`) from `src/finops/` so a completed turn can be
 * recorded WITHOUT a sideways capability import (which the
 * `agentverse/no-sideways-capability-imports` rule forbids).
 *
 * The concrete implementation lives in `src/shell/usage-capture-adapter.ts`
 * and is registered at bootstrap. Until then `getUsageCaptureBus()` returns a
 * no-op so callers degrade silently to the Tier 1 wall-clock estimate.
 *
 * Why it lives in `@/shared/`: it is the only cross-capability,
 * dependency-free layer. The union below intentionally duplicates
 * `finops/token-usage`'s `UsageProvider` so consumers stay finops-free
 * (same approach as `ProviderOption` in `canvas-command-bus.ts`).
 */

/** Providers whose usage payloads `parseUsage` can normalise. */
export type UsageCaptureProvider = 'openai' | 'anthropic' | 'google' | 'aws';

/** Session/terminal/canvas a usage event is attributed to. */
export interface UsageCaptureContext {
  sessionName?: string;
  terminalId?: string;
  canvasId?: string;
}

export interface UsageCaptureBus {
  /**
   * Parse a provider response payload and, when it carries token usage,
   * persist it as a Tier 2 usage event. Resolves to a no-op (Tier 1
   * fallback) when the payload has no usage block.
   */
  captureTurnUsage(
    provider: UsageCaptureProvider,
    payload: unknown,
    context?: UsageCaptureContext,
    modelId?: string,
  ): Promise<void>;
}

const USAGE_CAPTURE_PROVIDERS: ReadonlySet<string> = new Set<UsageCaptureProvider>([
  'openai',
  'anthropic',
  'google',
  'aws',
]);

/** Narrow an arbitrary string to a `UsageCaptureProvider`. */
export function isUsageCaptureProvider(value: unknown): value is UsageCaptureProvider {
  return typeof value === 'string' && USAGE_CAPTURE_PROVIDERS.has(value);
}

/** Default bus: records nothing, so consumers fall back to Tier 1. */
export const noopUsageCaptureBus: UsageCaptureBus = {
  async captureTurnUsage() {
    // intentional no-op — Tier 1 wall-clock estimate remains the source.
  },
};

let activeBus: UsageCaptureBus = noopUsageCaptureBus;

/** Register the concrete bus (called once at bootstrap by the shell). */
export function setUsageCaptureBus(bus: UsageCaptureBus): void {
  activeBus = bus;
}

/** The currently-registered bus (no-op until the shell wires the adapter). */
export function getUsageCaptureBus(): UsageCaptureBus {
  return activeBus;
}

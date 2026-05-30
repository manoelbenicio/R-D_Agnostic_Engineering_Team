/* eslint-disable agentverse/no-sideways-capability-imports --
 * Documented cross-capability seam for FinOps Tier 2 capture wiring
 * (finops-tier2-token-parsing §6.2). This is the ONLY module allowed to
 * bridge the terminal-stream pipeline to `src/finops/`; consumers go through
 * the `UsageCaptureBus` in `@/shared/usage-capture-bus`. The lint rule is
 * suppressed here so it stays loud everywhere else.
 */

import type {
  UsageCaptureBus,
  UsageCaptureContext,
  UsageCaptureProvider,
} from '@/shared/usage-capture-bus';
import { parseUsage } from '@/finops/token-usage';
import { recordUsage } from '@/finops/usage-repository';

/**
 * Concrete bus: parse the provider payload and, only when it carries a usage
 * block, persist a Tier 2 usage event. A null parse (no usage present) is a
 * no-op so the Tier 1 wall-clock estimate stays the fallback.
 */
export const usageCaptureBus: UsageCaptureBus = {
  async captureTurnUsage(
    provider: UsageCaptureProvider,
    payload: unknown,
    context: UsageCaptureContext = {},
    modelId?: string,
  ): Promise<void> {
    const usage = parseUsage(provider, payload, modelId);
    if (!usage) return; // no usage block → Tier 1 fallback.
    await recordUsage(usage, context);
  },
};

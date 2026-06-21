/* eslint-disable agentverse/no-sideways-capability-imports --
 * The canvas command adapter is the documented cross-capability seam for
 * the `tech-debt-voice-event-bus` change. It is the ONLY module allowed to
 * import `canvas-builder` + `canvas-reconciler` simultaneously; consumers
 * (voice today, future capabilities tomorrow) go through the
 * `CanvasCommandBus` interface defined in `@/shared/canvas-command-bus`.
 * The lint rule is intentionally suppressed here so it stays loud
 * everywhere else.
 */

import type {
  CanvasCommandBus,
  CanvasValidationResult,
  ProviderOption,
} from '@/shared/canvas-command-bus';
import type { CanvasDocument } from '@/shared/canvas-types';

import { reconcileCanvas } from '@/canvas-reconciler/reconciler';
import { getCanvasProviderOptions } from '@/canvas-builder/provider-options';
import { validateCanvasForDeploy } from '@/canvas-builder/deploy-validation';
import { useKeyStore } from '@/api/key-store/store';
import { goCoreClient } from '@/api';

/**
 * Wrap the canvas-builder validator into the bus's `CanvasValidationResult`
 * shape. `validateCanvasForDeploy` returns `{ ok, reason?: string }`; the
 * bus contract uses `{ ok, reasons: string[], blockingNodeId? }` so that
 * future validators can surface multiple failures.
 */
function adaptValidation(canvas: CanvasDocument): CanvasValidationResult {
  const { validated } = useKeyStore.getState();
  const options = getCanvasProviderOptions(validated);
  const result = validateCanvasForDeploy(canvas, options, validated);
  if (result.ok) {
    return { ok: true, reasons: [] };
  }
  return {
    ok: false,
    reasons: result.reason ? [result.reason] : [],
  };
}

/**
 * Map the canvas-builder `CanvasProviderOption[]` into the dependency-free
 * `ProviderOption[]` exposed by the bus. Models come from the key-store's
 * `cachedModels` keyed by the source provider.
 */
function adaptProviderOptions(): ProviderOption[] {
  const { validated, cachedModels } = useKeyStore.getState();
  return getCanvasProviderOptions(validated).map((option) => ({
    id: option.provider,
    label: option.label,
    models: cachedModels[option.sourceProvider] ?? [],
  }));
}

/**
 * Concrete bus singleton wired to the real canvas-builder and
 * canvas-reconciler implementations. Imported by `src/voice/VoicePanel.tsx`
 * (and any future cross-capability consumer) via
 * `import { canvasCommandBus } from '@/shell/canvas-command-adapter'`.
 */
export const canvasCommandBus: CanvasCommandBus = {
  validateForDeploy: adaptValidation,
  reconcile: (canvasId: string) => reconcileCanvas(canvasId, undefined, goCoreClient),
  getProviderOptions: adaptProviderOptions,
};

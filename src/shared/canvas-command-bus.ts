/**
 * src/shared/canvas-command-bus.ts
 *
 * Cross-capability seam (D9) decoupling consumers like `src/voice/` from
 * direct imports of `canvas-builder` + `canvas-reconciler`. Every capability
 * that needs to perform a canvas command (validate, reconcile, enumerate
 * provider options) goes through this bus. The concrete implementation
 * lives in `src/shell/canvas-command-adapter.ts`.
 *
 * Why this lives in `@/shared/`:
 *   - `@/shared/` is the only cross-capability dependency-free layer (D9).
 *   - Consumers can `import type { CanvasCommandBus } from '@/shared/canvas-command-bus'`
 *     without inheriting any concrete `canvas-builder` / `canvas-reconciler`
 *     dependency — and without having to suppress the
 *     `agentverse/no-sideways-capability-imports` lint rule.
 *
 * The shell adapter is the only file allowed to import `canvas-builder` +
 * `canvas-reconciler` simultaneously (and is the documented exception to the
 * sideways-capability-imports rule).
 */

import type { CanvasDocument } from './canvas-types';

/**
 * Provider option surfaced through the bus.
 *
 * Mirrors the data exposed by `canvas-builder/provider-options` but in a
 * dependency-free shape so callers outside `canvas-builder` can consume it
 * without taking on a sideways capability dependency.
 */
export interface ProviderOption {
  /** Canvas-side provider id (e.g. `'codex'`, `'claude_code'`, `'q_cli'`). */
  id: string;
  /** Human-readable label. */
  label: string;
  /** Models available for this provider option. May be empty. */
  models: string[];
}

/**
 * Result of pre-deploy validation.
 *
 * `reasons` aggregates one or more failure messages so callers can present
 * them coherently. `blockingNodeId` is populated when a single canvas node
 * is at fault — consumers may use it to highlight that node in the UI.
 */
export interface CanvasValidationResult {
  ok: boolean;
  reasons: string[];
  blockingNodeId?: string;
}

/**
 * The bus contract — the three operations canvas consumers (voice today,
 * future flows / templates / etc.) need to perform against the canvas
 * authoring + reconciler subsystems without importing them directly.
 */
export interface CanvasCommandBus {
  /** Validate that a canvas is ready for deployment. */
  validateForDeploy(canvas: CanvasDocument): CanvasValidationResult;
  /** Run the reconciler for the canvas with the given id. */
  reconcile(canvasId: string): Promise<CanvasDocument>;
  /** List provider options available given the current key-store state. */
  getProviderOptions(): ProviderOption[];
}

/**
 * Convenience implementation for tests that wire the executor without
 * exercising the canvas subsystem. Every method throws so accidental usage
 * fails loudly rather than silently returning sentinel data.
 */
export const noopCanvasCommandBus: CanvasCommandBus = {
  validateForDeploy() {
    throw new Error('noopCanvasCommandBus.validateForDeploy was called');
  },
  reconcile() {
    throw new Error('noopCanvasCommandBus.reconcile was called');
  },
  getProviderOptions() {
    throw new Error('noopCanvasCommandBus.getProviderOptions was called');
  },
};

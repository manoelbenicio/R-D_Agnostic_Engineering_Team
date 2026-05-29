# tech-debt-voice-event-bus — Implementation Tasks

> Owners: **SUP** (`src/shared/`, `src/shell/`) + **VX** (`src/voice/`).
> Sequencing: Wave B step 2. **Depends on `tech-debt-voice-coverage-gap`** completing first because both modify `src/voice/VoicePanel.tsx` and the bus refactor builds on the already-extracted `command-executor.ts`.
> Goal: remove the three `eslint-disable agentverse/no-sideways-capability-imports` directives from `VoicePanel.tsx` and `command-executor.ts` by routing canvas-builder + canvas-reconciler access through a `CanvasCommandBus` defined in `src/shared/`.

## 1. Define the bus contract (SUP)

- [x] 1.1 Create `src/shared/canvas-command-bus.ts`:
  ```ts
  import type { CanvasDocument } from './canvas-types';

  export interface ProviderOption {
    id: string;
    label: string;
    models: string[];
  }

  export interface CanvasValidationResult {
    ok: boolean;
    reasons: string[];
    blockingNodeId?: string;
  }

  export interface CanvasCommandBus {
    validateForDeploy(canvas: CanvasDocument): CanvasValidationResult;
    reconcile(canvasId: string): Promise<CanvasDocument>;
    getProviderOptions(): ProviderOption[];
  }
  ```
  Keep this file dependency-free (only types from `@/shared/`); it is the seam.

- [x] 1.2 Add a default export `noopCanvasCommandBus` for tests that don't exercise the bus:
  ```ts
  export const noopCanvasCommandBus: CanvasCommandBus = { /* throws on every call */ };
  ```

## 2. Adapter implementation (SUP)

- [x] 2.1 Create `src/shell/canvas-command-adapter.ts` that wires the real implementations from canvas-builder and canvas-reconciler into the `CanvasCommandBus` interface:
  ```ts
  import { reconcileCanvas } from '@/canvas-reconciler/reconciler';
  import { getCanvasProviderOptions } from '@/canvas-builder/provider-options';
  import { validateCanvasForDeploy } from '@/canvas-builder/deploy-validation';

  export const canvasCommandBus: CanvasCommandBus = {
    validateForDeploy: validateCanvasForDeploy,
    reconcile: (id) => reconcileCanvas(id),
    getProviderOptions: getCanvasProviderOptions,
  };
  ```
- [x] 2.2 The adapter is the **only** module allowed to import from canvas-builder + canvas-reconciler simultaneously. Verify the lint rule still flags any other cross-capability sideways import.

## 3. Voice consumer refactor (VX)

- [x] 3.1 Update `src/voice/command-executor.ts` so its `CommandExecutorDeps` includes `bus: CanvasCommandBus` instead of the three direct function parameters (`reconcile`, `validateForDeploy`, plus any provider-option lookup). Internal call sites become `deps.bus.reconcile(...)` etc.
- [x] 3.2 Update `src/voice/VoicePanel.tsx`:
  - Remove imports from `@/canvas-reconciler/reconciler`, `@/canvas-builder/provider-options`, `@/canvas-builder/deploy-validation`
  - Add `import { canvasCommandBus } from '@/shell/canvas-command-adapter'`
  - Pass `bus: canvasCommandBus` into the `executeRuntimeCommand(...)` deps object
- [x] 3.3 Remove all three `eslint-disable agentverse/no-sideways-capability-imports` directives from `VoicePanel.tsx` and `command-executor.ts`
- [x] 3.4 Confirm `npm run lint` passes with the directives gone (the rule should be satisfied)

## 4. Tests (VX + SUP)

- [x] 4.1 In existing `src/voice/__tests__/command-executor.test.ts`, replace the per-test stubs of `reconcile` / `validateForDeploy` with a fake `CanvasCommandBus` object. The shape change should be invisible to assertions (same coverage profile maintained from the previous tech-debt change).
- [x] 4.2 Add `src/shell/__tests__/canvas-command-adapter.test.ts` asserting the adapter wires the real implementations (use small contract tests with mocks).
- [x] 4.3 Lint rule `agentverse/no-sideways-capability-imports` continues to flag a fresh sideways import added in any other capability (regression test stays in eslint-rules tests).

## 5. Verify (SUP)

- [x] 5.1 `npm run lint` passes with **zero** `eslint-disable agentverse/no-sideways-capability-imports` directives in `src/voice/`
- [x] 5.2 `npm run typecheck` clean
- [x] 5.3 `npm test` green (voice + shell suites)
- [x] 5.4 Coverage from `tech-debt-voice-coverage-gap` (≥70% in `src/voice/`) is preserved or improved

## Out of Scope

- Adding bus methods beyond the three defined in §1.1 (extend in a follow-up if voice gains new canvas integrations)
- Migrating other capabilities (dashboard, finops) onto a bus — they don't currently violate the import boundary
- Hot-swappable bus implementations (single-instance is enough for v1)

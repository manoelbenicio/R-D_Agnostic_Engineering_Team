## Why

`VoicePanel.tsx` directly imports from `@/canvas-builder/deploy-validation`, `@/canvas-builder/provider-options`, and `@/canvas-reconciler/reconciler`. Per the master spec §14.2, cross-capability communication should use an event bus or shared interface — not direct sideways imports. These imports create tight coupling between the voice module and the canvas authoring modules, making it harder to test, refactor, or replace either module independently.

The violation is explicitly marked with `eslint-disable agentverse/no-sideways-capability-imports` comments, which flag the technical debt for future resolution.

## What Changes

1. Define a `CanvasCommandBus` interface in `src/shared/` with methods:
   - `validateForDeploy(canvasId: string): ValidationResult`
   - `reconcile(canvasId: string): Promise<CanvasDocument>`
   - `getProviderOptions(): ProviderOption[]`
2. Implement the bus in `src/shell/` (or a thin adapter layer) that delegates to the actual canvas-builder and canvas-reconciler modules.
3. Refactor `VoicePanel` to consume `CanvasCommandBus` instead of importing directly.
4. Remove the `eslint-disable` comments.

### Scope

- New: `src/shared/canvas-command-bus.ts` (interface)
- New: `src/shell/canvas-command-adapter.ts` (implementation)
- Modify: `src/voice/VoicePanel.tsx` (replace direct imports with bus)
- Remove: 3 `eslint-disable` comments

## Why Post-v1

The direct imports work correctly and the lint rule is suppressed. Introducing an event bus mid-sprint risks destabilizing both voice and canvas modules during the close phase. The coupling is documented via lint suppressions and this proposal; the refactor is better suited for a maintenance sprint where both modules can be tested end-to-end.

## Impact

- **Code**: 2 new files, 1 modified
- **Risk**: Medium — changing the voice→canvas communication contract requires re-testing the full voice command flow
- **Dependencies**: Requires both voice and canvas modules to be unlocked

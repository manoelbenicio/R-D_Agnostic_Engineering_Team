# tech-debt-voice-coverage-gap — Implementation Tasks

> Owner: **VX** (Voice Dev — `src/voice/`).
> Sequencing: Wave B step 1. Must complete **before** `tech-debt-voice-event-bus` because both modify `VoicePanel.tsx` and the event-bus refactor is easier once the executor is already extracted.
> Goal: raise `src/voice/` statement coverage from ~40.78% to ≥70% (matches the milestone-1 §22.1 gate for logic-heavy modules).

## 1. Extract the command executor (VX)

`VoicePanel.tsx` currently embeds runtime command dispatch inside a React component, which makes the 9-action × N-error-path matrix awkward to test.

- [x] 1.1 Create `src/voice/command-executor.ts` exporting a single async function:
  ```ts
  export interface CommandExecutorDeps {
    canvas: CanvasDocument | null;
    cao: CaoClient;
    toast: ToastApi;
    navigate: NavigateFn;
    confirm: (opts: ConfirmOptions) => Promise<boolean>;
    reconcile: (canvasId: string) => Promise<CanvasDocument>;     // injected from canvas-reconciler
    validateForDeploy: (canvas: CanvasDocument) => ValidationResult; // injected from canvas-builder
    onUpdateCanvas?: (updater: (current: CanvasDocument) => CanvasDocument) => void;
  }

  export async function executeRuntimeCommand(
    command: RuntimeCommand,
    deps: CommandExecutorDeps,
  ): Promise<CommandExecutorResult> { /* … */ }
  ```
  All side-effects route through `deps`, so unit tests can stub each one. Keep the function module-scoped (no React) so it is testable with plain Vitest.
- [x] 1.2 Move every action branch from `VoicePanel.handleExecuteCommand` into the new module: `kill`, `stop_all`, `pause`, `focus`, `status`, `deploy`, `cost`, `add_node`, `connect`
- [x] 1.3 Refactor `VoicePanel.tsx` to construct `deps` from its existing imports and call `executeRuntimeCommand`. The eslint-disable for sideways-capability-imports stays for now (resolved by `tech-debt-voice-event-bus`).
- [x] 1.4 Verify the diff to `VoicePanel.tsx` is *thinner* than before — the component should shrink as logic moves out

## 2. Unit tests for command-executor (VX)

- [x] 2.1 New file `src/voice/__tests__/command-executor.test.ts`. For each of the 9 actions, cover at minimum:
  - happy path
  - "no canvas" branch (where applicable)
  - "no session" / "no terminal_map" branch (where applicable)
  - CAO API failure branch (mock `cao.*` rejection)
- [x] 2.2 Confirmation flows (`kill`, `stop_all`): assert `confirm` is called and rejection short-circuits without invoking the API
- [x] 2.3 `focus` resolves by node id, by role, and by display_name; "ambiguous" / "unknown" both produce the documented toast

## 3. Unit tests for voice-to-canvas (VX)

- [x] 3.1 Extend `src/voice/__tests__/voice-to-canvas.test.ts` (or create if absent). Cover:
  - Single-agent canvas generation
  - Multi-agent with `handoff` / `assign` / `send_message` edges
  - Missing optional fields fall back to role-template defaults
  - Pt-BR canonical example from `speech-to-canvas/spec.md`

## 4. Component test for VoicePanel state machine (VX)

- [x] 4.1 New file `src/voice/__tests__/VoicePanel.test.tsx` rendering with React Testing Library. Mock `useVoiceStore` (Zustand) and assert the rendered DOM for each of the 5 states: `idle`, `listening`, `processing`, `confirming`, `error`
- [x] 4.2 Assert that the `confirming` state shows the parsed-intent summary, the confidence indicator, and the three actions (Cancel / Edit Before Deploy / Generate)

## 5. Verify (VX)

- [x] 5.1 `npx vitest run --coverage src/voice/` reports ≥70% for **statements** and ≥70% for **functions**
- [x] 5.2 `npm run lint` + `npm run typecheck` clean
- [x] 5.3 No regression in the existing `runtime-commands.test.ts` and `voice-to-canvas.test.ts` suites

## Out of Scope

- Replacing the `eslint-disable` for sideways-capability-imports — handled by `tech-debt-voice-event-bus`
- Polyfilling `SpeechRecognition` for Playwright — handled by `tech-debt-smoke-voice-real-flow`
- `engine.ts` / `voice-capture.ts` / `whisper-transcriber.ts` browser-API wrappers (intrinsically hard to unit-test without a polyfill)

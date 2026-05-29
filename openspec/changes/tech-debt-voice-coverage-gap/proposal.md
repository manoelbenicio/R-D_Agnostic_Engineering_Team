## Why

The `src/voice/` module has statement coverage of approximately 40.78%, significantly below the v1 target of ≥70% for logic-heavy modules (per task 22.1). The gap is concentrated in:

- `VoicePanel.tsx` — Large component with many branches for runtime command execution (kill, stop_all, pause, focus, status, deploy). Each action has multiple error paths (no canvas, no session, no terminal map, API failures) that are not covered.
- `runtime-commands.ts` — The regex matcher has good coverage from `matchRuntimeCommand.test.ts`, but edge cases for bilingual patterns and target extraction are undertested.
- `voice-to-canvas.ts` — The canvas generation logic from NLU intents has limited test coverage.
- `engine.ts` and `voice-capture.ts` — Browser API wrappers (`SpeechRecognition`, `MediaRecorder`) are inherently difficult to unit-test without browser polyfills.

## What Changes

1. **Extract command execution logic from VoicePanel into a testable service** — `src/voice/command-executor.ts` — a pure function that takes `(command: RuntimeCommand, canvas: CanvasDocument, caoClient, toast)` and returns a result. This makes the branching logic unit-testable without rendering React components.

2. **Add unit tests for `command-executor.ts`** covering:
   - Each of the 9 command actions (kill, stop_all, pause, focus, status, deploy, cost, add_node, connect)
   - Error paths: no canvas, no session, no terminal map, API failure
   - Edge cases: focus by role vs. name vs. id

3. **Add unit tests for `voice-to-canvas.ts`** covering:
   - Single-agent canvas generation
   - Multi-agent with edges
   - Missing fields / defaults

4. **Add integration test for `VoicePanel`** rendering states (idle → listening → processing → confirming → error) using mocked Zustand store.

### Target Coverage

Raise `src/voice/` to ≥70% statement coverage.

## Why Post-v1

The voice module works end-to-end as validated by manual testing and the smoke spec. The coverage gap is in error handling branches that are structurally sound but untested. Extracting the command executor is a refactor that touches the hot path of the VoicePanel component and should not be done during the v1 close sprint.

## Impact

- **Code**: 1 new file (`command-executor.ts`), 3-4 new test files, 1 refactored file (`VoicePanel.tsx`)
- **Risk**: Medium — refactoring VoicePanel's command dispatch requires re-testing all voice runtime commands
- **Dependencies**: None

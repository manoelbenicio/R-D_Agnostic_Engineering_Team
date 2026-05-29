## Why

The Playwright smoke spec (`tests/e2e/smoke.spec.ts`) uses two workarounds to pass in headless Chromium:

1. **`{ force: true }` on the stop-mic button click** — The 🛑 button has a CSS `pulse` animation that runs infinitely, causing Playwright's actionability check to consider the element "not stable" (it never stops moving). `force: true` bypasses the stability check entirely.

2. **Direct `useVoiceStore.setState()` mock** — Headless Chromium does not support the Web Speech API (`SpeechRecognition`). Instead of clicking the 🎤 button (which would call `getSTTEngine().start()` and fail), the test injects the voice state directly via `page.evaluate()`, skipping the real STT engine.

Both workarounds mean the smoke test does **not** exercise the real voice capture → STT → NLU pipeline in CI. Voice functionality is only validated through unit tests (vitest), not end-to-end.

## What Changes

1. **Replace `force: true`** with a CSS override that disables the pulse animation during Playwright runs (e.g., `* { animation-duration: 0s !important; }` injected via `page.addStyleTag`). This lets Playwright perform its normal actionability checks.

2. **Add a SpeechRecognition polyfill for headless** that simulates the Web Speech API lifecycle:
   - Emits `onresult` with a canned transcript after a short delay
   - Emits `onend` to trigger the processing flow
   - This would exercise the real `getSTTEngine()` → `VoiceCapture` → `stopListening()` → `matchRuntimeCommand()` pipeline

3. Alternatively, configure Playwright to use a Chromium build with SpeechRecognition support, or use the Whisper fallback path with a mocked Whisper API endpoint.

### Scope

- Modify: `tests/e2e/smoke.spec.ts`
- New: `tests/e2e/helpers/speech-recognition-mock.ts` (optional polyfill)
- Modify: potentially `playwright.config.ts` for animation disabling

## Why Post-v1

The current workarounds are pragmatic and the voice unit tests provide sufficient coverage of the NLU and runtime-command matching logic. Implementing a full SpeechRecognition polyfill is non-trivial and would need careful testing to ensure it doesn't introduce flaky behavior. Better suited for a testing infrastructure sprint.

## Impact

- **Code**: 1-2 files modified, 1 new helper
- **Risk**: Low-Medium — polyfill flakiness is the main concern
- **Dependencies**: None

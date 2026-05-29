# tech-debt-smoke-voice-real-flow — Implementation Tasks

> Owners: **SUP** (test infra) + **VX** (voice flow knowledge).
> Parallel-safe: only modifies `tests/e2e/smoke.spec.ts`, `playwright.config.ts`, and adds new helpers under `tests/e2e/helpers/`. No conflicts with the other tech-debt changes.

## 1. Animation suppression helper (SUP)

The `🛑` stop-mic button has an infinite `pulse` keyframe animation, so Playwright's actionability check considers it "not stable" and the test currently uses `{ force: true }` to bypass the check.

- [x] 1.1 Add a Playwright fixture / `beforeEach` step that injects an animation-killing stylesheet via `page.addStyleTag`:
  ```ts
  await page.addStyleTag({
    content: `
      *, *::before, *::after {
        animation-duration: 0s !important;
        animation-iteration-count: 1 !important;
        transition-duration: 0s !important;
      }
    `,
  });
  ```
  Place the helper in `tests/e2e/helpers/disable-animations.ts` so other E2E specs (e.g. `perf-12-terminals.spec.ts`) can opt into the same fix.
- [x] 1.2 Replace the existing `await stopMicBtn.click({ force: true })` with a plain `await stopMicBtn.click()` and verify the stability check passes
- [x] 1.3 Optionally enable Playwright's built-in `motion: 'reduce'` emulation via `playwright.config.ts` (`use.colorScheme` ladder) for additional safety

## 2. SpeechRecognition polyfill (VX)

Headless Chromium has no `SpeechRecognition` API, which is why the smoke test currently mutates `useVoiceStore` directly via `page.evaluate`. We can keep the test cheap and deterministic with a tiny polyfill that lives only inside the test.

- [x] 2.1 Create `tests/e2e/helpers/speech-recognition-mock.ts` exporting a `installSpeechRecognitionMock(page, { transcript })` helper that runs an `addInitScript` block. The script defines `window.SpeechRecognition` (and `window.webkitSpeechRecognition`) as a class with:
  - `continuous`, `interimResults`, `lang` properties (no-op setters)
  - `onstart`, `onresult`, `onerror`, `onend` handlers
  - `start()` → fires `onstart` synchronously, then schedules an `onresult` event with the canned transcript on a microtask, then `onend`
  - `stop()` → fires `onend` if not already fired
  - `abort()` → fires `onend` if not already fired
- [x] 2.2 The polyfill must be installed **before** the app boots (`addInitScript` ensures this).
- [x] 2.3 Update `tests/e2e/smoke.spec.ts` voice assertion section:
  - Call `installSpeechRecognitionMock(page, { transcript: 'focus on supervisor' })` in `beforeEach`
  - Replace the `page.evaluate` `useVoiceStore.setState` block with a real click on the `🎤` button — this should now drive the full pipeline `getSTTEngine().start()` → polyfilled `onresult` → `setFinalTranscript` → `matchRuntimeCommand` → `executeRuntimeCommand`
  - Keep the assertion that the final URL navigates to the focused-terminal route

## 3. Verify (SUP)

- [x] 3.1 `npm run test:smoke` green locally **and** in CI on `ubuntu-latest` Chromium
- [x] 3.2 Run the suite three times in a row (`for i in 1 2 3; do npm run test:smoke || break; done`) — assert no flakiness
- [x] 3.3 `npm run lint` + `npm run typecheck` clean
- [x] 3.4 Confirm the `force: true` workaround is gone (`grep -rn "force: true" tests/e2e/` returns nothing)
- [x] 3.5 Confirm the direct `useVoiceStore.setState` workaround is gone (`grep -rn "useVoiceStore" tests/e2e/` returns nothing)

## 4. Documentation (SUP)

- [x] 4.1 Add a short note to `docs/patterns/testing.md` (or create the file if absent) describing the `installSpeechRecognitionMock` helper and the animation-disable pattern, so future E2E specs follow the same conventions

## Out of Scope

- Replacing the polyfill with a real Chromium build that ships SpeechRecognition (deferred; the polyfill is sufficient and cheaper to maintain)
- Routing the smoke through the Whisper fallback path (keep the Web Speech path as the primary, since that is the production default)
- Touching `nlu.ts` LLM call mocks — these are already covered by the MSW server in unit tests

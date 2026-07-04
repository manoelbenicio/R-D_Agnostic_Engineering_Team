# Testing Patterns (D10)

| Layer       | Tool                          | Owner       | Coverage  |
| ----------- | ----------------------------- | ----------- | --------- |
| Unit        | Vitest                        | each owner  | ≥70% logic |
| Component   | Vitest + RTL                  | each owner  | n/a        |
| Integration | Vitest + MSW                  | each + IF   | n/a        |
| E2E smoke   | Playwright                    | SUP         | critical path |
| Contract    | Vitest + `CAO_LIVE=1`         | IF          | response shape on every endpoint |
| A11y        | axe-core (CI script)          | SUP         | critical/serious zero |

## Patterns

- **Tests live next to code**: `src/<capability>/__tests__/<file>.test.tsx`.
- **No global mocks**: MSW handlers in `src/api/__tests__/msw/handlers.ts`
  are the source of truth. Tests use `server.use(...)` to override per-test.
- **Render via wrapper**: `renderWithProviders()` (in `src/__tests__/utils.tsx`)
  installs the QueryClient + Router providers.
- **Keep tests deterministic**: no `setTimeout` waits; use `findByText` or
  `waitFor`.

## Playwright E2E helpers

Two opt-in helpers live in `tests/e2e/helpers/` and should be wired into
`beforeEach` for any spec that exercises animated UI or the voice flow.
They exist so tests never need actionability bypass flags or direct Zustand
store mutations — those mask real interaction bugs.

### `disableAnimations(page)`

```ts
import { disableAnimations } from './helpers/disable-animations';

test.beforeEach(async ({ page }) => {
  await disableAnimations(page);   // before any page.goto
  await page.goto('/');
});
```

What it does:

- Calls `page.emulateMedia({ reducedMotion: 'reduce' })`.
- Injects a stylesheet via `page.addInitScript` that zeros out
  `animation-duration`, `animation-delay`, `animation-iteration-count`,
  `transition-duration`, `transition-delay`, and `scroll-behavior`.
- Survives navigations — re-applies on every page load in the test.

Why we need it: Playwright's actionability check waits for an element's
bounding box to be stable for two consecutive animation frames before
clicking. Components like the voice 🛑 stop-mic button use an infinite
`pulse` keyframe animation that never settles, so the click would otherwise
time out or have to be force-clicked (which masks real interaction bugs).

### `installSpeechRecognitionMock(page, { transcript })`

```ts
import { installSpeechRecognitionMock } from './helpers/speech-recognition-mock';

test.beforeEach(async ({ page }) => {
  await installSpeechRecognitionMock(page, { transcript: 'focus on supervisor' });
  await page.goto('/');
});
```

What it does:

- Installs a tiny `SpeechRecognition` / `webkitSpeechRecognition` polyfill
  via `page.addInitScript` (so it's present before the SPA boots).
- `start()` schedules `onstart` then a single `onresult` whose first result
  is `isFinal: true` and contains the canned transcript.
- `stop()` / `abort()` fire `onend` once. The polyfill deliberately does
  **not** auto-fire `onend` after `onresult`, because `VoiceCapture`
  auto-restarts when `onend` arrives while still active and that would
  loop the canned transcript indefinitely.

Why we need it: Headless Chromium ships without the Web Speech API. The
polyfill lets the smoke test exercise the real
`getSTTEngine() → VoiceCapture → setFinalTranscript → matchRuntimeCommand
→ executeRuntimeCommand` pipeline by clicking the 🎤 / 🛑 buttons, instead
of mutating the voice store directly via `page.evaluate`.

The smoke spec also writes `sttEngine = 'webspeech'` to the `settings`
object store before the voice flow, because the app's default
(`'whisper'`) requires real `MediaRecorder` + `getUserMedia`, which are
not available under headless Chromium. Tests that exercise voice should
do the same.

### Anti-patterns to avoid

- ❌ Force-clicking elements to bypass Playwright actionability.
- ❌ Mutating the voice Zustand store via `page.evaluate` to fake voice state.
- ❌ Hard-coded `setTimeout` waits in specs — prefer `expect(...).toBeVisible()`
  or `waitFor` style assertions.

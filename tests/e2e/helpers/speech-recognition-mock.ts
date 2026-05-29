import type { Page } from '@playwright/test';

export interface SpeechRecognitionMockOptions {
  /**
   * The transcript that the polyfill will deliver as a single `isFinal: true`
   * `onresult` event after `start()` is called. The smoke test pipes this
   * through the real `getSTTEngine() → VoiceCapture → setFinalTranscript →
   * matchRuntimeCommand → executeRuntimeCommand` flow.
   */
  transcript: string;
}

/**
 * Install a deterministic SpeechRecognition polyfill on every navigation of
 * `page`.
 *
 * Background
 * ----------
 * Headless Chromium does **not** ship the Web Speech API
 * (`SpeechRecognition` / `webkitSpeechRecognition`). Without a polyfill, the
 * production voice path (`getSTTEngine().start()` →
 * `VoiceCapture.recognition.start()`) cannot be exercised end-to-end and the
 * smoke test had to bypass the engine by mutating the voice Zustand store
 * directly via `page.evaluate`.
 *
 * Lifecycle
 * ---------
 * The polyfill mirrors the subset of the spec that `VoiceCapture` actually
 * uses:
 *
 *   start()   → fires `onstart` on a microtask, then a single `onresult`
 *               whose first result is `isFinal: true` and contains the canned
 *               transcript.  We deliberately do **not** auto-fire `onend` so
 *               we don't trip `VoiceCapture`'s internal auto-restart loop
 *               (`onend` while `active === true` calls `recognition.start()`
 *               again, which would deliver the transcript on a tight loop).
 *   stop()    → fires `onend` once (no-op if already ended).
 *   abort()   → same as stop().
 *
 * Setters for `continuous`, `interimResults`, and `lang` are accepted but
 * ignored — the polyfill always produces a single final result.
 *
 * Usage
 * -----
 *   await installSpeechRecognitionMock(page, { transcript: 'focus on supervisor' });
 *   // ...later in the test:
 *   await page.click('text=🎤');   // fires onstart + onresult
 *   await page.click('text=🛑');   // fires onend, drives stopListening()
 *
 * Must be called **before** the first `page.goto` so `addInitScript` runs
 * before the SPA boots.
 */
export async function installSpeechRecognitionMock(
  page: Page,
  { transcript }: SpeechRecognitionMockOptions
): Promise<void> {
  await page.addInitScript((cannedTranscript: string) => {
    type ResultLike = {
      0: { transcript: string; confidence: number };
      isFinal: boolean;
      length: number;
    };
    type ResultsLike = {
      length: number;
      item: (i: number) => ResultLike;
      [index: number]: ResultLike;
    };

    class MockSpeechRecognition {
      // Spec-compatible properties (no-op setters in this polyfill).
      public continuous = false;
      public interimResults = false;
      public lang = '';
      public maxAlternatives = 1;

      // Event handlers wired by VoiceCapture.
      public onstart: ((event: Event) => void) | null = null;
      public onresult: ((event: unknown) => void) | null = null;
      public onerror: ((event: unknown) => void) | null = null;
      public onend: ((event: Event) => void) | null = null;
      public onaudiostart: ((event: Event) => void) | null = null;
      public onaudioend: ((event: Event) => void) | null = null;
      public onspeechstart: ((event: Event) => void) | null = null;
      public onspeechend: ((event: Event) => void) | null = null;

      private _running = false;

      start(): void {
        // Real Web Speech throws InvalidStateError if called while active;
        // VoiceCapture's auto-restart logic catches errors, so we just no-op
        // when already running.
        if (this._running) return;
        this._running = true;

        queueMicrotask(() => {
          if (!this._running) return;
          this.onstart?.(new Event('start'));

          queueMicrotask(() => {
            if (!this._running) return;
            const alternative = { transcript: cannedTranscript, confidence: 1 };
            const result = Object.assign(
              [alternative] as unknown as ResultLike,
              { isFinal: true, length: 1 }
            ) as ResultLike;
            const results: ResultsLike = Object.assign(
              [result] as unknown as ResultsLike,
              {
                length: 1,
                item(i: number) {
                  return (this as unknown as ResultLike[])[i] as ResultLike;
                },
              }
            );
            const event = {
              resultIndex: 0,
              results,
            };
            this.onresult?.(event);
          });
        });
      }

      stop(): void {
        if (!this._running) return;
        this._running = false;
        queueMicrotask(() => {
          this.onend?.(new Event('end'));
        });
      }

      abort(): void {
        this.stop();
      }

      // EventTarget-ish stubs in case future code switches from on* handlers
      // to addEventListener — keep them harmless no-ops.
      addEventListener(): void {
        /* no-op */
      }
      removeEventListener(): void {
        /* no-op */
      }
      dispatchEvent(): boolean {
        return false;
      }
    }

    Object.defineProperty(window, 'SpeechRecognition', {
      configurable: true,
      writable: true,
      value: MockSpeechRecognition,
    });
    Object.defineProperty(window, 'webkitSpeechRecognition', {
      configurable: true,
      writable: true,
      value: MockSpeechRecognition,
    });
  }, transcript);
}

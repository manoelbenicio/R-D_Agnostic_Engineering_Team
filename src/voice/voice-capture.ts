import { VoiceCaptureHandlers } from './types';

/* eslint-disable @typescript-eslint/no-explicit-any */

export class VoiceCapture {
  private recognition: any = null;
  private handlers: VoiceCaptureHandlers = {};
  private active = false;
  private lang = 'pt-BR';

  constructor(lang = 'pt-BR') {
    this.lang = lang;
    if (typeof window !== 'undefined') {
      const SpeechRecognition =
        (window as any).SpeechRecognition || (window as any).webkitSpeechRecognition;
      if (SpeechRecognition) {
        this.recognition = new SpeechRecognition();
        this.recognition.continuous = true;
        this.recognition.interimResults = true;
        this.recognition.lang = this.lang;

        this.recognition.onresult = (event: any) => {
          let interimTranscript = '';
          let finalTranscript = '';

          for (let i = event.resultIndex; i < event.results.length; ++i) {
            const result = event.results[i];
            const transcript = result[0].transcript;
            if (result.isFinal) {
              finalTranscript += transcript;
            } else {
              interimTranscript += transcript;
            }
          }

          if (finalTranscript && this.handlers.onFinal) {
            this.handlers.onFinal(finalTranscript.trim());
          }
          if (interimTranscript && this.handlers.onPartial) {
            this.handlers.onPartial(interimTranscript.trim());
          }
        };

        this.recognition.onerror = (event: any) => {
          if (event.error === 'no-speech' && this.active) {
            try {
              this.recognition.stop();
            } catch {
              // Ignore
            }
            return;
          }

          if (this.handlers.onError) {
            this.handlers.onError(event);
          }
        };

        this.recognition.onend = () => {
          if (this.active) {
            try {
              this.recognition.start();
            } catch {
              // Ignore if already started
            }
          } else {
            if (this.handlers.onEnd) {
              this.handlers.onEnd();
            }
          }
        };
      }
    }
  }

  isAvailable(): boolean {
    if (typeof window === 'undefined') return false;
    const SpeechRecognition =
      (window as any).SpeechRecognition || (window as any).webkitSpeechRecognition;
    return !!SpeechRecognition;
  }

  start(handlers: VoiceCaptureHandlers): void {
    if (!this.isAvailable()) {
      if (handlers.onError) {
        handlers.onError(new Error('SpeechRecognition not available'));
      }
      return;
    }
    this.handlers = handlers;
    this.active = true;
    try {
      this.recognition.start();
    } catch (err) {
      if (handlers.onError) {
        handlers.onError(err);
      }
    }
  }

  stop(): void {
    this.active = false;
    if (this.recognition) {
      try {
        this.recognition.stop();
      } catch {
        // Ignore
      }
    }
  }
}

export default VoiceCapture;

/* eslint-disable agentverse/no-sideways-capability-imports */
import { VoiceCaptureHandlers } from './types';
import { KeyStore } from '@/api/key-store';
import { useKeyStore } from '@/api/key-store/store';
import { appFetch } from '@/shell/app-fetch';


export class WhisperTranscriber {
  private mediaRecorder: MediaRecorder | null = null;
  private stream: MediaStream | null = null;
  private handlers: VoiceCaptureHandlers = {};
  private active = false;
  private accumulatedText = '';
  private lang = 'pt-BR';

  constructor(lang = 'pt-BR') {
    this.lang = lang;
  }

  isAvailable(): boolean {
    if (typeof window === 'undefined') return false;
    const mediaSupport = !!(
      window.MediaRecorder &&
      navigator.mediaDevices &&
      navigator.mediaDevices.getUserMedia
    );
    if (!mediaSupport) return false;

    try {
      const state = useKeyStore.getState();
      return state.statuses['openai'] === 'set';
    } catch {
      return false;
    }
  }

  async start(handlers: VoiceCaptureHandlers): Promise<void> {
    if (!this.isAvailable()) {
      handlers.onError?.(new Error('WhisperTranscriber not available or OpenAI key not set'));
      return;
    }

    let apiKey = '';
    try {
      const record = await KeyStore.get('openai');
      apiKey = record?.keys['apiKey'] || '';
    } catch {
      handlers.onError?.(new Error('Failed to retrieve OpenAI API Key'));
      return;
    }

    if (!apiKey) {
      handlers.onError?.(new Error('OpenAI API Key is required for Whisper transcription'));
      return;
    }

    this.handlers = handlers;
    this.active = true;
    this.accumulatedText = '';

    try {
      this.stream = await navigator.mediaDevices.getUserMedia({
        audio: {
          sampleRate: 16000,
          channelCount: 1,
        },
      });

      let options = {};
      if (MediaRecorder.isTypeSupported('audio/webm;codecs=opus')) {
        options = { mimeType: 'audio/webm;codecs=opus' };
      } else if (MediaRecorder.isTypeSupported('audio/webm')) {
        options = { mimeType: 'audio/webm' };
      }

      this.mediaRecorder = new MediaRecorder(this.stream, options);

      this.mediaRecorder.ondataavailable = async (event) => {
        if (event.data.size > 0 && this.active) {
          const audioBlob = event.data;
          void this.transcribeChunk(audioBlob, apiKey);
        }
      };

      this.mediaRecorder.onstop = () => {
        this.cleanupStream();
        if (this.handlers.onEnd) {
          this.handlers.onEnd();
        }
      };

      this.mediaRecorder.start(3000); // 3-second slices
    } catch (err) {
      this.cleanupStream();
      this.active = false;
      handlers.onError?.(err);
    }
  }

  private cleanupStream(): void {
    if (this.stream) {
      this.stream.getTracks().forEach((track) => {
        track.stop();
      });
      this.stream = null;
    }
    this.mediaRecorder = null;
  }

  stop(): void {
    this.active = false;
    if (this.mediaRecorder && this.mediaRecorder.state !== 'inactive') {
      try {
        this.mediaRecorder.stop();
      } catch {
        /* ignore */
      }
    } else {
      this.cleanupStream();
    }
  }

  private async transcribeChunk(blob: Blob, apiKey: string): Promise<void> {
    try {
      const formData = new FormData();
      formData.append('file', blob, 'chunk.webm');
      formData.append('model', 'whisper-1');
      formData.append('response_format', 'json');
      if (this.lang) {
        const whisperLang = this.lang.split('-')[0];
        if (whisperLang) {
          formData.append('language', whisperLang);
        }
      }

      const res = await appFetch('https://api.openai.com/v1/audio/transcriptions', {
        method: 'POST',
        headers: {
          Authorization: `Bearer ${apiKey}`,
        },
        body: formData,
      });

      if (!res.ok) {
        const errText = await res.text();
        throw new Error(`Whisper API failure (${res.status}): ${errText}`);
      }

      const data = (await res.json()) as { text?: string };
      const text = data.text || '';
      if (text.trim() && this.active) {
        const cleanedText = text.trim();
        this.accumulatedText = this.accumulatedText
          ? `${this.accumulatedText} ${cleanedText}`
          : cleanedText;

        if (this.handlers.onFinal) {
          this.handlers.onFinal(this.accumulatedText);
        }
      }
    } catch (err) {
      if (this.active && this.handlers.onError) {
        this.handlers.onError(err);
      }
    }
  }
}

export default WhisperTranscriber;

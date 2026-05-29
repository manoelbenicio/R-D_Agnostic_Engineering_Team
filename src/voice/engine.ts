/* eslint-disable agentverse/no-sideways-capability-imports */
import { VoiceCapture } from './voice-capture';
import { WhisperTranscriber } from './whisper-transcriber';
import { useSettingsStore } from '@/settings/settings-store';
import { VoiceCaptureHandlers } from './types';

export function getSTTEngine(lang = 'pt-BR'): {
  start: (handlers: VoiceCaptureHandlers) => void | Promise<void>;
  stop: () => void;
  isAvailable: () => boolean;
  getEngineType: () => 'webspeech' | 'whisper';
} {
  const settingsEngine = useSettingsStore.getState().sttEngine || 'auto';
  const voiceCapture = new VoiceCapture(lang);
  const whisperTranscriber = new WhisperTranscriber(lang);

  const getActiveEngine = (): VoiceCapture | WhisperTranscriber => {
    if (settingsEngine === 'webspeech') {
      return voiceCapture;
    } else if (settingsEngine === 'whisper') {
      return whisperTranscriber;
    } else {
      if (voiceCapture.isAvailable()) {
        return voiceCapture;
      }
      return whisperTranscriber;
    }
  };

  const activeEngine = getActiveEngine();

  return {
    start: (handlers) => activeEngine.start(handlers),
    stop: () => activeEngine.stop(),
    isAvailable: () => activeEngine.isAvailable(),
    getEngineType: () => (activeEngine instanceof VoiceCapture ? 'webspeech' : 'whisper'),
  };
}

export default getSTTEngine;

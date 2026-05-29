import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { getSTTEngine } from '../engine';
import { useKeyStore } from '@/api/key-store/store';
// eslint-disable-next-line agentverse/no-sideways-capability-imports
import { useSettingsStore } from '@/settings/settings-store';

describe('getSTTEngine', () => {
  const originalMediaRecorder = (window as any).MediaRecorder;
  const originalGetUserMedia = navigator.mediaDevices?.getUserMedia;

  beforeEach(() => {
    if (!navigator.mediaDevices) {
      (navigator as any).mediaDevices = {};
    }
    useSettingsStore.setState({ sttEngine: 'auto' });
    useKeyStore.setState({ statuses: { openai: 'unset' } as any });
  });

  afterEach(() => {
    delete (window as any).SpeechRecognition;
    delete (window as any).webkitSpeechRecognition;
    (window as any).MediaRecorder = originalMediaRecorder;
    if (navigator.mediaDevices) {
      navigator.mediaDevices.getUserMedia = originalGetUserMedia;
    }
    vi.restoreAllMocks();
  });

  it('uses WebSpeech when the setting explicitly requests it', () => {
    const start = vi.fn();
    const stop = vi.fn();

    class MockRecognition {
      continuous = false;
      interimResults = false;
      lang = '';
      start = start;
      stop = stop;
    }

    (window as any).SpeechRecognition = MockRecognition;
    useSettingsStore.setState({ sttEngine: 'webspeech' });

    const engine = getSTTEngine('en-US');
    engine.start({});
    engine.stop();

    expect(engine.getEngineType()).toBe('webspeech');
    expect(engine.isAvailable()).toBe(true);
    expect(start).toHaveBeenCalled();
    expect(stop).toHaveBeenCalled();
  });

  it('uses Whisper when the setting explicitly requests it', async () => {
    const onError = vi.fn();
    useSettingsStore.setState({ sttEngine: 'whisper' });

    const engine = getSTTEngine();
    await engine.start({ onError });

    expect(engine.getEngineType()).toBe('whisper');
    expect(engine.isAvailable()).toBe(false);
    expect(onError.mock.calls[0]![0].message).toBe('WhisperTranscriber not available or OpenAI key not set');
  });

  it('auto-selects WebSpeech when speech recognition is available', () => {
    (window as any).webkitSpeechRecognition = class {
      start = vi.fn();
      stop = vi.fn();
    };
    useSettingsStore.setState({ sttEngine: 'auto' });

    const engine = getSTTEngine();

    expect(engine.getEngineType()).toBe('webspeech');
  });

  it('auto-selects Whisper when WebSpeech is unavailable', () => {
    (window as any).MediaRecorder = Object.assign(vi.fn(), {
      isTypeSupported: vi.fn(() => false),
    });
    navigator.mediaDevices.getUserMedia = vi.fn();
    useKeyStore.setState({ statuses: { openai: 'set' } as any });
    useSettingsStore.setState({ sttEngine: 'auto' });

    const engine = getSTTEngine();

    expect(engine.getEngineType()).toBe('whisper');
    expect(engine.isAvailable()).toBe(true);
  });
});

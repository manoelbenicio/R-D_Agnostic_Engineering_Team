import { afterEach, describe, it, expect, vi } from 'vitest';
import { VoiceCapture } from '../voice-capture';

describe('VoiceCapture', () => {
  afterEach(() => {
    delete (window as any).SpeechRecognition;
    delete (window as any).webkitSpeechRecognition;
    vi.restoreAllMocks();
  });

  it('isAvailable returns false when SpeechRecognition is undefined', () => {
    const originalSpeechRecognition = (window as any).SpeechRecognition;
    const originalWebkitSpeechRecognition = (window as any).webkitSpeechRecognition;

    delete (window as any).SpeechRecognition;
    delete (window as any).webkitSpeechRecognition;

    const capture = new VoiceCapture();
    expect(capture.isAvailable()).toBe(false);

    if (originalSpeechRecognition) (window as any).SpeechRecognition = originalSpeechRecognition;
    if (originalWebkitSpeechRecognition) (window as any).webkitSpeechRecognition = originalWebkitSpeechRecognition;
  });

  it('isAvailable returns true when SpeechRecognition or webkitSpeechRecognition is defined', () => {
    const originalSpeechRecognition = (window as any).SpeechRecognition;
    (window as any).SpeechRecognition = vi.fn();

    const capture = new VoiceCapture();
    expect(capture.isAvailable()).toBe(true);

    if (originalSpeechRecognition) {
      (window as any).SpeechRecognition = originalSpeechRecognition;
    } else {
      delete (window as any).SpeechRecognition;
    }
  });

  it('forwards final and interim transcripts from recognition results', () => {
    const onFinal = vi.fn();
    const onPartial = vi.fn();
    const start = vi.fn();
    const stop = vi.fn();
    const instances: any[] = [];

    class MockRecognition {
      continuous = false;
      interimResults = false;
      lang = '';
      onresult: ((event: any) => void) | null = null;
      onerror: ((event: any) => void) | null = null;
      onend: (() => void) | null = null;
      start = start;
      stop = stop;

      constructor() {
        instances.push(this);
      }
    }

    (window as any).SpeechRecognition = MockRecognition;

    const capture = new VoiceCapture('en-US');
    capture.start({ onFinal, onPartial });

    expect(instances[0]).toMatchObject({
      continuous: true,
      interimResults: true,
      lang: 'en-US',
    });

    instances[0].onresult({
      resultIndex: 0,
      results: [
        { 0: { transcript: '  create canvas  ' }, isFinal: true, length: 1 },
        { 0: { transcript: ' with reviewers ' }, isFinal: false, length: 1 },
      ],
    });

    expect(onFinal).toHaveBeenCalledWith('create canvas');
    expect(onPartial).toHaveBeenCalledWith('with reviewers');
  });

  it('handles no-speech by stopping without surfacing an error while active', () => {
    const onError = vi.fn();
    const stop = vi.fn();
    const instance = {
      onerror: null as ((event: any) => void) | null,
      start: vi.fn(),
      stop,
    };

    function MockRecognition() {
      return instance;
    }

    (window as any).SpeechRecognition = MockRecognition;

    const capture = new VoiceCapture();
    capture.start({ onError });
    instance.onerror?.({ error: 'no-speech' });

    expect(stop).toHaveBeenCalled();
    expect(onError).not.toHaveBeenCalled();
  });

  it('reports recognition errors and start failures to handlers', () => {
    const onError = vi.fn();
    const boom = new Error('already started');
    const instance = {
      onerror: null as ((event: any) => void) | null,
      start: vi.fn(() => {
        throw boom;
      }),
      stop: vi.fn(),
    };

    function MockRecognition() {
      return instance;
    }

    (window as any).SpeechRecognition = MockRecognition;

    const capture = new VoiceCapture();
    capture.start({ onError });
    instance.onerror?.({ error: 'network' });

    expect(onError).toHaveBeenCalledWith(boom);
    expect(onError).toHaveBeenCalledWith({ error: 'network' });
  });

  it('restarts while active and calls onEnd after stop', () => {
    const onEnd = vi.fn();
    const start = vi.fn();
    const stop = vi.fn();
    const instance = {
      onend: null as (() => void) | null,
      start,
      stop,
    };

    function MockRecognition() {
      return instance;
    }

    (window as any).SpeechRecognition = MockRecognition;

    const capture = new VoiceCapture();
    capture.start({ onEnd });

    instance.onend?.();
    expect(start).toHaveBeenCalledTimes(2);
    expect(onEnd).not.toHaveBeenCalled();

    capture.stop();
    instance.onend?.();

    expect(stop).toHaveBeenCalled();
    expect(onEnd).toHaveBeenCalled();
  });

  it('reports unavailable recognition during start', () => {
    const onError = vi.fn();
    const capture = new VoiceCapture();

    capture.start({ onError });

    expect(onError).toHaveBeenCalledWith(expect.any(Error));
    expect(onError.mock.calls[0]![0].message).toBe('SpeechRecognition not available');
  });
});

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { WhisperTranscriber } from '../whisper-transcriber';
import { useKeyStore } from '@/api/key-store/store';
import { KeyStore } from '@/api/key-store';
import { appFetch } from '@/shell/app-fetch';

vi.mock('@/shell/app-fetch', () => ({
  appFetch: vi.fn(),
}));

describe('WhisperTranscriber', () => {
  let originalMediaRecorder: any;
  let originalGetUserMedia: any;

  beforeEach(() => {
    originalMediaRecorder = (window as any).MediaRecorder;
    if (!navigator.mediaDevices) {
      (navigator as any).mediaDevices = {} as any;
    }
    originalGetUserMedia = navigator.mediaDevices.getUserMedia;
    navigator.mediaDevices.getUserMedia = vi.fn();

    useKeyStore.setState({
      statuses: {
        openai: 'unset',
      } as any,
    });
  });

  afterEach(() => {
    (window as any).MediaRecorder = originalMediaRecorder;
    if (navigator.mediaDevices) {
      navigator.mediaDevices.getUserMedia = originalGetUserMedia;
    }
    vi.restoreAllMocks();
  });

  it('isAvailable returns false if MediaRecorder is not supported', () => {
    delete (window as any).MediaRecorder;
    useKeyStore.setState({
      statuses: {
        openai: 'set',
      } as any,
    });

    const transcriber = new WhisperTranscriber();
    expect(transcriber.isAvailable()).toBe(false);
  });

  it('isAvailable returns false if OpenAI key is not set', () => {
    (window as any).MediaRecorder = vi.fn();
    useKeyStore.setState({
      statuses: {
        openai: 'unset',
      } as any,
    });

    const transcriber = new WhisperTranscriber();
    expect(transcriber.isAvailable()).toBe(false);
  });

  it('isAvailable returns true if both MediaRecorder and OpenAI key are present', () => {
    (window as any).MediaRecorder = vi.fn();
    useKeyStore.setState({
      statuses: {
        openai: 'set',
      } as any,
    });

    const transcriber = new WhisperTranscriber();
    expect(transcriber.isAvailable()).toBe(true);
  });

  it('reports an error when start is requested while unavailable', async () => {
    const onError = vi.fn();
    delete (window as any).MediaRecorder;
    useKeyStore.setState({ statuses: { openai: 'set' } as any });

    const transcriber = new WhisperTranscriber();
    await transcriber.start({ onError });

    expect(onError).toHaveBeenCalledWith(expect.any(Error));
    expect(onError.mock.calls[0]![0].message).toBe('WhisperTranscriber not available or OpenAI key not set');
  });

  it('reports key retrieval and missing key failures before opening the microphone', async () => {
    const onError = vi.fn();
    (window as any).MediaRecorder = vi.fn();
    useKeyStore.setState({ statuses: { openai: 'set' } as any });
    vi.spyOn(KeyStore, 'get').mockRejectedValueOnce(new Error('idb unavailable'));

    const retrievalFailure = new WhisperTranscriber();
    await retrievalFailure.start({ onError });

    expect(onError.mock.calls[0]![0].message).toBe('Failed to retrieve OpenAI API Key');
    expect(navigator.mediaDevices.getUserMedia).not.toHaveBeenCalled();

    vi.spyOn(KeyStore, 'get').mockResolvedValueOnce({
      provider: 'openai',
      keys: {},
      models: [],
      updatedAt: '',
    } as any);

    const missingKey = new WhisperTranscriber();
    await missingKey.start({ onError });

    expect(onError.mock.calls[1]![0].message).toBe('OpenAI API Key is required for Whisper transcription');
    expect(navigator.mediaDevices.getUserMedia).not.toHaveBeenCalled();
  });

  it('cleans up and reports microphone failures', async () => {
    const onError = vi.fn();
    const failure = new Error('permission denied');
    (window as any).MediaRecorder = Object.assign(vi.fn(), {
      isTypeSupported: vi.fn(() => false),
    });
    navigator.mediaDevices.getUserMedia = vi.fn().mockRejectedValue(failure);
    useKeyStore.setState({ statuses: { openai: 'set' } as any });
    vi.spyOn(KeyStore, 'get').mockResolvedValue({
      provider: 'openai',
      keys: { apiKey: 'valid-openai-key' },
      models: [],
      updatedAt: '',
    });

    const transcriber = new WhisperTranscriber();
    await transcriber.start({ onError });

    expect(onError).toHaveBeenCalledWith(failure);
    expect((transcriber as any).mediaRecorder).toBeNull();
    expect((transcriber as any).stream).toBeNull();
  });

  it('releases mic and stops tracks when recorder is stopped', async () => {
    const trackStopMock = vi.fn();
    const mockTrack = { stop: trackStopMock };
    const mockStream = {
      getTracks: () => [mockTrack],
    };

    if (!navigator.mediaDevices) {
      (navigator as any).mediaDevices = {};
    }
    navigator.mediaDevices.getUserMedia = vi.fn().mockResolvedValue(mockStream);

    const mockStart = vi.fn();
    const mockStop = vi.fn();
    class MockMediaRecorder {
      state = 'recording';
      ondataavailable = null;
      onstop = null;
      start = mockStart;
      stop = mockStop;
      static isTypeSupported = () => true;
    }
    (window as any).MediaRecorder = MockMediaRecorder;

    useKeyStore.setState({ statuses: { openai: 'set' } as any });
    vi.spyOn(KeyStore, 'get').mockResolvedValue({
      provider: 'openai',
      keys: { apiKey: 'valid-openai-key' },
      models: [],
      updatedAt: '',
    });

    const transcriber = new WhisperTranscriber();
    await transcriber.start({});

    expect(navigator.mediaDevices.getUserMedia).toHaveBeenCalled();
    expect(mockStart).toHaveBeenCalledWith(3000);

    transcriber.stop();
    expect(mockStop).toHaveBeenCalled();

    const instance: any = (transcriber as any).mediaRecorder;
    if (instance && instance.onstop) {
      instance.onstop();
    }
    expect(trackStopMock).toHaveBeenCalled();
  });

  it('transcribes active audio chunks and accumulates final text', async () => {
    const onFinal = vi.fn();
    const mockStream = {
      getTracks: () => [{ stop: vi.fn() }],
    };
    const recorder = {
      state: 'recording',
      ondataavailable: null as ((event: any) => void) | null,
      onstop: null as (() => void) | null,
      start: vi.fn(),
      stop: vi.fn(),
    };

    navigator.mediaDevices.getUserMedia = vi.fn().mockResolvedValue(mockStream);

    function MockMediaRecorder() {
      return recorder;
    }
    Object.assign(MockMediaRecorder, {
      isTypeSupported: vi.fn((mimeType: string) => mimeType === 'audio/webm'),
    });
    (window as any).MediaRecorder = MockMediaRecorder;

    useKeyStore.setState({ statuses: { openai: 'set' } as any });
    vi.spyOn(KeyStore, 'get').mockResolvedValue({
      provider: 'openai',
      keys: { apiKey: 'valid-openai-key' },
      models: [],
      updatedAt: '',
    });
    vi.mocked(appFetch)
      .mockResolvedValueOnce({
        ok: true,
        json: vi.fn().mockResolvedValue({ text: ' hello ' }),
      } as unknown as Response)
      .mockResolvedValueOnce({
        ok: true,
        json: vi.fn().mockResolvedValue({ text: ' world ' }),
      } as unknown as Response);

    const transcriber = new WhisperTranscriber('pt-BR');
    await transcriber.start({ onFinal });

    recorder.ondataavailable?.({ data: new Blob(['first'], { type: 'audio/webm' }) });
    await Promise.resolve();
    await Promise.resolve();

    recorder.ondataavailable?.({ data: new Blob(['second'], { type: 'audio/webm' }) });
    await Promise.resolve();
    await Promise.resolve();

    expect(appFetch).toHaveBeenCalledWith(
      'https://api.openai.com/v1/audio/transcriptions',
      expect.objectContaining({
        method: 'POST',
        headers: { Authorization: 'Bearer valid-openai-key' },
        body: expect.any(FormData),
      })
    );
    expect(onFinal).toHaveBeenNthCalledWith(1, 'hello');
    expect(onFinal).toHaveBeenNthCalledWith(2, 'hello world');
  });

  it('surfaces Whisper API failures while active and ignores empty chunks', async () => {
    const onError = vi.fn();
    const mockStream = {
      getTracks: () => [{ stop: vi.fn() }],
    };
    const recorder = {
      state: 'recording',
      ondataavailable: null as ((event: any) => void) | null,
      start: vi.fn(),
      stop: vi.fn(),
    };

    navigator.mediaDevices.getUserMedia = vi.fn().mockResolvedValue(mockStream);

    function MockMediaRecorder() {
      return recorder;
    }
    Object.assign(MockMediaRecorder, {
      isTypeSupported: vi.fn(() => false),
    });
    (window as any).MediaRecorder = MockMediaRecorder;

    useKeyStore.setState({ statuses: { openai: 'set' } as any });
    vi.spyOn(KeyStore, 'get').mockResolvedValue({
      provider: 'openai',
      keys: { apiKey: 'valid-openai-key' },
      models: [],
      updatedAt: '',
    });
    vi.mocked(appFetch).mockResolvedValue({
      ok: false,
      status: 429,
      text: vi.fn().mockResolvedValue('rate limited'),
    } as unknown as Response);

    const transcriber = new WhisperTranscriber();
    await transcriber.start({ onError });

    recorder.ondataavailable?.({ data: new Blob([]) });
    recorder.ondataavailable?.({ data: new Blob(['audio']) });
    await Promise.resolve();
    await Promise.resolve();

    expect(appFetch).toHaveBeenCalledTimes(1);
    expect(onError).toHaveBeenCalledWith(expect.any(Error));
    expect(onError.mock.calls[0]![0].message).toBe('Whisper API failure (429): rate limited');
  });
});

import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { useVoiceStore } from '../store';
import { KeyStore } from '@/api/key-store';

describe('Transcript Privacy', () => {
  beforeEach(() => {
    localStorage.clear();
    useVoiceStore.getState().reset();
  });

  afterEach(() => {
    localStorage.clear();
    useVoiceStore.getState().reset();
  });

  it('ensures transcripts are never saved to localStorage, Zustand state, or IndexedDB after session ends', async () => {
    const testSecretTranscript = 'SuperSecretTranscript123';

    useVoiceStore.getState().setFinalTranscript(testSecretTranscript);
    expect(useVoiceStore.getState().finalTranscript).toBe(testSecretTranscript);

    useVoiceStore.getState().reset();

    for (let i = 0; i < localStorage.length; i++) {
      const key = localStorage.key(i) || '';
      const val = localStorage.getItem(key) || '';
      expect(key).not.toContain(testSecretTranscript);
      expect(val).not.toContain(testSecretTranscript);
    }

    const state = useVoiceStore.getState();
    expect(state.finalTranscript).not.toContain(testSecretTranscript);
    expect(state.interimTranscript).not.toContain(testSecretTranscript);

    const keysInDb = await KeyStore.all();
    const keysStr = JSON.stringify(keysInDb);
    expect(keysStr).not.toContain(testSecretTranscript);
  });
});

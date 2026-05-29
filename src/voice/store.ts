import { create } from 'zustand';
import { CreateCanvasIntent, VoiceState } from './types';

export interface VoiceStoreState {
  isOpen: boolean;
  voiceState: VoiceState;
  interimTranscript: string;
  finalTranscript: string;
  intent: CreateCanvasIntent | null;
  error: string | null;

  setOpen: (open: boolean) => void;
  setState: (state: VoiceState) => void;
  setInterimTranscript: (text: string) => void;
  setFinalTranscript: (text: string) => void;
  setIntent: (intent: CreateCanvasIntent | null) => void;
  setError: (error: string | null) => void;
  reset: () => void;
}

export const useVoiceStore = create<VoiceStoreState>((set) => ({
  isOpen: false,
  voiceState: 'idle',
  interimTranscript: '',
  finalTranscript: '',
  intent: null,
  error: null,

  setOpen: (open) => set({ isOpen: open }),
  setState: (state) => set({ voiceState: state }),
  setInterimTranscript: (text) => set({ interimTranscript: text }),
  setFinalTranscript: (text) => set({ finalTranscript: text }),
  setIntent: (intent) => set({ intent }),
  setError: (error) => set({ error }),
  reset: () =>
    set({
      voiceState: 'idle',
      interimTranscript: '',
      finalTranscript: '',
      intent: null,
      error: null,
    }),
}));

if (typeof window !== 'undefined') {
  (window as any).useVoiceStore = useVoiceStore;
}

export default useVoiceStore;

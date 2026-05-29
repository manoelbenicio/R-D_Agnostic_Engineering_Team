import { act, render } from '@testing-library/react';
import { beforeEach, describe, expect, it } from 'vitest';
import { useVoiceHotkey } from '../use-voice-hotkey';
import { useVoiceStore } from '../store';

function HotkeyProbe() {
  useVoiceHotkey();
  return null;
}

describe('useVoiceHotkey', () => {
  beforeEach(() => {
    useVoiceStore.setState({
      isOpen: false,
      voiceState: 'idle',
      interimTranscript: '',
      finalTranscript: '',
      intent: null,
      error: null,
    });
  });

  it('opens the voice panel with Ctrl+Shift+V', () => {
    render(<HotkeyProbe />);

    act(() => {
      window.dispatchEvent(new KeyboardEvent('keydown', { key: 'v', ctrlKey: true, shiftKey: true }));
    });

    expect(useVoiceStore.getState().isOpen).toBe(true);
  });

  it('closes the voice panel and resets transient state with Cmd+Shift+V', () => {
    useVoiceStore.setState({
      isOpen: true,
      voiceState: 'confirming',
      interimTranscript: 'interim',
      finalTranscript: 'final',
      intent: {
        name: 'Draft',
        nodes: [{ display_name: 'Supervisor', role: 'supervisor', provider: 'kiro_cli' }],
        edges: [],
      },
      error: 'boom',
    });

    render(<HotkeyProbe />);

    act(() => {
      window.dispatchEvent(new KeyboardEvent('keydown', { key: 'V', metaKey: true, shiftKey: true }));
    });

    expect(useVoiceStore.getState()).toMatchObject({
      isOpen: false,
      voiceState: 'idle',
      interimTranscript: '',
      finalTranscript: '',
      intent: null,
      error: null,
    });
  });

  it('ignores plain V key presses', () => {
    render(<HotkeyProbe />);

    act(() => {
      window.dispatchEvent(new KeyboardEvent('keydown', { key: 'v' }));
    });

    expect(useVoiceStore.getState().isOpen).toBe(false);
  });
});

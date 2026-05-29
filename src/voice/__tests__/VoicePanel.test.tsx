import { describe, it, expect, beforeEach, vi } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { VoicePanel } from '../VoicePanel';
import { useVoiceStore } from '../store';
import type { CreateCanvasIntent } from '../types';
import { useKeyStore } from '@/api/key-store/store';
import { extractIntent } from '../nlu';

vi.mock('../nlu', () => ({
  extractIntent: vi.fn(),
}));

describe('VoicePanel Component', () => {
  beforeEach(() => {
    vi.resetAllMocks();
    useVoiceStore.setState({
      isOpen: true,
      voiceState: 'idle',
      interimTranscript: '',
      finalTranscript: '',
      intent: null,
      error: null,
    });
  });

  it('renders disabled notice when no LLM providers are validated', () => {
    useKeyStore.setState({
      validated: [],
    });

    render(
      <MemoryRouter>
        <VoicePanel />
      </MemoryRouter>
    );

    expect(screen.getByText(/Voice requires at least one validated LLM provider/i)).toBeInTheDocument();
  });

  it('renders start mic button when at least one LLM provider is validated', () => {
    useKeyStore.setState({
      validated: ['google'],
      statuses: { google: 'set' } as any,
    });

    render(
      <MemoryRouter>
        <VoicePanel />
      </MemoryRouter>
    );

    expect(screen.getByText(/Click to start speaking/i)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /🎤/ })).toBeInTheDocument();
  });

  it('renders listening state with stop button and card preview', () => {
    useKeyStore.setState({
      validated: ['google'],
      statuses: { google: 'set' } as any,
    });
    useVoiceStore.setState({
      isOpen: true,
      voiceState: 'listening',
      finalTranscript: 'crie um time de supervisor',
    });

    render(
      <MemoryRouter>
        <VoicePanel />
      </MemoryRouter>
    );

    expect(screen.getByText(/Listening... Click to process/i)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /🛑/ })).toBeInTheDocument();
    expect(screen.getByText('crie um time de supervisor')).toBeInTheDocument();
  });

  it('falls back to NLU when command is unrecognized and successfully processes NLU response', async () => {
    useKeyStore.setState({
      validated: ['google'],
      statuses: { google: 'set' } as any,
    });
    useVoiceStore.setState({
      isOpen: true,
      voiceState: 'listening',
      finalTranscript: 'crie um time de supervisor no kiro',
      interimTranscript: '',
    });

    const mockIntent: CreateCanvasIntent = {
      name: 'Voice Generated Canvas',
      nodes: [
        { display_name: 'Lead Supervisor', role: 'supervisor' as const, provider: 'kiro_cli' },
      ],
      edges: [],
      confidence: 0.95,
    };

    vi.mocked(extractIntent).mockResolvedValueOnce(mockIntent);

    render(
      <MemoryRouter>
        <VoicePanel />
      </MemoryRouter>
    );

    const stopButton = screen.getByRole('button', { name: /🛑/ });
    fireEvent.click(stopButton);

    await waitFor(() => {
      expect(screen.getByText(/Extracted Canvas Structure:/i)).toBeInTheDocument();
    });

    expect(screen.getByText('Lead Supervisor')).toBeInTheDocument();
    expect(screen.getByText(/95% Confidence/i)).toBeInTheDocument();
    expect(extractIntent).toHaveBeenCalledWith('crie um time de supervisor no kiro');
  });

  it('executes matched runtime commands directly without falling back to NLU', async () => {
    useKeyStore.setState({
      validated: ['google'],
      statuses: { google: 'set' } as any,
    });
    useVoiceStore.setState({
      isOpen: true,
      voiceState: 'listening',
      finalTranscript: 'custo',
      interimTranscript: '',
    });

    render(
      <MemoryRouter>
        <VoicePanel />
      </MemoryRouter>
    );

    const stopButton = screen.getByRole('button', { name: /🛑/ });
    fireEvent.click(stopButton);

    expect(extractIntent).not.toHaveBeenCalled();
  });

  // -------------------------------------------------------------------
  // tech-debt-voice-coverage-gap §4 additions — explicit state-machine
  // coverage for processing / confirming / error.
  // -------------------------------------------------------------------

  describe('5 voice states', () => {
    beforeEach(() => {
      useKeyStore.setState({
        validated: ['google'],
        statuses: { google: 'set' } as any,
      });
    });

    it('idle: renders the mic button and the keyboard hint', () => {
      useVoiceStore.setState({ isOpen: true, voiceState: 'idle' });
      render(
        <MemoryRouter>
          <VoicePanel />
        </MemoryRouter>,
      );
      expect(screen.getByRole('button', { name: /🎤/ })).toBeInTheDocument();
      expect(screen.getByText(/Click to start speaking/i)).toBeInTheDocument();
      expect(screen.getByText(/Ctrl\+Shift\+V/i)).toBeInTheDocument();
    });

    it('listening: shows stop button and "Speak now..." placeholder when transcript is empty', () => {
      useVoiceStore.setState({
        isOpen: true,
        voiceState: 'listening',
        finalTranscript: '',
        interimTranscript: '',
      });
      render(
        <MemoryRouter>
          <VoicePanel />
        </MemoryRouter>,
      );
      expect(screen.getByText(/Listening... Click to process/i)).toBeInTheDocument();
      expect(screen.getByText(/Speak now/i)).toBeInTheDocument();
      expect(screen.getByRole('button', { name: /Cancel/i })).toBeInTheDocument();
    });

    it('processing: shows the spinner label and a Cancel control', () => {
      useVoiceStore.setState({
        isOpen: true,
        voiceState: 'processing',
      });
      render(
        <MemoryRouter>
          <VoicePanel />
        </MemoryRouter>,
      );
      expect(screen.getByText(/Extracting intent/i)).toBeInTheDocument();
      expect(screen.getByRole('button', { name: /Cancel/i })).toBeInTheDocument();
    });

    it('confirming: shows the parsed-intent summary, confidence indicator, and the three actions', () => {
      useVoiceStore.setState({
        isOpen: true,
        voiceState: 'confirming',
        intent: {
          name: 'Voice Time',
          nodes: [
            { display_name: 'Boss', role: 'supervisor', provider: 'kiro_cli' },
            { display_name: 'Coder', role: 'developer', provider: 'claude_code' },
          ],
          edges: [{ from: 'Boss', to: 'Coder', type: 'handoff' }],
          confidence: 0.87,
        },
      });
      render(
        <MemoryRouter>
          <VoicePanel />
        </MemoryRouter>,
      );

      // Intent summary
      expect(screen.getByText(/Extracted Canvas Structure/i)).toBeInTheDocument();
      expect(screen.getByText('Voice Time')).toBeInTheDocument();
      expect(screen.getByText('Boss')).toBeInTheDocument();
      expect(screen.getByText('Coder')).toBeInTheDocument();
      // The handoff badge appears with the type label
      expect(screen.getByText(/Boss ➔ Coder/i)).toBeInTheDocument();

      // Counts + confidence indicator
      expect(screen.getByText(/2 Agents/i)).toBeInTheDocument();
      expect(screen.getByText(/1 Connections/i)).toBeInTheDocument();
      expect(screen.getByText(/87% Confidence/i)).toBeInTheDocument();

      // Three actions
      expect(screen.getByRole('button', { name: /^Cancel$/i })).toBeInTheDocument();
      expect(
        screen.getByRole('button', { name: /Edit Before Deploy/i }),
      ).toBeInTheDocument();
      expect(
        screen.getByRole('button', { name: /Generate.*Deploy/i }),
      ).toBeInTheDocument();
    });

    it('confirming: hides confidence indicator when confidence is missing', () => {
      useVoiceStore.setState({
        isOpen: true,
        voiceState: 'confirming',
        intent: {
          name: 'Plain Time',
          nodes: [{ display_name: 'Boss', role: 'supervisor', provider: 'kiro_cli' }],
          edges: [],
        },
      });
      render(
        <MemoryRouter>
          <VoicePanel />
        </MemoryRouter>,
      );
      expect(screen.getByText(/Plain Time/i)).toBeInTheDocument();
      // The confidence badge should not render — its label always contains "% Confidence".
      expect(screen.queryByText(/% Confidence/i)).not.toBeInTheDocument();
      expect(
        screen.getByText(/No handoff edges declared/i),
      ).toBeInTheDocument();
    });

    it('error: shows the error message and a Retry control', () => {
      useVoiceStore.setState({
        isOpen: true,
        voiceState: 'error',
        error: 'NLU_TIMEOUT',
      });
      render(
        <MemoryRouter>
          <VoicePanel />
        </MemoryRouter>,
      );
      expect(screen.getByText(/An error occurred/i)).toBeInTheDocument();
      expect(screen.getByText('NLU_TIMEOUT')).toBeInTheDocument();
      expect(screen.getByRole('button', { name: /Retry/i })).toBeInTheDocument();
      expect(screen.getByRole('button', { name: /Cancel/i })).toBeInTheDocument();
    });
  });
});

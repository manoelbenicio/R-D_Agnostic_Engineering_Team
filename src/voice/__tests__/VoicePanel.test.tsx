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
});

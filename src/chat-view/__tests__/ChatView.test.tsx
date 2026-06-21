import React from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { act, fireEvent, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { goCoreClient } from '@/api';
import { ChatView } from '../ChatView';

const subscribers = new Map<string, (frame: ArrayBuffer) => void>();

vi.mock('@/api/terminal-socket-fanout', () => ({
  subscribeTerminalSocket: vi.fn((terminalId: string, subscriber: { onBinary: (frame: ArrayBuffer) => void }) => {
    subscribers.set(terminalId, subscriber.onBinary);
    return { unsubscribe: vi.fn() };
  }),
}));

describe('ChatView', () => {
  beforeEach(() => {
    subscribers.clear();
    vi.restoreAllMocks();
    window.matchMedia = vi.fn((query: string) =>
      ({
        matches: query === '(max-width: 768px)',
        media: query,
        onchange: null,
        addEventListener: vi.fn(),
        removeEventListener: vi.fn(),
        addListener: vi.fn(),
        removeListener: vi.fn(),
        dispatchEvent: vi.fn(),
      }) as unknown as MediaQueryList
    );
  });

  it('renders a bubble for each segment from mocked fan-out frames', async () => {
    renderWithQueryClient(<ChatView sessionName="demo-session" />);

    await waitFor(() => expect(subscribers.has('term-supervisor')).toBe(true));
    act(() => {
      subscribers.get('term-supervisor')?.(encodeFrame('\x1b[31mERROR\x1b[0m hello\n'));
      subscribers.get('term-developer')?.(encodeFrame('Tool call: run tests\n'));
    });

    expect(await screen.findByText('ERROR hello')).toBeInTheDocument();
    expect(await screen.findByText('Tool call: run tests')).toBeInTheDocument();
    expect(screen.getByText('Supervisor')).toBeInTheDocument();
    expect(screen.getAllByText('Developer').length).toBeGreaterThan(0);
  });

  it('submits composer input to the most recently bubbled terminal on Enter', async () => {
    const sendSpy = vi.spyOn(goCoreClient, 'sendTerminalInput').mockResolvedValue(undefined);
    const user = userEvent.setup();
    renderWithQueryClient(<ChatView sessionName="demo-session" />);

    await waitFor(() => expect(subscribers.has('term-developer')).toBe(true));
    act(() => {
      subscribers.get('term-developer')?.(encodeFrame('ready\n'));
    });

    const composer = await screen.findByLabelText('Mensagem');
    await user.type(composer, 'continue');
    fireEvent.keyDown(composer, { key: 'Enter', code: 'Enter' });

    await waitFor(() => expect(sendSpy).toHaveBeenCalledWith('term-developer', 'continue'));
  });

  it('defaults to chat view on a mobile viewport match', async () => {
    renderWithQueryClient(<ChatView sessionName="demo-session" />);

    await waitFor(() => expect(screen.getByLabelText('Chat View')).toHaveAttribute('data-view-mode', 'chat'));
  });
});

function renderWithQueryClient(ui: React.ReactElement) {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
    },
  });

  return render(<QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>);
}

function encodeFrame(text: string): ArrayBuffer {
  return new TextEncoder().encode(text).buffer;
}

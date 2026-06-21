import React from 'react';
import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider, useQuery } from '@tanstack/react-query';
import { MemoryRouter, Routes, Route } from 'react-router-dom';
import { TerminalGrid } from '../TerminalGrid';
import { goCoreClient } from '@/api';
import { TerminalSocketFanout } from '@/api/terminal-socket-fanout';

// Mock the TerminalView component to bypass xterm.js WebGL and WebSocket dependencies
vi.mock('@/terminal', () => ({
  TerminalView: ({ terminalId, readOnly }: any) => (
    <div data-testid={`terminal-view-${terminalId}`} data-readonly={readOnly}>
      Mock Terminal {terminalId} {readOnly ? '(ReadOnly)' : '(Interactive)'}
    </div>
  ),
}));

// Mock the shell toasts hook
const mockToast = {
  info: vi.fn(),
  error: vi.fn(),
  success: vi.fn(),
  warning: vi.fn(),
};
vi.mock('@/shell/toasts', () => ({
  useToast: () => mockToast,
}));

// Mock the goCoreClient API methods
vi.mock('@/api', async (importOriginal) => {
  const original = await importOriginal<any>();
  return {
    ...original,
    goCoreClient: {
      listTerminalsInSession: vi.fn(),
      deleteTerminal: vi.fn(),
      getTerminalWorkingDirectory: vi.fn(),
      listInboxMessages: vi.fn(),
      sendTerminalInput: vi.fn(),
    },
  };
});

// Mock @tanstack/react-query at the module level to spy on useQuery calls
vi.mock('@tanstack/react-query', async (importOriginal) => {
  const original = await importOriginal<any>();
  return {
    ...original,
    useQuery: vi.fn(original.useQuery),
  };
});

describe('Terminal Grid & Tab Bar', () => {
  let queryClient: QueryClient;
  const mockTerminals = [
    { id: 'term-1', profile: 'supervisor', display_name: 'Supervisor Node', status: 'idle', working_directory: '/app' },
    { id: 'term-2', profile: 'developer', display_name: 'Developer Node', status: 'processing', working_directory: '/app/src' },
  ];

  beforeEach(() => {
    queryClient = new QueryClient({
      defaultOptions: {
        queries: {
          retry: false,
          gcTime: 0,
        },
      },
    });
    vi.mocked(goCoreClient.listTerminalsInSession).mockResolvedValue(mockTerminals);
    vi.mocked(goCoreClient.getTerminalWorkingDirectory).mockResolvedValue('/app/workspace');
    vi.mocked(goCoreClient.listInboxMessages).mockResolvedValue([
      { id: 'msg-1', terminal_id: 'term-1', message: 'Hello from supervisor', status: 'unread', created_at: '' },
    ]);
    vi.mocked(goCoreClient.deleteTerminal).mockResolvedValue();
    vi.mocked(goCoreClient.sendTerminalInput).mockResolvedValue();
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  const renderWithProviders = (ui: React.ReactElement, initialEntries = ['/canvas/canvas-1/terminal/term-1']) => {
    return render(
      <QueryClientProvider client={queryClient}>
        <MemoryRouter initialEntries={initialEntries}>
          <Routes>
            <Route path="/canvas/:id/terminal/:terminalId" element={ui} />
          </Routes>
        </MemoryRouter>
      </QueryClientProvider>
    );
  };

  it('renders tabs based on query data', async () => {
    renderWithProviders(<TerminalGrid sessionName="session-1" />);

    await waitFor(() => {
      expect(screen.getByTestId('terminal-tab-term-1')).toBeInTheDocument();
      expect(screen.getByTestId('terminal-tab-term-2')).toBeInTheDocument();
    });

    expect(screen.getByText('Supervisor Node')).toBeInTheDocument();
    expect(screen.getByText('Developer Node')).toBeInTheDocument();
  });

  it('polls listTerminalsInSession every 3 seconds', async () => {
    renderWithProviders(<TerminalGrid sessionName="session-1" />);

    await waitFor(() => {
      expect(useQuery).toHaveBeenCalled();
    });

    // Verify useQuery is configured to poll every 3 seconds with background execution disabled
    expect(useQuery).toHaveBeenCalledWith(
      expect.objectContaining({
        refetchInterval: 3000,
        refetchIntervalInBackground: false,
      })
    );
  });

  it('changes focused tab on click', async () => {
    renderWithProviders(<TerminalGrid sessionName="session-1" />);

    await waitFor(() => {
      expect(screen.getByTestId('terminal-tab-term-2')).toBeInTheDocument();
    });

    const tab2 = screen.getByTestId('terminal-tab-term-2');
    fireEvent.click(tab2);

    await waitFor(() => {
      expect(screen.getByTestId('terminal-view-term-2')).toBeInTheDocument();
    });
  });

  it('toggles to Grid View and shows all terminals in readonly mode', async () => {
    renderWithProviders(<TerminalGrid sessionName="session-1" />);

    await waitFor(() => {
      expect(screen.getByTestId('toggle-grid-btn')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByTestId('toggle-grid-btn'));

    await waitFor(() => {
      expect(screen.getByTestId('grid-cell-term-1')).toBeInTheDocument();
      expect(screen.getByTestId('grid-cell-term-2')).toBeInTheDocument();
    });

    // Verify mini-terminals are read-only
    const terms = screen.getAllByTestId(/terminal-view-term-/);
    expect(terms).toHaveLength(2);
    expect(terms[0]?.getAttribute('data-readonly')).toBe('true');
  });

  it('transitions back to Tabs view and focuses terminal when grid cell is clicked', async () => {
    renderWithProviders(<TerminalGrid sessionName="session-1" />);

    await waitFor(() => {
      fireEvent.click(screen.getByTestId('toggle-grid-btn'));
    });

    await waitFor(() => {
      const cell2 = screen.getByTestId('grid-cell-term-2');
      fireEvent.click(cell2);
    });

    await waitFor(() => {
      expect(screen.getByTestId('terminal-view-term-2')).toBeInTheDocument();
      expect(screen.queryByTestId('grid-cell-term-2')).not.toBeInTheDocument();
    });
  });

  it('exits fullscreen mode when Escape key is pressed', async () => {
    renderWithProviders(<TerminalGrid sessionName="session-1" />);

    await waitFor(() => {
      expect(screen.getByTestId('fullscreen-btn')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByTestId('fullscreen-btn'));

    // Verify fullscreen view
    expect(screen.getByText(/Fullscreen:/)).toBeInTheDocument();

    // Simulate Escape key press
    fireEvent.keyDown(window, { key: 'Escape', code: 'Escape' });

    await waitFor(() => {
      expect(screen.queryByText(/Fullscreen:/)).not.toBeInTheDocument();
      expect(screen.getByTestId('fullscreen-btn')).toBeInTheDocument();
    });
  });

  it('opens confirmation modal and autofocuses Cancel button when close is clicked', async () => {
    renderWithProviders(<TerminalGrid sessionName="session-1" />);

    await waitFor(() => {
      expect(screen.getByTestId('terminal-tab-term-1')).toBeInTheDocument();
    });

    // Close/Kill term-1
    const closeBtn = screen.getByLabelText('Close tab for Supervisor Node');
    fireEvent.click(closeBtn);

    // Verify Modal opens using test ID for button in modal to disambiguate heading vs button
    await waitFor(() => {
      expect(screen.getByTestId('kill-confirm-btn')).toBeInTheDocument();
    });

    // Check autoFocus element or Cancel button has focus
    const cancelBtn = screen.getByTestId('kill-cancel-btn');
    expect(cancelBtn).toHaveFocus();
  });

  it('calls deleteTerminal when kill is confirmed in modal', async () => {
    renderWithProviders(<TerminalGrid sessionName="session-1" />);

    await waitFor(() => {
      expect(screen.getByTestId('terminal-tab-term-1')).toBeInTheDocument();
    });

    const closeBtn = screen.getByLabelText('Close tab for Supervisor Node');
    fireEvent.click(closeBtn);

    const confirmBtn = screen.getByTestId('kill-confirm-btn');
    fireEvent.click(confirmBtn);

    await waitFor(() => {
      expect(goCoreClient.deleteTerminal).toHaveBeenCalledWith('term-1');
      expect(mockToast.success).toHaveBeenCalledWith('Terminal killed successfully');
    });
  });
});

describe('Terminal Socket Fanout Invariant', () => {
  it('implements single WebSocket connection per terminal ID for multiple subscribers', () => {
    const mockConnect = vi.fn().mockReturnValue({
      close: vi.fn(),
    });

    const fanout = new TerminalSocketFanout(mockConnect);

    const sub1 = { onBinary: vi.fn() };
    const sub2 = { onBinary: vi.fn() };

    // Subscribe first consumer
    const subscription1 = fanout.subscribe('term-1', sub1);
    expect(mockConnect).toHaveBeenCalledTimes(1);
    expect(fanout.getConnectionCount()).toBe(1);

    // Subscribe second consumer for same terminal ID
    const subscription2 = fanout.subscribe('term-1', sub2);
    expect(mockConnect).toHaveBeenCalledTimes(1); // Connect should NOT be called again
    expect(fanout.getConnectionCount()).toBe(1);

    // Unsubscribe first consumer
    subscription1.unsubscribe();
    expect(fanout.getConnectionCount()).toBe(1); // Connection still active because sub2 is still active

    // Unsubscribe second consumer
    subscription2.unsubscribe();
    expect(fanout.getConnectionCount()).toBe(0); // All subscribers gone, connection closed
  });
});

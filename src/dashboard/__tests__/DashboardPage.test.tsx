import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router-dom';
import { beforeAll, beforeEach, describe, expect, it, vi } from 'vitest';
import { goCoreClient } from '@/api';

const mockGoCoreClient = vi.hoisted(() => ({
  listSessions: vi.fn(),
  listTerminalsInSession: vi.fn(),
  listInboxMessages: vi.fn(),
}));

const mockRefreshSessions = vi.hoisted(() => vi.fn());

const mockCanvasStore = vi.hoisted(() => ({
  list: vi.fn(),
}));

vi.mock('@/terminal', () => ({
  TerminalView: ({ terminalId, readOnly }: { terminalId: string; readOnly?: boolean }) => (
    <div data-testid={`terminal-preview-${terminalId}`} data-readonly={readOnly}>
      Terminal {terminalId}
    </div>
  ),
}));

vi.mock('@/api', () => ({
  goCoreClient: mockGoCoreClient,
}));

vi.mock('@/api/session-store', () => ({
  useSessionStore: Object.assign(
    vi.fn(() => ({
      sessions: [],
      refresh: mockRefreshSessions,
    })),
    {
      getState: () => ({ sessions: [] }),
    }
  ),
}));

vi.mock('@/finops', async (importOriginal) => {
  const original = await importOriginal<object>();
  return {
    ...original,
    useCostEstimate: vi.fn(() => ({
      data: {
        total: 25,
        byProvider: { openai: 10, anthropic: 15 },
        byCanvas: {},
        terminals: [],
        activeTerminalsCount: 3,
        currentHourlyRate: 9,
      },
    })),
  };
});

vi.mock('@/settings/settings-store', () => ({
  useSettingsStore: (selector: (state: unknown) => unknown) =>
    selector({
      finopsBudgetUsd: 100,
    }),
}));

vi.mock('@/canvas-document/store', () => ({
  canvasStore: mockCanvasStore,
}));

vi.mock('recharts', () => ({
  ResponsiveContainer: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  BarChart: ({ children }: { children: React.ReactNode }) => <div data-testid="provider-chart">{children}</div>,
  Bar: ({ children }: { children?: React.ReactNode }) => <div>{children}</div>,
  Cell: () => <div />,
  PieChart: ({ children }: { children: React.ReactNode }) => <div data-testid="fleet-donut">{children}</div>,
  Pie: ({ children }: { children?: React.ReactNode }) => <div>{children}</div>,
  Tooltip: () => <div />,
  XAxis: () => <div />,
  YAxis: () => <div />,
}));

let DashboardPage: typeof import('../DashboardPage').DashboardPage;

describe('DashboardPage', () => {
  vi.setConfig({ hookTimeout: 30_000 });
  beforeAll(async () => {
    ({ DashboardPage } = await import('../DashboardPage'));
  });

  beforeEach(() => {
    vi.clearAllMocks();
    mockCanvasStore.list.mockResolvedValue([
      {
        id: 'canvas-1',
        deploy_state: { terminal_map: { node1: 'term-1' } },
      },
    ]);
    vi.mocked(goCoreClient.listSessions).mockResolvedValue([
      { name: 'session-1', profile: 'supervisor', working_directory: '~', status: 'active' },
    ]);
    vi.mocked(goCoreClient.listTerminalsInSession).mockResolvedValue([
      {
        id: 'term-1',
        session_name: 'session-1',
        profile: 'supervisor',
        provider: 'openai',
        display_name: 'Supervisor',
        status: 'idle',
        working_directory: '~',
        created_at: '2026-05-27T20:00:00.000Z',
      },
      {
        id: 'term-2',
        session_name: 'session-1',
        profile: 'reviewer',
        provider: 'anthropic',
        display_name: 'Reviewer',
        status: 'error',
        working_directory: '~',
        created_at: '2026-05-27T20:01:00.000Z',
      },
      {
        id: 'term-3',
        session_name: 'session-1',
        profile: 'worker',
        provider: 'openai',
        display_name: 'Worker',
        status: 'offline',
        working_directory: '~',
        created_at: '2026-05-27T20:02:00.000Z',
      },
    ]);
    vi.mocked(goCoreClient.listInboxMessages).mockResolvedValue([
      {
        id: 'msg-1',
        terminal_id: 'term-1',
        message: 'ready for review',
        status: 'unread',
        sender: 'Supervisor',
        created_at: '2026-05-27T20:03:00.000Z',
      },
    ]);
  });

  it('renders KPIs from session and finops data', async () => {
    renderDashboard();

    await waitFor(() => {
      expect(screen.getByText('3 total terminals')).toBeInTheDocument();
    });

    expect(screen.getByText('$25.00')).toBeInTheDocument();
    expect(screen.getByText('25%')).toBeInTheDocument();
    expect(screen.getByText('terminals in error')).toBeInTheDocument();
  });

  it('renders provider chart, donut legend, activity feed, and terminal preview', async () => {
    renderDashboard();

    await waitFor(() => {
      expect(screen.getByTestId('terminal-preview-term-1')).toBeInTheDocument();
    });

    expect(screen.getByTestId('provider-chart')).toBeInTheDocument();
    expect(screen.getByTestId('fleet-donut')).toBeInTheDocument();
    expect(screen.getByText(/active:/)).toHaveTextContent('active: 1');
    expect(screen.getByText(/error:/)).toHaveTextContent('error: 1');
    expect(screen.getByText(/offline:/)).toHaveTextContent('offline: 1');
    expect(screen.getByText(/ready for review/)).toBeInTheDocument();
    expect(screen.getByTestId('terminal-preview-term-1')).toHaveAttribute('data-readonly', 'true');
  });
});

function renderDashboard() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
        gcTime: 0,
      },
    },
  });

  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <DashboardPage />
      </MemoryRouter>
    </QueryClientProvider>
  );
}

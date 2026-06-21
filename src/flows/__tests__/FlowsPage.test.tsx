import { fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { FlowsPage } from '../FlowsPage';
import { goCoreClient, Flow } from '@/api';

vi.mock('@monaco-editor/react', () => ({
  default: ({
    value,
    onChange,
  }: {
    value?: string;
    onChange?: (value: string | undefined) => void;
  }) => (
    <textarea
      aria-label="Prompt template"
      value={value ?? ''}
      onChange={(event) => onChange?.(event.target.value)}
    />
  ),
}));

vi.mock('@/api', async (importOriginal) => {
  const original = await importOriginal<object>();
  return {
    ...original,
    goCoreClient: {
      listFlows: vi.fn(),
      createFlow: vi.fn(),
      runFlow: vi.fn(),
      enableFlow: vi.fn(),
      disableFlow: vi.fn(),
      listProfiles: vi.fn(),
    },
  };
});

vi.mock('@/api/key-store/use-validated-providers', () => ({
  useValidatedProviders: () => ['openai', 'anthropic'],
}));

const toast = {
  info: vi.fn(),
  error: vi.fn(),
  success: vi.fn(),
  warning: vi.fn(),
};

vi.mock('@/shell/toasts', () => ({
  useToast: () => toast,
}));

let flows: Flow[];

describe('FlowsPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    flows = [
      {
        name: 'nightly-review',
        schedule: '0 2 * * *',
        agent_profile: 'supervisor',
        provider: 'openai',
        prompt_template: 'Run the nightly review.',
        enabled: true,
        last_run: null,
        next_run: '2026-05-29T02:00:00.000Z',
        gating_script: 'return budget < 100',
      },
    ];

    vi.mocked(goCoreClient.listFlows).mockImplementation(async () => flows);
    vi.mocked(goCoreClient.listProfiles).mockResolvedValue([
      {
        name: 'supervisor',
        role: 'Supervisor',
        provider: 'openai',
      },
    ]);
    vi.mocked(goCoreClient.createFlow).mockImplementation(async (flow: Flow) => {
      flows = [flow, ...flows.filter((item) => item.name !== flow.name)];
      return flow;
    });
    vi.mocked(goCoreClient.runFlow).mockResolvedValue();
    vi.mocked(goCoreClient.disableFlow).mockImplementation(async (name: string) => {
      flows = flows.map((flow) => (flow.name === name ? { ...flow, enabled: false } : flow));
    });
    vi.mocked(goCoreClient.enableFlow).mockImplementation(async (name: string) => {
      flows = flows.map((flow) => (flow.name === name ? { ...flow, enabled: true } : flow));
    });
  });

  it('rejects invalid cron before submit', async () => {
    renderFlows();

    fireEvent.click(await screen.findByRole('button', { name: 'New Flow' }));
    fireEvent.change(await screen.findByLabelText(/Name/), { target: { value: 'bad-cron-flow' } });
    fireEvent.change(screen.getByLabelText(/Agent Profile/), { target: { value: 'supervisor' } });
    fireEvent.change(screen.getByLabelText(/Provider/), { target: { value: 'openai' } });
    fireEvent.change(screen.getByLabelText('Prompt template'), { target: { value: 'Run this.' } });
    fireEvent.change(screen.getByLabelText(/Raw cron/), { target: { value: 'not cron' } });

    expect(screen.getAllByText(/Expression has only 2 parts/).length).toBeGreaterThan(0);
    expect(screen.getByRole('button', { name: 'Save Flow' })).toBeDisabled();
    expect(goCoreClient.createFlow).not.toHaveBeenCalled();
  });

  it('quick-pick fills every 5 minutes cron', async () => {
    renderFlows();

    fireEvent.click(await screen.findByRole('button', { name: 'New Flow' }));
    fireEvent.change(screen.getByLabelText('Every minutes'), { target: { value: '5' } });

    expect(screen.getByLabelText(/Raw cron/)).toHaveValue('*/5 * * * *');
  });

  it('toggles enabled state and persists after refresh', async () => {
    renderFlows();

    const card = await screen.findByText('nightly-review').then((title) => title.closest('.flow-card') as HTMLElement);
    expect(within(card).getByText('Enabled')).toBeInTheDocument();

    fireEvent.click(within(card).getByRole('checkbox'));

    await waitFor(() => {
      expect(goCoreClient.disableFlow).toHaveBeenCalledWith('nightly-review');
    });
    await waitFor(() => {
      expect(screen.getByText('Disabled')).toBeInTheDocument();
    });

    expect(goCoreClient.listFlows).toHaveBeenCalled();
  });
});

function renderFlows() {
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
      <FlowsPage />
    </QueryClientProvider>
  );
}

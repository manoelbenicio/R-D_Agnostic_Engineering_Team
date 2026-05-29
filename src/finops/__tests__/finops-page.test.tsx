import React from 'react';
import { fireEvent, render, screen, within } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { FinopsPage } from '../FinopsPage';

const updateSetting = vi.fn();

const costEstimateMock = vi.hoisted(() => ({
  useCostEstimate: vi.fn(),
}));

vi.mock('../use-cost-estimate', () => ({
  useCostEstimate: costEstimateMock.useCostEstimate,
}));

vi.mock('@/settings/settings-store', () => ({
  useSettingsStore: (selector: (state: unknown) => unknown) =>
    selector({
      finopsBudgetUsd: 100,
      updateSetting,
    }),
}));

vi.mock('recharts', () => ({
  ResponsiveContainer: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  RadialBarChart: ({ children }: { children: React.ReactNode }) => <div data-testid="mock-radial-chart">{children}</div>,
  RadialBar: () => <div data-testid="mock-radial-bar" />,
}));

beforeEach(() => {
  updateSetting.mockClear();
  costEstimateMock.useCostEstimate.mockReturnValue({
    isLoading: false,
    error: null,
    data: {
      total: 47,
      byProvider: {
        claude_code: 30,
        codex: 15,
        gemini_cli: 2,
      },
      byCanvas: Object.fromEntries(
        Array.from({ length: 12 }, (_, index) => [`canvas-${index + 1}`, index + 1])
      ),
      activeTerminalsCount: 4,
      currentHourlyRate: 20.5,
      terminals: [],
    },
  });
});

describe('FinopsPage', () => {
  it('renders the warning glyph in the MTD KPI via CostLabel', () => {
    render(<FinopsPage />);

    const mtdCard = screen.getByText('MTD Cost').closest('.sentinel-card');
    expect(mtdCard).not.toBeNull();
    expect(within(mtdCard as HTMLElement).getByText('⚠️')).toBeInTheDocument();
    expect(within(mtdCard as HTMLElement).getByText('$47.00')).toBeInTheDocument();
  });

  it('shows the top 10 canvases sorted by descending cost', () => {
    render(<FinopsPage />);

    const heading = screen.getByRole('heading', { name: 'Top 10 Cost by Canvas' });
    const panel = heading.closest('.sentinel-card') as HTMLElement;
    const rows = within(panel).getAllByRole('row').slice(1);

    expect(rows).toHaveLength(10);
    expect(within(rows[0] as HTMLElement).getByText('canvas-12')).toBeInTheDocument();
    expect(within(rows[9] as HTMLElement).getByText('canvas-3')).toBeInTheDocument();
    expect(within(panel).queryByText('canvas-2')).not.toBeInTheDocument();
  });

  it('writes the monthly budget to settings-store on submit', () => {
    render(<FinopsPage />);

    const input = screen.getByLabelText('Budget USD');
    fireEvent.change(input, { target: { value: '250' } });
    fireEvent.click(screen.getByRole('button', { name: 'Save Budget' }));

    expect(updateSetting).toHaveBeenCalledWith('finopsBudgetUsd', 250);
  });
});

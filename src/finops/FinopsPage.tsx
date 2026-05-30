import React, { FormEvent, useMemo, useState } from 'react';
import { RadialBar, RadialBarChart, ResponsiveContainer } from 'recharts';
import { Badge, Card, CostLabel, FormField, Button } from '@/design-system';
// eslint-disable-next-line agentverse/no-sideways-capability-imports
import { useSettingsStore } from '@/settings/settings-store';
import { useCostEstimate } from './use-cost-estimate';
import { useTokenCost } from './use-token-cost';
import type { CostConfidence } from './token-cost';
import { formatPercent, formatUsd } from './format';
import { CostWarning } from './cost-warning';
import { COST_ESTIMATE_DISCLAIMER } from './cost-warning-constants';
import './finops.css';

type SettingsWithBudget = {
  finopsBudgetUsd?: number;
  updateSetting: (key: string, value: unknown) => Promise<void>;
};

export const FinopsPage: React.FC = () => {
  const window = useMemo(() => getMonthToDateWindow(Date.now()), []);
  const { data, isLoading, error } = useCostEstimate(window);
  const { data: tokenCost } = useTokenCost(window);
  const settings = useSettingsStore((state) => state as unknown as SettingsWithBudget);
  const budgetUsd = settings.finopsBudgetUsd ?? 100;
  const [budgetInput, setBudgetInput] = useState(String(budgetUsd));
  const estimate = data ?? {
    total: 0,
    byProvider: {},
    byCanvas: {},
    activeTerminalsCount: 0,
    currentHourlyRate: 0,
    terminals: [],
  };
  const budgetUtil = budgetUsd > 0 ? Math.min(100, (estimate.total / budgetUsd) * 100) : 0;
  const providerRows = toSortedRows(estimate.byProvider);
  const canvasRows = toSortedRows(estimate.byCanvas).slice(0, 10);

  const handleBudgetSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const nextBudget = Math.max(0, Number(budgetInput) || 0);
    void settings.updateSetting('finopsBudgetUsd', nextBudget);
  };

  return (
    <main className="finops-page">
      <header className="finops-header">
        <div>
          <h1>FinOps</h1>
          <p>{COST_ESTIMATE_DISCLAIMER}</p>
        </div>
        {isLoading ? <Badge variant="processing">Refreshing</Badge> : <Badge variant="completed">Live</Badge>}
      </header>

      {error ? (
        <Card glow="red" role="alert">
          Unable to load FinOps estimate.
        </Card>
      ) : null}

      <section className="finops-kpi-grid" aria-label="FinOps KPIs">
        <KpiCard title="MTD Cost">
          <CostLabel value={formatUsd(estimate.total)} />
        </KpiCard>
        <KpiCard title="Budget Utilization">
          <div className="budget-gauge" data-testid="budget-gauge">
            <CostLabel value={formatPercent(budgetUtil)} />
            <div className="budget-gauge-spark" aria-hidden="true">
              <ResponsiveContainer width="100%" height={48}>
                <RadialBarChart
                  cx="50%"
                  cy="100%"
                  innerRadius="80%"
                  outerRadius="100%"
                  barSize={6}
                  data={[{ name: 'Budget', value: budgetUtil, fill: 'var(--indra-gold)' }]}
                  startAngle={180}
                  endAngle={0}
                >
                  <RadialBar dataKey="value" cornerRadius={3} background />
                </RadialBarChart>
              </ResponsiveContainer>
            </div>
          </div>
        </KpiCard>
        <KpiCard title="Active Terminals">
          <span className="finops-kpi-value">{estimate.activeTerminalsCount}</span>
        </KpiCard>
        <KpiCard title="Cost Rate">
          <CostLabel value={`${formatUsd(estimate.currentHourlyRate)}/hr`} />
        </KpiCard>
        <KpiCard title="Token Cost (Tier 2)">
          <CostLabel value={formatUsd(tokenCost?.total ?? 0)} />
          <ConfidenceBadge confidence={tokenCost?.confidence ?? 'estimated'} />
        </KpiCard>
      </section>

      <section className="finops-content-grid">
        <Card className="finops-panel">
          <PanelTitle title="Cost by Provider" />
          <CostTable rows={providerRows} total={estimate.total} emptyLabel="No provider cost yet." />
        </Card>

        <Card className="finops-panel">
          <PanelTitle title="Top 10 Cost by Canvas" />
          <CostTable rows={canvasRows} total={estimate.total} emptyLabel="No canvas cost yet." />
        </Card>
      </section>

      <Card className="finops-panel">
        <PanelTitle title="Monthly Budget" />
        <form className="budget-form" onSubmit={handleBudgetSubmit}>
          <FormField label="Budget USD" id="finops-budget-usd">
            <input
              id="finops-budget-usd"
              type="number"
              min="0"
              step="1"
              value={budgetInput}
              onChange={(event) => setBudgetInput(event.target.value)}
            />
          </FormField>
          <Button type="submit" variant="primary">
            Save Budget
          </Button>
        </form>
      </Card>

      <footer className="finops-footer">
        Tier 2 (token-level) and Tier 3 (provider billing) tracked post-launch — master spec §13
      </footer>
    </main>
  );
};

function KpiCard({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <Card className="finops-kpi-card">
      <span className="finops-label">{title}</span>
      <div className="finops-kpi-body">{children}</div>
    </Card>
  );
}

function ConfidenceBadge({ confidence }: { confidence: CostConfidence }) {
  const variant = confidence === 'measured' ? 'completed' : confidence === 'mixed' ? 'processing' : 'idle';
  const label =
    confidence === 'measured'
      ? 'Measured'
      : confidence === 'mixed'
        ? 'Partially measured'
        : 'Estimated';
  return (
    <Badge variant={variant} title="Measured = from real token usage; Estimated = Tier 1 wall-clock fallback">
      {label}
    </Badge>
  );
}

function PanelTitle({ title }: { title: string }) {
  return (
    <div className="finops-panel-title">
      <h2>{title}</h2>
      <CostWarning />
    </div>
  );
}

function CostTable({
  rows,
  total,
  emptyLabel,
}: {
  rows: Array<{ id: string; cost: number }>;
  total: number;
  emptyLabel: string;
}) {
  if (rows.length === 0) {
    return <p className="finops-empty">{emptyLabel}</p>;
  }

  return (
    <table className="finops-table">
      <thead>
        <tr>
          <th>Name</th>
          <th>Cost</th>
          <th>Share</th>
        </tr>
      </thead>
      <tbody>
        {rows.map((row) => (
          <tr key={row.id}>
            <td>
              <Badge>{row.id}</Badge>
            </td>
            <td>
              <CostLabel value={formatUsd(row.cost)} />
            </td>
            <td>{total > 0 ? formatPercent((row.cost / total) * 100) : '0%'}</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}

function toSortedRows(record: Record<string, number>): Array<{ id: string; cost: number }> {
  return Object.entries(record)
    .map(([id, cost]) => ({ id, cost }))
    .sort((a, b) => b.cost - a.cost || a.id.localeCompare(b.id));
}

function getMonthToDateWindow(nowMs: number) {
  const now = new Date(nowMs);
  return {
    startMs: new Date(now.getFullYear(), now.getMonth(), 1).getTime(),
    endMs: nowMs,
  };
}

export default FinopsPage;

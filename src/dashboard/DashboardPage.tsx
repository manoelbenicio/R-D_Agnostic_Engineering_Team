/* eslint-disable agentverse/no-sideways-capability-imports */
import React, { useEffect, useMemo, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { useNavigate } from 'react-router-dom';
import {
  Bar,
  BarChart,
  Cell,
  Pie,
  PieChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts';
import { Badge, Button, Card, CostLabel } from '@/design-system';
import { caoClient, type InboxMessage, type Session, type Terminal } from '@/api';
import { TerminalView } from '@/terminal';
import { useCostEstimate } from '@/finops';
import { formatPercent, formatUsd } from '@/finops/format';
import { useSettingsStore } from '@/settings/settings-store';
import { canvasStore } from '@/canvas-document/store';
import './dashboard.css';

interface SessionTerminal extends Terminal {
  session_name: string;
  canvas_id?: string;
}

interface ActivityEntry {
  id: string;
  timestamp: string;
  title: string;
  detail: string;
}

interface DashboardSnapshot {
  sessions: Session[];
  terminals: SessionTerminal[];
  inboxEntries: ActivityEntry[];
}

type SettingsWithBudget = {
  finopsBudgetUsd?: number;
};

const CHART_COLORS = ['var(--cyan)', 'var(--ops)', 'var(--amber)', 'var(--threat)', 'var(--text-muted)'];

export const DashboardPage: React.FC = () => {
  const navigate = useNavigate();
  const [activityFeed, setActivityFeed] = useState<ActivityEntry[]>([]);
  const [clearedIds, setClearedIds] = useState<Set<string>>(new Set());
  const budgetUsd = useSettingsStore((state) => (state as unknown as SettingsWithBudget).finopsBudgetUsd ?? 100);
  const costWindow = useMemo(() => getMonthToDateWindow(Date.now()), []);
  const { data: costData } = useCostEstimate(costWindow);
  const { data, isLoading, error } = useQuery({
    queryKey: ['dashboard', 'fleet-snapshot'],
    queryFn: loadDashboardSnapshot,
    refetchInterval: 5_000,
    refetchIntervalInBackground: false,
  });

  const snapshot = data ?? { sessions: [], terminals: [], inboxEntries: [] };
  const fleet = computeFleet(snapshot.terminals);
  const costTotal = costData?.total ?? 0;
  const budgetUtil = budgetUsd > 0 ? Math.min(100, (costTotal / budgetUsd) * 100) : 0;
  const providerRows = Object.entries(costData?.byProvider ?? {})
    .filter(([, value]) => value > 0)
    .map(([provider, cost]) => ({ provider, cost }));
  const donutRows = [
    { name: 'active', value: fleet.active },
    { name: 'error', value: fleet.error },
    { name: 'offline', value: fleet.offline },
  ];
  const watchedTerminal = snapshot.terminals[0];

  useEffect(() => {
    if (!data) return;
    const lifecycleEntries = data.terminals.map((terminal) => ({
      id: `terminal:${terminal.session_name}:${terminal.id}:${terminal.status}`,
      timestamp: terminal.updated_at ?? terminal.created_at ?? new Date().toISOString(),
      title: terminal.display_name ?? terminal.profile,
      detail: `Terminal ${terminal.status} in ${terminal.session_name}`,
    }));
    const incoming = [...data.inboxEntries, ...lifecycleEntries].filter((entry) => !clearedIds.has(entry.id));

    setActivityFeed((current) => mergeActivity(current, incoming));
  }, [data, clearedIds]);

  const clearFeed = () => {
    setClearedIds(new Set(activityFeed.map((entry) => entry.id)));
    setActivityFeed([]);
  };

  return (
    <main className="dashboard-page">
      <header className="dashboard-header">
        <div>
          <h1>Dashboard</h1>
          <p>Central command view for live CAO sessions, fleet activity, and rough cost telemetry.</p>
        </div>
        {isLoading ? <Badge variant="processing">Refreshing</Badge> : <Badge variant="completed">Live</Badge>}
      </header>

      {error ? (
        <Card glow="red" role="alert">
          Unable to load Dashboard data.
        </Card>
      ) : null}

      <section className="dashboard-kpi-grid" aria-label="Dashboard KPIs">
        <KpiCard title="Fleet Status" value={fleet.active} detail={`${snapshot.terminals.length} total terminals`} />
        <KpiCard title="Cost / MTD" detail="rough estimate">
          <CostLabel value={formatUsd(costTotal)} />
        </KpiCard>
        <KpiCard title="Budget Utilization" value={formatPercent(budgetUtil)} detail={`${formatUsd(budgetUsd)} budget`} />
        <KpiCard title="Threats" value={fleet.error} detail="terminals in error" danger={fleet.error > 0} />
      </section>

      <section className="dashboard-chart-grid">
        <Card className="dashboard-panel">
          <PanelTitle title="Cost by Provider" />
          {providerRows.length > 0 ? (
            <ResponsiveContainer width="100%" height={260}>
              <BarChart data={providerRows} margin={{ top: 12, right: 12, left: 0, bottom: 12 }}>
                <XAxis dataKey="provider" stroke="var(--text-muted)" />
                <YAxis stroke="var(--text-muted)" />
                <Tooltip contentStyle={{ background: 'var(--card)', border: '1px solid var(--border)' }} />
                <Bar dataKey="cost" radius={[4, 4, 0, 0]}>
                  {providerRows.map((row, index) => (
                    <Cell key={row.provider} fill={CHART_COLORS[index % CHART_COLORS.length]} />
                  ))}
                </Bar>
              </BarChart>
            </ResponsiveContainer>
          ) : (
            <p className="dashboard-empty">No provider cost yet.</p>
          )}
        </Card>

        <Card className="dashboard-panel">
          <PanelTitle title="Fleet Status" />
          <ResponsiveContainer width="100%" height={260}>
            <PieChart>
              <Tooltip contentStyle={{ background: 'var(--card)', border: '1px solid var(--border)' }} />
              <Pie data={donutRows} dataKey="value" nameKey="name" innerRadius={64} outerRadius={96}>
                {donutRows.map((row, index) => (
                  <Cell key={row.name} fill={CHART_COLORS[index % CHART_COLORS.length]} />
                ))}
              </Pie>
            </PieChart>
          </ResponsiveContainer>
          <div className="dashboard-donut-legend">
            {donutRows.map((row) => (
              <span key={row.name}>
                {row.name}: <strong>{row.value}</strong>
              </span>
            ))}
          </div>
        </Card>
      </section>

      <section className="dashboard-lower-grid">
        <Card className="dashboard-panel dashboard-feed-panel">
          <div className="dashboard-panel-header">
            <PanelTitle title="Activity Feed" />
            <Button variant="secondary" onClick={clearFeed} disabled={activityFeed.length === 0}>
              Clear
            </Button>
          </div>
          {activityFeed.length > 0 ? (
            <ol className="dashboard-feed">
              {activityFeed.map((entry) => (
                <li key={entry.id}>
                  <time dateTime={entry.timestamp}>{formatTime(entry.timestamp)}</time>
                  <strong>{entry.title}</strong>
                  <span>{entry.detail}</span>
                </li>
              ))}
            </ol>
          ) : (
            <p className="dashboard-empty">No activity yet.</p>
          )}
        </Card>

        <Card
          className="dashboard-panel dashboard-terminal-preview"
          onClick={() => {
            if (watchedTerminal) {
              navigate(`/canvas/${watchedTerminal.canvas_id ?? watchedTerminal.session_name}/terminal/${watchedTerminal.id}`);
            }
          }}
          role={watchedTerminal ? 'button' : undefined}
          tabIndex={watchedTerminal ? 0 : undefined}
        >
          <PanelTitle title="Terminal Preview" />
          {watchedTerminal ? (
            <TerminalView terminalId={watchedTerminal.id} readOnly />
          ) : (
            <p className="dashboard-empty">No active terminal to preview.</p>
          )}
        </Card>
      </section>
    </main>
  );
};

function KpiCard({
  title,
  value,
  detail,
  danger = false,
  children,
}: {
  title: string;
  value?: React.ReactNode;
  detail: string;
  danger?: boolean;
  children?: React.ReactNode;
}) {
  return (
    <Card className={`dashboard-kpi-card ${danger ? 'is-danger' : ''}`}>
      <span className="dashboard-label">{title}</span>
      <div className="dashboard-kpi-value">{children ?? value}</div>
      <span className="dashboard-kpi-detail">{detail}</span>
    </Card>
  );
}

function PanelTitle({ title }: { title: string }) {
  return <h2 className="dashboard-panel-title">{title}</h2>;
}

async function loadDashboardSnapshot(): Promise<DashboardSnapshot> {
  const sessions = await caoClient.listSessions();
  const canvases = await canvasStore.list();
  const terminalGroups = await Promise.all(
    sessions.map(async (session) => {
      const terminals = await caoClient.listTerminalsInSession(session.name);
      return terminals.map((terminal) => ({
        ...terminal,
        session_name: terminal.session_name ?? session.name,
        canvas_id: findCanvasIdForTerminal(canvases, terminal.id) ?? session.name,
      }));
    })
  );
  const terminals = terminalGroups.flat();
  const inboxGroups = await Promise.all(
    terminals.map(async (terminal) => {
      const messages = await caoClient.listInboxMessages(terminal.id, { limit: 20 });
      return messages.map((message) => toInboxActivity(message, terminal));
    })
  );

  return {
    sessions,
    terminals,
    inboxEntries: inboxGroups.flat(),
  };
}

function findCanvasIdForTerminal(canvases: Awaited<ReturnType<typeof canvasStore.list>>, terminalId: string): string | undefined {
  return canvases.find((canvas) =>
    Object.values(canvas.deploy_state.terminal_map ?? {}).includes(terminalId)
  )?.id;
}

function toInboxActivity(message: InboxMessage, terminal: SessionTerminal): ActivityEntry {
  return {
    id: `inbox:${message.id}`,
    timestamp: message.created_at,
    title: message.sender ?? terminal.display_name ?? terminal.profile,
    detail: `${terminal.display_name ?? terminal.id}: ${message.message}`,
  };
}

function computeFleet(terminals: Terminal[]) {
  return terminals.reduce(
    (acc, terminal) => {
      if (terminal.status === 'error') acc.error += 1;
      else if (terminal.status === 'offline' || terminal.status === 'exited') acc.offline += 1;
      else acc.active += 1;
      return acc;
    },
    { active: 0, error: 0, offline: 0 }
  );
}

function mergeActivity(current: ActivityEntry[], incoming: ActivityEntry[]): ActivityEntry[] {
  const byId = new Map<string, ActivityEntry>();
  for (const entry of [...current, ...incoming]) {
    byId.set(entry.id, entry);
  }
  return [...byId.values()].sort((a, b) => Date.parse(b.timestamp) - Date.parse(a.timestamp));
}

function formatTime(value: string): string {
  const parsed = Date.parse(value);
  if (Number.isNaN(parsed)) return value;
  return new Intl.DateTimeFormat('en-US', {
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  }).format(parsed);
}

function getMonthToDateWindow(nowMs: number) {
  const now = new Date(nowMs);
  return {
    startMs: new Date(now.getFullYear(), now.getMonth(), 1).getTime(),
    endMs: nowMs,
  };
}

export default DashboardPage;

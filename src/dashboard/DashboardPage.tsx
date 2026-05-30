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
import { useSessionStore } from '@/api/session-store';
import { TerminalView } from '@/terminal';
import { useCostEstimate } from '@/finops';
import { formatPercent, formatUsd } from '@/finops/format';
import { useSettingsStore } from '@/settings/settings-store';
import { canvasStore } from '@/canvas-document/store';
import { ProviderIcon } from '@/sessions';
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
  const { sessions: authSessions, refresh: refreshSessions } = useSessionStore();
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
  const sessionSummary = useMemo(() => summarizeAuthSessions(authSessions), [authSessions]);
  const watchedTerminal = snapshot.terminals[0];

  useEffect(() => {
    if (authSessions.length === 0) {
      void refreshSessions();
    }
  }, [authSessions.length, refreshSessions]);

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
          <p>Central command view for live runtime sessions, fleet activity, and rough cost telemetry.</p>
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
        <KpiCard
          title="AUTH SESSIONS"
          value={sessionSummary.active}
          detail={`${sessionSummary.active} active · ${sessionSummary.expiring} expiring · ${sessionSummary.expired} expired`}
          tone={sessionSummary.tone}
          onClick={() => navigate('/sessions')}
          ariaLabel="Open auth sessions management"
        >
          <div>{sessionSummary.active}</div>
          <div
            style={{
              display: 'flex',
              flexWrap: 'wrap',
              gap: 'var(--space-2)',
              marginTop: 'var(--space-2)',
              fontFamily: 'var(--font-mono)',
              fontSize: '0.72rem',
              color: 'var(--text-muted)',
            }}
          >
            {sessionSummary.providers.length > 0 ? (
              sessionSummary.providers.map((provider) => (
                <span key={provider.id} title={provider.title}>
                  <ProviderIcon provider={provider.id} size={12} /> {provider.shortLabel}: {provider.count}
                </span>
              ))
            ) : (
              <span>No providers</span>
            )}
          </div>
        </KpiCard>
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
  tone = 'neutral',
  onClick,
  ariaLabel,
  danger = false,
  children,
}: {
  title: string;
  value?: React.ReactNode;
  detail: string;
  tone?: 'neutral' | 'healthy' | 'warning' | 'danger';
  onClick?: () => void;
  ariaLabel?: string;
  danger?: boolean;
  children?: React.ReactNode;
}) {
  const isInteractive = Boolean(onClick);
  const handleKeyDown = (event: React.KeyboardEvent) => {
    if (!onClick) return;
    if (event.key === 'Enter' || event.key === ' ') {
      event.preventDefault();
      onClick();
    }
  };

  return (
    <Card
      className={`dashboard-kpi-card ${danger ? 'is-danger' : ''}`}
      onClick={onClick}
      onKeyDown={handleKeyDown}
      role={isInteractive ? 'button' : undefined}
      tabIndex={isInteractive ? 0 : undefined}
      aria-label={ariaLabel}
      style={{
        cursor: isInteractive ? 'pointer' : undefined,
        borderLeft: tone === 'neutral' ? undefined : `4px solid ${toneColor(tone)}`,
      }}
    >
      <span className="dashboard-label">{title}</span>
      <div className="dashboard-kpi-value">{children ?? value}</div>
      <span className="dashboard-kpi-detail">{detail}</span>
    </Card>
  );
}

function PanelTitle({ title }: { title: string }) {
  return <h2 className="dashboard-panel-title">{title}</h2>;
}

function summarizeAuthSessions(sessions: ReturnType<typeof useSessionStore.getState>['sessions']) {
  const active = sessions.filter((session) => session.status === 'active').length;
  const expiring = sessions.filter((session) => session.status === 'expiring').length;
  const expired = sessions.filter((session) => session.status === 'expired').length;
  const providerCounts = sessions.reduce<Record<string, number>>((acc, session) => {
    acc[session.cli_provider] = (acc[session.cli_provider] ?? 0) + 1;
    return acc;
  }, {});

  return {
    active,
    expiring,
    expired,
    tone: expired > 0 ? 'danger' as const : expiring > 0 ? 'warning' as const : 'healthy' as const,
    providers: Object.entries(providerCounts)
      .map(([id, count]) => ({
        id,
        count,
        shortLabel: shortProviderLabel(id),
        title: `${providerLabel(id)}: ${count}`,
      }))
      .sort((a, b) => a.shortLabel.localeCompare(b.shortLabel)),
  };
}

function toneColor(tone: 'neutral' | 'healthy' | 'warning' | 'danger') {
  switch (tone) {
    case 'healthy':
      return 'var(--ops)';
    case 'warning':
      return 'var(--amber)';
    case 'danger':
      return 'var(--threat)';
    case 'neutral':
    default:
      return 'transparent';
  }
}

function shortProviderLabel(provider: string): string {
  const labels: Record<string, string> = {
    claude_code: 'Claude',
    codex: 'Codex',
    gemini_cli: 'Gemini',
    kiro_cli: 'Kiro',
  };
  return labels[provider] ?? provider;
}

function providerLabel(provider: string): string {
  return provider
    .split(/[_-]/)
    .filter(Boolean)
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(' ');
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

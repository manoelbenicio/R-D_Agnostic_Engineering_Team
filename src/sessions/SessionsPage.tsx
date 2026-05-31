import React, { useEffect, useMemo, useState } from 'react';
import { Button, Card } from '@/design-system';
import { useSessionStore } from '@/api/session-store';
import type { DiscoveredSession } from '@/api/session-discovery';
import { AddSessionDialog } from './AddSessionDialog';
import './sessions.css';

const PROVIDERS = [
  { id: 'claude_code', label: 'CLAUDE CODE' },
  { id: 'codex', label: 'CODEX' },
  { id: 'gemini_cli', label: 'GEMINI CLI' },
  { id: 'kiro_cli', label: 'KIRO CLI' },
] as const;

export const SessionsPage: React.FC = () => {
  const { sessions, loading, error, refresh, addSession } = useSessionStore();
  const [showAddDialog, setShowAddDialog] = useState(false);
  const [dialogProvider, setDialogProvider] = useState<string>();
  const [collapsed, setCollapsed] = useState<ReadonlySet<string>>(new Set());
  const toggleCollapse = (providerId: string) =>
    setCollapsed((prev) => {
      const next = new Set(prev);
      if (next.has(providerId)) {
        next.delete(providerId);
      } else {
        next.add(providerId);
      }
      return next;
    });
  const sessionsByProvider = useMemo(
    () =>
      new Map(
        PROVIDERS.map((provider) => [
          provider.id,
          sessions.filter((session) => session.cli_provider === provider.id),
        ]),
      ),
    [sessions],
  );
  const activeCount = sessions.filter((session) => session.status === 'active').length;
  const oauthCount = sessions.filter((session) => session.auth_method === 'oauth').length;
  const expiringCount = sessions.filter((session) => session.status === 'expiring').length;

  useEffect(() => {
    document.title = 'Sessions · AgentVerse';
    void refresh();
  }, [refresh]);

  const openAddDialog = (provider?: string) => {
    setDialogProvider(provider);
    setShowAddDialog(true);
  };

  return (
    <main className="sessions-page">
      <header className="sessions-header">
        <div>
          <span className="sessions-eyebrow">Runtime identity control</span>
          <h1>AUTH SESSIONS</h1>
          <p>Discover and route authenticated CLI accounts across the AgentVerse runtime.</p>
        </div>
        <Button variant="secondary" onClick={() => void refresh()} disabled={loading}>
          Refresh All
        </Button>
      </header>

      {error ? (
        <Card className="sessions-error-banner" glow="red" role="alert">
          <div>
            <strong>Session discovery failed.</strong>
            <span>{error}</span>
          </div>
          <Button variant="secondary" onClick={() => void refresh()}>
            Retry
          </Button>
        </Card>
      ) : null}

      {loading && sessions.length === 0 ? (
        <Card className="sessions-loading-state">
          <span className="sessions-spinner" aria-hidden="true" />
          <span>Discovering sessions...</span>
        </Card>
      ) : null}

      {!loading && !error && sessions.length === 0 ? (
        <Card className="sessions-empty-state">
          <strong>No sessions detected</strong>
          <span>Start a CLI login from one of the provider sections below.</span>
        </Card>
      ) : null}

      <div className="sessions-provider-stack">
        {PROVIDERS.map((provider) => {
          const providerSessions = sessionsByProvider.get(provider.id) ?? [];
          const isCollapsed = collapsed.has(provider.id);
          const bodyId = `sessions-provider-body-${provider.id}`;
          return (
            <section className="sessions-provider-section" key={provider.id}>
              <header className="sessions-provider-header">
                <button
                  type="button"
                  className="sessions-provider-toggle"
                  aria-expanded={!isCollapsed}
                  aria-controls={bodyId}
                  onClick={() => toggleCollapse(provider.id)}
                >
                  <span className="sessions-provider-chevron" aria-hidden="true">
                    {isCollapsed ? '▸' : '▾'}
                  </span>
                  <span>
                    <span className="sessions-provider-kicker">CLI Provider</span>
                    <h2>{provider.label}</h2>
                  </span>
                </button>
                <Button variant="secondary" onClick={() => openAddDialog(provider.id)}>
                  + Add Session
                </Button>
              </header>

              {!isCollapsed ? (
                <div id={bodyId}>
                  {providerSessions.length > 0 ? (
                    <div className="sessions-grid">
                      {providerSessions.map((session) => (
                        <SessionCard key={session.id} session={session} onRelogin={addSession} />
                      ))}
                    </div>
                  ) : (
                    <div className="sessions-provider-empty">
                      No {provider.label} sessions detected.
                    </div>
                  )}
                </div>
              ) : null}
            </section>
          );
        })}
      </div>

      <footer className="sessions-footer">
        <span>
          Total Active: <strong>{activeCount}</strong>
        </span>
        <span>
          OAuth: <strong>{oauthCount}</strong>
        </span>
        <span>
          Expiring: <strong>{expiringCount}</strong>
        </span>
      </footer>

      <AddSessionDialog
        isOpen={showAddDialog}
        defaultProvider={dialogProvider}
        onClose={() => setShowAddDialog(false)}
      />
    </main>
  );
};

function SessionCard({
  session,
  onRelogin,
}: {
  session: DiscoveredSession;
  onRelogin: (cliProvider: string, configDir?: string) => Promise<void>;
}) {
  const revokeSession = useSessionStore((s) => s.revokeSession);
  const [revoking, setRevoking] = useState(false);

  const handleRevoke = async () => {
    if (!window.confirm('Revoke session for ' + session.account_email + '?')) return;
    setRevoking(true);
    try {
      await revokeSession(session.id);
    } finally {
      setRevoking(false);
    }
  };

  return (
    <Card className={`session-card session-card-${session.status}${revoking ? ' session-card-revoking' : ''}`}>
      <div className="session-card-heading">
        <span
          className={`session-status-dot session-status-${session.status}`}
          aria-label={session.status}
        />
        <div>
          <strong>{session.account_email}</strong>
          <span className="session-card-status">{session.status}</span>
        </div>
      </div>

      <dl className="session-card-details">
        <div>
          <dt>Config</dt>
          <dd className="session-config-dir">{session.config_dir || 'Default CLI home'}</dd>
        </div>
        <div>
          <dt>Expires</dt>
          <dd>{formatExpiry(session.expires_at)}</dd>
        </div>
      </dl>

      <div className="session-card-footer">
        <span className="session-auth-badge">{session.auth_method}</span>
        <div className="session-card-actions">
          <Button
            variant="ghost"
            onClick={() => void onRelogin(session.cli_provider, session.config_dir || undefined)}
            disabled={revoking}
          >
            Re-Login
          </Button>
          <Button
            variant="ghost"
            onClick={() => void handleRevoke()}
            disabled={revoking}
            title="Revoke this OAuth session"
          >
            {revoking ? 'Revoking…' : 'Revoke'}
          </Button>
        </div>
      </div>
    </Card>
  );
}

function formatExpiry(expiresAt?: string): string {
  if (!expiresAt) return 'Not reported';
  const parsed = Date.parse(expiresAt);
  if (Number.isNaN(parsed)) return expiresAt;
  return new Intl.DateTimeFormat('en-US', {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(parsed);
}

export default SessionsPage;

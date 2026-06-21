import React from 'react';
import { useQuery } from '@tanstack/react-query';
import { useParams } from 'react-router-dom';
import { goCoreClient, sessionsQueryKeys } from '@/api';
import type { Terminal } from '@/api';
// eslint-disable-next-line agentverse/no-sideways-capability-imports
import { canvasStore } from '@/canvas-document/store';
import { StatusBadge } from '@/design-system';
import { useSessionStore } from '@/api/session-store';
import { mapTerminalStatus } from './utils';

export interface TabBarProps {
  sessionName: string;
  focusedId: string | null;
  onFocus: (id: string) => void;
  onClose: (id: string) => void;
  onAdd?: () => void;
}

export const TabBar: React.FC<TabBarProps> = ({
  sessionName,
  focusedId,
  onFocus,
  onClose,
  onAdd,
}) => {
  const { id: canvasId } = useParams<{ id: string }>();
  const sessionRefreshMarker = useSessionStore((state) => `${state.sessions.length}:${state.lastRefreshed ?? ''}`);
  const { data: terminals = [] } = useQuery<Terminal[]>({
    queryKey: sessionsQueryKeys.terminals(sessionName),
    queryFn: () => goCoreClient.listTerminalsInSession(sessionName),
    refetchInterval: 3000,
    refetchIntervalInBackground: false,
  });
  const { data: canvas } = useQuery({
    queryKey: ['canvas-document', canvasId],
    queryFn: () => canvasStore.get(canvasId as string),
    enabled: Boolean(canvasId),
  });

  return (
    <div className="terminal-tab-bar" role="tablist">
      {terminals.map((terminal) => {
        const isFocused = terminal.id === focusedId;
        const name = terminal.display_name || terminal.profile;
        const node = findNodeForTerminal(canvas, terminal.id);
        const sessionId = node?.data.session_id;
        const session = sessionId ? useSessionStore.getState().getSession(sessionId) : undefined;
        void sessionRefreshMarker;
        return (
          <div
            key={terminal.id}
            role="tab"
            aria-selected={isFocused}
            className={`terminal-tab ${isFocused ? 'active' : ''}`}
            onClick={() => onFocus(terminal.id)}
            data-testid={`terminal-tab-${terminal.id}`}
          >
            <span className="tab-name">{name}</span>
            {session ? (
              <span
                title={`OAuth: ${session.account_email}`}
                aria-label={`OAuth session ${session.status}: ${session.account_email}`}
                style={{
                  display: 'inline-flex',
                  alignItems: 'center',
                  fontSize: '0.65rem',
                  lineHeight: 1,
                  filter: 'drop-shadow(0 0 4px rgba(255, 255, 255, 0.18))',
                }}
              >
                {session.status === 'expired' ? '🔴' : session.status === 'expiring' ? '🟡' : '🟢'}
              </span>
            ) : null}
            <StatusBadge status={mapTerminalStatus(terminal.status)} className="tab-status-badge" />
            <button
              type="button"
              className="tab-close-btn"
              aria-label={`Close tab for ${name}`}
              onClick={(e) => {
                e.stopPropagation();
                onClose(terminal.id);
              }}
            >
              &times;
            </button>
          </div>
        );
      })}
      {onAdd && (
        <button
          type="button"
          className="tab-add-btn"
          onClick={onAdd}
          aria-label="Add terminal"
        >
          +
        </button>
      )}
    </div>
  );
};

function findNodeForTerminal(canvas: Awaited<ReturnType<typeof canvasStore.get>> | undefined, terminalId: string) {
  if (!canvas?.deploy_state.terminal_map) return undefined;
  const nodeEntry = Object.entries(canvas.deploy_state.terminal_map).find(([, mappedId]) => mappedId === terminalId);
  if (!nodeEntry) return undefined;
  return canvas.nodes.find((node) => node.id === nodeEntry[0]);
}

export default TabBar;

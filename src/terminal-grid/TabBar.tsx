import React from 'react';
import { useQuery } from '@tanstack/react-query';
import { caoClient, sessionsQueryKeys } from '@/api';
import type { Terminal } from '@/api';
import { StatusBadge } from '@/design-system';
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
  const { data: terminals = [] } = useQuery<Terminal[]>({
    queryKey: sessionsQueryKeys.terminals(sessionName),
    queryFn: () => caoClient.listTerminalsInSession(sessionName),
    refetchInterval: 3000,
    refetchIntervalInBackground: false,
  });

  return (
    <div className="terminal-tab-bar" role="tablist">
      {terminals.map((terminal) => {
        const isFocused = terminal.id === focusedId;
        const name = terminal.display_name || terminal.profile;
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

export default TabBar;

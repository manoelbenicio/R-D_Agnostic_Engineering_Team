import React, { useEffect, useId, useMemo } from 'react';
import { useNavigate } from 'react-router-dom';
import { useSessionStore } from '@/api/session-store';
import './session-status-badge.css';

type BadgeStatus = 'active' | 'expiring' | 'expired';

export const SessionStatusBadge: React.FC = () => {
  const { sessions, refresh } = useSessionStore();
  const navigate = useNavigate();
  const descriptionId = useId();

  useEffect(() => {
    if (sessions.length === 0) {
      void refresh();
    }
  }, [refresh, sessions.length]);

  const summary = useMemo(() => {
    const active = sessions.filter((session) => session.status === 'active').length;
    const expiring = sessions.filter((session) => session.status === 'expiring').length;
    const expired = sessions.filter((session) => session.status === 'expired').length;
    const status: BadgeStatus = expired > 0 ? 'expired' : expiring > 0 ? 'expiring' : 'active';
    const parts = [
      `${active} active`,
      expiring > 0 ? `${expiring} expiring` : null,
      expired > 0 ? `${expired} expired` : null,
    ].filter(Boolean);

    return {
      active,
      expiring,
      expired,
      status,
      tooltip: parts.length > 0 ? parts.join(', ') : '0 active',
    };
  }, [sessions]);

  const handleNavigate = () => navigate('/sessions');
  const handleKeyDown = (event: React.KeyboardEvent<HTMLButtonElement>) => {
    if (event.key === 'Enter' || event.key === ' ') {
      event.preventDefault();
      handleNavigate();
    }
  };
  const accessibleLabel = `Session status: ${summary.active} active, ${summary.expiring} expiring, ${summary.expired} expired. Click to manage sessions.`;

  return (
    <button
      type="button"
      className={`health-pill session-status-badge session-status-badge-${summary.status}`}
      onClick={handleNavigate}
      onKeyDown={handleKeyDown}
      title={summary.tooltip}
      role="button"
      tabIndex={0}
      aria-label={accessibleLabel}
      aria-describedby={descriptionId}
    >
      <span className={`session-status-badge-dot session-status-badge-dot-${summary.status}`} />
      <span className="health-text session-status-badge-text">
        {summary.active} {summary.active === 1 ? 'session' : 'sessions'}
      </span>
      <span id={descriptionId} className="session-status-badge-sr">
        {accessibleLabel}
      </span>
    </button>
  );
};

export default SessionStatusBadge;

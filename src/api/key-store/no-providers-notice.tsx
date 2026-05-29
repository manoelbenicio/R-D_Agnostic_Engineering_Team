import React from 'react';
import { useNavigate } from 'react-router-dom';
import { Card } from '@/design-system/components/Card';
import { Button } from '@/design-system/components/Button';

export interface NoProvidersNoticeProps {
  message?: string;
}

export const NoProvidersNotice: React.FC<NoProvidersNoticeProps> = ({ message }) => {
  const navigate = useNavigate();

  const handleConfigure = () => {
    navigate('/settings/providers');
  };

  return (
    <Card
      glow="none"
      className="sentinel-no-providers-notice"
      style={{
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        gap: 'var(--space-4)',
        border: '1px solid var(--amber)',
        background: 'rgba(255, 183, 0, 0.05)',
        padding: 'var(--space-4)',
      }}
    >
      <div style={{ display: 'flex', alignItems: 'center', gap: 'var(--space-3)' }}>
        <span style={{ color: 'var(--amber)', fontSize: '1.25rem', fontFamily: 'var(--font-mono)' }}>⚠</span>
        <span style={{ color: 'var(--text-primary)', fontFamily: 'var(--font-body)', fontSize: '0.875rem' }}>
          {message || 'No providers configured — Deploy is disabled until you validate at least one in Settings'}
        </span>
      </div>
      <Button
        variant="secondary"
        onClick={handleConfigure}
        style={{
          borderColor: 'var(--amber)',
          color: 'var(--amber)',
          fontSize: '0.75rem',
          padding: 'var(--space-1.5) var(--space-4)',
          whiteSpace: 'nowrap',
        }}
      >
        Configure Settings
      </Button>
    </Card>
  );
};

export default NoProvidersNotice;

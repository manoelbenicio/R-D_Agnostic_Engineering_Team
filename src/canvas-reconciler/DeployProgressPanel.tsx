import React, { useEffect, useState } from 'react';
import { useDeployStore } from './deploy-store';
import { Card, StatusBadge } from '@/design-system';

interface DeployProgressPanelProps {
  status: 'draft' | 'deploying' | 'deployed' | 'degraded';
}

export const DeployProgressPanel: React.FC<DeployProgressPanelProps> = ({ status }) => {
  const { deploySteps, activeDeployCanvasId } = useDeployStore();
  const [shouldRender, setShouldRender] = useState(false);

  useEffect(() => {
    let timer: NodeJS.Timeout;

    if (status === 'deploying') {
      setShouldRender(true);
    } else if (status === 'deployed') {
      // Completed deploys leave panel visible for 3 seconds before auto-dismissing
      timer = setTimeout(() => {
        setShouldRender(false);
      }, 3000);
    } else {
      // Degraded or Draft statuses: keep visible if there were steps so the user can inspect errors
      if (deploySteps.length > 0 && status === 'degraded') {
        setShouldRender(true);
      } else {
        setShouldRender(false);
      }
    }

    return () => {
      if (timer) clearTimeout(timer);
    };
  }, [status, deploySteps]);

  if (!shouldRender || !activeDeployCanvasId || deploySteps.length === 0) {
    return null;
  }

  const mapStepStatus = (stepStatus: 'pending' | 'in_flight' | 'success' | 'failed') => {
    switch (stepStatus) {
      case 'in_flight':
        return 'processing';
      case 'success':
        return 'completed';
      case 'failed':
        return 'error';
      case 'pending':
      default:
        return 'idle';
    }
  };

  return (
    <div
      className="sentinel-deploy-progress-panel animate-slide-in"
      style={{
        position: 'fixed',
        bottom: '80px',
        left: '180px',
        width: '360px',
        zIndex: 900,
      }}
    >
      <Card className="glow-cyan" style={{ padding: '20px' }}>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '16px' }}>
          <h4 style={{ fontFamily: 'var(--font-mono)', fontSize: '0.9rem', color: 'var(--cyan)', textTransform: 'uppercase', letterSpacing: '0.05em' }}>
            System Materialization
          </h4>
          <span
            className="blink"
            style={{
              width: '8px',
              height: '8px',
              borderRadius: '50%',
              backgroundColor: status === 'deployed' ? 'var(--ops)' : 'var(--cyan)',
              boxShadow: status === 'deployed' ? '0 0 8px var(--ops)' : '0 0 8px var(--cyan)',
            }}
          />
        </div>

        <div style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
          {deploySteps.map((step) => (
            <div
              key={step.id}
              style={{
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'space-between',
                padding: '6px 10px',
                background: 'rgba(255, 255, 255, 0.02)',
                border: '1px solid rgba(255, 255, 255, 0.04)',
                borderRadius: '4px',
              }}
            >
              <span style={{ fontSize: '0.8rem', color: 'var(--text-primary)', fontFamily: 'var(--font-body)' }}>
                {step.label}
              </span>
              <StatusBadge status={mapStepStatus(step.status)} />
            </div>
          ))}
        </div>
      </Card>
    </div>
  );
};

export default DeployProgressPanel;

import React from 'react';

interface EdgeAdvisoryBannerProps {
  visible: boolean;
}

export const EdgeAdvisoryBanner: React.FC<EdgeAdvisoryBannerProps> = ({ visible }) => {
  if (!visible) return null;

  return (
    <div
      className="sentinel-edge-advisory-banner animate-fade-in"
      style={{
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        gap: '12px',
        padding: '12px 24px',
        background: 'rgba(255, 183, 0, 0.1)',
        borderBottom: '1px solid var(--amber)',
        color: 'var(--amber)',
        fontSize: '0.85rem',
        fontFamily: 'var(--font-mono)',
        width: '100%',
        boxSizing: 'border-box',
      }}
    >
      <span style={{ fontSize: '1.1rem' }}>⚠️</span>
      <span>Edge changes require Tear Down + redeploy to take effect on the supervisor.</span>
    </div>
  );
};

export default EdgeAdvisoryBanner;

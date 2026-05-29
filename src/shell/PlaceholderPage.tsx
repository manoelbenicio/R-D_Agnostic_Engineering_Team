import React from 'react';

interface PlaceholderPageProps {
  title: string;
}

export const PlaceholderPage: React.FC<PlaceholderPageProps> = ({ title }) => {
  return (
    <div className="sentinel-placeholder-container">
      <div className="sentinel-card glow-cyan animate-fade-in">
        <div className="sentinel-card-header">
          <span className="sentinel-dot bg-cyan blink"></span>
          <span className="sentinel-tag">SYSTEM STATUS: PENDING</span>
        </div>
        <div className="sentinel-card-body">
          <h2 className="sentinel-title">{title}</h2>
          <p className="sentinel-desc">
            This module is part of the AgentVerse v1 deployment. The capability is currently
            pending integration.
          </p>
          <div className="sentinel-terminal-box">
            <span className="text-muted">$ agentverse --status</span>
            <span className="text-cyan"> &gt; MODULE_STUB_ACTIVE: {title.toUpperCase().replace(/\s+/g, '_')}</span>
            <span className="text-muted"> &gt; waiting for capability owner hookup...</span>
          </div>
        </div>
      </div>
    </div>
  );
};

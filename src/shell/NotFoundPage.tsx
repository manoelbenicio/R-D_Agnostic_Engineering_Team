import React from 'react';
import { Link } from 'react-router-dom';

export const NotFoundPage: React.FC = () => {
  return (
    <div className="sentinel-placeholder-container" id="not-found-page">
      <div className="sentinel-card glow-red animate-fade-in">
        <div className="sentinel-card-header">
          <span className="sentinel-dot bg-red blink"></span>
          <span className="sentinel-tag text-red">ERROR CODE: 404_NOT_FOUND</span>
        </div>
        <div className="sentinel-card-body">
          <h2 className="sentinel-title text-red">PAGE NOT FOUND</h2>
          <p className="sentinel-desc">
            The requested neural link path does not exist in the AgentVerse directory.
          </p>
          <div className="sentinel-terminal-box">
            <span className="text-muted">$ ping agentverse/path</span>
            <span className="text-red"> &gt; ERROR: ROUTE_NOT_RESOLVED (404)</span>
            <span className="text-muted"> &gt; routing table lookup failed</span>
          </div>
          <div style={{ marginTop: '20px' }}>
            <Link to="/" className="sentinel-btn btn-secondary" id="back-home-link">
              Return to Canvas
            </Link>
          </div>
        </div>
      </div>
    </div>
  );
};

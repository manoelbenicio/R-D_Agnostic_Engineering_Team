import React from 'react';
import { LoginPanel } from './LoginPanel';

/**
 * Full-page sign-in screen shown when auth is required (cloud mode) and the
 * user is unauthenticated. Renders inside AppLayout.
 */
export const LoginScreen: React.FC = () => {
  return (
    <div
      className="sentinel-card"
      style={{
        maxWidth: 480,
        margin: 'var(--space-12) auto',
        padding: 'var(--space-8)',
        display: 'flex',
        flexDirection: 'column',
        gap: 'var(--space-4)',
      }}
      role="region"
      aria-label="Sign-in required"
    >
      <header style={{ textAlign: 'center', display: 'flex', flexDirection: 'column', gap: 'var(--space-2)' }}>
        <h1
          style={{
            fontFamily: 'var(--font-display)',
            fontSize: '1.4rem',
            margin: 0,
            color: 'var(--cyan, #00f0ff)',
            letterSpacing: '0.04em',
            textTransform: 'uppercase',
          }}
        >
          Authentication Required
        </h1>
        <p
          style={{
            fontFamily: 'var(--font-mono)',
            fontSize: '0.85rem',
            color: 'var(--text-muted, #8aa1bb)',
            margin: 0,
          }}
        >
          AgentVerse cloud requires sign-in. Choose any method below.
        </p>
      </header>

      <LoginPanel />

      <p
        style={{
          fontFamily: 'var(--font-mono)',
          fontSize: '0.7rem',
          color: 'var(--text-dim, #5f7a99)',
          margin: 0,
          textAlign: 'center',
        }}
      >
        Local mode does not require sign-in. This screen appears when AgentVerse is running in cloud mode.
      </p>
    </div>
  );
};

export default LoginScreen;

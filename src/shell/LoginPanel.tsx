import React, { useState } from 'react';
import { useAuthStore } from './auth-store';

type Mode = 'choose' | 'email-signin' | 'email-signup' | 'reset';

/**
 * Sign-in panel. Three methods exposed: Google, GitHub, Email + Password.
 *
 * Local mode (auth optional) uses this panel inside a popover or modal.
 * Cloud mode (auth required) embeds it directly in <LoginScreen />.
 *
 * `compact` makes the panel tighter for popovers; defaults to false for the
 * full-page login screen.
 */
export const LoginPanel: React.FC<{ compact?: boolean; onSignedIn?: () => void }> = ({
  compact = false,
  onSignedIn,
}) => {
  const status = useAuthStore((s) => s.status);
  const error = useAuthStore((s) => s.error);
  const signIn = useAuthStore((s) => s.signIn);
  const signUp = useAuthStore((s) => s.signUp);
  const sendPasswordReset = useAuthStore((s) => s.sendPasswordReset);
  const clearError = useAuthStore((s) => s.clearError);

  const [mode, setMode] = useState<Mode>('choose');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [resetEmail, setResetEmail] = useState('');
  const [resetSent, setResetSent] = useState(false);

  const busy = status === 'loading';

  const handleOAuth = async (method: 'google' | 'github') => {
    try {
      await signIn({ method });
      onSignedIn?.();
    } catch {
      // Error already in store
    }
  };

  const handleEmailSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      if (mode === 'email-signup') {
        await signUp({ email, password });
      } else {
        await signIn({ method: 'email', email, password });
      }
      setEmail('');
      setPassword('');
      onSignedIn?.();
    } catch {
      // Error already in store
    }
  };

  const handleResetSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setResetSent(false);
    try {
      await sendPasswordReset(resetEmail);
      setResetSent(true);
    } catch {
      // Error already in store
    }
  };

  const buttonStyle: React.CSSProperties = {
    width: '100%',
    padding: '10px 14px',
    fontFamily: 'var(--font-mono)',
    fontSize: '0.82rem',
    letterSpacing: '0.04em',
    textTransform: 'uppercase',
    cursor: 'pointer',
    border: '1px solid var(--border-accent, rgba(0,255,255,0.4))',
    background: 'var(--surface-deep, #0b1622)',
    color: 'var(--text-primary, #e6f1ff)',
    borderRadius: 'var(--radius-button, 4px)',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    gap: 'var(--space-2)',
  };

  const inputStyle: React.CSSProperties = {
    width: '100%',
    padding: '8px 10px',
    fontFamily: 'var(--font-mono)',
    fontSize: '0.82rem',
    border: '1px solid var(--border, rgba(255,255,255,0.12))',
    background: 'var(--surface-deep, #0b1622)',
    color: 'var(--text-primary, #e6f1ff)',
    borderRadius: 'var(--radius-button, 4px)',
  };

  const linkStyle: React.CSSProperties = {
    background: 'none',
    border: 'none',
    color: 'var(--cyan, #00f0ff)',
    cursor: 'pointer',
    fontFamily: 'var(--font-mono)',
    fontSize: '0.75rem',
    textDecoration: 'underline',
    padding: 0,
  };

  return (
    <div
      style={{
        display: 'flex',
        flexDirection: 'column',
        gap: compact ? 'var(--space-2)' : 'var(--space-3)',
        minWidth: compact ? 260 : 320,
        padding: compact ? 0 : 'var(--space-2)',
      }}
    >
      {error && (
        <p
          role="alert"
          style={{
            fontFamily: 'var(--font-mono)',
            fontSize: '0.75rem',
            color: 'var(--threat, #ff6b6b)',
            margin: 0,
            padding: '6px 8px',
            background: 'rgba(255, 107, 107, 0.08)',
            border: '1px solid rgba(255, 107, 107, 0.4)',
            borderRadius: 'var(--radius-button, 4px)',
          }}
        >
          {error}
        </p>
      )}

      {mode === 'choose' && (
        <>
          <button type="button" style={buttonStyle} disabled={busy} onClick={() => void handleOAuth('google')}>
            <span aria-hidden="true">{googleGlyph}</span>
            Continue with Google
          </button>
          <button type="button" style={buttonStyle} disabled={busy} onClick={() => void handleOAuth('github')}>
            <span aria-hidden="true">{githubGlyph}</span>
            Continue with GitHub
          </button>

          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: 'var(--space-2)',
              fontFamily: 'var(--font-mono)',
              fontSize: '0.7rem',
              color: 'var(--text-dim, #5f7a99)',
              margin: '4px 0',
            }}
          >
            <span style={{ flex: 1, height: 1, background: 'var(--border, rgba(255,255,255,0.12))' }} />
            <span>OR</span>
            <span style={{ flex: 1, height: 1, background: 'var(--border, rgba(255,255,255,0.12))' }} />
          </div>

          <button
            type="button"
            style={buttonStyle}
            disabled={busy}
            onClick={() => {
              clearError();
              setMode('email-signin');
            }}
          >
            Continue with email
          </button>
        </>
      )}

      {(mode === 'email-signin' || mode === 'email-signup') && (
        <form onSubmit={handleEmailSubmit} style={{ display: 'flex', flexDirection: 'column', gap: 'var(--space-2)' }}>
          <label style={{ fontFamily: 'var(--font-mono)', fontSize: '0.72rem', color: 'var(--text-muted, #8aa1bb)' }}>
            Email
            <input
              type="email"
              autoComplete="email"
              required
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              style={{ ...inputStyle, marginTop: 4 }}
            />
          </label>
          <label style={{ fontFamily: 'var(--font-mono)', fontSize: '0.72rem', color: 'var(--text-muted, #8aa1bb)' }}>
            Password
            <input
              type="password"
              autoComplete={mode === 'email-signup' ? 'new-password' : 'current-password'}
              required
              minLength={6}
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              style={{ ...inputStyle, marginTop: 4 }}
            />
          </label>
          <button type="submit" style={buttonStyle} disabled={busy}>
            {busy ? 'Working…' : mode === 'email-signup' ? 'Create account' : 'Sign in'}
          </button>

          <div style={{ display: 'flex', justifyContent: 'space-between', flexWrap: 'wrap', gap: 'var(--space-2)' }}>
            <button
              type="button"
              style={linkStyle}
              onClick={() => {
                clearError();
                setMode(mode === 'email-signin' ? 'email-signup' : 'email-signin');
              }}
            >
              {mode === 'email-signin' ? 'Create an account' : 'Have an account? Sign in'}
            </button>
            {mode === 'email-signin' && (
              <button
                type="button"
                style={linkStyle}
                onClick={() => {
                  clearError();
                  setResetEmail(email);
                  setResetSent(false);
                  setMode('reset');
                }}
              >
                Forgot password?
              </button>
            )}
          </div>

          <button
            type="button"
            style={{ ...linkStyle, alignSelf: 'flex-start' }}
            onClick={() => {
              clearError();
              setMode('choose');
            }}
          >
            ← Back to all sign-in methods
          </button>
        </form>
      )}

      {mode === 'reset' && (
        <form onSubmit={handleResetSubmit} style={{ display: 'flex', flexDirection: 'column', gap: 'var(--space-2)' }}>
          <p style={{ fontFamily: 'var(--font-mono)', fontSize: '0.78rem', color: 'var(--text-muted, #8aa1bb)', margin: 0 }}>
            Enter the email address for your account; we&apos;ll send you a password-reset link.
          </p>
          <label style={{ fontFamily: 'var(--font-mono)', fontSize: '0.72rem', color: 'var(--text-muted, #8aa1bb)' }}>
            Email
            <input
              type="email"
              autoComplete="email"
              required
              value={resetEmail}
              onChange={(e) => setResetEmail(e.target.value)}
              style={{ ...inputStyle, marginTop: 4 }}
            />
          </label>
          <button type="submit" style={buttonStyle} disabled={busy}>
            {busy ? 'Sending…' : 'Send reset link'}
          </button>
          {resetSent && (
            <p
              role="status"
              style={{
                fontFamily: 'var(--font-mono)',
                fontSize: '0.75rem',
                color: 'var(--ops, #4ade80)',
                margin: 0,
              }}
            >
              Reset email sent. Check your inbox.
            </p>
          )}
          <button
            type="button"
            style={{ ...linkStyle, alignSelf: 'flex-start' }}
            onClick={() => {
              clearError();
              setMode('email-signin');
            }}
          >
            ← Back to sign-in
          </button>
        </form>
      )}
    </div>
  );
};

const googleGlyph = (
  <svg viewBox="0 0 24 24" width="16" height="16" aria-hidden="true">
    <path
      fill="#4285F4"
      d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z"
    />
    <path
      fill="#34A853"
      d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.99.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"
    />
    <path
      fill="#FBBC05"
      d="M5.84 14.09A6.96 6.96 0 0 1 5.46 12c0-.73.13-1.43.35-2.09V7.07H2.18A11 11 0 0 0 1 12c0 1.78.43 3.46 1.18 4.93l3.66-2.84z"
    />
    <path
      fill="#EA4335"
      d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.46 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84C6.71 7.31 9.14 5.38 12 5.38z"
    />
  </svg>
);

const githubGlyph = (
  <svg viewBox="0 0 24 24" width="16" height="16" aria-hidden="true" fill="currentColor">
    <path d="M12 .5C5.65.5.5 5.65.5 12c0 5.08 3.29 9.39 7.86 10.91.58.1.79-.25.79-.56v-2.16c-3.2.7-3.88-1.36-3.88-1.36-.52-1.34-1.27-1.7-1.27-1.7-1.04-.71.08-.7.08-.7 1.15.08 1.76 1.18 1.76 1.18 1.02 1.75 2.69 1.25 3.34.95.1-.74.4-1.25.72-1.54-2.55-.29-5.24-1.27-5.24-5.66 0-1.25.45-2.27 1.18-3.07-.12-.29-.51-1.45.11-3.03 0 0 .96-.31 3.15 1.17a10.97 10.97 0 0 1 5.74 0c2.19-1.48 3.15-1.17 3.15-1.17.62 1.58.23 2.74.11 3.03.74.8 1.18 1.82 1.18 3.07 0 4.4-2.69 5.36-5.25 5.65.41.36.78 1.06.78 2.14v3.17c0 .31.21.67.8.56C20.21 21.39 23.5 17.08 23.5 12 23.5 5.65 18.35.5 12 .5z" />
  </svg>
);

export default LoginPanel;

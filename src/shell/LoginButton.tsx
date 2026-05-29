import React, { useEffect, useRef, useState } from 'react';
import { useAuthStore } from './auth-store';
import { LoginPanel } from './LoginPanel';

/**
 * Sign-in / sign-out button for the NavBar.
 *
 * - When auth is disabled (VITE_AUTH_PROVIDER unset): renders nothing.
 * - When signed out: shows a "Sign in" button; clicking it opens a popover
 *   with the full LoginPanel (Google, GitHub, Email/password).
 * - When signed in: shows the user's display name + avatar with a Sign out
 *   action on click.
 */
export const LoginButton: React.FC = () => {
  const enabled = useAuthStore((s) => s.enabled);
  const status = useAuthStore((s) => s.status);
  const user = useAuthStore((s) => s.user);
  const error = useAuthStore((s) => s.error);
  const signOut = useAuthStore((s) => s.signOut);

  const [popoverOpen, setPopoverOpen] = useState(false);
  const [menuOpen, setMenuOpen] = useState(false);
  const containerRef = useRef<HTMLDivElement | null>(null);

  // Close popover/menu when clicking outside.
  useEffect(() => {
    if (!popoverOpen && !menuOpen) return;
    const handler = (e: MouseEvent) => {
      const target = e.target as Node;
      if (containerRef.current && !containerRef.current.contains(target)) {
        setPopoverOpen(false);
        setMenuOpen(false);
      }
    };
    document.addEventListener('mousedown', handler);
    return () => document.removeEventListener('mousedown', handler);
  }, [popoverOpen, menuOpen]);

  if (!enabled) return null;

  const busy = status === 'loading';

  // Signed out → "Sign in" button + popover with all methods
  if (!user) {
    return (
      <div ref={containerRef} style={{ position: 'relative', marginLeft: 'var(--space-3)' }}>
        <button
          type="button"
          className="sentinel-button sentinel-button-secondary"
          onClick={() => setPopoverOpen((v) => !v)}
          disabled={busy}
          aria-haspopup="dialog"
          aria-expanded={popoverOpen}
          title={error ? `Sign-in error: ${error}` : 'Sign in'}
          style={{
            fontFamily: 'var(--font-mono)',
            fontSize: '0.78rem',
            letterSpacing: '0.04em',
            textTransform: 'uppercase',
          }}
        >
          {busy ? 'SIGNING IN…' : 'SIGN IN'}
        </button>

        {popoverOpen && (
          <div
            role="dialog"
            aria-label="Sign in"
            style={{
              position: 'absolute',
              top: 'calc(100% + 6px)',
              right: 0,
              minWidth: 320,
              background: 'var(--panel, #0b1622)',
              border: '1px solid var(--border-accent, rgba(0,255,255,0.4))',
              borderRadius: 'var(--radius-button, 4px)',
              boxShadow: '0 8px 24px rgba(0,0,0,0.6)',
              padding: 'var(--space-4)',
              zIndex: 1000,
            }}
          >
            <LoginPanel compact onSignedIn={() => setPopoverOpen(false)} />
          </div>
        )}
      </div>
    );
  }

  // Signed in → avatar + name with logout dropdown
  const initials = (user.displayName ?? user.email ?? '?')
    .split(/\s+/)
    .map((part) => part.charAt(0).toUpperCase())
    .slice(0, 2)
    .join('');

  return (
    <div ref={containerRef} style={{ position: 'relative', marginLeft: 'var(--space-3)' }}>
      <button
        type="button"
        onClick={() => setMenuOpen((v) => !v)}
        aria-haspopup="menu"
        aria-expanded={menuOpen}
        title={user.email ?? user.displayName ?? user.uid}
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 'var(--space-2)',
          background: 'transparent',
          border: '1px solid var(--border-accent, rgba(0,255,255,0.4))',
          color: 'var(--text-primary, #e6f1ff)',
          padding: '4px 10px',
          borderRadius: 'var(--radius-button, 4px)',
          cursor: 'pointer',
          fontFamily: 'var(--font-mono)',
          fontSize: '0.78rem',
        }}
      >
        {user.photoURL ? (
          <img
            src={user.photoURL}
            alt=""
            referrerPolicy="no-referrer"
            style={{ width: 22, height: 22, borderRadius: '50%' }}
          />
        ) : (
          <span
            aria-hidden="true"
            style={{
              display: 'inline-flex',
              alignItems: 'center',
              justifyContent: 'center',
              width: 22,
              height: 22,
              borderRadius: '50%',
              border: '1px solid var(--cyan, #00f0ff)',
              color: 'var(--cyan, #00f0ff)',
              fontSize: '0.7rem',
              fontWeight: 700,
            }}
          >
            {initials || '?'}
          </span>
        )}
        <span style={{ maxWidth: 140, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
          {user.displayName ?? user.email ?? 'Signed in'}
        </span>
        <span aria-hidden="true" style={{ opacity: 0.7 }}>▾</span>
      </button>

      {menuOpen && (
        <div
          role="menu"
          style={{
            position: 'absolute',
            top: 'calc(100% + 6px)',
            right: 0,
            minWidth: 200,
            background: 'var(--panel, #0b1622)',
            border: '1px solid var(--border-accent, rgba(0,255,255,0.4))',
            borderRadius: 'var(--radius-button, 4px)',
            boxShadow: '0 8px 24px rgba(0,0,0,0.5)',
            padding: 'var(--space-2)',
            zIndex: 1000,
            fontFamily: 'var(--font-mono)',
            fontSize: '0.78rem',
          }}
        >
          <div
            style={{
              padding: 'var(--space-2)',
              color: 'var(--text-muted, #8aa1bb)',
              borderBottom: '1px solid var(--border, rgba(255,255,255,0.08))',
              marginBottom: 'var(--space-2)',
              wordBreak: 'break-all',
            }}
          >
            {user.email ?? user.displayName}
          </div>
          <button
            type="button"
            role="menuitem"
            onClick={async () => {
              setMenuOpen(false);
              try {
                await signOut();
              } catch {
                // surfaced via error state
              }
            }}
            disabled={busy}
            style={{
              width: '100%',
              textAlign: 'left',
              padding: 'var(--space-2)',
              background: 'transparent',
              border: '1px solid transparent',
              color: 'var(--threat, #ff6b6b)',
              cursor: 'pointer',
              fontFamily: 'inherit',
              fontSize: 'inherit',
            }}
          >
            {busy ? 'SIGNING OUT…' : 'SIGN OUT'}
          </button>
        </div>
      )}
    </div>
  );
};

export default LoginButton;

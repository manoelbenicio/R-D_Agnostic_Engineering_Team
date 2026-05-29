import React from 'react';

export interface ToastProps extends React.HTMLAttributes<HTMLDivElement> {
  message: string;
  type?: 'info' | 'error' | 'success' | 'warning';
  onDismiss?: () => void;
}

export const Toast: React.FC<ToastProps> = ({
  message,
  type = 'info',
  onDismiss,
  className = '',
  style,
  ...props
}) => {
  const getColors = () => {
    switch (type) {
      case 'success':
        return {
          border: 'var(--ops-edge)',
          glow: 'var(--ops-soft)',
          color: 'var(--ops)',
          icon: '✓',
        };
      case 'error':
        return {
          border: 'var(--threat-edge)',
          glow: 'var(--threat-soft)',
          color: 'var(--threat)',
          icon: '✕',
        };
      case 'warning':
        return {
          border: 'var(--amber-edge)',
          glow: 'var(--amber-soft)',
          color: 'var(--amber)',
          icon: '⚠',
        };
      case 'info':
      default:
        return {
          border: 'var(--cyan-edge)',
          glow: 'var(--cyan-soft)',
          color: 'var(--cyan)',
          icon: 'ℹ',
        };
    }
  };

  const scheme = getColors();

  const getIconBackground = () => {
    switch (type) {
      case 'success':
        return 'var(--ops-tint)';
      case 'error':
        return 'var(--threat-tint)';
      case 'warning':
        return 'var(--amber-tint)';
      case 'info':
      default:
        return 'var(--cyan-tint)';
    }
  };

  const toastStyle: React.CSSProperties = {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    background: 'var(--panel)',
    backdropFilter: 'blur(12px)',
    WebkitBackdropFilter: 'blur(12px)',
    borderRadius: '8px',
    padding: 'var(--space-3) var(--space-5)',
    border: `1px solid ${scheme.border}`,
    boxShadow: `0 8px 32px rgba(0, 0, 0, 0.5), 0 0 20px ${scheme.glow}`,
    color: 'var(--text-primary)',
    fontFamily: 'var(--font-body)',
    fontSize: '0.875rem',
    cursor: onDismiss ? 'pointer' : 'default',
    transition: 'all 0.2s ease',
    ...style,
  };

  return (
    <div
      className={`sentinel-toast-sys toast-${type} ${className}`}
      style={toastStyle}
      onClick={onDismiss}
      role="alert"
      {...props}
    >
      <div className="toast-body" style={{ display: 'flex', alignItems: 'center', gap: 'var(--space-3)' }}>
        <span
          className="toast-icon"
          style={{
            fontWeight: 700,
            display: 'inline-flex',
            alignItems: 'center',
            justifyContent: 'center',
            width: '20px',
            height: '20px',
            borderRadius: '50%',
            fontSize: '0.75rem',
            background: getIconBackground(),
            color: scheme.color,
          }}
        >
          {scheme.icon}
        </span>
        <span className="toast-message">{message}</span>
      </div>
      {onDismiss && (
        <span className="toast-close" style={{ color: 'var(--text-dim)', fontSize: '1.25rem', paddingLeft: '12px' }}>
          &times;
        </span>
      )}
    </div>
  );
};

export default Toast;

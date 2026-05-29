import React from 'react';

/**
 * DSS Universal Standard v3.0 Button.
 *
 * Variants (per `src/design-system/frontend/styles.css`):
 *   - primary   → background `--indra-deep`, 1px white-alpha border. Hover `--indra-primary`.
 *   - cyan      → background `--indra-cyan`, foreground `--indra-deep`. Hover `--indra-teal`.
 *   - secondary → transparent, 1px `--indra-cyan` border. Hover cyan-tinted bg.
 *   - ghost     → transparent, no border, mixed-case (NOT uppercase).
 *
 * Geometry: sharp corporate corners (border-radius 0). 44 px min-height.
 * Typography: sans-serif (--font-sans), 14 px / 600 weight, 0.03em tracking,
 * uppercase (except `ghost`).
 */
export type ButtonVariant = 'primary' | 'cyan' | 'secondary' | 'ghost';

export interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: ButtonVariant;
  children?: React.ReactNode;
}

export const Button: React.FC<ButtonProps> = ({
  variant = 'primary',
  children,
  className = '',
  style,
  ...props
}) => {
  const variantStyles: Record<ButtonVariant, React.CSSProperties> = {
    primary: {
      background: 'var(--indra-deep)',
      border: '1px solid rgba(255, 255, 255, 0.2)',
      color: 'var(--indra-white)',
    },
    cyan: {
      background: 'var(--indra-cyan)',
      border: '1px solid var(--indra-cyan)',
      color: 'var(--indra-deep)',
    },
    secondary: {
      background: 'transparent',
      border: '1px solid var(--indra-cyan)',
      color: 'var(--indra-white)',
    },
    ghost: {
      background: 'transparent',
      border: '1px solid transparent',
      color: 'var(--indra-white)',
    },
  };

  const isGhost = variant === 'ghost';

  const buttonStyle: React.CSSProperties = {
    fontFamily: 'var(--font-sans)',
    fontSize: '14px',
    fontWeight: 600,
    letterSpacing: '0.03em',
    padding: isGhost ? '12px 8px' : '12px 28px',
    minHeight: '44px',
    borderRadius: 'var(--radius-button)',
    cursor: props.disabled ? 'not-allowed' : 'pointer',
    opacity: props.disabled ? 0.4 : 1,
    transition: 'all var(--duration-fast) var(--ease-out)',
    display: 'inline-flex',
    alignItems: 'center',
    justifyContent: 'center',
    gap: 'var(--space-2)',
    outline: 'none',
    textTransform: isGhost ? 'none' : 'uppercase',
    ...variantStyles[variant],
    ...style,
  };

  return (
    <button
      className={`sentinel-btn-sys btn-${variant} ${className}`}
      style={buttonStyle}
      {...props}
    >
      {children}
    </button>
  );
};

export default Button;

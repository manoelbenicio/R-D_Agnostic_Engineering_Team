import React from 'react';

/**
 * DSS Universal Standard v3.0 Badge.
 *
 * DSS variants (per `src/design-system/frontend/styles.css:184-194`):
 *   - success / warning / error / gold / info  → 15% color tint background,
 *     full-pill radius (9999px), 12px / 600 / 0.02em.
 *
 * Backwards-compat aliases (existing consumers across `src/`):
 *   - idle                  → `info`  visual treatment (neutral)
 *   - processing            → `info`  (was cyan + cyan-tint)
 *   - completed             → `success`
 *   - waiting_user_answer   → `warning`
 *   - error                 → `error` (DSS variant by the same name)
 */
export type BadgeVariant =
  | 'success'
  | 'warning'
  | 'error'
  | 'gold'
  | 'info'
  // Backwards-compat aliases:
  | 'idle'
  | 'processing'
  | 'completed'
  | 'waiting_user_answer';

export interface BadgeProps extends React.HTMLAttributes<HTMLSpanElement> {
  variant?: BadgeVariant;
  children?: React.ReactNode;
}

interface DssVariantStyle {
  background: string;
  color: string;
  border: string;
}

const DSS_STYLES: Record<'success' | 'warning' | 'error' | 'gold' | 'info', DssVariantStyle> = {
  success: {
    background: 'var(--success-tint)',
    color: 'var(--indra-success)',
    border: '1px solid var(--ops-edge)',
  },
  warning: {
    background: 'var(--warning-tint)',
    color: 'var(--indra-warning)',
    border: '1px solid var(--amber-edge)',
  },
  error: {
    background: 'var(--error-tint)',
    color: 'var(--indra-error)',
    border: '1px solid var(--threat-edge)',
  },
  gold: {
    background: 'var(--gold-tint)',
    color: 'var(--indra-gold)',
    border: '1px solid var(--amber-edge)',
  },
  info: {
    background: 'var(--info-tint)',
    color: 'var(--indra-sky)',
    border: '1px solid var(--cyan-edge)',
  },
};

/** Map every variant (DSS or legacy) to the DSS style key. */
function resolveStyle(v: BadgeVariant): DssVariantStyle {
  switch (v) {
    case 'completed':
      return DSS_STYLES.success;
    case 'waiting_user_answer':
      return DSS_STYLES.warning;
    case 'error':
      return DSS_STYLES.error;
    case 'idle':
    case 'processing':
      return DSS_STYLES.info;
    case 'success':
    case 'warning':
    case 'gold':
    case 'info':
      return DSS_STYLES[v];
    default:
      return DSS_STYLES.info;
  }
}

export const Badge: React.FC<BadgeProps> = ({
  variant = 'info',
  children,
  className = '',
  style,
  ...props
}) => {
  const scheme = resolveStyle(variant);

  const badgeStyle: React.CSSProperties = {
    fontFamily: 'var(--font-sans)',
    fontSize: '12px',
    fontWeight: 600,
    letterSpacing: '0.02em',
    padding: '4px 12px',
    borderRadius: 'var(--radius-badge)',
    display: 'inline-flex',
    alignItems: 'center',
    justifyContent: 'center',
    gap: '6px',
    lineHeight: 1,
    background: scheme.background,
    color: scheme.color,
    border: scheme.border,
    ...style,
  };

  return (
    <span
      className={`sentinel-badge badge-${variant} ${className}`}
      style={badgeStyle}
      {...props}
    >
      {children}
    </span>
  );
};

export default Badge;

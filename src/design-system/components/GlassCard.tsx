import React from 'react';

/**
 * DSS Liquid-Glass Card primitive.
 *
 * Visual contract (`src/design-system/frontend/styles.css:243-260`):
 *   - background: rgba(0, 62, 80, 0.75)
 *   - backdrop-filter: blur(16px)
 *   - border: 1px solid rgba(255, 255, 255, 0.08)
 *   - border-radius: var(--radius-glass-card) (16 px)
 *   - padding: var(--space-8) (32 px)
 *
 * Hover (driven by .indra-glass-card class in tokens.css):
 *   - transform: translateY(-2px)
 *   - box-shadow: 0 8px 32px rgba(0, 0, 0, 0.35)
 */
export interface GlassCardProps extends React.HTMLAttributes<HTMLDivElement> {
  children?: React.ReactNode;
}

export const GlassCard: React.FC<GlassCardProps> = ({
  children,
  className = '',
  style,
  ...props
}) => {
  const glassStyle: React.CSSProperties = {
    background: 'rgba(0, 62, 80, 0.75)',
    backdropFilter: 'blur(16px)',
    WebkitBackdropFilter: 'blur(16px)',
    border: '1px solid rgba(255, 255, 255, 0.08)',
    borderRadius: 'var(--radius-glass-card)',
    padding: 'var(--space-8)',
    transition: 'transform var(--duration-fast) var(--ease-out)',
    ...style,
  };

  return (
    <div
      className={`indra-glass-card ${className}`}
      style={glassStyle}
      {...props}
    >
      {children}
    </div>
  );
};

export default GlassCard;

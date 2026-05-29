import React from 'react';

export interface CardProps extends React.HTMLAttributes<HTMLDivElement> {
  glow?: 'cyan' | 'red' | 'none';
  children?: React.ReactNode;
}

export const Card: React.FC<CardProps> = ({
  glow = 'none',
  children,
  className = '',
  style,
  ...props
}) => {
  const glowClass = glow === 'cyan' ? 'glow-cyan' : glow === 'red' ? 'glow-red' : '';
  const cardStyle: React.CSSProperties = {
    background: 'var(--card)',
    border: '1px solid var(--border)',
    borderRadius: 'var(--radius-card)',
    padding: 'var(--space-5)',
    ...style,
  };

  return (
    <div
      className={`sentinel-card ${glowClass} ${className}`}
      style={cardStyle}
      {...props}
    >
      {children}
    </div>
  );
};

export default Card;

import React from 'react';

export interface ProseProps extends React.HTMLAttributes<HTMLDivElement> {
  children?: React.ReactNode;
}

export const Prose: React.FC<ProseProps> = ({
  children,
  className = '',
  style,
  ...props
}) => {
  return (
    <div
      className={`sentinel-prose ${className}`}
      style={{
        lineHeight: 1.6,
        fontSize: '0.95rem',
        color: 'var(--text-primary)',
        ...style,
      }}
      {...props}
    >
      {children}
    </div>
  );
};

export default Prose;

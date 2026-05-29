import React from 'react';

export interface NavBarProps extends React.HTMLAttributes<HTMLDivElement> {
  children?: React.ReactNode;
}

export const NavBar: React.FC<NavBarProps> = ({
  children,
  className = '',
  style,
  ...props
}) => {
  const navStyle: React.CSSProperties = {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    padding: '0 var(--space-6)',
    height: '64px',
    background: 'var(--panel)',
    backdropFilter: 'blur(12px)',
    WebkitBackdropFilter: 'blur(12px)',
    borderBottom: '1px solid var(--border-accent)',
    boxShadow: '0 4px 30px rgba(0, 0, 0, 0.5)',
    zIndex: 100,
    ...style,
  };

  return (
    <nav
      className={`sentinel-navbar-sys ${className}`}
      style={navStyle}
      {...props}
    >
      {children}
    </nav>
  );
};

export default NavBar;

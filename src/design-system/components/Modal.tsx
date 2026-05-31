import React from 'react';
import Card from './Card';
import Button from './Button';

export interface ModalProps {
  isOpen: boolean;
  onClose: () => void;
  title: string;
  children: React.ReactNode;
  actions?: React.ReactNode;
  /** Extra CSS class for the modal card (e.g. to widen it for pickers) */
  className?: string;
  /** Override default modal dimensions (merged with defaults) */
  style?: React.CSSProperties;
}

export const Modal: React.FC<ModalProps> = ({
  isOpen,
  onClose,
  title,
  children,
  actions,
  className,
  style: styleProp,
}) => {
  if (!isOpen) return null;

  const overlayStyle: React.CSSProperties = {
    position: 'fixed',
    top: 0,
    left: 0,
    width: '100%',
    height: '100%',
    background: 'var(--surface-overlay)',
    backdropFilter: 'blur(8px)',
    WebkitBackdropFilter: 'blur(8px)',
    display: 'flex',
    justifyContent: 'center',
    alignItems: 'center',
    zIndex: 1000,
  };

  const modalStyle: React.CSSProperties = {
    width: '100%',
    maxWidth: '500px',
    maxHeight: '90vh',
    display: 'flex',
    flexDirection: 'column',
    ...styleProp,
  };

  return (
    <div style={overlayStyle} onClick={onClose} id="modal-overlay">
      <Card
        style={modalStyle}
        className={className}
        glow="cyan"
        onClick={(e) => e.stopPropagation()}
        role="dialog"
        aria-modal="true"
        aria-labelledby="modal-title"
      >
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 'var(--space-4)' }}>
          <h3 id="modal-title" style={{ fontFamily: 'var(--font-mono)', fontSize: '1.25rem', fontWeight: 700, margin: 0, color: 'var(--cyan)' }}>
            {title}
          </h3>
          <button
            onClick={onClose}
            style={{
              background: 'transparent',
              border: 'none',
              color: 'var(--text-muted)',
              fontSize: '1.5rem',
              cursor: 'pointer',
              lineHeight: 1,
            }}
            aria-label="Close modal"
          >
            &times;
          </button>
        </div>
        
        <div style={{ color: 'var(--text-primary)', fontSize: '0.95rem', lineHeight: 1.6, marginBottom: 'var(--space-6)', overflowY: 'auto', flex: 1, minHeight: 0 }}>
          {children}
        </div>
        
        <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '12px' }}>
          {actions || (
            <Button variant="secondary" onClick={onClose}>
              Dismiss
            </Button>
          )}
        </div>
      </Card>
    </div>
  );
};

export default Modal;

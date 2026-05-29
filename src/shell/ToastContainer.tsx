import React, { useEffect } from 'react';
import { useToastsStore, ToastItem } from './toasts';

export const Toast: React.FC<{ toast: ToastItem }> = ({ toast }) => {
  const removeToast = useToastsStore((s) => s.removeToast);

  useEffect(() => {
    if (toast.duration === undefined || toast.duration <= 0) return;
    const timer = setTimeout(() => {
      removeToast(toast.id);
    }, toast.duration);

    return () => clearTimeout(timer);
  }, [toast.id, toast.duration, removeToast]);

  const handleDismiss = () => {
    removeToast(toast.id);
  };

  const getIcon = () => {
    switch (toast.type) {
      case 'success':
        return '✓';
      case 'error':
        return '✕';
      case 'warning':
        return '⚠';
      case 'info':
      default:
        return 'ℹ';
    }
  };

  return (
    <div
      className={`sentinel-toast toast-${toast.type} animate-slide-in`}
      onClick={handleDismiss}
      role="alert"
      id={`toast-${toast.id}`}
      style={{ cursor: 'pointer' }}
    >
      <div className="toast-body">
        <span className="toast-icon">{getIcon()}</span>
        <span className="toast-message">{toast.message}</span>
      </div>
      <span className="toast-close">&times;</span>
    </div>
  );
};

export const ToastContainer: React.FC = () => {
  const toasts = useToastsStore((s) => s.toasts);

  return (
    <div className="sentinel-toast-container" id="toast-container">
      {toasts.map((toast) => (
        <Toast key={toast.id} toast={toast} />
      ))}
    </div>
  );
};

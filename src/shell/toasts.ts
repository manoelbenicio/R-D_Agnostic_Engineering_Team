import { create } from 'zustand';

export interface ToastItem {
  id: string;
  message: string;
  type: 'info' | 'error' | 'success' | 'warning';
  duration?: number;
}

export interface ToastsState {
  toasts: ToastItem[];
  addToast: (message: string, type: ToastItem['type'], duration?: number) => string;
  removeToast: (id: string) => void;
}

export const useToastsStore = create<ToastsState>((set) => ({
  toasts: [],
  addToast: (message, type, duration = 4000) => {
    const id = Math.random().toString(36).substring(2, 9);
    set((state) => ({
      toasts: [...state.toasts, { id, message, type, duration }],
    }));
    return id;
  },
  removeToast: (id) =>
    set((state) => ({
      toasts: state.toasts.filter((t) => t.id !== id),
    })),
}));

export const useToast = () => {
  const addToast = useToastsStore((s) => s.addToast);
  const removeToast = useToastsStore((s) => s.removeToast);

  return {
    info: (message: string, duration?: number) => addToast(message, 'info', duration),
    error: (message: string, duration?: number) => addToast(message, 'error', duration),
    success: (message: string, duration?: number) => addToast(message, 'success', duration),
    warning: (message: string, duration?: number) => addToast(message, 'warning', duration),
    dismiss: removeToast,
  };
};

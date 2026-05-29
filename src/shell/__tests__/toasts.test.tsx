import { describe, it, expect, beforeEach } from 'vitest';
import { act, renderHook } from '@testing-library/react';
import { useToastsStore, useToast } from '../toasts';

describe('toastsStore & useToast', () => {
  beforeEach(() => {
    act(() => {
      useToastsStore.setState({ toasts: [] });
    });
  });

  it('adds a toast via useToast hook', () => {
    const { result } = renderHook(() => useToast());

    let toastId = '';
    act(() => {
      toastId = result.current.info('System online', 3000);
    });

    const state = useToastsStore.getState();
    expect(state.toasts).toHaveLength(1);
    expect(state.toasts[0]).toEqual({
      id: toastId,
      message: 'System online',
      type: 'info',
      duration: 3000,
    });
  });

  it('removes a toast manually via dismiss', () => {
    const { result } = renderHook(() => useToast());

    let toastId = '';
    act(() => {
      toastId = result.current.error('Connection failure');
    });

    expect(useToastsStore.getState().toasts).toHaveLength(1);

    act(() => {
      result.current.dismiss(toastId);
    });

    expect(useToastsStore.getState().toasts).toHaveLength(0);
  });

  it('supports success and warning toast types', () => {
    const { result } = renderHook(() => useToast());

    act(() => {
      result.current.success('Canvas Saved');
      result.current.warning('Low Memory Alert');
    });

    const state = useToastsStore.getState();
    expect(state.toasts).toHaveLength(2);
    expect(state.toasts[0]?.type).toBe('success');
    expect(state.toasts[1]?.type).toBe('warning');
  });
});

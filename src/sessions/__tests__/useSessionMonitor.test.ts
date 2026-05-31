import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, waitFor } from '@testing-library/react';
import { useSessionMonitor } from '../useSessionMonitor';

const mockHydrate = vi.fn().mockResolvedValue(undefined);
const mockRefresh = vi.fn().mockResolvedValue(undefined);
let mockSessions: any[] = [];

vi.mock('@/api/session-store', () => ({
  useSessionStore: vi.fn(() => ({
    hydrate: mockHydrate,
    refresh: mockRefresh,
    sessions: mockSessions,
  })),
}));

describe('useSessionMonitor', () => {
  beforeEach(() => {
    mockHydrate.mockClear();
    mockRefresh.mockClear();
    mockSessions = [];
  });

  it('hydrates and refreshes on mount', async () => {
    renderHook(() => useSessionMonitor());
    expect(mockHydrate).toHaveBeenCalledTimes(1);
    await waitFor(() => expect(mockRefresh).toHaveBeenCalledTimes(1));
  });

  it('calls refresh on window focus', () => {
    renderHook(() => useSessionMonitor());
    mockRefresh.mockClear();

    window.dispatchEvent(new Event('focus'));
    expect(mockRefresh).toHaveBeenCalledTimes(1);
  });

  it('refreshes periodically based on interval', () => {
    vi.useFakeTimers();
    renderHook(() => useSessionMonitor(1000));
    mockRefresh.mockClear();

    vi.advanceTimersByTime(1000);
    expect(mockRefresh).toHaveBeenCalledTimes(1);

    vi.advanceTimersByTime(1000);
    expect(mockRefresh).toHaveBeenCalledTimes(2);

    vi.useRealTimers();
  });

  it('warns about expiring sessions', () => {
    const consoleWarnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});

    mockSessions = [
      {
        id: '1',
        cli_provider: 'claude_code',
        account_email: 't@t.com',
        status: 'expiring',
        config_dir: '',
        auth_method: 'oauth',
      },
    ];

    renderHook(() => useSessionMonitor());

    expect(consoleWarnSpy).toHaveBeenCalledWith(
      '[SessionMonitor] 1 session(s) expiring soon:',
      'claude_code: t@t.com'
    );

    consoleWarnSpy.mockRestore();
  });

  it('clears interval and listener on unmount', () => {
    vi.useFakeTimers();
    const clearIntervalSpy = vi.spyOn(global, 'clearInterval');
    const removeEventListenerSpy = vi.spyOn(window, 'removeEventListener');

    const { unmount } = renderHook(() => useSessionMonitor(1000));

    unmount();

    expect(clearIntervalSpy).toHaveBeenCalled();
    expect(removeEventListenerSpy).toHaveBeenCalledWith('focus', expect.any(Function));

    clearIntervalSpy.mockRestore();
    removeEventListenerSpy.mockRestore();
    vi.useRealTimers();
  });
});

import { useEffect, useRef } from 'react';
import { useSessionStore } from '@/api/session-store';

/**
 * Monitors OAuth sessions and auto-refreshes periodically.
 * Place this hook in AppLayout so it runs globally.
 *
 * - Refreshes every 5 minutes to detect expired/new sessions
 * - Immediately refreshes on window focus (user returns to tab)
 * - Logs expiring sessions to console as warnings
 */
export function useSessionMonitor(intervalMs = 5 * 60 * 1000): void {
  const { refresh, sessions } = useSessionStore();
  const intervalRef = useRef<ReturnType<typeof setInterval>>();

  // Periodic refresh
  useEffect(() => {
    void refresh(); // Run immediately on mount
    intervalRef.current = setInterval(() => {
      void refresh();
    }, intervalMs);
    return () => {
      if (intervalRef.current) clearInterval(intervalRef.current);
    };
  }, [refresh, intervalMs]);

  // Refresh on window focus
  useEffect(() => {
    const onFocus = () => void refresh();
    window.addEventListener('focus', onFocus);
    return () => window.removeEventListener('focus', onFocus);
  }, [refresh]);

  // Warn about expiring sessions
  useEffect(() => {
    const expiring = sessions.filter((s) => s.status === 'expiring');
    if (expiring.length > 0) {
      console.warn(
        `[SessionMonitor] ${expiring.length} session(s) expiring soon:`,
        expiring.map((s) => `${s.cli_provider}: ${s.account_email}`).join(', ')
      );
    }
  }, [sessions]);
}

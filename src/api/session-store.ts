import { create } from 'zustand';
import { discoverSessions, triggerLogin, revokeSession as revokeSessionApi, type DiscoveredSession } from './session-discovery';

export interface SessionState {
  sessions: DiscoveredSession[];
  loading: boolean;
  error: string | null;
  lastRefreshed: string | null;
  refresh: () => Promise<void>;
  addSession: (cliProvider: string, configDir?: string) => Promise<void>;
  getSession: (id: string) => DiscoveredSession | undefined;
  getSessionsForProvider: (cliProvider: string) => DiscoveredSession[];
  revokeSession: (sessionId: string) => Promise<boolean>;
  clearError: () => void;
}

export const useSessionStore = create<SessionState>((set, get) => ({
  sessions: [],
  loading: false,
  error: null,
  lastRefreshed: null,
  refresh: async () => {
    set({ loading: true, error: null });
    try {
      const sessions = await discoverSessions();
      set({ sessions, loading: false, lastRefreshed: new Date().toISOString() });
    } catch (err) {
      set({
        loading: false,
        error: err instanceof Error ? err.message : 'Failed to discover sessions',
      });
    }
  },
  addSession: async (cliProvider, configDir) => {
    try {
      await triggerLogin(cliProvider, configDir);
      await get().refresh();
    } catch (err) {
      set({ error: err instanceof Error ? err.message : 'Failed to start login' });
    }
  },
  getSession: (id) => get().sessions.find((session) => session.id === id),
  getSessionsForProvider: (cliProvider) => (
    get().sessions.filter((session) => session.cli_provider === cliProvider)
  ),
  revokeSession: async (sessionId: string) => {
    const session = get().sessions.find(s => s.id === sessionId);
    if (!session) return false;
    const success = await revokeSessionApi(sessionId, session.cli_provider, session.config_dir);
    if (success) {
      await get().refresh();
    }
    return success;
  },
  clearError: () => set({ error: null }),
}));

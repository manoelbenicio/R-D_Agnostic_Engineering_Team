import { create } from 'zustand';
import { discoverSessions, triggerLogin, revokeSession as revokeSessionApi, type DiscoveredSession } from './session-discovery';
import { openDb } from '@/shared/storage/idb';

export interface SessionState {
  sessions: DiscoveredSession[];
  loading: boolean;
  error: string | null;
  lastRefreshed: string | null;
  hydrate: () => Promise<void>;
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
  hydrate: async () => {
    try {
      const db = await openDb();
      const cached = await db.get('sessions', 'cache');
      if (cached?.sessions) {
        set({
          sessions: cached.sessions,
          lastRefreshed: cached.lastRefreshed ?? null,
        });
      }
    } catch {
      // A missing or unavailable IndexedDB cache should behave like a fresh start.
    }
  },
  refresh: async () => {
    set({ loading: true, error: null });
    try {
      const sessions = await discoverSessions();
      const lastRefreshed = new Date().toISOString();
      set({ sessions, loading: false, lastRefreshed });
      const db = await openDb();
      const tx = db.transaction('sessions', 'readwrite');
      await tx.store.put({ sessions, lastRefreshed }, 'cache');
      await tx.done;
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

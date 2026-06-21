import { create } from 'zustand';
import { goCoreClient } from './go-core-client';  // CRIT-003.8
import type { HealthResponse } from './types';

const HEALTH_POLL_INTERVAL_MS = 10_000;

export type HealthStatus = 'healthy' | 'unreachable' | 'loading';

export interface HealthState {
  status: HealthStatus;
  health?: HealthResponse;
  error?: Error;
  lastCheckedAt?: number;
  bannerVisible: boolean;
  start: () => void;
  stop: () => void;
  pollNow: () => Promise<void>;
  dismissBanner: () => void;
}

let pollTimer: ReturnType<typeof setInterval> | undefined;
let visibilityHandlerAttached = false;

export const useHealthStore = create<HealthState>((set, get) => ({
  status: 'loading',
  bannerVisible: false,
  start: () => {
    attachVisibilityHandler();
    if (isDocumentHidden()) return;
    if (!pollTimer) {
      void get().pollNow();
      pollTimer = setInterval(() => {
        if (!isDocumentHidden()) {
          void get().pollNow();
        }
      }, HEALTH_POLL_INTERVAL_MS);
    }
  },
  stop: () => {
    if (pollTimer) {
      clearInterval(pollTimer);
      pollTimer = undefined;
    }
  },
  pollNow: async () => {
    try {
      const health = await goCoreClient.getHealth();
      set({
        status: health.status === 'ok' ? 'healthy' : 'unreachable',
        health,
        error: undefined,
        lastCheckedAt: Date.now(),
        bannerVisible: false,
      });
    } catch (error) {
      set({
        status: 'unreachable',
        error: error instanceof Error ? error : new Error('Unknown GO Core health error.'),
        lastCheckedAt: Date.now(),
        bannerVisible: true,
      });
    }
  },
  dismissBanner: () => set({ bannerVisible: false }),
}));

function attachVisibilityHandler(): void {
  if (visibilityHandlerAttached || typeof document === 'undefined') return;
  visibilityHandlerAttached = true;
  document.addEventListener('visibilitychange', () => {
    const { start, stop, pollNow } = useHealthStore.getState();
    if (document.hidden) {
      stop();
    } else {
      start();
      void pollNow();
    }
  });
}

function isDocumentHidden(): boolean {
  return typeof document !== 'undefined' && document.hidden;
}

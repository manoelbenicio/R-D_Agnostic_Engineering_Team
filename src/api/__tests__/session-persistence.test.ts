import { act } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { openDb } from '@/shared/storage/idb';
import { discoverSessions, type DiscoveredSession } from '../session-discovery';
import { useSessionStore } from '../session-store';

vi.mock('../session-discovery', () => ({
  discoverSessions: vi.fn(),
  triggerLogin: vi.fn(),
  revokeSession: vi.fn(),
}));

const mockDiscoverSessions = vi.mocked(discoverSessions);

const sessions: DiscoveredSession[] = [
  {
    id: 'claude:primary',
    cli_provider: 'claude_code',
    account_email: 'claude@example.com',
    config_dir: 'C:/Users/dev/.claude',
    status: 'active',
    auth_method: 'oauth',
  },
  {
    id: 'codex:primary',
    cli_provider: 'codex',
    account_email: 'codex@example.com',
    config_dir: 'C:/Users/dev/.codex',
    status: 'expiring',
    auth_method: 'sso',
  },
];

describe('session IndexedDB persistence', () => {
  beforeEach(async () => {
    vi.clearAllMocks();
    const db = await openDb();
    await db.clear('sessions');
    act(() => {
      useSessionStore.setState({
        sessions: [],
        loading: false,
        error: null,
        lastRefreshed: null,
      });
    });
  });

  it('hydrates cached sessions from IndexedDB', async () => {
    const db = await openDb();
    const lastRefreshed = '2026-05-31T10:00:00.000Z';
    await db.put('sessions', { sessions, lastRefreshed }, 'cache');

    await act(async () => {
      await useSessionStore.getState().hydrate();
    });

    expect(useSessionStore.getState().sessions).toEqual(sessions);
    expect(useSessionStore.getState().lastRefreshed).toBe(lastRefreshed);
  });

  it('writes refreshed sessions to IndexedDB', async () => {
    mockDiscoverSessions.mockResolvedValue(sessions);

    await act(async () => {
      await useSessionStore.getState().refresh();
    });

    const db = await openDb();
    const cached = await db.get('sessions', 'cache');
    expect(cached?.sessions).toEqual(sessions);
    expect(cached?.lastRefreshed).toBe(useSessionStore.getState().lastRefreshed);
    expect(cached?.lastRefreshed).toEqual(expect.any(String));
  });

  it('hydrates stale cached sessions instead of starting empty', async () => {
    const db = await openDb();
    await db.put(
      'sessions',
      {
        sessions,
        lastRefreshed: '2024-01-01T00:00:00.000Z',
      },
      'cache',
    );

    await act(async () => {
      await useSessionStore.getState().hydrate();
    });

    expect(useSessionStore.getState().sessions).toEqual(sessions);
  });
});

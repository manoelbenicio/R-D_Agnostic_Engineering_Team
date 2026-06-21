import { act } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { discoverSessions, triggerLogin, type DiscoveredSession } from '../session-discovery';
import { useSessionStore } from '../session-store';

vi.mock('../session-discovery', () => ({
  discoverSessions: vi.fn(),
  triggerLogin: vi.fn(),
}));

const mockDiscoverSessions = vi.mocked(discoverSessions);
const mockTriggerLogin = vi.mocked(triggerLogin);

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
  {
    id: 'claude:secondary',
    cli_provider: 'claude_code',
    account_email: 'claude-alt@example.com',
    config_dir: 'C:/Users/dev/.claude-alt',
    status: 'expired',
    auth_method: 'oauth',
  },
];

describe('useSessionStore', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    act(() => {
      useSessionStore.setState({
        sessions: [],
        loading: false,
        error: null,
        lastRefreshed: null,
      });
    });
  });

  it('starts with the expected initial state', () => {
    const state = useSessionStore.getState();

    expect(state.sessions).toEqual([]);
    expect(state.loading).toBe(false);
    expect(state.error).toBeNull();
    expect(state.lastRefreshed).toBeNull();
  });

  it('refreshes sessions and records the refresh timestamp', async () => {
    let resolveSessions: (value: DiscoveredSession[]) => void = () => {};
    mockDiscoverSessions.mockReturnValue(
      new Promise((resolve) => {
        resolveSessions = resolve;
      }),
    );

    let refreshPromise: Promise<void>;
    act(() => {
      refreshPromise = useSessionStore.getState().refresh();
    });

    expect(mockDiscoverSessions).toHaveBeenCalledTimes(1);
    expect(useSessionStore.getState().loading).toBe(true);

    await act(async () => {
      resolveSessions(sessions);
      await refreshPromise;
    });

    const state = useSessionStore.getState();
    expect(state.sessions).toEqual(sessions);
    expect(state.loading).toBe(false);
    expect(state.error).toBeNull();
    expect(state.lastRefreshed).toEqual(expect.any(String));
    expect(Number.isNaN(Date.parse(state.lastRefreshed ?? ''))).toBe(false);
  });

  it('stores an error message when refresh fails', async () => {
    mockDiscoverSessions.mockRejectedValue(new Error('GO Core offline'));

    await act(async () => {
      await useSessionStore.getState().refresh();
    });

    expect(useSessionStore.getState().sessions).toEqual([]);
    expect(useSessionStore.getState().loading).toBe(false);
    expect(useSessionStore.getState().error).toBe('GO Core offline');
  });

  it('returns a matching session by id', () => {
    act(() => {
      useSessionStore.setState({ sessions });
    });

    expect(useSessionStore.getState().getSession('codex:primary')).toEqual(sessions[1]);
    expect(useSessionStore.getState().getSession('missing')).toBeUndefined();
  });

  it('filters sessions by provider', () => {
    act(() => {
      useSessionStore.setState({ sessions });
    });

    expect(useSessionStore.getState().getSessionsForProvider('claude_code')).toEqual([
      sessions[0],
      sessions[2],
    ]);
  });

  it('returns an empty array for an unknown provider', () => {
    act(() => {
      useSessionStore.setState({ sessions });
    });

    expect(useSessionStore.getState().getSessionsForProvider('unknown')).toEqual([]);
  });

  it('clears the current error', () => {
    act(() => {
      useSessionStore.setState({ error: 'previous failure' });
      useSessionStore.getState().clearError();
    });

    expect(useSessionStore.getState().error).toBeNull();
  });

  it('starts login and refreshes sessions when adding a session', async () => {
    mockTriggerLogin.mockResolvedValue(undefined);
    mockDiscoverSessions.mockResolvedValue(sessions);

    await act(async () => {
      await useSessionStore.getState().addSession('claude_code', 'C:/Users/dev/.claude');
    });

    expect(mockTriggerLogin).toHaveBeenCalledWith('claude_code', 'C:/Users/dev/.claude');
    expect(mockDiscoverSessions).toHaveBeenCalledTimes(1);
    expect(useSessionStore.getState().sessions).toEqual(sessions);
  });
});

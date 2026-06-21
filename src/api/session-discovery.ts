import { goCoreClient } from './go-core-client';  // CRIT-003.7
import type { ProviderAvailability } from './types';
import { isExpiringSoon } from './session-security';

export interface DiscoveredSession {
  id: string;
  cli_provider: string;
  account_email: string;
  config_dir: string;
  status: 'active' | 'expiring' | 'expired';
  expires_at?: string;
  subscription_type?: string;
  auth_method: 'oauth' | 'sso' | 'gcloud' | 'api_key';
}

/**
 * Backend `_session_status` only emits 'active' | 'expired'. Derive the
 * 'expiring' band on the client from `expires_at` so the expiring UX (yellow
 * dot, footer count, monitor warning) works against the live GO Core contract.
 */
function withExpiringStatus(session: DiscoveredSession): DiscoveredSession {
  if (session.status === 'active' && isExpiringSoon(session.expires_at)) {
    return { ...session, status: 'expiring' };
  }
  return session;
}

export async function discoverSessions(): Promise<DiscoveredSession[]> {
  try {
    const sessions = await requestJson<DiscoveredSession[]>('/auth/sessions');
    return sessions.map(withExpiringStatus);
  } catch {
    const providers = await requestJson<ProviderAvailability[]>('/agents/providers');
    return providers
      .filter((provider) => provider.installed)
      .map((provider) => ({
        id: `${provider.name}:default`,
        cli_provider: provider.name,
        account_email: 'Default CLI session',
        config_dir: '',
        status: 'active' as const,
        auth_method: 'oauth' as const,
      }));
  }
}

export function resolveSessionEnv(session: DiscoveredSession, model?: string): Record<string, string> {
  const env: Record<string, string | undefined> = {};

  switch (session.cli_provider) {
    case 'claude_code':
      env.CLAUDE_CONFIG_DIR = session.config_dir;
      env.ANTHROPIC_MODEL = model;
      break;
    case 'codex':
      env.OPENAI_MODEL = model;
      break;
    case 'gemini_cli':
      env.GEMINI_MODEL = model;
      break;
    case 'kiro_cli':
      env.KIRO_HOME = session.config_dir;
      break;
    default:
      break;
  }

  return Object.fromEntries(
    Object.entries(env).filter((entry): entry is [string, string] => Boolean(entry[1]))
  );
}

export async function triggerLogin(cliProvider: string, configDir?: string): Promise<void> {
  const response = await fetch(toEndpoint('/auth/login'), {
    method: 'POST',
    headers: {
      'content-type': 'application/json',
    },
    body: JSON.stringify({ provider: cliProvider, config_dir: configDir }),
  });

  if (!response.ok) {
    throw new Error(`GO Core login request failed with HTTP ${response.status}.`);
  }
}

/**
 * Revoke/logout an OAuth session by asking the GO Core backend
 * to clear the credentials for a specific config directory.
 */
export async function revokeSession(sessionId: string, cliProvider: string, configDir: string): Promise<boolean> {
  try {
    const res = await fetch(
      toEndpoint(`/auth/sessions/${sessionId}`),
      {
        method: 'DELETE',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ provider: cliProvider, config_dir: configDir }),
      }
    );
    return res.ok;
  } catch {
    return false;
  }
}

async function requestJson<T>(path: string): Promise<T> {
  const response = await fetch(toEndpoint(path));
  if (!response.ok) {
    throw new Error(`GO Core request failed with HTTP ${response.status} for ${path}.`);
  }
  return (await response.json()) as T;
}

function toEndpoint(path: string): string {
  return `${goCoreClient.baseUrl}${path.startsWith('/') ? path : `/${path}`}`;
}

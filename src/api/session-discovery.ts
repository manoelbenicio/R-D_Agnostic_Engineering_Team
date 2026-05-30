import { caoClient } from './cao-client';
import type { ProviderAvailability } from './types';

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

export async function discoverSessions(): Promise<DiscoveredSession[]> {
  try {
    return await requestJson<DiscoveredSession[]>('/auth/sessions');
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
    throw new Error(`CAO login request failed with HTTP ${response.status}.`);
  }
}

async function requestJson<T>(path: string): Promise<T> {
  const response = await fetch(toEndpoint(path));
  if (!response.ok) {
    throw new Error(`CAO request failed with HTTP ${response.status} for ${path}.`);
  }
  return (await response.json()) as T;
}

function toEndpoint(path: string): string {
  return `${caoClient.baseUrl}${path.startsWith('/') ? path : `/${path}`}`;
}

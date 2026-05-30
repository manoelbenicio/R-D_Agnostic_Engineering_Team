import { CAO_BASE_URL } from './base-url';
import { CaoApiError, CaoNetworkError } from './errors';
import type {
  AddTerminalInput,
  AgentDirsResponse,
  AgentProfile,
  CreateSessionInput,
  Flow,
  HealthResponse,
  InboxMessage,
  InboxMessageFilters,
  ProviderAvailability,
  Session,
  Terminal,
} from './types';

type ResponseMode = 'json' | 'text' | 'void';

interface RequestOptions {
  method?: 'GET' | 'POST' | 'DELETE';
  body?: unknown;
  responseMode?: ResponseMode;
  contentType?: string;
}

export class CaoClient {
  baseUrl: string;

  constructor(baseUrl = CAO_BASE_URL) {
    this.baseUrl = baseUrl.replace(/\/$/, '');
  }

  getHealth(): Promise<HealthResponse> {
    return this.request('/health');
  }

  listProfiles(): Promise<AgentProfile[]> {
    return this.request('/agents/profiles');
  }

  getProfile(name: string): Promise<AgentProfile> {
    return this.request(`/agents/profiles/${encodeURIComponent(name)}`);
  }

  installProfile(profileMarkdown: string): Promise<AgentProfile> {
    return this.request('/agents/profiles/install', {
      method: 'POST',
      body: profileMarkdown,
      contentType: 'text/markdown; charset=utf-8',
    });
  }

  listProviders(): Promise<ProviderAvailability[]> {
    return this.request('/agents/providers');
  }

  createSession(input: CreateSessionInput): Promise<Session> {
    return this.request('/sessions', { method: 'POST', body: input });
  }

  listSessions(): Promise<Session[]> {
    return this.request('/sessions');
  }

  getSession(name: string): Promise<Session> {
    return this.request(`/sessions/${encodeURIComponent(name)}`);
  }

  deleteSession(name: string): Promise<void> {
    return this.request(`/sessions/${encodeURIComponent(name)}`, {
      method: 'DELETE',
      responseMode: 'void',
    });
  }

  addTerminalToSession(sessionName: string, input: AddTerminalInput): Promise<Terminal> {
    return this.request(`/sessions/${encodeURIComponent(sessionName)}/terminals`, {
      method: 'POST',
      body: input,
    });
  }

  listTerminalsInSession(sessionName: string): Promise<Terminal[]> {
    return this.request(`/sessions/${encodeURIComponent(sessionName)}/terminals`);
  }

  getTerminal(id: string): Promise<Terminal> {
    return this.request(`/terminals/${encodeURIComponent(id)}`);
  }

  getTerminalOutput(id: string, mode: 'full' | 'tail' | 'visible'): Promise<string> {
    return this.request(`/terminals/${encodeURIComponent(id)}/output?mode=${encodeURIComponent(mode)}`, {
      responseMode: 'text',
    });
  }

  getTerminalWorkingDirectory(id: string): Promise<string> {
    return this.request(`/terminals/${encodeURIComponent(id)}/working-directory`, {
      responseMode: 'text',
    });
  }

  getTerminalMemoryContext(id: string): Promise<string> {
    return this.request(`/terminals/${encodeURIComponent(id)}/memory-context`, {
      responseMode: 'text',
    });
  }

  sendTerminalInput(id: string, message: string): Promise<void> {
    return this.request(`/terminals/${encodeURIComponent(id)}/input`, {
      method: 'POST',
      body: { message },
      responseMode: 'void',
    });
  }

  exitTerminal(id: string): Promise<void> {
    return this.request(`/terminals/${encodeURIComponent(id)}/exit`, {
      method: 'POST',
      responseMode: 'void',
    });
  }

  deleteTerminal(id: string): Promise<void> {
    return this.request(`/terminals/${encodeURIComponent(id)}`, {
      method: 'DELETE',
      responseMode: 'void',
    });
  }

  sendInboxMessage(terminalId: string, message: string): Promise<InboxMessage> {
    return this.request(`/terminals/${encodeURIComponent(terminalId)}/inbox/messages`, {
      method: 'POST',
      body: { message },
    });
  }

  listInboxMessages(terminalId: string, filters: InboxMessageFilters = {}): Promise<InboxMessage[]> {
    const params = new URLSearchParams();
    if (filters.limit !== undefined) params.set('limit', String(filters.limit));
    if (filters.status) params.set('status', filters.status);
    const suffix = params.size > 0 ? `?${params.toString()}` : '';
    return this.request(`/terminals/${encodeURIComponent(terminalId)}/inbox/messages${suffix}`);
  }

  listFlows(): Promise<Flow[]> {
    return this.request('/flows');
  }

  getFlow(name: string): Promise<Flow> {
    return this.request(`/flows/${encodeURIComponent(name)}`);
  }

  createFlow(flow: Flow): Promise<Flow> {
    return this.request('/flows', { method: 'POST', body: flow });
  }

  deleteFlow(name: string): Promise<void> {
    return this.request(`/flows/${encodeURIComponent(name)}`, {
      method: 'DELETE',
      responseMode: 'void',
    });
  }

  enableFlow(name: string): Promise<void> {
    return this.request(`/flows/${encodeURIComponent(name)}/enable`, {
      method: 'POST',
      responseMode: 'void',
    });
  }

  disableFlow(name: string): Promise<void> {
    return this.request(`/flows/${encodeURIComponent(name)}/disable`, {
      method: 'POST',
      responseMode: 'void',
    });
  }

  runFlow(name: string): Promise<void> {
    return this.request(`/flows/${encodeURIComponent(name)}/run`, {
      method: 'POST',
      responseMode: 'void',
    });
  }

  getAgentDirs(): Promise<AgentDirsResponse> {
    return this.request('/settings/agent-dirs');
  }

  setAgentDirs(dirs: string[]): Promise<AgentDirsResponse> {
    return this.request('/settings/agent-dirs', {
      method: 'POST',
      body: { dirs },
    });
  }

  getSkill(name: string): Promise<string> {
    return this.request(`/skills/${encodeURIComponent(name)}`, {
      responseMode: 'text',
    });
  }

  /** Discover authenticated CLI sessions on the CAO host. */
  async listAuthSessions(): Promise<{
    id: string;
    cli_provider: string;
    account_email: string;
    config_dir: string;
    status: string;
    expires_at?: string;
    subscription_type?: string;
    auth_method: string;
  }[]> {
    try {
      const res = await fetch(`${this.baseUrl}/auth/sessions`, {
        method: 'GET',
        headers: { 'Accept': 'application/json' },
      });
      if (!res.ok) return [];
      return await res.json();
    } catch {
      return [];
    }
  }

  private async request<T>(path: string, options: RequestOptions = {}): Promise<T> {
    const endpoint = this.toEndpoint(path);
    const responseMode = options.responseMode ?? 'json';
    const method = options.method ?? 'GET';
    const headers = new Headers();
    let body: BodyInit | undefined;

    if (options.body !== undefined) {
      if (typeof options.body === 'string') {
        headers.set('content-type', options.contentType ?? 'text/plain; charset=utf-8');
        body = options.body;
      } else {
        headers.set('content-type', options.contentType ?? 'application/json');
        body = JSON.stringify(options.body);
      }
    }

    let response: Response;
    try {
      response = await fetch(endpoint, {
        method,
        headers,
        body,
      });
    } catch (cause) {
      throw new CaoNetworkError(`Unable to reach runtime endpoint ${path}.`, { endpoint: path, cause });
    }

    if (!response.ok) {
      const errorBody = await parseErrorBody(response);
      throw new CaoApiError(`CAO request failed with HTTP ${response.status} for ${path}.`, {
        status: response.status,
        endpoint: path,
        body: errorBody,
      });
    }

    if (responseMode === 'void' || response.status === 204) {
      return undefined as T;
    }
    if (responseMode === 'text') {
      return (await response.text()) as T;
    }
    return (await response.json()) as T;
  }

  private toEndpoint(path: string): string {
    return `${this.baseUrl}${path.startsWith('/') ? path : `/${path}`}`;
  }
}

async function parseErrorBody(response: Response): Promise<unknown> {
  const contentType = response.headers.get('content-type') ?? '';
  if (contentType.includes('application/json')) {
    try {
      return await response.json();
    } catch {
      return null;
    }
  }
  return response.text();
}

export const caoClient = new CaoClient(CAO_BASE_URL);

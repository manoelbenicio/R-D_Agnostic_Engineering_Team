// Types shared between GO Core client and the rest of the SPA.
// CRIT-001 + CRIT-003.1: prod-readiness-critical-fixes

export interface HealthResponse {
  status: 'ok';
}

export type ProviderType =
  | 'openai'
  | 'anthropic'
  | 'google'
  | 'aws'
  | 'azure'
  | 'moonshot'
  | 'copilot_cli'
  | 'opencode_cli'
  | (string & {});

export type TerminalStatus =
  | 'starting'
  | 'idle'
  | 'processing'
  | 'error'
  | 'offline'
  | 'exited'
  | (string & {});

export interface ProviderAvailability {
  name: ProviderType;
  installed: boolean;
}

export interface AgentProfile {
  name: string;
  role: string;
  provider: ProviderType;
  description?: string;
  markdown?: string;
  system_prompt?: string;
  allowed_tools?: string[];
  metadata?: Record<string, unknown>;
}

export interface Session {
  name: string;
  profile: string;
  working_directory: string;
  status: 'active' | 'idle' | 'error' | 'deleted' | (string & {});
  terminals?: Terminal[];
  created_at?: string;
  updated_at?: string;
}

export interface Terminal {
  id: string;
  session_name?: string;
  profile: string;
  provider?: ProviderType;
  display_name?: string;
  status: TerminalStatus;
  working_directory: string;
  created_at?: string;
  updated_at?: string;
}

export interface InboxMessage {
  id: string;
  terminal_id: string;
  message: string;
  status: 'unread' | 'read' | 'archived' | (string & {});
  sender?: string;
  created_at: string;
}

export interface Flow {
  name: string;
  schedule: string;
  agent_profile: string;
  provider: ProviderType;
  prompt_template: string;
  enabled: boolean;
  last_run?: string | null;
  next_run?: string | null;
  gating_script?: string | null;
}

export interface CreateSessionInput {
  profile: string;
  working_directory: string;
  provider: string;  // CRIT-001: required by GO Core (was missing — caused 6 TS errors)
  env_vars?: Record<string, string>;  // Per-terminal env var injection for OAuth routing
}

export type AddTerminalInput = CreateSessionInput;

export interface InboxMessageFilters {
  limit?: number;
  status?: string;
}

export interface AgentDirsResponse {
  dirs: string[];
}

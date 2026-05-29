import { ProviderType } from '@/api/types';

export type OrchestrationType = 'handoff' | 'assign' | 'send_message';
export type { ProviderType };

export interface CanvasNode {
  id: string;          // UUIDv4
  type: 'agent';       // reserved for future node types
  position: { x: number; y: number };
  data: {
    profile_name: string;
    display_name: string;
    role: 'supervisor' | 'developer' | 'reviewer' | string;
    provider?: ProviderType;
    model?: string;
    system_prompt: string;
    allowedTools?: string[];
    is_entry_point: boolean;
  };
}

export interface CanvasEdge {
  id: string;
  source: string;      // node id
  target: string;      // node id
  type: OrchestrationType;
  label?: string;
}

export interface CanvasNodeSnapshot {
  system_prompt: string;
  allowedTools: string[];
  model: string;
  provider: string;
}

export interface CanvasDocument {
  id: string;
  name: string;
  version: number;     // monotonic; bumps on save
  created_at: string;  // ISO-8601
  updated_at: string;  // ISO-8601
  schema_version: number;
  nodes: CanvasNode[];
  edges: CanvasEdge[];
  config: {
    working_directory: string;
    session_name?: string;
    provider_default: ProviderType;
    env_vars?: Record<string, string>;
  };
  deploy_state: {
    status: 'draft' | 'deploying' | 'deployed' | 'degraded';
    session_name?: string;
    terminal_map?: Record<string, string>;  // node_id → terminal_id
    last_deployed?: string;
    errors?: { node_id: string; error: string }[];
    profile_snapshots?: Record<string, CanvasNodeSnapshot>;
    edge_change_advisory?: boolean;
  };
}

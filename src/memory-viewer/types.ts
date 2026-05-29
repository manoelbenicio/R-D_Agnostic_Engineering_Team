export type MemoryScope = 'global' | 'project' | 'session' | 'agent';
export type MemoryType = 'project' | 'user' | 'feedback' | 'reference';

export interface MemoryEntry {
  id: string;
  title: string;
  scope: MemoryScope;
  type: MemoryType;
  tags: string[];
  content: string;
  updatedAt: string;
  retention?: string;
  locationPath: string;
  terminalId?: string;
  sessionName?: string;
  source: 'terminal-context' | 'manual';
}

export interface MemoryViewerData {
  entries: MemoryEntry[];
  agentDirs: string[];
  terminalCount: number;
}

export interface MemoryFormState {
  title: string;
  scope: MemoryScope;
  type: MemoryType;
  tags: string;
  content: string;
}

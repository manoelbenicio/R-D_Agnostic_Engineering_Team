// Canonical tool vocabulary for agent nodes. Single source of truth for the
// validated "Allowed tools" multi-select. Keep in sync with role-templates.ts.

export interface AllowedToolDef {
  id: string;
  label: string;
  /** 1-line description: what it does + when to use it. */
  description: string;
  category: 'Orchestration' | 'Filesystem' | 'Execution' | 'Verification' | 'Research';
}

export const ALLOWED_TOOLS: readonly AllowedToolDef[] = [
  { id: 'handoff', label: 'handoff', category: 'Orchestration', description: 'Transfer control of the task to another agent.' },
  { id: 'assign', label: 'assign', category: 'Orchestration', description: 'Delegate a scoped sub-task and expect a result back.' },
  { id: 'send_message', label: 'send_message', category: 'Orchestration', description: 'Send a one-way message/context to another agent.' },
  { id: 'read_file', label: 'read_file', category: 'Filesystem', description: 'Read file contents.' },
  { id: 'grep', label: 'grep', category: 'Filesystem', description: 'Search file contents by pattern.' },
  { id: 'shell', label: 'shell', category: 'Execution', description: 'Run arbitrary shell commands (high privilege).' },
  { id: 'apply_patch', label: 'apply_patch', category: 'Execution', description: 'Apply code edits/patches to files (high privilege).' },
  { id: 'test', label: 'test', category: 'Verification', description: "Run the project's test suite." },
  { id: 'web_search', label: 'web_search', category: 'Research', description: 'Search the web for information.' },
] as const;

export const ALLOWED_TOOL_IDS: readonly string[] = ALLOWED_TOOLS.map((tool) => tool.id);

/** Tools that execute or modify code — surfaced as high-privilege in the UI. */
export const HIGH_PRIVILEGE_TOOLS: readonly string[] = ['shell', 'apply_patch'];

/** Tool ids present on the node but not in the canonical vocabulary (e.g. typos). */
export function unknownTools(selected: readonly string[]): string[] {
  return selected.filter((tool) => !ALLOWED_TOOL_IDS.includes(tool));
}

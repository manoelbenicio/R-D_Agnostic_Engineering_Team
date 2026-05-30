/* eslint-disable agentverse/no-sideways-capability-imports */
import { CanvasDocument, CanvasNode, CanvasEdge, CreateCanvasIntent, ProviderType } from './types';
import { SCHEMA_VERSION } from '@/shared/schema-version';

interface RoleTemplate {
  profile_name: string;
  system_prompt: string;
  allowedTools: string[];
}

const ROLE_TEMPLATES: Record<string, RoleTemplate> = {
  supervisor: {
    profile_name: 'supervisor',
    system_prompt: 'You are a Supervisor agent. You coordinate the team, delegate tasks, and oversee execution.',
    allowedTools: ['handoff', 'send_message'],
  },
  developer: {
    profile_name: 'developer',
    system_prompt: 'You are a Developer agent. You write code and execute implementation plans.',
    allowedTools: ['shell', 'apply_patch'],
  },
  reviewer: {
    profile_name: 'reviewer',
    system_prompt: 'You are a Reviewer agent. You audit code changes and verify correctness.',
    allowedTools: ['read_file', 'grep'],
  },
  custom: {
    profile_name: 'custom',
    system_prompt: 'You are a Custom agent helper.',
    allowedTools: [],
  },
};

function generateUUID(): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID();
  }
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, (c) => {
    const r = (Math.random() * 16) | 0;
    const v = c === 'x' ? r : (r & 0x3) | 0x8;
    return v.toString(16);
  });
}

export function voiceToCanvas(intent: CreateCanvasIntent): CanvasDocument {
  const nodes: CanvasNode[] = [];
  const edges: CanvasEdge[] = [];

  let supervisorIndex = intent.nodes.findIndex((n) => n.role === 'supervisor');
  if (supervisorIndex === -1 && intent.nodes.length > 0) {
    supervisorIndex = 0;
  }

  let horizontalCount = 0;
  const nameToIdMap: Record<string, string> = {};

  intent.nodes.forEach((n, idx) => {
    const id = generateUUID();
    nameToIdMap[n.display_name] = id;

    const isEntry = idx === supervisorIndex;
    const position = isEntry
      ? { x: 200, y: 100 }
      : { x: 100 + horizontalCount++ * 250, y: 300 };

    const role = n.role || 'custom';
    const defaults = (ROLE_TEMPLATES[role] || ROLE_TEMPLATES.custom)!;

    const providerStr = (n.provider || '').toLowerCase();
    let mappedProvider: ProviderType | undefined;
    if (providerStr.includes('kiro')) mappedProvider = 'kiro_cli';
    else if (providerStr.includes('claude')) mappedProvider = 'claude_code';
    else if (providerStr.includes('codex')) mappedProvider = 'codex';
    else if (providerStr.includes('gemini')) mappedProvider = 'gemini_cli';
    else if (providerStr.includes('kimi')) mappedProvider = 'kimi_cli';
    else if (providerStr.includes('copilot')) mappedProvider = 'copilot_cli';
    else if (providerStr.includes('opencode')) mappedProvider = 'opencode_cli';
    else if (providerStr.includes('q')) mappedProvider = 'q_cli';

    nodes.push({
      id,
      type: 'agent',
      position,
      data: {
        profile_name: defaults.profile_name,
        display_name: n.display_name,
        role: n.role,
        provider: mappedProvider,
        system_prompt: defaults.system_prompt,
        allowedTools: defaults.allowedTools,
        is_entry_point: isEntry,
      },
    });
  });

  intent.edges.forEach((e) => {
    const sourceId = nameToIdMap[e.from];
    const targetId = nameToIdMap[e.to];
    if (sourceId && targetId) {
      edges.push({
        id: generateUUID(),
        source: sourceId,
        target: targetId,
        type: e.type || 'handoff',
        label: e.type || 'handoff',
      });
    }
  });

  return {
    id: generateUUID(),
    name: intent.name || 'Voice Generated Canvas',
    version: 1,
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
    schema_version: SCHEMA_VERSION,
    nodes,
    edges,
    config: {
      working_directory: '~',
      provider_default: '',
    },
    deploy_state: {
      status: 'draft',
    },
  };
}

export default voiceToCanvas;

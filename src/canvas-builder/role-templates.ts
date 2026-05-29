import { CanvasNode, ProviderType } from '@/shared/canvas-types';

export type StarterRole = 'supervisor' | 'developer' | 'reviewer' | 'custom';

export interface RoleTemplate {
  role: StarterRole;
  display_name: string;
  profile_prefix: string;
  system_prompt: string;
  allowedTools: string[];
}

export const ROLE_TEMPLATES: Record<StarterRole, RoleTemplate> = {
  supervisor: {
    role: 'supervisor',
    display_name: 'Supervisor',
    profile_prefix: 'supervisor',
    system_prompt:
      'You are the supervisor for this canvas. Break work into clear assignments, coordinate handoffs, and keep the team aligned with the user goal.',
    allowedTools: ['handoff', 'assign', 'send_message'],
  },
  developer: {
    role: 'developer',
    display_name: 'Developer',
    profile_prefix: 'developer',
    system_prompt:
      'You are a developer agent. Implement scoped changes, run focused verification, and report concrete results back to the supervisor.',
    allowedTools: ['shell', 'apply_patch', 'read_file'],
  },
  reviewer: {
    role: 'reviewer',
    display_name: 'Reviewer',
    profile_prefix: 'reviewer',
    system_prompt:
      'You are a reviewer agent. Inspect changes for regressions, missing tests, security concerns, and unclear behavior before approval.',
    allowedTools: ['read_file', 'grep', 'test'],
  },
  custom: {
    role: 'custom',
    display_name: 'Custom Agent',
    profile_prefix: 'custom',
    system_prompt: '',
    allowedTools: [],
  },
};

export function createAgentNode({
  role,
  position,
  hasEntryPoint,
  provider,
}: {
  role: StarterRole;
  position: { x: number; y: number };
  hasEntryPoint: boolean;
  provider?: ProviderType;
}): CanvasNode {
  const template = ROLE_TEMPLATES[role];
  const id = crypto.randomUUID();
  const isEntryPoint = role === 'supervisor' && !hasEntryPoint;

  return {
    id,
    type: 'agent',
    position,
    data: {
      profile_name: `${template.profile_prefix}-${id.slice(0, 8)}`,
      display_name: template.display_name,
      role,
      provider,
      system_prompt: template.system_prompt,
      allowedTools: [...template.allowedTools],
      is_entry_point: isEntryPoint,
    },
  };
}

export function roleFromValue(value: string): StarterRole {
  if (value === 'supervisor' || value === 'developer' || value === 'reviewer') {
    return value;
  }
  return 'custom';
}

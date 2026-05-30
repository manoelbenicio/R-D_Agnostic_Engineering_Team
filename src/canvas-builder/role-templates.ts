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
      `You are the Supervisor. You are the strategic coordinator of a multi-agent development team.

## Operating Protocol
1. DECOMPOSE: Break the user's goal into 2-4 concrete tasks with clear acceptance criteria
2. ASSIGN: Delegate each task to the right developer with specific file paths, function signatures, and expected behavior
3. SEQUENCE: If tasks have dependencies, assign them in waves — don't assign dependent work until prerequisites are done
4. VERIFY: After developers report completion, assign the Reviewer to inspect ALL changes holistically
5. SYNTHESIZE: Collect the Reviewer's feedback, decide if rework is needed, and report final status to the user

## Assignment Format
When assigning tasks, always include:
- **Task**: One-sentence description
- **Files**: Specific files to create/modify
- **Acceptance criteria**: How to verify it's done
- **Constraints**: What NOT to change

## Rules
- Never implement code yourself — you coordinate only
- If the goal is ambiguous, ask the user for clarification before assigning
- If a developer reports failure, analyze the error and reassign with adjusted instructions
- Keep the user informed of progress at each phase transition`,
    allowedTools: ['handoff', 'assign', 'send_message'],
  },
  developer: {
    role: 'developer',
    display_name: 'Developer',
    profile_prefix: 'developer',
    system_prompt:
      `You are a Developer agent. You are a high-velocity implementation agent.

## Operating Protocol
1. READ the assignment carefully — understand the exact scope and acceptance criteria
2. EXPLORE the existing codebase first (read_file, grep) to understand conventions and patterns
3. IMPLEMENT the changes using apply_patch or shell commands
4. VERIFY by running tests, linters, or the application itself
5. REPORT results back to the supervisor with: files changed, what was implemented, verification output, and any concerns

## Sub-Agent Scaling
You can spawn sub-agents to parallelize work. Use this when:
- You receive multiple independent tasks that can run concurrently
- A task is large enough to benefit from parallel file edits
- You need a research sub-agent to investigate while you implement
Spawn up to 3 sub-agents for parallel workstreams. Each sub-agent inherits your capabilities.
Coordinate their outputs and merge results before reporting back.

## Rules
- Stay within your assigned scope — don't refactor unrelated code
- Match existing code style and conventions
- Always verify your changes before reporting completion
- Write concise commit-ready code — no TODOs or placeholders`,
    allowedTools: ['shell', 'apply_patch', 'read_file', 'grep', 'test'],
  },
  reviewer: {
    role: 'reviewer',
    display_name: 'Reviewer',
    profile_prefix: 'reviewer',
    system_prompt:
      `You are a Reviewer agent. You are the quality gate for this team.

## Review Protocol
1. CONTEXT: Read the supervisor's original goal and acceptance criteria
2. SCAN: Read every file that was changed or created
3. ANALYZE: Check correctness, completeness, security, integration, tests, and style
4. VERDICT: Report one of:
   - ✅ APPROVED — All criteria met
   - ⚠️ CHANGES REQUESTED — List specific issues with file:line references
   - ❌ REJECTED — Fundamental problems requiring re-implementation

## Sub-Agent Scaling
You can spawn sub-agents to parallelize review work. Use this when:
- Multiple files/components need independent deep review
- You need a research sub-agent to check best practices or security advisories
- End-to-end testing needs to run in parallel with code review
Spawn up to 3 sub-agents for parallel review tracks. Consolidate their findings into your final verdict.

## For each issue found report:
- **Severity**: Critical / Warning / Nit
- **File**: exact file path and line number
- **Issue**: what's wrong
- **Fix**: suggested correction

## Rules
- Be thorough but fair — don't block on style nits if functionality is correct
- Always verify claims by reading the actual code, never trust summaries
- Think step-by-step through complex logic before judging it`,
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

/**
 * 12 high-contrast colors for dark mode, curated for colorblind accessibility.
 * Colors auto-cycle as agents are added. Users can override in the config panel.
 */
export const AGENT_COLOR_PALETTE = [
  '#00b0bd', // cyan (default/supervisor)
  '#f59e0b', // amber
  '#a78bfa', // violet
  '#34d399', // emerald
  '#f472b6', // pink
  '#60a5fa', // blue
  '#fb923c', // orange
  '#4ade80', // green
  '#c084fc', // purple
  '#fbbf24', // yellow
  '#38bdf8', // sky
  '#fb7185', // rose
] as const;

export function createAgentNode({
  role,
  position,
  hasEntryPoint,
  provider,
  nodeIndex = 0,
}: {
  role: StarterRole;
  position: { x: number; y: number };
  hasEntryPoint: boolean;
  provider?: ProviderType;
  nodeIndex?: number;
}): CanvasNode {
  const template = ROLE_TEMPLATES[role];
  const id = crypto.randomUUID();
  const isEntryPoint = role === 'supervisor' && !hasEntryPoint;
  const color = AGENT_COLOR_PALETTE[nodeIndex % AGENT_COLOR_PALETTE.length];

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
      color,
    },
  };
}

export function roleFromValue(value: string): StarterRole {
  if (value === 'supervisor' || value === 'developer' || value === 'reviewer') {
    return value;
  }
  return 'custom';
}

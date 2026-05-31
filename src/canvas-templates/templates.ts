import { CanvasDocument, CanvasEdge, CanvasNode, OrchestrationType, ProviderType } from '@/shared/canvas-types';
// eslint-disable-next-line agentverse/no-sideways-capability-imports
import { SCHEMA_VERSION } from '@/shared/schema-version';

export interface CanvasTemplate {
  id: string;
  name: string;
  description: string;
  agent_count: number;
  primary_edge_type: OrchestrationType | 'none';
  est_cost_per_hour_usd: number;
  document: CanvasDocument;
}

const TEMPLATE_CREATED_AT = '2026-05-27T00:00:00.000Z';
const OPUS_CLASS_COST_PER_AGENT_HOUR = 15;

const TEMPLATES_NODE_IDS: Record<string, number> = {
  'code-review-pipeline': 1,
  'bug-triage': 101,
  'documentation-sprint': 201,
  'full-stack-dev': 301,
  'data-pipeline': 401,
  'security-audit': 501,
  'devops-pipeline': 601,
  'research-team': 701,
  'enterprise-squad': 801,
  'blank-canvas': 901,
};

// ---------------------------------------------------------------------------
// Full Stack Dev Blueprint — custom 5-agent topology with hardened prompts
// Must be defined BEFORE the TEMPLATES array since it's called at module init.
// ---------------------------------------------------------------------------

const BLUEPRINT_PROMPTS = {
  coordinator: `You are the Delivery Coordinator for a multi-agent software team. You own the user's goal end to end. You coordinate specialists; you never write code yourself.

## Why this matters
Reliable delivery depends on clear ownership and verifiable results. You keep one source of truth for scope and status so work does not drift or get lost.

## Operating protocol
1. DECOMPOSE the user's goal into 2-4 concrete tasks, each with explicit acceptance criteria.
2. STRESS-TEST the plan: assign it to the Adversarial Reviewer first and fold its critique in before delegating implementation.
3. ASSIGN each task to the right specialist using the assign tool. Include:
   - Task: one sentence
   - Files: specific paths to create/modify
   - Acceptance criteria: how completion is verified
   - Constraints: what must NOT change
4. PARALLELIZE independent tasks (assign them in the same wave). Sequence only when one task genuinely depends on another's output.
5. VERIFY: after specialists report back, assign the Reviewer to inspect all changes against the original acceptance criteria.
6. SYNTHESIZE the verdicts, decide if rework is needed, and report final status to the user.

## Scope discipline
- Implement EXACTLY and ONLY what the user requested. Do not expand scope.
- If a task surfaces new work, call it out as optional; do not silently add it.

## Ambiguity
- If the goal is ambiguous, ask at most 1-3 precise clarifying questions before assigning. If still unclear, state your best-guess interpretation and proceed.

## Reporting
- Keep user updates to 1-2 sentences per phase transition, each with a concrete outcome.
- If a specialist reports failure, diagnose the cause and reassign with adjusted instructions rather than retrying blindly.

Only the targets listed in your Canvas Topology block are reachable. Delegate work with assign; reserve handoff for transferring final ownership.`,

  backendDev: `You are a Backend Developer agent - a high-velocity, verification-first implementation specialist for server-side and API work.

## Why this matters
Your changes must be correct and self-verified before you report back, because the Coordinator and Reviewers act on your word.

## Operating protocol
1. READ the assignment: understand the exact scope and acceptance criteria.
2. EXPLORE first - use read_file and grep (in parallel where independent) to learn existing conventions, interfaces, and dependencies before editing.
3. IMPLEMENT with apply_patch / shell, matching existing code style.
4. VERIFY by running the relevant tests (test) and the build; do not report success until verification passes.
5. REPORT back with: files changed, what was implemented, exact verification output, and any risks or follow-ups.

## Scope discipline
- Stay strictly within the assigned scope. Do not refactor unrelated code.
- No TODOs or placeholders - ship commit-ready code.

## After any write
Restate: WHAT changed, WHERE (file paths), and the VALIDATION you ran.

## Uncertainty
- If acceptance criteria are unclear, state your assumption explicitly and proceed with the simplest valid interpretation. Never fabricate file paths, line numbers, or test results.

If your API surface changes in a way the frontend depends on, send_message the Frontend Developer with the contract delta.`,

  frontendDev: `You are a Frontend Developer agent - a verification-first implementation specialist for UI and client-side work.

## Why this matters
Your changes must be correct, accessible, and self-verified before you report back, because the Coordinator and Reviewers act on your word.

## Operating protocol
1. READ the assignment: exact scope and acceptance criteria.
2. EXPLORE first - read_file and grep (in parallel where independent) to learn the existing design system, components, and conventions. Reuse them; do not invent new UI primitives, colors, or tokens unless explicitly requested.
3. IMPLEMENT with apply_patch / shell, matching existing patterns.
4. VERIFY by running tests (test) and the build; confirm the UI renders before reporting success.
5. REPORT back with: files changed, what was implemented, verification output, and any risks.

## Scope discipline
- Implement EXACTLY and ONLY what was assigned. No extra features, no UX embellishments, no uncontrolled styling.

## After any write
Restate: WHAT changed, WHERE (file paths), and the VALIDATION you ran.

## Uncertainty
- If a requirement is ambiguous, choose the simplest valid interpretation and state the assumption. Never fabricate results.`,

  reviewerQA: `You are the Reviewer - the quality gate for this team. You inspect and judge; you do not modify code.

## Why this matters
You are the last check before work reaches the user. A missed defect is a shipped defect, so verify claims against the actual code rather than trusting summaries.

## Review protocol
1. CONTEXT: read the Coordinator's original goal and acceptance criteria.
2. SCAN: read every changed or created file (read_file, grep - parallelize across independent files).
3. ANALYZE: correctness, completeness vs. acceptance criteria, security, integration, test coverage, and style.
4. VERIFY: run the test suite (test) to confirm the developers' claims.
5. VERDICT - return exactly one:
   - APPROVED - all criteria met
   - CHANGES REQUESTED - list issues with file:line, severity, and suggested fix
   - REJECTED - fundamental problems requiring re-implementation

## For each issue
Report Severity (Critical / Warning / Nit), File (path:line), Issue, and Fix.

## Discipline
- Be thorough but fair; do not block on nits if functionality is correct.
- Always confirm by reading the actual code and running tests. If something cannot be verified, say so explicitly rather than assuming it passes.`,

  adversarialReviewer: `You are the Adversarial Reviewer ("red team") for this team, with deep SharePoint and Copilot Studio domain expertise. Your job is to challenge plans and changes, not to coordinate or implement.

## Why this matters
A dedicated challenger catches flawed assumptions, missing edge cases, and domain-specific risks before they become shipped defects. You make the team's output more robust by disagreeing well.

## Protocol
1. Read the Coordinator's plan and any proposed changes (read_file, grep).
2. Attack the plan constructively. For each concern report:
   - Risk: what could go wrong
   - Evidence: the specific file/assumption/requirement it stems from
   - Severity: Critical / Warning / Nit
   - Recommended mitigation
3. Apply SharePoint / Copilot Studio domain checks specifically: permissions and tenant scoping, connector/data-source limits, throttling, governance and compliance constraints.
4. Return your critique to the Coordinator via send_message. Do not assign work and do not modify code.

## Discipline
- Be specific and grounded - verify against the actual artifacts, never assume.
- Where you are uncertain, say so explicitly rather than asserting a risk you cannot evidence. Distinguish facts from inference.`,
};

function fullStackDevBlueprint(): CanvasTemplate {
  const id = 'full-stack-dev';
  const uuid = '10000000-0000-4000-8000-000000000004';
  const coordId    = '20000000-0000-4000-8000-000000000301';
  const backendId  = '20000000-0000-4000-8000-000000000302';
  const frontendId = '20000000-0000-4000-8000-000000000303';
  const reviewerId = '20000000-0000-4000-8000-000000000304';
  const adversaryId = '20000000-0000-4000-8000-000000000305';

  const nodes: CanvasNode[] = [
    {
      id: coordId, type: 'agent', position: { x: 120, y: 200 },
      data: {
        profile_name: 'coordinator-fsd-1', display_name: 'Principal AI Solutions Engineering',
        role: 'supervisor', provider: 'kiro_cli' as ProviderType, model: 'opus-4.8',
        system_prompt: BLUEPRINT_PROMPTS.coordinator,
        allowedTools: ['assign', 'handoff', 'send_message'],
        is_entry_point: true, color: '#00b0bd',
      },
    },
    {
      id: backendId, type: 'agent', position: { x: 520, y: 60 },
      data: {
        profile_name: 'backend-dev-fsd-2', display_name: 'Principal Backend Developer',
        role: 'developer', provider: 'codex' as ProviderType, model: 'codex-5.5-high-thinking',
        system_prompt: BLUEPRINT_PROMPTS.backendDev,
        allowedTools: ['shell', 'apply_patch', 'read_file', 'grep', 'test'],
        is_entry_point: false, color: '#34d399',
      },
    },
    {
      id: frontendId, type: 'agent', position: { x: 520, y: 430 },
      data: {
        profile_name: 'frontend-dev-fsd-3', display_name: 'Frontend Developer',
        role: 'developer', provider: 'codex' as ProviderType, model: 'codex-5.5-high-thinking',
        system_prompt: BLUEPRINT_PROMPTS.frontendDev,
        allowedTools: ['shell', 'apply_patch', 'read_file', 'grep', 'test'],
        is_entry_point: false, color: '#f472b6',
      },
    },
    {
      id: reviewerId, type: 'agent', position: { x: 920, y: 200 },
      data: {
        profile_name: 'reviewer-qa-fsd-4', display_name: 'Reviewer QA',
        role: 'reviewer', provider: 'gemini_cli' as ProviderType, model: 'gemini-3.5-flash-high-thinking',
        system_prompt: BLUEPRINT_PROMPTS.reviewerQA,
        allowedTools: ['read_file', 'grep', 'test'],
        is_entry_point: false, color: '#a78bfa',
      },
    },
    {
      id: adversaryId, type: 'agent', position: { x: 120, y: 430 },
      data: {
        profile_name: 'adversary-fsd-5', display_name: 'PA SharePoint Copilot Studio SME adversary',
        role: 'reviewer', provider: 'kiro_cli' as ProviderType, model: 'opus-4.7',
        system_prompt: BLUEPRINT_PROMPTS.adversarialReviewer,
        allowedTools: ['read_file', 'grep', 'send_message'],
        is_entry_point: false, color: '#f59e0b',
      },
    },
  ];

  const edges: CanvasEdge[] = [
    { id: `${id}-edge-1`, source: coordId, target: backendId, type: 'assign', label: 'assign' },
    { id: `${id}-edge-2`, source: coordId, target: frontendId, type: 'assign', label: 'assign' },
    { id: `${id}-edge-3`, source: coordId, target: reviewerId, type: 'assign', label: 'assign' },
    { id: `${id}-edge-4`, source: coordId, target: adversaryId, type: 'assign', label: 'assign' },
    { id: `${id}-edge-5`, source: backendId, target: frontendId, type: 'send_message', label: 'send_message' },
  ];

  return {
    id, name: 'Full Stack Dev',
    description: 'Coordinator + Backend/Frontend developers + QA Reviewer + Adversarial SharePoint SME.',
    agent_count: 5, primary_edge_type: 'assign',
    est_cost_per_hour_usd: 5 * OPUS_CLASS_COST_PER_AGENT_HOUR,
    document: {
      id: uuid, name: 'Full Stack Dev', version: 1,
      created_at: TEMPLATE_CREATED_AT, updated_at: TEMPLATE_CREATED_AT,
      schema_version: SCHEMA_VERSION, nodes, edges,
      config: { working_directory: '~', provider_default: '' as ProviderType },
      deploy_state: { status: 'draft' },
    },
  };
}

export const TEMPLATES: CanvasTemplate[] = [
  template({
    id: 'code-review-pipeline',
    uuid: '10000000-0000-4000-8000-000000000001',
    name: 'Code Review Pipeline',
    description: 'Supervisor routes implementation to a developer and final verification to a reviewer.',
    roles: ['supervisor', 'developer', 'reviewer'],
    edges: [
      [0, 1, 'handoff'],
      [1, 2, 'handoff'],
    ],
  }),
  template({
    id: 'bug-triage',
    uuid: '10000000-0000-4000-8000-000000000002',
    name: 'Bug Triage',
    description: 'Classify a defect, assign a fix, and verify the result before closing.',
    roles: ['supervisor', 'developer', 'reviewer'],
    edges: [
      [0, 1, 'assign'],
      [1, 2, 'handoff'],
    ],
  }),
  template({
    id: 'documentation-sprint',
    uuid: '10000000-0000-4000-8000-000000000003',
    name: 'Documentation Sprint',
    description: 'Draft, review, and polish docs for a bounded product or engineering area.',
    roles: ['supervisor', 'developer', 'reviewer'],
    edges: [
      [0, 1, 'handoff'],
      [1, 2, 'send_message'],
    ],
  }),
  fullStackDevBlueprint(),
  template({
    id: 'data-pipeline',
    uuid: '10000000-0000-4000-8000-000000000005',
    name: 'Data Pipeline',
    description: 'Split ingestion, transformation, and validation work across specialized agents.',
    roles: ['supervisor', 'developer', 'developer', 'reviewer'],
    labels: ['Supervisor', 'Ingestion Developer', 'Transform Developer', 'Data Reviewer'],
    edges: [
      [0, 1, 'assign'],
      [1, 2, 'handoff'],
      [2, 3, 'handoff'],
    ],
  }),
  template({
    id: 'security-audit',
    uuid: '10000000-0000-4000-8000-000000000006',
    name: 'Security Audit',
    description: 'Review architecture, inspect code, and produce actionable security findings.',
    roles: ['supervisor', 'reviewer', 'developer', 'reviewer'],
    labels: ['Supervisor', 'Threat Reviewer', 'Fix Developer', 'Final Reviewer'],
    edges: [
      [0, 1, 'assign'],
      [1, 2, 'handoff'],
      [2, 3, 'handoff'],
    ],
  }),
  template({
    id: 'devops-pipeline',
    uuid: '10000000-0000-4000-8000-000000000007',
    name: 'DevOps Pipeline',
    description: 'Plan deployment, implement automation, and review operational readiness.',
    roles: ['supervisor', 'developer', 'reviewer'],
    labels: ['Supervisor', 'DevOps Developer', 'Ops Reviewer'],
    edges: [
      [0, 1, 'assign'],
      [1, 2, 'handoff'],
    ],
  }),
  template({
    id: 'research-team',
    uuid: '10000000-0000-4000-8000-000000000008',
    name: 'Research Team',
    description: 'Coordinate research, synthesis, and review for exploratory work.',
    roles: ['supervisor', 'custom', 'reviewer'],
    labels: ['Supervisor', 'Research Agent', 'Synthesis Reviewer'],
    edges: [
      [0, 1, 'send_message'],
      [1, 2, 'send_message'],
    ],
  }),
  template({
    id: 'enterprise-squad',
    uuid: '10000000-0000-4000-8000-000000000009',
    name: 'Enterprise Squad',
    description: 'A larger squad for implementation, platform, security, and review tracks.',
    roles: ['supervisor', 'developer', 'developer', 'developer', 'reviewer', 'reviewer'],
    labels: ['Supervisor', 'Frontend Developer', 'Backend Developer', 'Platform Developer', 'Security Reviewer', 'QA Reviewer'],
    edges: [
      [0, 1, 'assign'],
      [0, 2, 'assign'],
      [0, 3, 'assign'],
      [1, 5, 'handoff'],
      [2, 5, 'handoff'],
      [3, 4, 'handoff'],
      [4, 5, 'send_message'],
    ],
  }),
  template({
    id: 'blank-canvas',
    uuid: '10000000-0000-4000-8000-000000000010',
    name: 'Blank Canvas',
    description: 'Start from an empty draft and build the team by hand.',
    roles: [],
    edges: [],
  }),
];

export function instantiateTemplate(templateId: string): CanvasDocument {
  const selected = TEMPLATES.find((candidate) => candidate.id === templateId);
  if (!selected) {
    throw new Error(`Unknown canvas template: ${templateId}`);
  }

  const now = new Date().toISOString();
  const nodeIdMap = new Map<string, string>();
  const nodes = selected.document.nodes.map((node) => {
    const id = crypto.randomUUID();
    nodeIdMap.set(node.id, id);
    return {
      ...node,
      id,
      data: {
        ...node.data,
        profile_name: `${node.data.role}-${id.slice(0, 8)}`,
      },
    };
  });

  const edges = selected.document.edges.map((edge) => ({
    ...edge,
    id: crypto.randomUUID(),
    source: nodeIdMap.get(edge.source) ?? edge.source,
    target: nodeIdMap.get(edge.target) ?? edge.target,
  }));

  return {
    ...selected.document,
    id: crypto.randomUUID(),
    name: `${selected.name} (copy)`,
    version: 1,
    created_at: now,
    updated_at: now,
    nodes,
    edges,
    deploy_state: { status: 'draft' },
  };
}

export function formatTemplateCost(value: number): string {
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
    maximumFractionDigits: 2,
  }).format(value);
}

type RoleName = 'supervisor' | 'developer' | 'reviewer' | 'custom';
type EdgeTuple = [sourceIndex: number, targetIndex: number, type: OrchestrationType];

function template({
  id,
  uuid,
  name,
  description,
  roles,
  labels,
  edges,
}: {
  id: string;
  uuid: string;
  name: string;
  description: string;
  roles: RoleName[];
  labels?: string[];
  edges: EdgeTuple[];
}): CanvasTemplate {
  const nodes = roles.map((role, index) =>
    templateNode({
      templateId: id,
      index,
      role,
      label: labels?.[index] ?? defaultDisplayName(role),
    })
  );
  const canvasEdges = edges.map(([sourceIndex, targetIndex, type], index) =>
    templateEdge({
      templateId: id,
      index,
      source: nodes[sourceIndex]?.id ?? '',
      target: nodes[targetIndex]?.id ?? '',
      type,
    })
  );
  const primaryEdgeType = mostCommonEdgeType(canvasEdges);

  return {
    id,
    name,
    description,
    agent_count: nodes.length,
    primary_edge_type: primaryEdgeType,
    est_cost_per_hour_usd: nodes.length * OPUS_CLASS_COST_PER_AGENT_HOUR,
    document: {
      id: uuid,
      name,
      version: 1,
      created_at: TEMPLATE_CREATED_AT,
      updated_at: TEMPLATE_CREATED_AT,
      schema_version: SCHEMA_VERSION,
      nodes,
      edges: canvasEdges,
      config: {
        working_directory: '~',
        provider_default: '',
      },
      deploy_state: { status: 'draft' },
    },
  };
}

function templateNode({
  templateId,
  index,
  role,
  label,
}: {
  templateId: string;
  index: number;
  role: RoleName;
  label: string;
}): CanvasNode {
  const column = index === 0 ? 0 : ((index - 1) % 3) + 1;
  const row = index === 0 ? 1 : Math.floor((index - 1) / 3);
  const nodeUuid = `20000000-0000-4000-8000-${String((TEMPLATES_NODE_IDS[templateId] ?? 1) + index).padStart(12, '0')}`;

  return {
    id: nodeUuid,
    type: 'agent',
    position: { x: 120 + column * 280, y: 80 + row * 170 },
    data: {
      profile_name: `${role}-${templateId}-${index + 1}`,
      display_name: label,
      role,
      provider: undefined,
      model: '',
      system_prompt: systemPromptForRole(role, label),
      allowedTools: allowedToolsForRole(role),
      is_entry_point: index === 0,
    },
  };
}

function templateEdge({
  templateId,
  index,
  source,
  target,
  type,
}: {
  templateId: string;
  index: number;
  source: string;
  target: string;
  type: OrchestrationType;
}): CanvasEdge {
  return {
    id: `${templateId}-edge-${index + 1}`,
    source,
    target,
    type,
    label: type,
  };
}

function defaultDisplayName(role: RoleName): string {
  if (role === 'custom') return 'Custom Agent';
  return `${role.charAt(0).toUpperCase()}${role.slice(1)}`;
}

function systemPromptForRole(role: RoleName, label: string): string {
  if (role === 'supervisor') {
    return `You are ${label}. Coordinate the canvas team, delegate work, and keep outcomes aligned with the user's goal.`;
  }
  if (role === 'reviewer') {
    return `You are ${label}. Review outputs carefully, identify risks, and request concrete fixes when needed.`;
  }
  if (role === 'developer') {
    return `You are ${label}. Execute assigned implementation work and report verified results.`;
  }
  return `You are ${label}. Follow the supervisor's instructions and communicate concise progress.`;
}

function allowedToolsForRole(role: RoleName): string[] {
  if (role === 'supervisor') return ['handoff', 'assign', 'send_message'];
  if (role === 'reviewer') return ['read_file', 'grep', 'test'];
  if (role === 'developer') return ['shell', 'apply_patch', 'read_file'];
  return [];
}


function mostCommonEdgeType(edges: CanvasEdge[]): OrchestrationType | 'none' {
  if (edges.length === 0) return 'none';
  const counts = edges.reduce<Record<OrchestrationType, number>>(
    (acc, edge) => ({ ...acc, [edge.type]: acc[edge.type] + 1 }),
    { handoff: 0, assign: 0, send_message: 0 }
  );
  return (Object.entries(counts).sort((a, b) => b[1] - a[1])[0]?.[0] ?? 'handoff') as OrchestrationType;
}

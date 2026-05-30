import { CanvasDocument, CanvasEdge, CanvasNode, OrchestrationType } from '@/shared/canvas-types';
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
  template({
    id: 'full-stack-dev',
    uuid: '10000000-0000-4000-8000-000000000004',
    name: 'Full Stack Dev',
    description: 'Coordinate frontend, backend, and review work for a vertical feature.',
    roles: ['supervisor', 'developer', 'developer', 'reviewer'],
    labels: ['Supervisor', 'Frontend Developer', 'Backend Developer', 'Reviewer'],
    edges: [
      [0, 1, 'assign'],
      [0, 2, 'assign'],
      [1, 3, 'handoff'],
      [2, 3, 'handoff'],
    ],
  }),
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

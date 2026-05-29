import { z } from 'zod';
import { CanvasDocument } from '@/shared/canvas-types';

export class CanvasParseError extends Error {
  paths: string[];

  constructor(zodError: z.ZodError) {
    const paths = zodError.issues.map((issue) => issue.path.join('.'));
    super(`Canvas parsing failed. Invalid paths: ${paths.join(', ')}`);
    this.name = 'CanvasParseError';
    this.paths = paths;
    Object.setPrototypeOf(this, CanvasParseError.prototype);
  }
}

const providerTypeSchema = z.string();

const orchestrationTypeSchema = z.enum(['handoff', 'assign', 'send_message']);

const canvasNodeSchema = z.object({
  id: z.string().uuid(),
  type: z.literal('agent'),
  position: z.object({
    x: z.number(),
    y: z.number(),
  }),
  data: z.object({
    profile_name: z.string(),
    display_name: z.string(),
    role: z.string(),
    provider: providerTypeSchema.optional(),
    model: z.string().optional(),
    system_prompt: z.string(),
    allowedTools: z.array(z.string()).optional(),
    is_entry_point: z.boolean(),
  }),
});

const canvasEdgeSchema = z.object({
  id: z.string(),
  source: z.string(),
  target: z.string(),
  type: orchestrationTypeSchema,
  label: z.string().optional(),
});

export const canvasDocumentSchema: z.ZodType<CanvasDocument> = z.object({
  id: z.string().uuid(),
  name: z.string(),
  version: z.number().int().nonnegative(),
  created_at: z.string(),
  updated_at: z.string(),
  schema_version: z.number().int().nonnegative(),
  nodes: z.array(canvasNodeSchema),
  edges: z.array(canvasEdgeSchema),
  config: z.object({
    working_directory: z.string(),
    session_name: z.string().optional(),
    provider_default: providerTypeSchema,
    env_vars: z.record(z.string(), z.string()).optional(),
  }),
  deploy_state: z.object({
    status: z.enum(['draft', 'deploying', 'deployed', 'degraded']),
    session_name: z.string().optional(),
    terminal_map: z.record(z.string(), z.string()).optional(),
    last_deployed: z.string().optional(),
    errors: z.array(
      z.object({
        node_id: z.string(),
        error: z.string(),
      })
    ).optional(),
    profile_snapshots: z.record(
      z.string(),
      z.object({
        system_prompt: z.string(),
        allowedTools: z.array(z.string()),
        model: z.string(),
        provider: z.string(),
      })
    ).optional(),
    edge_change_advisory: z.boolean().optional(),
  }),
}).superRefine((data, ctx) => {
  // Check for self loops
  data.edges.forEach((edge, index) => {
    if (edge.source === edge.target) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        message: 'Self loops are not allowed',
        path: ['edges', index],
      });
    }
  });

  // Check for dangling edges
  const nodeIds = new Set(data.nodes.map(n => n.id));
  data.edges.forEach((edge, index) => {
    if (!nodeIds.has(edge.source)) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        message: `Edge source ${edge.source} does not reference a valid node`,
        path: ['edges', index, 'source'],
      });
    }
    if (!nodeIds.has(edge.target)) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        message: `Edge target ${edge.target} does not reference a valid node`,
        path: ['edges', index, 'target'],
      });
    }
  });
});

export function parseCanvasDocument(input: unknown): CanvasDocument {
  const result = canvasDocumentSchema.safeParse(input);
  if (!result.success) {
    throw new CanvasParseError(result.error);
  }
  return result.data;
}

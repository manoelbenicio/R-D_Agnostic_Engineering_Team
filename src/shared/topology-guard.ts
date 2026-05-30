/**
 * src/shared/topology-guard.ts
 *
 * Runtime topology guard for the `validation-proxy` change (R3 defense).
 *
 * v1 only mitigates R3 at the prompt level: the supervisor's system prompt
 * lists allowed `handoff` / `assign` / `send_message` targets (see
 * `docs/canvas-topology-prompt.md`). That reduces risk but cannot *prevent*
 * the model from emitting a communication call to an agent outside the
 * declared canvas edges.
 *
 * This module turns the deployed `CanvasDocument` into a queryable topology
 * and validates each orchestration call against it. A call is allowed only
 * when an edge of the matching `OrchestrationType` exists from source to
 * target. Everything else is blocked with an explicit, auditable reason.
 *
 * Pure + dependency-free (only `@/shared` types) so it can run SPA-side
 * today and be lifted into a CAO-side proxy later without changes.
 */

import type {
  CanvasDocument,
  CanvasNode,
  OrchestrationType,
} from './canvas-types';

/** A single allowed communication edge, keyed by both endpoints' identities. */
export interface TopologyEdge {
  type: OrchestrationType;
  source: string;
  target: string;
}

/**
 * Compiled, queryable view of a canvas's communication topology.
 *
 * `aliases` maps every identity a runtime call might use (node id,
 * `profile_name`, generated `<profile_name>_<node_id_with_underscores>`, and
 * `display_name`) back to the canonical node id, so the guard tolerates the
 * different identifiers CAO and the supervisor prompt use interchangeably.
 */
export interface CanvasTopology {
  canvasId: string;
  nodeIds: Set<string>;
  aliases: Map<string, string>;
  /** Set of `${type}:${sourceId}->${targetId}` for O(1) lookup. */
  allowed: Set<string>;
  edges: TopologyEdge[];
}

export type TopologyViolationCode =
  | 'unknown-source'
  | 'unknown-target'
  | 'edge-not-allowed';

export interface OrchestrationCall {
  action: OrchestrationType;
  /** Source agent identity (node id, profile name, or display name). */
  source: string;
  /** Target agent identity (node id, profile name, or display name). */
  target: string;
}

export interface TopologyValidationResult {
  ok: boolean;
  reason?: string;
  code?: TopologyViolationCode;
}

/** Generated profile name convention shared with the reconciler. */
function generatedProfileName(node: CanvasNode): string {
  return `${node.data.profile_name}_${node.id.replace(/-/g, '_')}`;
}

function edgeKey(type: OrchestrationType, source: string, target: string): string {
  return `${type}:${source}->${target}`;
}

/**
 * Compile a `CanvasDocument` into a `CanvasTopology`. Only the three v1
 * control edges (`handoff`, `assign`, `send_message`) are considered; other
 * edge types (e.g. future `data-flow`) are ignored.
 */
export function buildCanvasTopology(canvas: CanvasDocument): CanvasTopology {
  const nodeIds = new Set<string>();
  const aliases = new Map<string, string>();

  for (const node of canvas.nodes) {
    nodeIds.add(node.id);
    // Every identity a call might reference resolves back to the node id.
    aliases.set(node.id, node.id);
    if (node.data.profile_name) aliases.set(node.data.profile_name, node.id);
    if (node.data.display_name) aliases.set(node.data.display_name, node.id);
    aliases.set(generatedProfileName(node), node.id);
  }

  const allowed = new Set<string>();
  const edges: TopologyEdge[] = [];
  for (const edge of canvas.edges) {
    if (!nodeIds.has(edge.source) || !nodeIds.has(edge.target)) continue;
    allowed.add(edgeKey(edge.type, edge.source, edge.target));
    edges.push({ type: edge.type, source: edge.source, target: edge.target });
  }

  return { canvasId: canvas.id, nodeIds, aliases, allowed, edges };
}

/**
 * Validate an orchestration call against a compiled topology.
 *
 * Resolves source/target through the alias table, then checks that an edge
 * of the requested `action` exists between them. Returns an explicit,
 * auditable reason on every rejection.
 */
export function validateOrchestrationCall(
  topology: CanvasTopology,
  call: OrchestrationCall,
): TopologyValidationResult {
  const source = topology.aliases.get(call.source);
  if (!source) {
    return {
      ok: false,
      code: 'unknown-source',
      reason: `Source agent "${call.source}" is not a node in canvas ${topology.canvasId}.`,
    };
  }

  const target = topology.aliases.get(call.target);
  if (!target) {
    return {
      ok: false,
      code: 'unknown-target',
      reason: `Target agent "${call.target}" is not a node in canvas ${topology.canvasId}.`,
    };
  }

  if (!topology.allowed.has(edgeKey(call.action, source, target))) {
    return {
      ok: false,
      code: 'edge-not-allowed',
      reason: `No "${call.action}" edge from "${call.source}" to "${call.target}" exists in canvas ${topology.canvasId}. Add the edge in the Canvas Builder or correct the call.`,
    };
  }

  return { ok: true };
}

/** Convenience: compile + validate in one call (e.g. for one-off checks). */
export function validateAgainstCanvas(
  canvas: CanvasDocument,
  call: OrchestrationCall,
): TopologyValidationResult {
  return validateOrchestrationCall(buildCanvasTopology(canvas), call);
}

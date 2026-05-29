import { CanvasNode, CanvasEdge } from '@/shared/canvas-types';

export function selfLoopValidator(edges: CanvasEdge[]): { ok: true } | { ok: false; offenders: string[] } {
  const offenders: string[] = [];
  for (const edge of edges) {
    if (edge.source === edge.target) {
      offenders.push(edge.id);
    }
  }
  if (offenders.length > 0) {
    return { ok: false, offenders };
  }
  return { ok: true };
}

export function danglingEdgeValidator(
  nodes: CanvasNode[],
  edges: CanvasEdge[]
): { ok: true } | { ok: false; offenders: string[] } {
  const nodeIds = new Set(nodes.map((n) => n.id));
  const offenders: string[] = [];
  for (const edge of edges) {
    if (!nodeIds.has(edge.source) || !nodeIds.has(edge.target)) {
      offenders.push(edge.id);
    }
  }
  if (offenders.length > 0) {
    return { ok: false, offenders };
  }
  return { ok: true };
}

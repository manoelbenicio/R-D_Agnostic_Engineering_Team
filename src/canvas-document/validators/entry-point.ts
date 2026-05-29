import { CanvasDocument } from '@/shared/canvas-types';

export function validateEntryPointForSave(doc: CanvasDocument): { ok: true } | { ok: false; offenders: string[] } {
  const entryPoints = doc.nodes.filter(node => node.data.is_entry_point);
  if (entryPoints.length > 1) {
    return {
      ok: false,
      offenders: entryPoints.map(node => node.id),
    };
  }
  return { ok: true };
}

export function validateEntryPointForDeploy(doc: CanvasDocument):
  | { ok: true }
  | { ok: false; error: 'no entry point'; offenders: string[] }
  | { ok: false; error: 'multiple entry points'; offenders: string[] } {
  const entryPoints = doc.nodes.filter(node => node.data.is_entry_point);
  if (entryPoints.length === 0) {
    return {
      ok: false,
      error: 'no entry point',
      offenders: [],
    };
  }
  if (entryPoints.length > 1) {
    return {
      ok: false,
      error: 'multiple entry points',
      offenders: entryPoints.map(node => node.id),
    };
  }
  return { ok: true };
}

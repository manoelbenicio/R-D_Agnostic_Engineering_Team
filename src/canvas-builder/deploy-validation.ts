import { ProviderType } from '@/api/key-store/registry';
import { CanvasDocument } from '@/shared/canvas-types';
import { CanvasProviderOption } from './provider-options';

export interface DeployValidationResult {
  ok: boolean;
  reason?: string;
}

export function validateCanvasForDeploy(
  doc: CanvasDocument,
  providerOptions: CanvasProviderOption[],
  validatedProviders: ProviderType[]
): DeployValidationResult {
  if (doc.nodes.length === 0) {
    return { ok: false, reason: 'Canvas is empty - add at least one agent block' };
  }

  const entryPoints = doc.nodes.filter((node) => node.data.is_entry_point);
  if (entryPoints.length === 0) {
    return { ok: false, reason: 'No entry-point supervisor - promote one node to entry point' };
  }
  if (entryPoints.length > 1) {
    return {
      ok: false,
      reason: `Multiple entry points - fix nodes ${entryPoints.map((node) => node.id).join(', ')}`,
    };
  }

  for (const node of doc.nodes) {
    if (!node.data.provider) {
      return {
        ok: false,
        reason: `Node ${node.id} has no provider configured`,
      };
    }

    const option = providerOptions.find((candidate) => candidate.provider === node.data.provider);
    if (!option || !validatedProviders.includes(option.sourceProvider)) {
      return {
        ok: false,
        reason: `Node ${node.id} references unconfigured provider ${node.data.provider}`,
      };
    }

    if (!node.data.model) {
      return {
        ok: false,
        reason: `Node ${node.id}: Pick a model for this node`,
      };
    }
  }

  return { ok: true };
}

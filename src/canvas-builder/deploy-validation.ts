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
  validatedProviders: ProviderType[],
  /**
   * Canvas providers whose CLI is installed (and OAuth-authenticated) on the
   * CAO server — from `listProviders()` where `installed === true`. A node is
   * deployable if its CLI is installed OR a BYOK key is validated; BYOK is
   * optional because CLIs like codex/kiro authenticate via their own OAuth
   * session inside the runtime, not via an AgentVerse-held API key.
   */
  installedCliProviders: ReadonlyArray<string> = []
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

  const installed = new Set(installedCliProviders);

  for (const node of doc.nodes) {
    if (!node.data.provider) {
      return {
        ok: false,
        reason: `Node ${node.id} has no provider configured`,
      };
    }

    // Path 1: the CLI for this canvas provider is installed/authenticated on CAO (OAuth).
    const cliInstalled = installed.has(node.data.provider);

    // Path 2: a BYOK key for the underlying source provider is validated.
    const option = providerOptions.find((candidate) => candidate.provider === node.data.provider);
    const byokValidated = !!option && validatedProviders.includes(option.sourceProvider);

    if (!cliInstalled && !byokValidated) {
      return {
        ok: false,
        reason: `Node ${node.id}: provider ${node.data.provider} is neither installed on the runtime (OAuth) nor configured with an API key`,
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

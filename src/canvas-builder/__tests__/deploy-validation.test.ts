import { describe, it, expect } from 'vitest';
import { validateCanvasForDeploy } from '../deploy-validation';
import { getCanvasProviderOptions } from '../provider-options';
import type { CanvasDocument } from '@/shared/canvas-types';

function canvasWith(provider: string, model = 'm'): CanvasDocument {
  return {
    id: 'c1',
    name: 'C',
    version: 1,
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
    schema_version: 1,
    nodes: [
      {
        id: 'n1',
        type: 'agent',
        position: { x: 0, y: 0 },
        data: {
          profile_name: 'supervisor',
          display_name: 'Sup',
          role: 'supervisor',
          provider: provider as CanvasDocument['nodes'][number]['data']['provider'],
          model,
          system_prompt: '',
          is_entry_point: true,
        },
      },
    ],
    edges: [],
    config: { working_directory: '/tmp', provider_default: 'anthropic' },
    deploy_state: { status: 'draft' },
  };
}

describe('validateCanvasForDeploy — OAuth/CLI path', () => {
  it('allows deploy when the CLI is installed on GO Core, with NO BYOK key', () => {
    const doc = canvasWith('codex');
    const res = validateCanvasForDeploy(doc, getCanvasProviderOptions([]), [], ['codex']);
    expect(res.ok).toBe(true);
  });

  it('still allows deploy via BYOK when no CLI is installed', () => {
    const doc = canvasWith('codex');
    // openai validated → provider option "codex" maps to sourceProvider openai
    const res = validateCanvasForDeploy(doc, getCanvasProviderOptions(['openai']), ['openai'], []);
    expect(res.ok).toBe(true);
  });

  it('blocks when neither CLI installed nor BYOK configured', () => {
    const doc = canvasWith('codex');
    const res = validateCanvasForDeploy(doc, getCanvasProviderOptions([]), [], []);
    expect(res.ok).toBe(false);
    expect(res.reason).toContain('neither installed');
  });

  it('blocks when CLI installed but model missing', () => {
    const doc = canvasWith('kiro_cli', '');
    const res = validateCanvasForDeploy(doc, getCanvasProviderOptions([]), [], ['kiro_cli']);
    expect(res.ok).toBe(false);
    expect(res.reason).toContain('Pick a model');
  });
});

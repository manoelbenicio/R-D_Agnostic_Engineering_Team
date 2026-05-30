import { describe, it, expect } from 'vitest';
import type { CanvasDocument } from '../canvas-types';
import {
  buildCanvasTopology,
  validateOrchestrationCall,
  validateAgainstCanvas,
} from '../topology-guard';

const SUP = '00000000-0000-4000-8000-000000000001';
const DEV = '00000000-0000-4000-8000-000000000002';
const REV = '00000000-0000-4000-8000-000000000003';

function makeCanvas(): CanvasDocument {
  return {
    id: 'canvas-1',
    name: 'Test',
    version: 1,
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
    schema_version: 1,
    nodes: [
      node(SUP, 'supervisor', 'Supervisor Node', true),
      node(DEV, 'developer', 'Developer Node', false),
      node(REV, 'reviewer', 'Reviewer Node', false),
    ],
    edges: [
      { id: 'e1', source: SUP, target: DEV, type: 'handoff' },
      { id: 'e2', source: SUP, target: REV, type: 'send_message' },
    ],
    config: { working_directory: '/tmp', provider_default: 'anthropic' },
    deploy_state: { status: 'deployed' },
  };
}

function node(id: string, profile: string, display: string, entry: boolean): CanvasDocument['nodes'][number] {
  return {
    id,
    type: 'agent',
    position: { x: 0, y: 0 },
    data: {
      profile_name: profile,
      display_name: display,
      role: profile,
      provider: 'anthropic',
      model: 'claude',
      system_prompt: '',
      is_entry_point: entry,
    },
  };
}

describe('topology-guard', () => {
  it('allows a call along a declared edge (by node id)', () => {
    const t = buildCanvasTopology(makeCanvas());
    expect(validateOrchestrationCall(t, { action: 'handoff', source: SUP, target: DEV }).ok).toBe(true);
  });

  it('resolves identities by profile_name and display_name', () => {
    const t = buildCanvasTopology(makeCanvas());
    expect(
      validateOrchestrationCall(t, { action: 'send_message', source: 'supervisor', target: 'Reviewer Node' }).ok,
    ).toBe(true);
  });

  it('resolves the generated <profile>_<id> alias', () => {
    const t = buildCanvasTopology(makeCanvas());
    const genTarget = `developer_${DEV.replace(/-/g, '_')}`;
    expect(validateOrchestrationCall(t, { action: 'handoff', source: SUP, target: genTarget }).ok).toBe(true);
  });

  it('blocks an edge that exists but with the wrong action type', () => {
    const t = buildCanvasTopology(makeCanvas());
    const res = validateOrchestrationCall(t, { action: 'assign', source: SUP, target: DEV });
    expect(res.ok).toBe(false);
    expect(res.code).toBe('edge-not-allowed');
  });

  it('blocks a call to a node with no edge from the source', () => {
    const t = buildCanvasTopology(makeCanvas());
    const res = validateOrchestrationCall(t, { action: 'handoff', source: DEV, target: REV });
    expect(res.ok).toBe(false);
    expect(res.code).toBe('edge-not-allowed');
  });

  it('blocks an unknown source', () => {
    const t = buildCanvasTopology(makeCanvas());
    const res = validateOrchestrationCall(t, { action: 'handoff', source: 'ghost', target: DEV });
    expect(res.ok).toBe(false);
    expect(res.code).toBe('unknown-source');
  });

  it('blocks an unknown target', () => {
    const t = buildCanvasTopology(makeCanvas());
    const res = validateOrchestrationCall(t, { action: 'handoff', source: SUP, target: 'ghost' });
    expect(res.ok).toBe(false);
    expect(res.code).toBe('unknown-target');
  });

  it('ignores edges referencing missing nodes when compiling', () => {
    const canvas = makeCanvas();
    canvas.edges.push({ id: 'e3', source: SUP, target: 'missing', type: 'handoff' });
    const t = buildCanvasTopology(canvas);
    expect(t.edges).toHaveLength(2);
  });

  it('validateAgainstCanvas compiles + checks in one call', () => {
    expect(validateAgainstCanvas(makeCanvas(), { action: 'handoff', source: SUP, target: DEV }).ok).toBe(true);
  });

  it('emits an auditable reason string on rejection', () => {
    const res = validateAgainstCanvas(makeCanvas(), { action: 'assign', source: SUP, target: REV });
    expect(res.ok).toBe(false);
    expect(res.reason).toContain('assign');
    expect(res.reason).toContain('canvas-1');
  });
});

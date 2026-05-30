import { describe, it, expect, vi } from 'vitest';
import type { CanvasDocument } from '../canvas-types';
import { createValidationProxy, type ViolationRecord } from '../validation-proxy';

const SUP = '00000000-0000-4000-8000-000000000001';
const DEV = '00000000-0000-4000-8000-000000000002';
const REV = '00000000-0000-4000-8000-000000000003';

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
    deploy_state: {
      status: 'deployed',
      session_name: 'sess-1',
      terminal_map: { [SUP]: 'term-sup', [DEV]: 'term-dev', [REV]: 'term-rev' },
    },
  };
}

describe('validation-proxy', () => {
  it('allows a call along a declared edge', () => {
    const proxy = createValidationProxy(makeCanvas());
    expect(proxy.guardOrchestration({ action: 'handoff', source: SUP, target: DEV }).allowed).toBe(true);
  });

  it('denies a call with no edge between source and target', () => {
    const proxy = createValidationProxy(makeCanvas());
    const d = proxy.guardOrchestration({ action: 'handoff', source: DEV, target: REV });
    expect(d.allowed).toBe(false);
    expect(d.code).toBe('edge-not-allowed');
    expect(d.reason).toContain('handoff');
  });

  it('denies an existing edge invoked with the wrong action type', () => {
    const proxy = createValidationProxy(makeCanvas());
    const d = proxy.guardOrchestration({ action: 'assign', source: SUP, target: DEV });
    expect(d.allowed).toBe(false);
    expect(d.code).toBe('edge-not-allowed');
  });

  it('resolves terminal ids back to node ids via terminal_map', () => {
    const proxy = createValidationProxy(makeCanvas());
    const d = proxy.guardOrchestration({ action: 'send_message', source: 'term-sup', target: 'term-rev' });
    expect(d.allowed).toBe(true);
  });

  it('fires onViolation with an auditable record on denial', () => {
    const records: ViolationRecord[] = [];
    const onViolation = vi.fn((r: ViolationRecord) => records.push(r));
    const proxy = createValidationProxy(makeCanvas(), { onViolation, now: () => 'T0' });

    const d = proxy.guardOrchestration({ action: 'handoff', source: 'term-sup', target: 'term-rev' });

    expect(d.allowed).toBe(false);
    expect(onViolation).toHaveBeenCalledTimes(1);
    expect(records[0]).toMatchObject({
      timestamp: 'T0',
      canvasId: 'canvas-1',
      code: 'edge-not-allowed',
      call: { source: 'term-sup', target: 'term-rev' },
      resolved: { source: SUP, target: REV },
    });
  });

  it('does not fire onViolation when the call is allowed', () => {
    const onViolation = vi.fn();
    const proxy = createValidationProxy(makeCanvas(), { onViolation });
    proxy.guardOrchestration({ action: 'handoff', source: SUP, target: DEV });
    expect(onViolation).not.toHaveBeenCalled();
  });
});

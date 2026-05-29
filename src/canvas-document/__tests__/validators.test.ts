import { describe, expect, it } from 'vitest';
import {
  selfLoopValidator,
  danglingEdgeValidator,
  validateEntryPointForSave,
  validateEntryPointForDeploy,
} from '../validators';
import { CanvasNode, CanvasEdge, CanvasDocument } from '@/shared/canvas-types';

describe('Canvas Validators', () => {
  const node1: CanvasNode = {
    id: 'node-1',
    type: 'agent',
    position: { x: 0, y: 0 },
    data: { profile_name: 'p1', display_name: 'n1', role: 'developer', system_prompt: '', is_entry_point: false },
  };

  const node2: CanvasNode = {
    id: 'node-2',
    type: 'agent',
    position: { x: 0, y: 0 },
    data: { profile_name: 'p2', display_name: 'n2', role: 'developer', system_prompt: '', is_entry_point: false },
  };

  describe('selfLoopValidator', () => {
    it('passes when there are no self loops', () => {
      const edges: CanvasEdge[] = [
        { id: 'edge-1', source: 'node-1', target: 'node-2', type: 'handoff' },
      ];
      expect(selfLoopValidator(edges)).toEqual({ ok: true });
    });

    it('rejects edges that are self loops', () => {
      const edges: CanvasEdge[] = [
        { id: 'edge-1', source: 'node-1', target: 'node-1', type: 'handoff' },
        { id: 'edge-2', source: 'node-1', target: 'node-2', type: 'assign' },
      ];
      expect(selfLoopValidator(edges)).toEqual({
        ok: false,
        offenders: ['edge-1'],
      });
    });
  });

  describe('danglingEdgeValidator', () => {
    it('passes when all edges point to existing nodes', () => {
      const nodes = [node1, node2];
      const edges: CanvasEdge[] = [
        { id: 'edge-1', source: 'node-1', target: 'node-2', type: 'handoff' },
      ];
      expect(danglingEdgeValidator(nodes, edges)).toEqual({ ok: true });
    });

    it('rejects edges referencing missing nodes', () => {
      const nodes = [node1];
      const edges: CanvasEdge[] = [
        { id: 'edge-1', source: 'node-1', target: 'node-2', type: 'handoff' },
      ];
      expect(danglingEdgeValidator(nodes, edges)).toEqual({
        ok: false,
        offenders: ['edge-1'],
      });
    });
  });

  describe('entry point validators', () => {
    const createDoc = (nodeData: Partial<CanvasNode['data']>[]): CanvasDocument => ({
      id: 'doc-id',
      name: 'doc',
      version: 1,
      created_at: '',
      updated_at: '',
      schema_version: 1,
      nodes: nodeData.map((data, index) => ({
        id: `node-${index}`,
        type: 'agent',
        position: { x: 0, y: 0 },
        data: {
          profile_name: 'p',
          display_name: 'n',
          role: 'developer',
          system_prompt: '',
          is_entry_point: false,
          ...data,
        },
      })),
      edges: [],
      config: { working_directory: '', provider_default: 'openai' },
      deploy_state: { status: 'draft' },
    });

    describe('validateEntryPointForSave', () => {
      it('allows zero entry points', () => {
        const doc = createDoc([{ is_entry_point: false }, { is_entry_point: false }]);
        expect(validateEntryPointForSave(doc)).toEqual({ ok: true });
      });

      it('allows exactly one entry point', () => {
        const doc = createDoc([{ is_entry_point: true }, { is_entry_point: false }]);
        expect(validateEntryPointForSave(doc)).toEqual({ ok: true });
      });

      it('rejects multiple entry points', () => {
        const doc = createDoc([{ is_entry_point: true }, { is_entry_point: true }]);
        expect(validateEntryPointForSave(doc)).toEqual({
          ok: false,
          offenders: ['node-0', 'node-1'],
        });
      });
    });

    describe('validateEntryPointForDeploy', () => {
      it('rejects zero entry points', () => {
        const doc = createDoc([{ is_entry_point: false }, { is_entry_point: false }]);
        expect(validateEntryPointForDeploy(doc)).toEqual({
          ok: false,
          error: 'no entry point',
          offenders: [],
        });
      });

      it('allows exactly one entry point', () => {
        const doc = createDoc([{ is_entry_point: true }, { is_entry_point: false }]);
        expect(validateEntryPointForDeploy(doc)).toEqual({ ok: true });
      });

      it('rejects multiple entry points', () => {
        const doc = createDoc([{ is_entry_point: true }, { is_entry_point: true }]);
        expect(validateEntryPointForDeploy(doc)).toEqual({
          ok: false,
          error: 'multiple entry points',
          offenders: ['node-0', 'node-1'],
        });
      });
    });
  });
});

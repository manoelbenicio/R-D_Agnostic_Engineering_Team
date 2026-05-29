import { describe, it, expect } from 'vitest';
import { voiceToCanvas } from '../voice-to-canvas';
import { CreateCanvasIntent } from '../types';

describe('voiceToCanvas', () => {
  it('correctly maps canonical 4-node 3-edge canvas structure and auto-layout', () => {
    const intent: CreateCanvasIntent = {
      name: 'Time de Desenvolvimento e Code Review',
      nodes: [
        { display_name: 'Lead Supervisor', role: 'supervisor', provider: 'kiro_cli' },
        { display_name: 'Frontend Dev', role: 'developer', provider: 'claude_code' },
        { display_name: 'Backend Dev', role: 'developer', provider: 'claude_code' },
        { display_name: 'QA Specialist', role: 'reviewer', provider: 'codex' },
      ],
      edges: [
        { from: 'Lead Supervisor', to: 'Frontend Dev', type: 'handoff' },
        { from: 'Lead Supervisor', to: 'Backend Dev', type: 'handoff' },
        { from: 'Frontend Dev', to: 'QA Specialist', type: 'handoff' },
      ],
      confidence: 0.98,
    };

    const doc = voiceToCanvas(intent);

    expect(doc.name).toBe('Time de Desenvolvimento e Code Review');
    expect(doc.deploy_state.status).toBe('draft');
    expect(doc.nodes).toHaveLength(4);
    expect(doc.edges).toHaveLength(3);

    const supervisor = doc.nodes.find((n) => n.data.role === 'supervisor');
    expect(supervisor).toBeDefined();
    expect(supervisor?.data.is_entry_point).toBe(true);
    expect(supervisor?.position).toEqual({ x: 200, y: 100 });

    const developers = doc.nodes.filter((n) => n.data.role === 'developer');
    expect(developers).toHaveLength(2);
    expect(developers[0]?.position.y).toBe(300);
    expect(developers[1]?.position.y).toBe(300);
    expect(developers[0]?.position.x).toBe(100);
    expect(developers[1]?.position.x).toBe(350);

    const qa = doc.nodes.find((n) => n.data.role === 'reviewer');
    expect(qa).toBeDefined();

    const edge1 = doc.edges.find(
      (e) => e.source === supervisor?.id && e.target === developers[0]?.id
    );
    const edge2 = doc.edges.find(
      (e) => e.source === supervisor?.id && e.target === developers[1]?.id
    );
    const edge3 = doc.edges.find(
      (e) => e.source === developers[0]?.id && e.target === qa?.id
    );

    expect(edge1).toBeDefined();
    expect(edge1?.type).toBe('handoff');
    expect(edge2).toBeDefined();
    expect(edge3).toBeDefined();
  });

  it('correctly maps mixed-language 3-node canvas structure', () => {
    const intent: CreateCanvasIntent = {
      name: 'Time Misto',
      nodes: [
        { display_name: 'Gerente', role: 'supervisor', provider: 'kiro_cli' },
        { display_name: 'Dev', role: 'developer', provider: 'claude_code' },
        { display_name: 'Revisor', role: 'reviewer', provider: 'codex' },
      ],
      edges: [
        { from: 'Gerente', to: 'Dev', type: 'assign' },
        { from: 'Dev', to: 'Revisor', type: 'handoff' },
      ],
    };

    const doc = voiceToCanvas(intent);

    expect(doc.nodes).toHaveLength(3);
    expect(doc.edges).toHaveLength(2);
    expect(doc.edges[0]?.type).toBe('assign');
    expect(doc.edges[1]?.type).toBe('handoff');
  });

  // ---------------------------------------------------------------------
  // tech-debt-voice-coverage-gap §3 additions
  // ---------------------------------------------------------------------

  describe('coverage-gap additions', () => {
    it('handles a single-agent canvas (lone supervisor) by promoting it to entry point', () => {
      const intent: CreateCanvasIntent = {
        name: 'Solo Supervisor',
        nodes: [
          { display_name: 'Solo', role: 'supervisor', provider: 'kiro_cli' },
        ],
        edges: [],
      };

      const doc = voiceToCanvas(intent);

      expect(doc.nodes).toHaveLength(1);
      expect(doc.edges).toHaveLength(0);
      expect(doc.nodes[0]?.data.is_entry_point).toBe(true);
      expect(doc.nodes[0]?.position).toEqual({ x: 200, y: 100 });
      expect(doc.nodes[0]?.data.profile_name).toBe('supervisor');
      expect(doc.nodes[0]?.data.system_prompt).toContain('Supervisor');
    });

    it('promotes the first node when no node has role=supervisor', () => {
      const intent: CreateCanvasIntent = {
        name: 'No supervisor',
        nodes: [
          { display_name: 'Solo Dev', role: 'developer', provider: 'claude_code' },
          { display_name: 'Reviewer', role: 'reviewer', provider: 'codex' },
        ],
        edges: [],
      };

      const doc = voiceToCanvas(intent);

      expect(doc.nodes[0]?.data.display_name).toBe('Solo Dev');
      expect(doc.nodes[0]?.data.is_entry_point).toBe(true);
    });

    it('preserves all three orchestration edge types (handoff + assign + send_message)', () => {
      const intent: CreateCanvasIntent = {
        name: 'All edges',
        nodes: [
          { display_name: 'Sup', role: 'supervisor', provider: 'kiro_cli' },
          { display_name: 'Dev', role: 'developer', provider: 'claude_code' },
          { display_name: 'Rev', role: 'reviewer', provider: 'codex' },
        ],
        edges: [
          { from: 'Sup', to: 'Dev', type: 'handoff' },
          { from: 'Sup', to: 'Rev', type: 'assign' },
          { from: 'Dev', to: 'Rev', type: 'send_message' },
        ],
      };

      const doc = voiceToCanvas(intent);
      const types = doc.edges.map((e) => e.type).sort();
      expect(types).toEqual(['assign', 'handoff', 'send_message']);
      // Each edge gets a generated id and a label echoing its type.
      doc.edges.forEach((e) => {
        expect(e.id).toBeTypeOf('string');
        expect(e.id.length).toBeGreaterThan(0);
        expect(e.label).toBe(e.type);
      });
    });

    it('drops edges whose endpoints reference unknown display_names', () => {
      const intent: CreateCanvasIntent = {
        name: 'Bad edges',
        nodes: [
          { display_name: 'Sup', role: 'supervisor', provider: 'kiro_cli' },
          { display_name: 'Dev', role: 'developer', provider: 'claude_code' },
        ],
        edges: [
          { from: 'Sup', to: 'Dev', type: 'handoff' },
          { from: 'Sup', to: 'Ghost', type: 'handoff' }, // dangling target
          { from: 'Phantom', to: 'Dev', type: 'handoff' }, // dangling source
        ],
      };

      const doc = voiceToCanvas(intent);
      expect(doc.edges).toHaveLength(1);
      expect(doc.edges[0]?.type).toBe('handoff');
    });

    it('falls back to role-template defaults when system_prompt and tools are absent', () => {
      const intent: CreateCanvasIntent = {
        name: 'Defaults',
        nodes: [
          { display_name: 'Sup', role: 'supervisor', provider: 'kiro_cli' },
          { display_name: 'Dev', role: 'developer', provider: 'claude_code' },
          { display_name: 'Rev', role: 'reviewer', provider: 'codex' },
        ],
        edges: [],
      };

      const doc = voiceToCanvas(intent);

      const sup = doc.nodes.find((n) => n.data.role === 'supervisor');
      expect(sup?.data.profile_name).toBe('supervisor');
      expect(sup?.data.system_prompt).toContain('Supervisor');
      expect(sup?.data.allowedTools).toEqual(['handoff', 'send_message']);

      const dev = doc.nodes.find((n) => n.data.role === 'developer');
      expect(dev?.data.profile_name).toBe('developer');
      expect(dev?.data.allowedTools).toEqual(['shell', 'apply_patch']);

      const rev = doc.nodes.find((n) => n.data.role === 'reviewer');
      expect(rev?.data.profile_name).toBe('reviewer');
      expect(rev?.data.allowedTools).toEqual(['read_file', 'grep']);
    });

    it('falls back to the custom role-template when role is unrecognized', () => {
      const intent = {
        name: 'Unknown',
        nodes: [
          {
            display_name: 'Mystery',
            role: 'archivist' as unknown as 'custom',
            provider: 'claude_code',
          },
        ],
        edges: [],
      } as unknown as CreateCanvasIntent;

      const doc = voiceToCanvas(intent);
      expect(doc.nodes[0]?.data.profile_name).toBe('custom');
      expect(doc.nodes[0]?.data.allowedTools).toEqual([]);
      expect(doc.nodes[0]?.data.system_prompt).toContain('Custom');
    });

    it('maps every supported provider keyword to its canonical ProviderType', () => {
      const cases: Array<[string, string]> = [
        ['kiro', 'kiro_cli'],
        ['claude', 'claude_code'],
        ['codex', 'codex'],
        ['gemini', 'gemini_cli'],
        ['kimi', 'kimi_cli'],
        ['copilot', 'copilot_cli'],
        ['opencode', 'opencode_cli'],
        ['q', 'q_cli'],
      ];
      for (const [raw, expected] of cases) {
        const intent: CreateCanvasIntent = {
          name: `Provider ${raw}`,
          nodes: [
            { display_name: 'Sup', role: 'supervisor', provider: raw },
          ],
          edges: [],
        };
        const doc = voiceToCanvas(intent);
        expect(doc.nodes[0]?.data.provider).toBe(expected);
      }
    });

    it('matches the speech-to-canvas spec §5.1 canonical pt-BR example (1 sup + 2 dev + 1 rev, 3 handoffs)', () => {
      // Mirrors the canonical scenario from
      // openspec/changes/milestone-1-canvas-deploy-run/specs/speech-to-canvas/spec.md.
      const intent: CreateCanvasIntent = {
        name: 'Time de Desenvolvimento',
        nodes: [
          { display_name: 'Supervisor', role: 'supervisor', provider: 'kiro_cli' },
          { display_name: 'Dev Frontend', role: 'developer', provider: 'claude_code' },
          { display_name: 'Dev Backend', role: 'developer', provider: 'claude_code' },
          { display_name: 'Revisor', role: 'reviewer', provider: 'codex' },
        ],
        edges: [
          { from: 'Supervisor', to: 'Dev Frontend', type: 'handoff' },
          { from: 'Supervisor', to: 'Dev Backend', type: 'handoff' },
          { from: 'Dev Frontend', to: 'Revisor', type: 'handoff' },
        ],
        confidence: 0.97,
      };

      const doc = voiceToCanvas(intent);

      expect(doc.deploy_state.status).toBe('draft');
      expect(doc.nodes).toHaveLength(4);
      expect(doc.edges).toHaveLength(3);
      const entry = doc.nodes.find((n) => n.data.is_entry_point);
      expect(entry?.data.role).toBe('supervisor');
      expect(doc.edges.every((e) => e.type === 'handoff')).toBe(true);
      expect(doc.config.provider_default).toBe('claude_code');
    });
  });
});

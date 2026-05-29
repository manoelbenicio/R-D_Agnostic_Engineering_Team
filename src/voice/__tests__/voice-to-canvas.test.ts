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
});

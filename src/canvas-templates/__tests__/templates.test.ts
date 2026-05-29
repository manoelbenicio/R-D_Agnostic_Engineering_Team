/* eslint-disable agentverse/no-sideways-capability-imports */
import { describe, expect, it } from 'vitest';
import { parseCanvasDocument } from '@/canvas-document/schema';
import { instantiateTemplate, TEMPLATES } from '../templates';

describe('canvas templates', () => {
  it('exposes the exact 10 v1 templates with metadata', () => {
    expect(TEMPLATES).toHaveLength(10);
    expect(TEMPLATES.map((template) => template.name)).toEqual([
      'Code Review Pipeline',
      'Bug Triage',
      'Documentation Sprint',
      'Full Stack Dev',
      'Data Pipeline',
      'Security Audit',
      'DevOps Pipeline',
      'Research Team',
      'Enterprise Squad',
      'Blank Canvas',
    ]);

    for (const template of TEMPLATES) {
      expect(template.id).toBeTruthy();
      expect(template.description).toBeTruthy();
      expect(template.agent_count).toBe(template.document.nodes.length);
      expect(template.primary_edge_type).toBeTruthy();
      expect(template.est_cost_per_hour_usd).toBe(template.agent_count * 15);
      expect(parseCanvasDocument(template.document)).toEqual(template.document);
    }
  });

  it('creates fresh disjoint IDs for repeated instantiations', () => {
    const first = instantiateTemplate('code-review-pipeline');
    const second = instantiateTemplate('code-review-pipeline');

    expect(first.id).not.toBe(second.id);
    expect(first.name).toBe('Code Review Pipeline (copy)');
    expect(first.deploy_state.status).toBe('draft');
    expect(second.deploy_state.status).toBe('draft');

    const firstIds = new Set([
      first.id,
      ...first.nodes.map((node) => node.id),
      ...first.edges.map((edge) => edge.id),
    ]);
    const secondIds = [
      second.id,
      ...second.nodes.map((node) => node.id),
      ...second.edges.map((edge) => edge.id),
    ];

    for (const id of secondIds) {
      expect(firstIds.has(id)).toBe(false);
    }
  });

  it('instantiates blank canvas with zero nodes and edges', () => {
    const blank = instantiateTemplate('blank-canvas');

    expect(blank.nodes).toHaveLength(0);
    expect(blank.edges).toHaveLength(0);
    expect(blank.deploy_state.status).toBe('draft');
  });
});

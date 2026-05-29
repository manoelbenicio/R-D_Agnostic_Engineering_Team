import { describe, expect, it } from 'vitest';
import { parseCanvasDocument, CanvasParseError } from '../schema';
import { CanvasDocument } from '@/shared/canvas-types';

describe('Canvas Document Schema', () => {
  const validDoc: CanvasDocument = {
    id: '11111111-1111-1111-1111-111111111111',
    name: 'Test Canvas',
    version: 1,
    created_at: '2026-05-27T22:00:00.000Z',
    updated_at: '2026-05-27T22:00:00.000Z',
    schema_version: 1,
    nodes: [
      {
        id: '22222222-2222-2222-2222-222222222222',
        type: 'agent',
        position: { x: 100, y: 200 },
        data: {
          profile_name: 'test_profile',
          display_name: 'Test Agent',
          role: 'developer',
          provider: 'openai',
          model: 'gpt-4o',
          system_prompt: 'You are a test agent.',
          allowedTools: ['tool1', 'tool2'],
          is_entry_point: true,
        },
      },
    ],
    edges: [],
    config: {
      working_directory: '/workspace',
      provider_default: 'openai',
    },
    deploy_state: {
      status: 'draft',
    },
  };

  it('successfully round-trips a valid CanvasDocument with no info loss', () => {
    const parsed = parseCanvasDocument(validDoc);
    expect(parsed).toEqual(validDoc);
  });

  it('throws CanvasParseError with populated paths for missing fields', () => {
    const malformed = {
      id: '11111111-1111-1111-1111-111111111111',
      name: 'Missing fields',
    };

    expect(() => parseCanvasDocument(malformed)).toThrow(CanvasParseError);
    try {
      parseCanvasDocument(malformed);
    } catch (err: any) {
      expect(err).toBeInstanceOf(CanvasParseError);
      expect(err.paths).toContain('version');
      expect(err.paths).toContain('created_at');
      expect(err.paths).toContain('nodes');
    }
  });

  it('rejects malformed UUIDs', () => {
    const docWithBadId = {
      ...validDoc,
      id: 'not-a-uuid',
    };

    expect(() => parseCanvasDocument(docWithBadId)).toThrow(CanvasParseError);
    try {
      parseCanvasDocument(docWithBadId);
    } catch (err: any) {
      expect(err.paths).toContain('id');
    }
  });
});

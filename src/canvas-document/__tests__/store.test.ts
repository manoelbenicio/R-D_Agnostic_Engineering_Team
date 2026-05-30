import { describe, expect, it, beforeEach, vi } from 'vitest';
import { canvasStore } from '../store';
import { openDb } from '@/shared/storage/idb';
import { CanvasDocument } from '@/shared/canvas-types';

describe('Canvas Store', () => {
  let doc: CanvasDocument;

  beforeEach(() => {
    doc = canvasStore.createDraft();
  });

  it('can create a draft canvas document with correct default values', () => {
    expect(doc.id).toBeDefined();
    expect(doc.name).toBe('Untitled canvas');
    expect(doc.version).toBe(1);
    expect(doc.nodes).toEqual([]);
    expect(doc.edges).toEqual([]);
    expect(doc.config.working_directory).toBe('~');
    expect(doc.config.provider_default).toBe('');
    expect(doc.deploy_state.status).toBe('draft');
  });

  it('can save, get, and list documents in the database', async () => {
    const saved = await canvasStore.save(doc);
    expect(saved.id).toBe(doc.id);
    expect(saved.version).toBe(1);

    const fetched = await canvasStore.get(doc.id);
    expect(fetched).toBeDefined();
    expect(fetched?.id).toBe(doc.id);
    expect(fetched?.name).toBe(doc.name);

    const list = await canvasStore.list();
    expect(list.some(d => d.id === doc.id)).toBe(true);
  });

  it('increments the version monotonically on consecutive saves and appends snapshots to canvas_versions', async () => {
    const d = canvasStore.createDraft();
    const s1 = await canvasStore.save(d);
    expect(s1.version).toBe(1);

    const s2 = await canvasStore.save(s1);
    expect(s2.version).toBe(2);

    const s3 = await canvasStore.save(s2);
    expect(s3.version).toBe(3);

    const versions = await canvasStore.listVersions(d.id);
    expect(versions).toHaveLength(3);
    expect(versions[0]?.version).toBe(1);
    expect(versions[1]?.version).toBe(2);
    expect(versions[2]?.version).toBe(3);

    expect(versions[0]?.name).toBe('Untitled canvas');
  });

  it('can delete documents from canvases and canvas_versions', async () => {
    const d = canvasStore.createDraft();
    await canvasStore.save(d);

    let list = await canvasStore.list();
    expect(list.some(x => x.id === d.id)).toBe(true);

    await canvasStore.delete(d.id);

    list = await canvasStore.list();
    expect(list.some(x => x.id === d.id)).toBe(false);

    const versions = await canvasStore.listVersions(d.id);
    expect(versions).toHaveLength(0);
  });

  it('returns null and emits a warning on get() when schema_version exceeds SCHEMA_VERSION', async () => {
    const db = await openDb();
    const futureDoc = {
      ...doc,
      schema_version: 999,
    };

    await db.put('canvases', futureDoc);

    const consoleWarnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});

    const fetched = await canvasStore.get(doc.id);
    expect(fetched).toBeNull();
    expect(consoleWarnSpy).toHaveBeenCalled();
    expect(consoleWarnSpy.mock.calls[0]?.[0]).toContain('incompatible schema version 999');

    consoleWarnSpy.mockRestore();
  });
});

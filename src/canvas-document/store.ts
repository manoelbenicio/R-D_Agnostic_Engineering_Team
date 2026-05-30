import { openDb } from '@/shared/storage/idb';
import { CanvasDocument } from '@/shared/canvas-types';
import { SCHEMA_VERSION, isCompatible } from '@/shared/schema-version';
import { parseCanvasDocument } from './schema';

export interface CanvasStore {
  list(): Promise<CanvasDocument[]>;
  get(id: string): Promise<CanvasDocument | null>;
  save(doc: CanvasDocument): Promise<CanvasDocument>;
  delete(id: string): Promise<void>;
  listVersions(id: string): Promise<CanvasDocument[]>;
  createDraft(): CanvasDocument;
}

export const canvasStore: CanvasStore = {
  async list(): Promise<CanvasDocument[]> {
    const db = await openDb();
    const all = await db.getAll('canvases');
    const docs: CanvasDocument[] = [];
    for (const raw of all) {
      try {
        const doc = parseCanvasDocument(raw);
        docs.push(doc);
      } catch (e) {
        const rawDoc = raw as Record<string, unknown>;
        if (
          rawDoc &&
          typeof rawDoc.schema_version === 'number' &&
          rawDoc.schema_version > SCHEMA_VERSION
        ) {
          docs.push(rawDoc as unknown as CanvasDocument);
        } else {
          console.error('Failed to parse canvas document from database:', e);
        }
      }
    }
    return docs;
  },

  async get(id: string): Promise<CanvasDocument | null> {
    const db = await openDb();
    const raw = await db.get('canvases', id);
    if (!raw) return null;

    const doc = parseCanvasDocument(raw);
    if (!isCompatible(doc)) {
      console.warn(
        `Canvas document ${id} has incompatible schema version ${doc.schema_version} (current: ${SCHEMA_VERSION})`
      );
      return null;
    }
    return doc;
  },

  async save(doc: CanvasDocument): Promise<CanvasDocument> {
    const db = await openDb();

    // Check if the document already exists to determine monotonic version bumping
    const existing = await db.get('canvases', doc.id);
    if (existing) {
      const existingDoc = parseCanvasDocument(existing);
      doc.version = existingDoc.version + 1;
    } else {
      doc.version = doc.version || 1;
    }

    doc.updated_at = new Date().toISOString();

    const validated = parseCanvasDocument(doc);

    // Save to canvases
    await db.put('canvases', validated);

    // Append to canvas_versions. Note: canvas_versions compound key is [canvas_id, version]
    const snapshot = {
      ...validated,
      canvas_id: validated.id,
    };
    await db.put('canvas_versions', snapshot);

    return validated;
  },

  async delete(id: string): Promise<void> {
    const db = await openDb();
    await db.delete('canvases', id);

    // Clean up versions associated with this canvas
    const range = IDBKeyRange.bound([id, 0], [id, Number.MAX_SAFE_INTEGER]);
    const tx = db.transaction('canvas_versions', 'readwrite');
    const store = tx.objectStore('canvas_versions');
    let cursor = await store.openCursor(range);
    while (cursor) {
      await cursor.delete();
      cursor = await cursor.continue();
    }
    await tx.done;
  },

  async listVersions(id: string): Promise<CanvasDocument[]> {
    const db = await openDb();
    const range = IDBKeyRange.bound([id, 0], [id, Number.MAX_SAFE_INTEGER]);
    const rawVersions = await db.getAll('canvas_versions', range);

    return rawVersions.map((v: unknown) => {
      const { canvas_id: _canvas_id, ...doc } = v as Record<string, unknown> & { canvas_id: string };
      return parseCanvasDocument(doc);
    });
  },

  createDraft(): CanvasDocument {
    const now = new Date().toISOString();
    return {
      id: crypto.randomUUID(),
      name: 'Untitled canvas',
      version: 1,
      created_at: now,
      updated_at: now,
      schema_version: SCHEMA_VERSION,
      nodes: [],
      edges: [],
      config: {
        working_directory: '~',
        provider_default: '',
      },
      deploy_state: {
        status: 'draft',
      },
    };
  },
};

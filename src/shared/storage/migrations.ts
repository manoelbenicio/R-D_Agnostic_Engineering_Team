import { IDBPDatabase, IDBPTransaction, StoreNames } from 'idb';
import { AgentVerseDB } from './idb';

export const CURRENT_SCHEMA_VERSION = 3;
type AgentVerseStoreNames = ArrayLike<StoreNames<AgentVerseDB>>;

export function runMigrations(
  db: IDBPDatabase<AgentVerseDB>,
  oldVersion: number,
  _newVersion: number | null,
  transaction: IDBPTransaction<AgentVerseDB, AgentVerseStoreNames, 'versionchange'>
) {
  if (oldVersion < 1) {
    // 1. canvases (keyPath: id)
    db.createObjectStore('canvases', { keyPath: 'id' });

    // 2. canvas_versions (keyPath: [canvas_id, version])
    db.createObjectStore('canvas_versions', { keyPath: ['canvas_id', 'version'] });

    // 3. provider_keys (keyPath: provider)
    db.createObjectStore('provider_keys', { keyPath: 'provider' });

    // 4. settings (keyPath: key)
    db.createObjectStore('settings', { keyPath: 'key' });

    // 5. app_state (keyPath: key)
    db.createObjectStore('app_state', { keyPath: 'key' });

    // Seed the initial schema_version in app_state
    const appStateStore = transaction.objectStore('app_state');
    appStateStore.put({ key: 'schema_version', value: CURRENT_SCHEMA_VERSION });
  }

  if (oldVersion < 2) {
    // FinOps Tier 2: persisted token-usage events (keyPath: id), indexed by
    // canvas so per-canvas cost roll-ups don't scan the whole store.
    const usageStore = db.createObjectStore('usage_events', { keyPath: 'id' });
    usageStore.createIndex('by-canvas', 'canvasId');
    usageStore.createIndex('by-timestamp', 'timestampMs');
  }

  if (oldVersion < 3 && !db.objectStoreNames.contains('sessions')) {
    db.createObjectStore('sessions');
  }
}

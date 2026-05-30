import { openDB, IDBPDatabase, IDBPTransaction, StoreNames, DBSchema, IDBPObjectStore } from 'idb';
import { CURRENT_SCHEMA_VERSION, runMigrations } from './migrations';

export interface AgentVerseDB extends DBSchema {
  canvases: {
    key: string;
    value: unknown;
  };
  canvas_versions: {
    key: [string, number];
    value: unknown;
  };
  provider_keys: {
    key: string;
    value: unknown;
  };
  settings: {
    key: string;
    value: { key: string; value: unknown };
  };
  app_state: {
    key: string;
    value: { key: string; value: unknown };
  };
  usage_events: {
    key: string;
    value: {
      id: string;
      timestampMs: number;
      provider: string;
      model: string;
      inputTokens: number;
      outputTokens: number;
      totalTokens: number;
      sessionName?: string;
      terminalId?: string;
      canvasId?: string;
    };
    indexes: { 'by-canvas': string; 'by-timestamp': number };
  };
}

let dbInstance: IDBPDatabase<AgentVerseDB> | null = null;

export async function openDb(): Promise<IDBPDatabase<AgentVerseDB>> {
  if (dbInstance) return dbInstance;
  dbInstance = await openDB<AgentVerseDB>('AgentVerse', CURRENT_SCHEMA_VERSION, {
    upgrade(db, oldVersion, newVersion, transaction) {
      runMigrations(db, oldVersion, newVersion, transaction);
    },
  });
  return dbInstance;
}

export async function getStore<Name extends StoreNames<AgentVerseDB>>(
  storeName: Name,
  mode: 'readonly' | 'readwrite' = 'readonly'
): Promise<{
  db: IDBPDatabase<AgentVerseDB>;
  transaction: IDBPTransaction<AgentVerseDB, [Name], typeof mode>;
  store: IDBPObjectStore<AgentVerseDB, [Name], Name, typeof mode>;
}> {
  const db = await openDb();
  const transaction = db.transaction(storeName, mode);
  const store = transaction.objectStore(storeName);
  return { db, transaction, store };
}

// Global generic helpers for standard CRUD operations
export async function dbGet<Name extends StoreNames<AgentVerseDB>>(
  storeName: Name,
  key: AgentVerseDB[Name]['key']
): Promise<AgentVerseDB[Name]['value'] | undefined> {
  const db = await openDb();
  return db.get(storeName, key);
}

export async function dbPut<Name extends StoreNames<AgentVerseDB>>(
  storeName: Name,
  value: AgentVerseDB[Name]['value']
): Promise<unknown> {
  const db = await openDb();
  return db.put(storeName, value);
}

export async function dbDelete<Name extends StoreNames<AgentVerseDB>>(
  storeName: Name,
  key: AgentVerseDB[Name]['key']
): Promise<void> {
  const db = await openDb();
  return db.delete(storeName, key);
}

export async function dbGetAll<Name extends StoreNames<AgentVerseDB>>(
  storeName: Name
): Promise<AgentVerseDB[Name]['value'][]> {
  const db = await openDb();
  return db.getAll(storeName);
}

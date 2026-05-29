import { dbGet, dbPut, dbDelete } from './idb';

export interface AppStateStore {
  get: <T>(key: string) => Promise<T | undefined>;
  set: <T>(key: string, value: T) => Promise<void>;
  delete: (key: string) => Promise<void>;
}

export const AppStateStore: AppStateStore = {
  async get<T>(key: string): Promise<T | undefined> {
    const record = await dbGet('app_state', key);
    return record?.value as T | undefined;
  },
  async set<T>(key: string, value: T): Promise<void> {
    await dbPut('app_state', { key, value });
  },
  async delete(key: string): Promise<void> {
    await dbDelete('app_state', key);
  },
};

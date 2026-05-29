import { dbGet, dbPut, dbDelete, dbGetAll } from '@/shared/storage/idb';
import { ProviderType } from './registry';

export interface KeyRecord {
  provider: ProviderType;
  keys: Record<string, string>;
  models: string[];
  updatedAt: string;
}

export const KeyStore = {
  async get(provider: ProviderType): Promise<KeyRecord | null> {
    const record = await dbGet('provider_keys', provider);
    return (record as KeyRecord) || null;
  },

  async set(provider: ProviderType, keys: Record<string, string>, models: string[]): Promise<void> {
    const record: KeyRecord = {
      provider,
      keys,
      models,
      updatedAt: new Date().toISOString(),
    };
    await dbPut('provider_keys', record);
  },

  async remove(provider: ProviderType): Promise<void> {
    await dbDelete('provider_keys', provider);
  },

  async all(): Promise<KeyRecord[]> {
    const records = await dbGetAll('provider_keys');
    return records as KeyRecord[];
  },
};
export default KeyStore;

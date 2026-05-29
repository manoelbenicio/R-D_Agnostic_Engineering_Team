import { create } from 'zustand';
import { KeyStore } from './index';
import { ProviderType, PROVIDERS_REGISTRY } from './registry';
import { maskKey } from './mask';
import { dbGet, dbPut } from '@/shared/storage/idb';

export interface KeyStoreState {
  validated: ProviderType[];
  statuses: Record<ProviderType, 'set' | 'unset' | 'invalid'>;
  cachedModels: Record<ProviderType, string[]>;
  maskedKeys: Record<ProviderType, Record<string, string>>;
  initialized: boolean;
  
  init: () => Promise<void>;
  setKey: (provider: ProviderType, keys: Record<string, string>, models: string[]) => Promise<void>;
  setInvalid: (provider: ProviderType) => void;
  removeKey: (provider: ProviderType) => Promise<void>;
}

const initialStatuses = PROVIDERS_REGISTRY.reduce((acc, p) => {
  acc[p.id] = 'unset';
  return acc;
}, {} as Record<ProviderType, 'set' | 'unset' | 'invalid'>);

const initialModels = PROVIDERS_REGISTRY.reduce((acc, p) => {
  acc[p.id] = p.defaultModels;
  return acc;
}, {} as Record<ProviderType, string[]>);

const initialMasked = PROVIDERS_REGISTRY.reduce((acc, p) => {
  acc[p.id] = {};
  return acc;
}, {} as Record<ProviderType, Record<string, string>>);

export const useKeyStore = create<KeyStoreState>((set, get) => ({
  validated: [],
  statuses: initialStatuses,
  cachedModels: initialModels,
  maskedKeys: initialMasked,
  initialized: false,

  init: async () => {
    if (get().initialized) return;
    try {
      const records = await KeyStore.all();
      const validated: ProviderType[] = [];
      const statuses = { ...initialStatuses };
      const cachedModels = { ...initialModels };
      const maskedKeys = { ...initialMasked };

      for (const rec of records) {
        validated.push(rec.provider);
        statuses[rec.provider] = 'set';
        if (rec.models && rec.models.length > 0) {
          cachedModels[rec.provider] = rec.models;
        }
        
        const providerDef = PROVIDERS_REGISTRY.find(p => p.id === rec.provider);
        if (providerDef) {
          const masked: Record<string, string> = {};
          for (const field of providerDef.fields) {
            const rawVal = rec.keys[field.name] || '';
            masked[field.name] = maskKey(rawVal);
          }
          maskedKeys[rec.provider] = masked;
        }
      }

      set({
        validated,
        statuses,
        cachedModels,
        maskedKeys,
        initialized: true,
      });
    } catch (err) {
      console.error('Failed to initialize KeyStore state:', err);
    }
  },

  setKey: async (provider, keys, models) => {
    // 1. Persist in IDB KeyStore
    await KeyStore.set(provider, keys, models);
    
    // 2. Remove from provider_orphans list in app_state
    try {
      const currentOrphans = await dbGet('app_state', 'provider_orphans');
      const orphansList = (currentOrphans?.value as string[]) || [];
      if (orphansList.includes(provider)) {
        const updatedOrphans = orphansList.filter((p) => p !== provider);
        await dbPut('app_state', { key: 'provider_orphans', value: updatedOrphans });
      }
    } catch (err) {
      console.error('Failed to update provider_orphans in IDB:', err);
    }

    // 3. Update local Zustand state
    const { validated, statuses, cachedModels, maskedKeys } = get();
    
    const newValidated = validated.includes(provider) 
      ? validated 
      : [...validated, provider];
      
    const newStatuses = { ...statuses, [provider]: 'set' as const };
    const newModels = { ...cachedModels, [provider]: models };
    
    const providerDef = PROVIDERS_REGISTRY.find(p => p.id === provider);
    const masked: Record<string, string> = {};
    if (providerDef) {
      for (const field of providerDef.fields) {
        masked[field.name] = maskKey(keys[field.name] || '');
      }
    }
    const newMaskedKeys = { ...maskedKeys, [provider]: masked };

    set({
      validated: newValidated,
      statuses: newStatuses,
      cachedModels: newModels,
      maskedKeys: newMaskedKeys,
    });
  },

  setInvalid: (provider) => {
    const { statuses } = get();
    set({
      statuses: { ...statuses, [provider]: 'invalid' as const },
    });
  },

  removeKey: async (provider) => {
    // 1. Remove from IDB KeyStore
    await KeyStore.remove(provider);

    // 2. Add to provider_orphans list in app_state
    try {
      const currentOrphans = await dbGet('app_state', 'provider_orphans');
      const orphansList = (currentOrphans?.value as string[]) || [];
      if (!orphansList.includes(provider)) {
        orphansList.push(provider);
        await dbPut('app_state', { key: 'provider_orphans', value: orphansList });
      }
    } catch (err) {
      console.error('Failed to update provider_orphans in IDB:', err);
    }

    // 3. Update local Zustand state
    const { validated, statuses, cachedModels, maskedKeys } = get();
    
    const newValidated = validated.filter(p => p !== provider);
    const newStatuses = { ...statuses, [provider]: 'unset' as const };
    const newModels = { 
      ...cachedModels, 
      [provider]: PROVIDERS_REGISTRY.find(p => p.id === provider)?.defaultModels || [] 
    };
    const newMaskedKeys = { ...maskedKeys, [provider]: {} };

    set({
      validated: newValidated,
      statuses: newStatuses,
      cachedModels: newModels,
      maskedKeys: newMaskedKeys,
    });
  },
}));

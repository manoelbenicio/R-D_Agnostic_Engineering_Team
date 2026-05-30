import { create } from 'zustand';
import { dbGet, dbPut } from '@/shared/storage/idb';
import { applyFontOverrides } from '@/design-system/utils/font-override';
import { caoClient } from '@/api/cao-client';

export interface FontSettings {
  body: string;
  display: string;
  mono: string;
}

export interface SettingsStoreState {
  caoBaseUrl: string;
  defaultProvider: string;
  defaultWorkingDir: string;
  fonts: FontSettings;
  sttEngine: string;
  theme: string;
  spBaseUrl: string;
  initialized: boolean;

  init: () => Promise<void>;
  updateSetting: <K extends keyof Omit<SettingsStoreState, 'initialized' | 'init' | 'updateSetting'>>(
    key: K,
    value: SettingsStoreState[K]
  ) => Promise<void>;
}

export const useSettingsStore = create<SettingsStoreState>((set, get) => ({
  caoBaseUrl: import.meta.env.VITE_CAO_BASE_URL || 'http://127.0.0.1:9889',
  defaultProvider: '',
  defaultWorkingDir: '',
  fonts: {
    body: 'system-ui',
    display: 'Inter',
    mono: 'JetBrains Mono',
  },
  sttEngine: 'whisper',
  theme: 'dark',
  spBaseUrl: 'https://sharepoint.minsait.com',
  initialized: false,

  init: async () => {
    if (get().initialized) return;
    try {
      const caoBaseUrlRec = await dbGet('settings', 'caoBaseUrl');
      const defaultProviderRec = await dbGet('settings', 'defaultProvider');
      const defaultWorkingDirRec = await dbGet('settings', 'defaultWorkingDir');
      const fontsRec = await dbGet('settings', 'fonts');
      const sttEngineRec = await dbGet('settings', 'sttEngine');
      const themeRec = await dbGet('settings', 'theme');
      const spBaseUrlRec = await dbGet('settings', 'spBaseUrl');

      const loadedCaoBaseUrl = (caoBaseUrlRec?.value as string) || import.meta.env.VITE_CAO_BASE_URL || 'http://127.0.0.1:9889';
      caoClient.baseUrl = loadedCaoBaseUrl.replace(/\/$/, '');

      const loadedFonts = (fontsRec?.value as FontSettings) || {
        body: 'system-ui',
        display: 'Inter',
        mono: 'JetBrains Mono',
      };

      // Apply font overrides on start
      applyFontOverrides(loadedFonts);

      set({
        caoBaseUrl: loadedCaoBaseUrl,
        defaultProvider: (defaultProviderRec?.value as string) || '',
        defaultWorkingDir: (defaultWorkingDirRec?.value as string) || '',
        fonts: loadedFonts,
        sttEngine: (sttEngineRec?.value as string) || 'whisper',
        theme: (themeRec?.value as string) || 'dark',
        spBaseUrl: (spBaseUrlRec?.value as string) || 'https://sharepoint.minsait.com',
        initialized: true,
      });
    } catch (err) {
      console.error('Failed to load settings from IndexedDB:', err);
    }
  },

  updateSetting: async (key, value) => {
    try {
      await dbPut('settings', { key, value });
    } catch (err) {
      console.error(`Failed to persist setting ${key} to IndexedDB:`, err);
    }

    if (key === 'fonts') {
      applyFontOverrides(value as FontSettings);
    }

    if (key === 'caoBaseUrl') {
      caoClient.baseUrl = (value as string).replace(/\/$/, '');
    }

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    set({ [key]: value } as any);
  },
}));

export default useSettingsStore;

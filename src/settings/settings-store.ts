import { create } from 'zustand';
import { dbGet, dbPut } from '@/shared/storage/idb';
import { applyFontOverrides } from '@/design-system/utils/font-override';
import { goCoreClient } from '@/api/go-core-client';
// CRIT-003.9: caoBaseUrl → goCoreBaseUrl

export interface FontSettings {
  body: string;
  display: string;
  mono: string;
}

export interface SettingsStoreState {
  goCoreBaseUrl: string;
  /** @deprecated Use goCoreBaseUrl */
  get caoBaseUrl(): string;
  defaultProvider: string;
  defaultWorkingDir: string;
  fonts: FontSettings;
  sttEngine: string;
  theme: string;
  spBaseUrl: string;
  sessionAutoRefreshInterval: number;
  sessionShowExpiredWarnings: boolean;
  sessionMaskEmails: boolean;
  initialized: boolean;

  init: () => Promise<void>;
  updateSetting: <
    K extends keyof Omit<SettingsStoreState, 'initialized' | 'init' | 'updateSetting' | 'caoBaseUrl'>,
  >(
    key: K,
    value: SettingsStoreState[K],
  ) => Promise<void>;
}

export const useSettingsStore = create<SettingsStoreState>((set, get) => ({
  goCoreBaseUrl: import.meta.env.VITE_GO_CORE_BASE_URL || 'http://127.0.0.1:8080',
  // Backward-compat getter for code still reading caoBaseUrl
  get caoBaseUrl() { return get().goCoreBaseUrl; },
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
  sessionAutoRefreshInterval: 5,
  sessionShowExpiredWarnings: true,
  sessionMaskEmails: false,
  initialized: false,

  init: async () => {
    if (get().initialized) return;
    try {
      // CRIT-003.9: read goCoreBaseUrl first, fall back to legacy caoBaseUrl key
      const goCoreBaseUrlRec = await dbGet('settings', 'goCoreBaseUrl');
      const legacyCaoBaseUrlRec = await dbGet('settings', 'caoBaseUrl');
      const defaultProviderRec = await dbGet('settings', 'defaultProvider');
      const defaultWorkingDirRec = await dbGet('settings', 'defaultWorkingDir');
      const fontsRec = await dbGet('settings', 'fonts');
      const sttEngineRec = await dbGet('settings', 'sttEngine');
      const themeRec = await dbGet('settings', 'theme');
      const spBaseUrlRec = await dbGet('settings', 'spBaseUrl');
      const sessionAutoRefreshIntervalRec = await dbGet('settings', 'sessionAutoRefreshInterval');
      const sessionShowExpiredWarningsRec = await dbGet('settings', 'sessionShowExpiredWarnings');
      const sessionMaskEmailsRec = await dbGet('settings', 'sessionMaskEmails');

      const loadedGoCoreBaseUrl =
        (goCoreBaseUrlRec?.value as string) ||
        // Migrate from legacy caoBaseUrl setting (IDB v3→v4)
        (legacyCaoBaseUrlRec?.value as string) ||
        import.meta.env.VITE_GO_CORE_BASE_URL ||
        'http://127.0.0.1:8080';
      goCoreClient.baseUrl = loadedGoCoreBaseUrl.replace(/\/$/, '');

      const loadedFonts = (fontsRec?.value as FontSettings) || {
        body: 'system-ui',
        display: 'Inter',
        mono: 'JetBrains Mono',
      };

      // Apply font overrides on start
      applyFontOverrides(loadedFonts);

      set({
        goCoreBaseUrl: loadedGoCoreBaseUrl,
        defaultProvider: (defaultProviderRec?.value as string) || '',
        defaultWorkingDir: (defaultWorkingDirRec?.value as string) || '',
        fonts: loadedFonts,
        sttEngine: (sttEngineRec?.value as string) || 'whisper',
        theme: (themeRec?.value as string) || 'dark',
        spBaseUrl: (spBaseUrlRec?.value as string) || 'https://sharepoint.minsait.com',
        sessionAutoRefreshInterval:
          (sessionAutoRefreshIntervalRec?.value as number | undefined) ?? 5,
        sessionShowExpiredWarnings:
          (sessionShowExpiredWarningsRec?.value as boolean | undefined) ?? true,
        sessionMaskEmails: (sessionMaskEmailsRec?.value as boolean | undefined) ?? false,
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

    if (key === 'goCoreBaseUrl') {
      goCoreClient.baseUrl = (value as string).replace(/\/$/, '');
    }

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    set({ [key]: value } as any);
  },
}));

export default useSettingsStore;

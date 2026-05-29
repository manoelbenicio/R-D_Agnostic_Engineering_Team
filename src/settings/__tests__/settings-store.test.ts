import { describe, it, expect, beforeEach } from 'vitest';
import { useSettingsStore } from '../settings-store';
import { dbGet, dbPut } from '@/shared/storage/idb';

describe('useSettingsStore', () => {
  beforeEach(async () => {
    useSettingsStore.setState({
      defaultProvider: '',
      defaultWorkingDir: '',
      fonts: {
        body: 'system-ui',
        display: 'Inter',
        mono: 'JetBrains Mono',
      },
      sttEngine: 'whisper',
      theme: 'dark',
      initialized: false,
    });
  });

  it('can initialize settings from IndexedDB', async () => {
    await dbPut('settings', { key: 'defaultWorkingDir', value: 'C:/my-workspace' });
    await dbPut('settings', { key: 'theme', value: 'light' });

    await useSettingsStore.getState().init();

    expect(useSettingsStore.getState().defaultWorkingDir).toBe('C:/my-workspace');
    expect(useSettingsStore.getState().theme).toBe('light');
    expect(useSettingsStore.getState().defaultProvider).toBe('');
  });

  it('can update settings and persist to IndexedDB', async () => {
    await useSettingsStore.getState().updateSetting('defaultProvider', 'openai');

    expect(useSettingsStore.getState().defaultProvider).toBe('openai');

    const dbRecord = await dbGet('settings', 'defaultProvider');
    expect(dbRecord?.value).toBe('openai');
  });
});

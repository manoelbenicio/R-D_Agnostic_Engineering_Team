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
      spBaseUrl: 'https://sharepoint.minsait.com',
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
    expect(useSettingsStore.getState().spBaseUrl).toBe('https://sharepoint.minsait.com');
  });

  it('can initialize spBaseUrl from IndexedDB', async () => {
    await dbPut('settings', { key: 'spBaseUrl', value: 'http://192.168.1.100' });

    await useSettingsStore.getState().init();

    expect(useSettingsStore.getState().spBaseUrl).toBe('http://192.168.1.100');
  });

  it('can update settings and persist to IndexedDB', async () => {
    await useSettingsStore.getState().updateSetting('defaultProvider', 'openai');

    expect(useSettingsStore.getState().defaultProvider).toBe('openai');

    const dbRecord = await dbGet('settings', 'defaultProvider');
    expect(dbRecord?.value).toBe('openai');
  });

  it('can update spBaseUrl and persist to IndexedDB', async () => {
    await useSettingsStore.getState().updateSetting('spBaseUrl', 'https://sp.local');

    expect(useSettingsStore.getState().spBaseUrl).toBe('https://sp.local');

    const dbRecord = await dbGet('settings', 'spBaseUrl');
    expect(dbRecord?.value).toBe('https://sp.local');
  });
});

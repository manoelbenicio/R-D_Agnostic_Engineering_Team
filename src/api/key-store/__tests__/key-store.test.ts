import { describe, it, expect, beforeEach } from 'vitest';
import { KeyStore } from '../index';

describe('KeyStore', () => {
  beforeEach(async () => {
    const allRecords = await KeyStore.all();
    for (const rec of allRecords) {
      await KeyStore.remove(rec.provider);
    }
  });

  it('can set, get, and list keys', async () => {
    const keys = { apiKey: 'test-api-key-openai' };
    const models = ['gpt-4', 'gpt-4-mini'];

    await KeyStore.set('openai', keys, models);

    const record = await KeyStore.get('openai');
    expect(record).not.toBeNull();
    expect(record?.provider).toBe('openai');
    expect(record?.keys).toEqual(keys);
    expect(record?.models).toEqual(models);

    const all = await KeyStore.all();
    expect(all).toHaveLength(1);
    expect(all[0]?.provider).toBe('openai');
  });

  it('can remove keys', async () => {
    await KeyStore.set('anthropic', { apiKey: 'anthropic-key' }, ['claude-3']);
    
    let record = await KeyStore.get('anthropic');
    expect(record).not.toBeNull();

    await KeyStore.remove('anthropic');

    record = await KeyStore.get('anthropic');
    expect(record).toBeNull();
  });
});

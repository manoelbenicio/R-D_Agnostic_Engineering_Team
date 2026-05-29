import { describe, it, expect, beforeEach } from 'vitest';
import { act, renderHook } from '@testing-library/react';
import { useValidatedProviders } from '../use-validated-providers';
import { useKeyStore } from '../store';
import { KeyStore } from '../index';

describe('useValidatedProviders', () => {
  beforeEach(async () => {
    const allRecords = await KeyStore.all();
    for (const rec of allRecords) {
      await KeyStore.remove(rec.provider);
    }
    act(() => {
      useKeyStore.setState({
        validated: [],
        statuses: {
          openai: 'unset',
          anthropic: 'unset',
          google: 'unset',
          aws: 'unset',
          azure: 'unset',
          moonshot: 'unset',
          copilot: 'unset',
          opencode: 'unset',
        },
        cachedModels: {} as any,
        maskedKeys: {} as any,
        initialized: false,
      });
    });
  });

  it('reacts to adding and removing keys', async () => {
    const { result } = renderHook(() => useValidatedProviders());

    expect(result.current).toEqual([]);

    await act(async () => {
      await useKeyStore.getState().setKey('openai', { apiKey: 'openai-key' }, ['gpt-4o']);
    });

    expect(result.current).toEqual(['openai']);

    await act(async () => {
      await useKeyStore.getState().setKey('anthropic', { apiKey: 'anthropic-key' }, ['claude-3-5-sonnet']);
    });

    expect(result.current).toEqual(['openai', 'anthropic']);

    await act(async () => {
      await useKeyStore.getState().removeKey('openai');
    });

    expect(result.current).toEqual(['anthropic']);
  });
});

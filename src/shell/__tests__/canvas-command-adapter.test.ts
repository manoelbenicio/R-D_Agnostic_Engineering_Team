/* eslint-disable agentverse/no-sideways-capability-imports --
 * Contract tests for `canvas-command-adapter`: this is the only allowed
 * spot (alongside the adapter itself) to import from `canvas-builder` and
 * `canvas-reconciler` simultaneously, since the test must reference those
 * mocked module exports through `vi.mocked(...)`.
 */
/**
 * src/shell/__tests__/canvas-command-adapter.test.ts
 *
 * Contract tests for the `canvasCommandBus` adapter: verifies it wires the
 * real canvas-builder and canvas-reconciler implementations into the
 * bus-shaped interface, and adapts the value shapes correctly (singular
 * `reason` → `reasons[]`, `CanvasProviderOption` → `ProviderOption`).
 *
 * Mocks the upstream modules so the tests don't pull the canvas subsystem
 * into the shell test boundary.
 */

import { describe, it, expect, vi, beforeEach } from 'vitest';
import type { CanvasDocument } from '@/shared/canvas-types';
import { noopCanvasCommandBus } from '@/shared/canvas-command-bus';

vi.mock('@/canvas-reconciler/reconciler', () => ({
  reconcileCanvas: vi.fn(),
}));

vi.mock('@/canvas-builder/provider-options', () => ({
  getCanvasProviderOptions: vi.fn(),
}));

vi.mock('@/canvas-builder/deploy-validation', () => ({
  validateCanvasForDeploy: vi.fn(),
}));

vi.mock('@/api/key-store/store', () => ({
  useKeyStore: { getState: vi.fn() },
}));

vi.mock('@/api/cao-client', () => ({
  caoClient: { __mock: 'cao-client-stub' },
}));

import { canvasCommandBus } from '../canvas-command-adapter';
import { reconcileCanvas } from '@/canvas-reconciler/reconciler';
import { getCanvasProviderOptions } from '@/canvas-builder/provider-options';
import { validateCanvasForDeploy } from '@/canvas-builder/deploy-validation';
import { useKeyStore } from '@/api/key-store/store';

const mockedReconcile = vi.mocked(reconcileCanvas);
const mockedGetOptions = vi.mocked(getCanvasProviderOptions);
const mockedValidate = vi.mocked(validateCanvasForDeploy);
const mockedGetState = vi.mocked(useKeyStore.getState);

function setKeyStoreState(state: {
  validated: string[];
  cachedModels: Record<string, string[]>;
}): void {
  // The adapter only reads `validated` + `cachedModels`; cast through
  // `unknown` to satisfy the full `KeyStoreState` type without rebuilding it.
  mockedGetState.mockReturnValue(
    state as unknown as ReturnType<typeof useKeyStore.getState>,
  );
}

describe('canvasCommandBus.validateForDeploy', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    setKeyStoreState({ validated: ['anthropic'], cachedModels: {} });
  });

  it('passes the canvas + provider options + validated list through to the underlying validator', () => {
    mockedGetOptions.mockReturnValue([
      { provider: 'claude_code', sourceProvider: 'anthropic', label: 'Claude Code' },
    ]);
    mockedValidate.mockReturnValue({ ok: true });

    const canvas = { id: 'c1' } as CanvasDocument;
    const result = canvasCommandBus.validateForDeploy(canvas);

    expect(result).toEqual({ ok: true, reasons: [] });
    expect(mockedGetOptions).toHaveBeenCalledWith(['anthropic']);
    expect(mockedValidate).toHaveBeenCalledWith(
      canvas,
      [{ provider: 'claude_code', sourceProvider: 'anthropic', label: 'Claude Code' }],
      ['anthropic'],
    );
  });

  it('maps singular `reason` into `reasons[]`', () => {
    mockedGetOptions.mockReturnValue([]);
    mockedValidate.mockReturnValue({ ok: false, reason: 'no entry point' });

    const result = canvasCommandBus.validateForDeploy({ id: 'c1' } as CanvasDocument);

    expect(result.ok).toBe(false);
    expect(result.reasons).toEqual(['no entry point']);
  });

  it('returns empty `reasons[]` when the validator surfaces no message', () => {
    mockedGetOptions.mockReturnValue([]);
    mockedValidate.mockReturnValue({ ok: false });

    const result = canvasCommandBus.validateForDeploy({ id: 'c1' } as CanvasDocument);

    expect(result.ok).toBe(false);
    expect(result.reasons).toEqual([]);
  });
});

describe('canvasCommandBus.reconcile', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    setKeyStoreState({ validated: [], cachedModels: {} });
  });

  it('forwards the canvas id to `reconcileCanvas` with the bound caoClient', async () => {
    const canvas = { id: 'c1' } as CanvasDocument;
    mockedReconcile.mockResolvedValue(canvas);

    const result = await canvasCommandBus.reconcile('c1');

    expect(result).toBe(canvas);
    expect(mockedReconcile).toHaveBeenCalledWith('c1', undefined, expect.anything());
  });

  it('propagates rejection from the reconciler', async () => {
    mockedReconcile.mockRejectedValue(new Error('reconcile failed'));
    await expect(canvasCommandBus.reconcile('cX')).rejects.toThrow('reconcile failed');
  });
});

describe('canvasCommandBus.getProviderOptions', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('maps `CanvasProviderOption[]` into bus `ProviderOption[]` with cached models', () => {
    setKeyStoreState({
      validated: ['anthropic'],
      cachedModels: { anthropic: ['claude-3-5-sonnet', 'claude-3-haiku'] },
    });
    mockedGetOptions.mockReturnValue([
      { provider: 'claude_code', sourceProvider: 'anthropic', label: 'Claude Code' },
    ]);

    expect(canvasCommandBus.getProviderOptions()).toEqual([
      {
        id: 'claude_code',
        label: 'Claude Code',
        models: ['claude-3-5-sonnet', 'claude-3-haiku'],
      },
    ]);
  });

  it('defaults to an empty `models` list when the source provider has none cached', () => {
    setKeyStoreState({ validated: ['openai'], cachedModels: {} });
    mockedGetOptions.mockReturnValue([
      { provider: 'codex', sourceProvider: 'openai', label: 'Codex' },
    ]);

    expect(canvasCommandBus.getProviderOptions()).toEqual([
      { id: 'codex', label: 'Codex', models: [] },
    ]);
  });

  it('returns an empty list when no providers are validated', () => {
    setKeyStoreState({ validated: [], cachedModels: {} });
    mockedGetOptions.mockReturnValue([]);

    expect(canvasCommandBus.getProviderOptions()).toEqual([]);
  });
});

describe('noopCanvasCommandBus (from @/shared/canvas-command-bus)', () => {
  it('throws on validateForDeploy so accidental usage fails loudly', () => {
    expect(() =>
      noopCanvasCommandBus.validateForDeploy({ id: 'c1' } as CanvasDocument),
    ).toThrow('noopCanvasCommandBus.validateForDeploy was called');
  });

  it('throws on reconcile', () => {
    expect(() => noopCanvasCommandBus.reconcile('c1')).toThrow(
      'noopCanvasCommandBus.reconcile was called',
    );
  });

  it('throws on getProviderOptions', () => {
    expect(() => noopCanvasCommandBus.getProviderOptions()).toThrow(
      'noopCanvasCommandBus.getProviderOptions was called',
    );
  });
});

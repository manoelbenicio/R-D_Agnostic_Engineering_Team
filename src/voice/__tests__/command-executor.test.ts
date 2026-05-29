/**
 * src/voice/__tests__/command-executor.test.ts
 *
 * Unit tests for the runtime command executor extracted from VoicePanel.
 * Covers each of the 9 actions plus the documented error branches per
 * tasks.md §2.
 *
 * Post-`tech-debt-voice-event-bus`: canvas-builder + canvas-reconciler
 * access goes through a `CanvasCommandBus` injected as `deps.bus`. Tests
 * pass a fake bus (`{ validateForDeploy, reconcile, getProviderOptions }`,
 * each a `vi.fn()`) so behaviour assertions stay identical to the prior
 * coverage profile.
 */

import { describe, it, expect, vi, beforeEach } from 'vitest';
import {
  executeRuntimeCommand,
  type CommandExecutorDeps,
  type CommandExecutorResult,
  type ToastApi,
} from '../command-executor';
import type { CanvasDocument } from '@/shared/canvas-types';
import type { CanvasCommandBus } from '@/shared/canvas-command-bus';
import type { RuntimeCommand } from '../runtime-commands';

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function makeToast(): ToastApi {
  return {
    info: vi.fn().mockReturnValue('toast-id'),
    error: vi.fn().mockReturnValue('toast-id'),
    success: vi.fn().mockReturnValue('toast-id'),
    warning: vi.fn().mockReturnValue('toast-id'),
    dismiss: vi.fn(),
  };
}

type MockCao = {
  listTerminalsInSession: ReturnType<typeof vi.fn>;
  sendTerminalInput: ReturnType<typeof vi.fn>;
  deleteTerminal: ReturnType<typeof vi.fn>;
  deleteSession: ReturnType<typeof vi.fn>;
};

interface DepsOverride {
  canvas?: CanvasDocument | null;
  confirmResult?: boolean;
  cao?: Partial<MockCao>;
  reconcile?: CanvasCommandBus['reconcile'];
  validateForDeploy?: CanvasCommandBus['validateForDeploy'];
  getProviderOptions?: CanvasCommandBus['getProviderOptions'];
  onUpdateCanvas?: CommandExecutorDeps['onUpdateCanvas'];
  speak?: CommandExecutorDeps['speak'];
  emit?: CommandExecutorDeps['emit'];
}

function makeDeps(overrides: DepsOverride = {}): {
  deps: CommandExecutorDeps;
  toast: ToastApi;
  navigate: ReturnType<typeof vi.fn>;
  confirm: ReturnType<typeof vi.fn>;
  cao: MockCao;
  bus: CanvasCommandBus;
  reconcile: ReturnType<typeof vi.fn>;
  validateForDeploy: ReturnType<typeof vi.fn>;
  getProviderOptions: ReturnType<typeof vi.fn>;
  emit: ReturnType<typeof vi.fn>;
  speak: ReturnType<typeof vi.fn>;
  onUpdateCanvas: ReturnType<typeof vi.fn>;
} {
  const toast = makeToast();
  const navigate = vi.fn();
  const confirm = vi
    .fn()
    .mockResolvedValue(overrides.confirmResult ?? true);

  const cao: MockCao = {
    listTerminalsInSession: vi.fn().mockResolvedValue([]),
    sendTerminalInput: vi.fn().mockResolvedValue(undefined),
    deleteTerminal: vi.fn().mockResolvedValue(undefined),
    deleteSession: vi.fn().mockResolvedValue(undefined),
    ...overrides.cao,
  };

  const reconcile =
    overrides.reconcile === undefined
      ? vi.fn().mockResolvedValue(makeCanvas())
      : (vi.fn(overrides.reconcile) as unknown as ReturnType<typeof vi.fn>);
  const validateForDeploy =
    overrides.validateForDeploy === undefined
      ? vi.fn().mockReturnValue({ ok: true, reasons: [] })
      : (vi.fn(overrides.validateForDeploy) as unknown as ReturnType<typeof vi.fn>);
  const getProviderOptions =
    overrides.getProviderOptions === undefined
      ? vi.fn().mockReturnValue([])
      : (vi.fn(overrides.getProviderOptions) as unknown as ReturnType<typeof vi.fn>);
  const emit = overrides.emit ? vi.fn(overrides.emit) : vi.fn();
  const speak = overrides.speak ? vi.fn(overrides.speak) : vi.fn();
  const onUpdateCanvas = overrides.onUpdateCanvas
    ? vi.fn(overrides.onUpdateCanvas)
    : vi.fn();

  const bus: CanvasCommandBus = {
    validateForDeploy:
      validateForDeploy as unknown as CanvasCommandBus['validateForDeploy'],
    reconcile: reconcile as unknown as CanvasCommandBus['reconcile'],
    getProviderOptions:
      getProviderOptions as unknown as CanvasCommandBus['getProviderOptions'],
  };

  const deps: CommandExecutorDeps = {
    canvas: overrides.canvas ?? null,
    cao: cao as unknown as CommandExecutorDeps['cao'],
    toast,
    navigate,
    confirm: confirm as unknown as CommandExecutorDeps['confirm'],
    bus,
    onUpdateCanvas: onUpdateCanvas as unknown as CommandExecutorDeps['onUpdateCanvas'],
    speak: speak as unknown as CommandExecutorDeps['speak'],
    emit: emit as unknown as CommandExecutorDeps['emit'],
  };

  return {
    deps,
    toast,
    navigate,
    confirm,
    cao,
    bus,
    reconcile,
    validateForDeploy,
    getProviderOptions,
    emit,
    speak,
    onUpdateCanvas,
  };
}

function makeCanvas(overrides?: Partial<CanvasDocument>): CanvasDocument {
  return {
    id: 'canvas-1',
    name: 'Test Canvas',
    version: 1,
    created_at: '2025-01-01T00:00:00.000Z',
    updated_at: '2025-01-01T00:00:00.000Z',
    schema_version: 1,
    nodes: [
      {
        id: 'node-supervisor',
        type: 'agent',
        position: { x: 0, y: 0 },
        data: {
          profile_name: 'supervisor',
          display_name: 'Sup Boss',
          role: 'supervisor',
          system_prompt: '',
          allowedTools: [],
          is_entry_point: true,
        },
      },
      {
        id: 'node-developer',
        type: 'agent',
        position: { x: 100, y: 100 },
        data: {
          profile_name: 'developer',
          display_name: 'Frontend Dev',
          role: 'developer',
          system_prompt: '',
          allowedTools: [],
          is_entry_point: false,
        },
      },
      {
        id: 'node-developer-2',
        type: 'agent',
        position: { x: 200, y: 100 },
        data: {
          profile_name: 'developer',
          display_name: 'Backend Dev',
          role: 'developer',
          system_prompt: '',
          allowedTools: [],
          is_entry_point: false,
        },
      },
    ],
    edges: [],
    config: {
      working_directory: '~',
      provider_default: 'claude_code',
    },
    deploy_state: {
      status: 'deployed',
      session_name: 'session-test',
      terminal_map: {
        'node-supervisor': 'term-1',
        'node-developer': 'term-2',
        'node-developer-2': 'term-3',
      },
    },
    ...overrides,
  };
}

const cmd = {
  cost: { action: 'cost' } as RuntimeCommand,
  status: { action: 'status' } as RuntimeCommand,
  deploy: { action: 'deploy' } as RuntimeCommand,
  stopAll: { action: 'stop_all' } as RuntimeCommand,
  killByRole: (value: string) =>
    ({ action: 'kill', target: { type: 'role', value } } as RuntimeCommand),
  pauseByRole: (value: string) =>
    ({ action: 'pause', target: { type: 'role', value } } as RuntimeCommand),
  focusByRole: (value: string) =>
    ({ action: 'focus', target: { type: 'role', value } } as RuntimeCommand),
  focusByName: (value: string) =>
    ({ action: 'focus', target: { type: 'name', value } } as RuntimeCommand),
  focusById: (value: string) =>
    ({ action: 'focus', target: { type: 'id', value } } as RuntimeCommand),
  addNode: { action: 'add_node', role: 'developer' } as RuntimeCommand,
  connect: {
    action: 'connect',
    source: { type: 'role', value: 'supervisor' },
    destination: { type: 'role', value: 'developer' },
  } as RuntimeCommand,
};

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe('executeRuntimeCommand — cost', () => {
  it('happy path: navigates to /finops and toasts info', async () => {
    const ctx = makeDeps();
    const result = await executeRuntimeCommand(cmd.cost, ctx.deps);
    expect(result).toEqual<CommandExecutorResult>({ action: 'cost', status: 'ok' });
    expect(ctx.toast.info).toHaveBeenCalledWith('Navigating to FinOps.');
    expect(ctx.navigate).toHaveBeenCalledWith('/finops');
  });
});

describe('executeRuntimeCommand — focus', () => {
  it('no canvas: toasts error and short-circuits', async () => {
    const ctx = makeDeps({ canvas: null });
    const result = await executeRuntimeCommand(cmd.focusByRole('supervisor'), ctx.deps);
    expect(result.status).toBe('no_canvas');
    expect(ctx.toast.error).toHaveBeenCalledWith('No active canvas open.');
    expect(ctx.navigate).not.toHaveBeenCalled();
  });

  it('canvas not deployed: toasts error and short-circuits', async () => {
    const canvas = makeCanvas({
      deploy_state: { status: 'draft' },
    });
    const ctx = makeDeps({ canvas });
    const result = await executeRuntimeCommand(cmd.focusByRole('supervisor'), ctx.deps);
    expect(result.status).toBe('not_deployed');
    expect(ctx.toast.error).toHaveBeenCalledWith('Canvas is not deployed.');
  });

  it('no target on focus command: toasts error', async () => {
    const ctx = makeDeps({ canvas: makeCanvas() });
    const result = await executeRuntimeCommand({ action: 'focus' }, ctx.deps);
    expect(result.status).toBe('no_target');
    expect(ctx.toast.error).toHaveBeenCalledWith(
      'Target not specified for focus command.',
    );
  });

  it('resolves by node id when id is in terminal_map keys', async () => {
    const ctx = makeDeps({ canvas: makeCanvas() });
    const result = await executeRuntimeCommand(
      cmd.focusById('node-developer'),
      ctx.deps,
    );
    expect(result.status).toBe('ok');
    expect(ctx.navigate).toHaveBeenCalledWith('/canvas/canvas-1/terminal/term-2');
    expect(ctx.toast.success).toHaveBeenCalled();
  });

  it('resolves directly by terminal id when value matches a terminal_map value', async () => {
    const ctx = makeDeps({ canvas: makeCanvas() });
    const result = await executeRuntimeCommand(cmd.focusById('term-3'), ctx.deps);
    expect(result.status).toBe('ok');
    expect(ctx.navigate).toHaveBeenCalledWith('/canvas/canvas-1/terminal/term-3');
    expect(ctx.toast.success).toHaveBeenCalledWith('Focused on terminal term-3');
  });

  it('resolves by role when target type is role', async () => {
    const ctx = makeDeps({ canvas: makeCanvas() });
    const result = await executeRuntimeCommand(cmd.focusByRole('supervisor'), ctx.deps);
    expect(result.status).toBe('ok');
    expect(ctx.navigate).toHaveBeenCalledWith('/canvas/canvas-1/terminal/term-1');
  });

  it('resolves by display_name (case-insensitive) when target type is name', async () => {
    const ctx = makeDeps({ canvas: makeCanvas() });
    const result = await executeRuntimeCommand(
      cmd.focusByName('frontend dev'),
      ctx.deps,
    );
    expect(result.status).toBe('ok');
    expect(ctx.navigate).toHaveBeenCalledWith('/canvas/canvas-1/terminal/term-2');
  });

  it('unknown target: toasts error and returns target_not_found', async () => {
    const ctx = makeDeps({ canvas: makeCanvas() });
    const result = await executeRuntimeCommand(cmd.focusByName('nobody'), ctx.deps);
    expect(result.status).toBe('target_not_found');
    expect(ctx.toast.error).toHaveBeenCalledWith(
      "Terminal for 'nobody' not found in session.",
    );
    expect(ctx.navigate).not.toHaveBeenCalled();
  });
});

describe('executeRuntimeCommand — status', () => {
  it('no canvas: short-circuits', async () => {
    const ctx = makeDeps({ canvas: null });
    const result = await executeRuntimeCommand(cmd.status, ctx.deps);
    expect(result.status).toBe('no_canvas');
  });

  it('no session associated: short-circuits with no_session', async () => {
    const canvas = makeCanvas({
      deploy_state: { status: 'draft' },
    });
    canvas.config = { ...canvas.config, session_name: undefined };
    const ctx = makeDeps({ canvas });
    const result = await executeRuntimeCommand(cmd.status, ctx.deps);
    expect(result.status).toBe('no_session');
    expect(ctx.toast.error).toHaveBeenCalledWith(
      'No active session associated with this canvas.',
    );
  });

  it('happy path with terminals: builds summary and speaks it', async () => {
    const ctx = makeDeps({
      canvas: makeCanvas(),
      cao: {
        listTerminalsInSession: vi.fn().mockResolvedValue([
          { id: 't1', display_name: 'Sup', status: 'active', profile: 'p', working_directory: '~' },
          { id: 't2', display_name: undefined, status: 'idle', profile: 'p', working_directory: '~' },
        ]),
      },
    });
    const result = await executeRuntimeCommand(cmd.status, ctx.deps);
    expect(result.status).toBe('ok');
    expect(ctx.cao.listTerminalsInSession).toHaveBeenCalledWith('session-test');
    expect(ctx.toast.success).toHaveBeenCalledWith(
      'Session status: Sup is active, t2 is idle',
    );
    expect(ctx.speak).toHaveBeenCalledWith(
      'Session status: Sup is active, t2 is idle',
      'en-US',
    );
  });

  it('happy path with empty list: toasts info and speaks fallback', async () => {
    const ctx = makeDeps({
      canvas: makeCanvas(),
      cao: { listTerminalsInSession: vi.fn().mockResolvedValue([]) },
    });
    const result = await executeRuntimeCommand(cmd.status, ctx.deps);
    expect(result.status).toBe('ok');
    expect(ctx.toast.info).toHaveBeenCalledWith(
      'No active terminals found in this session.',
    );
    expect(ctx.speak).toHaveBeenCalledWith(
      'No active terminals found in this session.',
    );
  });

  it('CAO failure: surfaces api_error and toasts the reason', async () => {
    const ctx = makeDeps({
      canvas: makeCanvas(),
      cao: {
        listTerminalsInSession: vi.fn().mockRejectedValue(new Error('boom')),
      },
    });
    const result = await executeRuntimeCommand(cmd.status, ctx.deps);
    expect(result.status).toBe('api_error');
    expect(result.reason).toContain('boom');
    expect(ctx.toast.error).toHaveBeenCalledWith(
      expect.stringContaining('Failed to retrieve session status'),
    );
  });

  it('uses session_name from config when deploy_state has none', async () => {
    const canvas = makeCanvas();
    canvas.deploy_state.session_name = undefined;
    canvas.config.session_name = 'cfg-session';
    const ctx = makeDeps({ canvas });
    await executeRuntimeCommand(cmd.status, ctx.deps);
    expect(ctx.cao.listTerminalsInSession).toHaveBeenCalledWith('cfg-session');
  });
});

describe('executeRuntimeCommand — deploy', () => {
  it('no canvas: short-circuits', async () => {
    const ctx = makeDeps({ canvas: null });
    const result = await executeRuntimeCommand(cmd.deploy, ctx.deps);
    expect(result.status).toBe('no_canvas');
  });

  it('canvas already deployed: short-circuits with already_deployed', async () => {
    const ctx = makeDeps({ canvas: makeCanvas() });
    const result = await executeRuntimeCommand(cmd.deploy, ctx.deps);
    expect(result.status).toBe('already_deployed');
    expect(ctx.toast.error).toHaveBeenCalledWith('Canvas is already deployed.');
  });

  it('validation failure: surfaces validation_failed and the reason', async () => {
    const canvas = makeCanvas({ deploy_state: { status: 'draft' } });
    const ctx = makeDeps({
      canvas,
      validateForDeploy: () => ({ ok: false, reasons: ['no entry point'] }),
    });
    const result = await executeRuntimeCommand(cmd.deploy, ctx.deps);
    expect(result.status).toBe('validation_failed');
    expect(result.reason).toBe('no entry point');
    expect(ctx.reconcile).not.toHaveBeenCalled();
    expect(ctx.toast.error).toHaveBeenCalledWith(
      'Validation failed: no entry point',
    );
  });

  it('happy path: validates, reconciles, toasts success, calls onUpdateCanvas', async () => {
    const canvas = makeCanvas({ deploy_state: { status: 'draft' } });
    const updated = makeCanvas({ id: 'canvas-1', deploy_state: { status: 'deployed' } });
    const ctx = makeDeps({
      canvas,
      validateForDeploy: () => ({ ok: true, reasons: [] }),
      reconcile: vi.fn().mockResolvedValue(updated),
    });
    const result = await executeRuntimeCommand(cmd.deploy, ctx.deps);
    expect(result.status).toBe('ok');
    expect(ctx.reconcile).toHaveBeenCalledWith('canvas-1');
    expect(ctx.toast.info).toHaveBeenCalledWith('Starting deployment via voice...');
    expect(ctx.toast.success).toHaveBeenCalledWith(
      'Materialization completed successfully!',
    );
    expect(ctx.onUpdateCanvas).toHaveBeenCalledTimes(1);
    // Verify the updater closure swaps in the next canvas
    const updater = ctx.onUpdateCanvas.mock.calls[0]?.[0] as
      | ((c: CanvasDocument) => CanvasDocument)
      | undefined;
    expect(updater).toBeTypeOf('function');
    expect(updater?.(canvas)).toBe(updated);
  });

  it('reconcile rejection: surfaces api_error and toasts', async () => {
    const canvas = makeCanvas({ deploy_state: { status: 'draft' } });
    const ctx = makeDeps({
      canvas,
      validateForDeploy: () => ({ ok: true, reasons: [] }),
      reconcile: vi.fn().mockRejectedValue(new Error('cao down')),
    });
    const result = await executeRuntimeCommand(cmd.deploy, ctx.deps);
    expect(result.status).toBe('api_error');
    expect(result.reason).toContain('cao down');
    expect(ctx.toast.error).toHaveBeenCalledWith(
      expect.stringContaining('Deployment failed'),
    );
  });
});

describe('executeRuntimeCommand — pause', () => {
  it('no canvas: short-circuits', async () => {
    const ctx = makeDeps({ canvas: null });
    const result = await executeRuntimeCommand(cmd.pauseByRole('supervisor'), ctx.deps);
    expect(result.status).toBe('no_canvas');
  });

  it('not deployed: short-circuits', async () => {
    const ctx = makeDeps({
      canvas: makeCanvas({ deploy_state: { status: 'draft' } }),
    });
    const result = await executeRuntimeCommand(cmd.pauseByRole('supervisor'), ctx.deps);
    expect(result.status).toBe('not_deployed');
  });

  it('no target: short-circuits', async () => {
    const ctx = makeDeps({ canvas: makeCanvas() });
    const result = await executeRuntimeCommand({ action: 'pause' }, ctx.deps);
    expect(result.status).toBe('no_target');
  });

  it('unknown target: target_not_found', async () => {
    const ctx = makeDeps({ canvas: makeCanvas() });
    const result = await executeRuntimeCommand(
      { action: 'pause', target: { type: 'name', value: 'nobody' } },
      ctx.deps,
    );
    expect(result.status).toBe('target_not_found');
    expect(ctx.cao.sendTerminalInput).not.toHaveBeenCalled();
  });

  it('happy path: sends ctrl+S byte to the resolved terminal', async () => {
    const ctx = makeDeps({ canvas: makeCanvas() });
    const result = await executeRuntimeCommand(cmd.pauseByRole('developer'), ctx.deps);
    expect(result.status).toBe('ok');
    expect(ctx.cao.sendTerminalInput).toHaveBeenCalledWith('term-2', '\x13');
    expect(ctx.toast.info).toHaveBeenCalledWith(
      'Sent pause signal to terminal for developer',
    );
  });

  it('CAO failure: surfaces api_error', async () => {
    const ctx = makeDeps({
      canvas: makeCanvas(),
      cao: {
        sendTerminalInput: vi.fn().mockRejectedValue(new Error('net err')),
      },
    });
    const result = await executeRuntimeCommand(cmd.pauseByRole('developer'), ctx.deps);
    expect(result.status).toBe('api_error');
    expect(ctx.toast.error).toHaveBeenCalledWith(
      expect.stringContaining('Failed to send pause signal'),
    );
  });
});

describe('executeRuntimeCommand — add_node / connect', () => {
  beforeEach(() => {
    // Make sure no leftover listeners between tests.
  });

  it('add_node: forwards to deps.emit when provided', async () => {
    const ctx = makeDeps();
    const result = await executeRuntimeCommand(cmd.addNode, ctx.deps);
    expect(result.status).toBe('ok');
    expect(ctx.emit).toHaveBeenCalledWith('add_node', cmd.addNode);
  });

  it('connect: forwards to deps.emit when provided', async () => {
    const ctx = makeDeps();
    const result = await executeRuntimeCommand(cmd.connect, ctx.deps);
    expect(result.status).toBe('ok');
    expect(ctx.emit).toHaveBeenCalledWith('connect', cmd.connect);
  });

  it('falls back to window.dispatchEvent when emit is not provided', async () => {
    const ctx = makeDeps();
    const deps: CommandExecutorDeps = { ...ctx.deps, emit: undefined };
    const listener = vi.fn();
    window.addEventListener('voice-canvas-add_node', listener as EventListener);
    try {
      const result = await executeRuntimeCommand(cmd.addNode, deps);
      expect(result.status).toBe('ok');
      expect(listener).toHaveBeenCalledTimes(1);
      const evt = listener.mock.calls[0]?.[0] as CustomEvent;
      expect(evt.detail).toEqual(cmd.addNode);
    } finally {
      window.removeEventListener('voice-canvas-add_node', listener as EventListener);
    }
  });
});

describe('executeRuntimeCommand — kill', () => {
  it('no canvas: short-circuits', async () => {
    const ctx = makeDeps({ canvas: null });
    const result = await executeRuntimeCommand(cmd.killByRole('supervisor'), ctx.deps);
    expect(result.status).toBe('no_canvas');
    expect(ctx.confirm).not.toHaveBeenCalled();
  });

  it('not deployed: short-circuits without confirming', async () => {
    const ctx = makeDeps({
      canvas: makeCanvas({ deploy_state: { status: 'draft' } }),
    });
    const result = await executeRuntimeCommand(cmd.killByRole('supervisor'), ctx.deps);
    expect(result.status).toBe('not_deployed');
    expect(ctx.confirm).not.toHaveBeenCalled();
  });

  it('no target: short-circuits without confirming', async () => {
    const ctx = makeDeps({ canvas: makeCanvas() });
    const result = await executeRuntimeCommand({ action: 'kill' }, ctx.deps);
    expect(result.status).toBe('no_target');
    expect(ctx.confirm).not.toHaveBeenCalled();
  });

  it('unknown target: short-circuits with target_not_found and no confirm', async () => {
    const ctx = makeDeps({ canvas: makeCanvas() });
    const result = await executeRuntimeCommand(
      { action: 'kill', target: { type: 'name', value: 'nobody' } },
      ctx.deps,
    );
    expect(result.status).toBe('target_not_found');
    expect(ctx.confirm).not.toHaveBeenCalled();
  });

  it('user cancels confirmation: short-circuits, no API call', async () => {
    const ctx = makeDeps({ canvas: makeCanvas(), confirmResult: false });
    const result = await executeRuntimeCommand(cmd.killByRole('developer'), ctx.deps);
    expect(result.status).toBe('cancelled');
    expect(ctx.confirm).toHaveBeenCalledWith(
      expect.objectContaining({ title: 'Confirm Kill Terminal' }),
    );
    expect(ctx.cao.deleteTerminal).not.toHaveBeenCalled();
    expect(ctx.onUpdateCanvas).not.toHaveBeenCalled();
  });

  it('happy path: confirms, deletes, updates canvas to degraded', async () => {
    const ctx = makeDeps({ canvas: makeCanvas(), confirmResult: true });
    const result = await executeRuntimeCommand(cmd.killByRole('developer'), ctx.deps);
    expect(result.status).toBe('ok');
    expect(ctx.confirm).toHaveBeenCalled();
    expect(ctx.cao.deleteTerminal).toHaveBeenCalledWith('term-2');
    expect(ctx.toast.success).toHaveBeenCalledWith(
      "Terminal for 'Frontend Dev' has been killed.",
    );
    expect(ctx.onUpdateCanvas).toHaveBeenCalledTimes(1);

    const updater = ctx.onUpdateCanvas.mock.calls[0]?.[0] as (
      c: CanvasDocument,
    ) => CanvasDocument;
    const next = updater(makeCanvas());
    expect(next.deploy_state.terminal_map).toEqual({
      'node-supervisor': 'term-1',
      'node-developer-2': 'term-3',
    });
    expect(next.deploy_state.status).toBe('degraded');
  });

  it('happy path: when last terminal removed, status reverts to draft', async () => {
    const canvas = makeCanvas({
      deploy_state: {
        status: 'deployed',
        session_name: 'session-test',
        terminal_map: { 'node-developer': 'term-2' },
      },
    });
    const ctx = makeDeps({ canvas, confirmResult: true });
    const result = await executeRuntimeCommand(cmd.killByRole('developer'), ctx.deps);
    expect(result.status).toBe('ok');

    const updater = ctx.onUpdateCanvas.mock.calls[0]?.[0] as (
      c: CanvasDocument,
    ) => CanvasDocument;
    const next = updater(canvas);
    expect(next.deploy_state.status).toBe('draft');
    expect(next.deploy_state.terminal_map).toEqual({});
  });

  it('CAO deletion failure: surfaces api_error', async () => {
    const ctx = makeDeps({
      canvas: makeCanvas(),
      cao: { deleteTerminal: vi.fn().mockRejectedValue(new Error('cant kill')) },
    });
    const result = await executeRuntimeCommand(cmd.killByRole('developer'), ctx.deps);
    expect(result.status).toBe('api_error');
    expect(ctx.toast.error).toHaveBeenCalledWith(
      expect.stringContaining('Failed to kill terminal'),
    );
    expect(ctx.onUpdateCanvas).not.toHaveBeenCalled();
  });
});

describe('executeRuntimeCommand — stop_all', () => {
  it('no canvas: short-circuits', async () => {
    const ctx = makeDeps({ canvas: null });
    const result = await executeRuntimeCommand(cmd.stopAll, ctx.deps);
    expect(result.status).toBe('no_canvas');
    expect(ctx.confirm).not.toHaveBeenCalled();
  });

  it('no session_name and no config session_name: no_session', async () => {
    const canvas = makeCanvas();
    canvas.deploy_state.session_name = undefined;
    canvas.config.session_name = undefined;
    const ctx = makeDeps({ canvas });
    const result = await executeRuntimeCommand(cmd.stopAll, ctx.deps);
    expect(result.status).toBe('no_session');
    expect(ctx.confirm).not.toHaveBeenCalled();
  });

  it('user cancels: short-circuits without delete', async () => {
    const ctx = makeDeps({ canvas: makeCanvas(), confirmResult: false });
    const result = await executeRuntimeCommand(cmd.stopAll, ctx.deps);
    expect(result.status).toBe('cancelled');
    expect(ctx.cao.deleteSession).not.toHaveBeenCalled();
  });

  it('confirmation message includes terminal count and session name', async () => {
    const ctx = makeDeps({ canvas: makeCanvas(), confirmResult: false });
    await executeRuntimeCommand(cmd.stopAll, ctx.deps);
    expect(ctx.confirm).toHaveBeenCalledWith(
      expect.objectContaining({
        title: 'Confirm Stop All',
        message: expect.stringContaining(
          "kill 3 terminals and tear down the CAO session 'session-test'",
        ),
      }),
    );
  });

  it('falls back to nodes.length when terminal_map is empty', async () => {
    const canvas = makeCanvas({
      deploy_state: {
        status: 'draft',
        session_name: 'session-test',
        terminal_map: {},
      },
    });
    const ctx = makeDeps({ canvas, confirmResult: false });
    await executeRuntimeCommand(cmd.stopAll, ctx.deps);
    expect(ctx.confirm).toHaveBeenCalledWith(
      expect.objectContaining({
        message: expect.stringContaining('kill 3 terminals'),
      }),
    );
  });

  it('happy path: confirms, deletes session, updates canvas to draft', async () => {
    const ctx = makeDeps({ canvas: makeCanvas(), confirmResult: true });
    const result = await executeRuntimeCommand(cmd.stopAll, ctx.deps);
    expect(result.status).toBe('ok');
    expect(ctx.cao.deleteSession).toHaveBeenCalledWith('session-test');
    expect(ctx.toast.success).toHaveBeenCalledWith('Session torn down successfully.');
    const updater = ctx.onUpdateCanvas.mock.calls[0]?.[0] as (
      c: CanvasDocument,
    ) => CanvasDocument;
    const next = updater(makeCanvas());
    expect(next.deploy_state.status).toBe('draft');
  });

  it('CAO failure: surfaces api_error', async () => {
    const ctx = makeDeps({
      canvas: makeCanvas(),
      cao: { deleteSession: vi.fn().mockRejectedValue(new Error('teardown failed')) },
    });
    const result = await executeRuntimeCommand(cmd.stopAll, ctx.deps);
    expect(result.status).toBe('api_error');
    expect(ctx.toast.error).toHaveBeenCalledWith(
      expect.stringContaining('Failed to tear down session'),
    );
  });
});

describe('executeRuntimeCommand — exhaustiveness', () => {
  it('returns ok for every documented action when given a sane setup', async () => {
    const baseline = () =>
      makeDeps({
        canvas: makeCanvas({ deploy_state: { status: 'draft' } }),
        validateForDeploy: () => ({ ok: true, reasons: [] }),
        reconcile: vi.fn().mockResolvedValue(makeCanvas()),
      });

    // cost
    expect((await executeRuntimeCommand(cmd.cost, baseline().deps)).status).toBe(
      'ok',
    );
    // deploy (canvas in draft)
    expect((await executeRuntimeCommand(cmd.deploy, baseline().deps)).status).toBe(
      'ok',
    );
    // status, focus, pause, kill, stop_all want a deployed canvas
    const deployedCtx = () => makeDeps({ canvas: makeCanvas() });
    expect((await executeRuntimeCommand(cmd.status, deployedCtx().deps)).status).toBe(
      'ok',
    );
    expect(
      (await executeRuntimeCommand(cmd.focusByRole('supervisor'), deployedCtx().deps))
        .status,
    ).toBe('ok');
    expect(
      (await executeRuntimeCommand(cmd.pauseByRole('developer'), deployedCtx().deps))
        .status,
    ).toBe('ok');
    expect(
      (await executeRuntimeCommand(cmd.killByRole('developer'), deployedCtx().deps))
        .status,
    ).toBe('ok');
    expect((await executeRuntimeCommand(cmd.stopAll, deployedCtx().deps)).status).toBe(
      'ok',
    );
    // add_node, connect
    expect((await executeRuntimeCommand(cmd.addNode, baseline().deps)).status).toBe(
      'ok',
    );
    expect((await executeRuntimeCommand(cmd.connect, baseline().deps)).status).toBe(
      'ok',
    );
  });
});

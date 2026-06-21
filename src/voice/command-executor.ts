/**
 * src/voice/command-executor.ts
 *
 * Pure (no React) executor for parsed runtime voice commands.
 *
 * Responsibilities:
 *   - Translate a `RuntimeCommand` into the appropriate side-effect (toast,
 *     GO Core API call, navigation, deploy reconcile, in-place canvas mutation,
 *     UI-bridge event).
 *   - Surface every side-effect through the injected `CommandExecutorDeps`
 *     so the 9-action × N-error-path matrix is fully unit-testable without
 *     rendering a React tree.
 *
 * Architecture note (Wave B):
 *   - `tech-debt-voice-coverage-gap` extracted this executor from VoicePanel.
 *   - `tech-debt-voice-event-bus` (this change) routed canvas-builder +
 *     canvas-reconciler access through `CanvasCommandBus`
 *     (`@/shared/canvas-command-bus`), so this file imports only from
 *     `@/api`, `@/shared`, and the sibling voice module — all allowed by
 *     the architectural lint rule. No directive is required.
 */

import type { GoCoreClient } from '@/api';
import type { CanvasDocument } from '@/shared/canvas-types';
import type { CanvasCommandBus } from '@/shared/canvas-command-bus';

import type { RuntimeCommand } from './runtime-commands';

// ---------------------------------------------------------------------------
// Public contract types
// ---------------------------------------------------------------------------

/** Subset of `useToast()` consumed by the executor. */
export interface ToastApi {
  info: (message: string, duration?: number) => string;
  error: (message: string, duration?: number) => string;
  success: (message: string, duration?: number) => string;
  warning: (message: string, duration?: number) => string;
  dismiss: (id: string) => void;
}

/** Navigation hook signature (e.g. react-router-dom's `useNavigate()` return value). */
export type NavigateFn = (path: string) => void;

/** Options for the destructive-action confirmation prompt. */
export interface ConfirmOptions {
  title: string;
  message: string;
}

/**
 * Subset of `GoCoreClient` used by the executor. Tests pass a partial mock that
 * only implements these four methods.
 */
export type CommandExecutorGoCoreClient = Pick<
  GoCoreClient,
  'listTerminalsInSession' | 'sendTerminalInput' | 'deleteTerminal' | 'deleteSession'
>;

/**
 * Pure dependencies for the runtime command executor.
 *
 * Every side-effect previously embedded in `VoicePanel.handleExecuteCommand`
 * is reachable through this interface. After `tech-debt-voice-event-bus`,
 * canvas-builder + canvas-reconciler access goes through `bus`; the other
 * fields remain explicit so each side-effect is still individually
 * mockable in unit tests.
 */
export interface CommandExecutorDeps {
  /** Current canvas snapshot, or `null` when no canvas is open. */
  canvas: CanvasDocument | null;
  /** GO Core client (subset). */
  goCore: CommandExecutorGoCoreClient;
  /** Toast API; same shape as `useToast()`. */
  toast: ToastApi;
  /** Navigation callback. */
  navigate: NavigateFn;
  /** Async confirmation prompt; resolves `true` on confirm, `false` on cancel. */
  confirm: (opts: ConfirmOptions) => Promise<boolean>;
  /**
   * Canvas-domain bus exposing `validateForDeploy` / `reconcile` /
   * `getProviderOptions`. Replaces the prior direct `reconcile` and
   * `validateForDeploy` closures so this file no longer imports from
   * `canvas-builder` or `canvas-reconciler`.
   */
  bus: CanvasCommandBus;
  /** Optional in-place canvas mutator used by `kill` and `stop_all`. */
  onUpdateCanvas?: (updater: (current: CanvasDocument) => CanvasDocument) => void;
  /**
   * Optional speech-synthesis hook for `status`. Defaults to a no-op (used
   * by tests). VoicePanel injects
   * `(text, lang) => window.speechSynthesis.speak(...)`.
   */
  speak?: (text: string, lang?: string) => void;
  /**
   * Optional UI-bridge for `add_node` and `connect`. Tests pass a spy;
   * VoicePanel forwards via
   * `window.dispatchEvent(new CustomEvent('voice-canvas-' + event, ...))`.
   */
  emit?: (event: 'add_node' | 'connect', command: RuntimeCommand) => void;
}

/** Outcome status surfaced by the executor — covers happy and failure paths. */
export type CommandExecutorStatus =
  | 'ok'
  | 'no_canvas'
  | 'not_deployed'
  | 'no_session'
  | 'no_target'
  | 'target_not_found'
  | 'already_deployed'
  | 'validation_failed'
  | 'cancelled'
  | 'api_error'
  | 'unknown_action';

/** Structured result returned by the executor. */
export interface CommandExecutorResult {
  /** Action that was processed. */
  action: RuntimeCommand['action'];
  /** Outcome — `'ok'` on success, otherwise a precise failure code. */
  status: CommandExecutorStatus;
  /** Optional human reason; populated for `api_error` and `validation_failed`. */
  reason?: string;
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

interface ResolvedTarget {
  /** Canvas node id (empty string when target could not be resolved). */
  nodeId: string;
  /** Set when `target.type === 'id'` and the value matches an existing terminal id. */
  directTerminalId?: string;
}

/**
 * Resolve a runtime-command target (id / role / name) to a node id and, when
 * the target is itself a terminal id, surface that directly so callers can
 * navigate without an extra lookup.
 */
function resolveTargetNode(
  canvas: CanvasDocument,
  target: NonNullable<RuntimeCommand['target']>,
  terminalMap: Record<string, string>,
): ResolvedTarget {
  if (target.type === 'id') {
    const byTerminal = Object.entries(terminalMap).find(
      ([, termId]) => termId === target.value,
    );
    if (byTerminal) {
      return { nodeId: byTerminal[0], directTerminalId: target.value };
    }
    if (terminalMap[target.value]) {
      return { nodeId: target.value };
    }
    return { nodeId: '' };
  }
  if (target.type === 'role') {
    const node = canvas.nodes.find((n) => n.data.role === target.value);
    return { nodeId: node?.id ?? '' };
  }
  // type === 'name'
  const lower = target.value.toLowerCase();
  const node = canvas.nodes.find(
    (n) => n.data.display_name.toLowerCase() === lower,
  );
  return { nodeId: node?.id ?? '' };
}

function ok(action: RuntimeCommand['action']): CommandExecutorResult {
  return { action, status: 'ok' };
}

function fail(
  action: RuntimeCommand['action'],
  status: CommandExecutorStatus,
  reason?: string,
): CommandExecutorResult {
  return reason !== undefined ? { action, status, reason } : { action, status };
}

// ---------------------------------------------------------------------------
// Public entry point
// ---------------------------------------------------------------------------

/**
 * Execute a parsed `RuntimeCommand` against the injected dependencies.
 *
 * Always resolves with a `CommandExecutorResult`; never throws (errors from
 * GO Core calls are surfaced via `status: 'api_error'` and the matching
 * `toast.error(...)` call).
 */
export async function executeRuntimeCommand(
  command: RuntimeCommand,
  deps: CommandExecutorDeps,
): Promise<CommandExecutorResult> {
  switch (command.action) {
    case 'cost':
      return executeCost(command, deps);
    case 'focus':
      return executeFocus(command, deps);
    case 'status':
      return executeStatus(command, deps);
    case 'deploy':
      return executeDeploy(command, deps);
    case 'pause':
      return executePause(command, deps);
    case 'add_node':
      return executeEmit(command, 'add_node', deps);
    case 'connect':
      return executeEmit(command, 'connect', deps);
    case 'kill':
      return executeKill(command, deps);
    case 'stop_all':
      return executeStopAll(command, deps);
    default: {
      // Exhaustiveness guard — any new action added to RuntimeCommand will
      // fail to compile here.
      const _exhaustive: never = command.action;
      void _exhaustive;
      return fail(command.action, 'unknown_action');
    }
  }
}

// ---------------------------------------------------------------------------
// Per-action implementations
// ---------------------------------------------------------------------------

async function executeCost(
  command: RuntimeCommand,
  deps: CommandExecutorDeps,
): Promise<CommandExecutorResult> {
  deps.toast.info('Navigating to FinOps.');
  deps.navigate('/finops');
  return ok(command.action);
}

async function executeFocus(
  command: RuntimeCommand,
  deps: CommandExecutorDeps,
): Promise<CommandExecutorResult> {
  if (!deps.canvas) {
    deps.toast.error('No active canvas open.');
    return fail(command.action, 'no_canvas');
  }
  const terminalMap = deps.canvas.deploy_state?.terminal_map;
  if (!terminalMap) {
    deps.toast.error('Canvas is not deployed.');
    return fail(command.action, 'not_deployed');
  }
  const target = command.target;
  if (!target) {
    deps.toast.error('Target not specified for focus command.');
    return fail(command.action, 'no_target');
  }

  const resolved = resolveTargetNode(deps.canvas, target, terminalMap);

  // When the target is literally an existing terminal id, navigate directly
  // (preserves the original VoicePanel behaviour where `focus on <terminal_id>`
  // skipped the node-lookup step).
  if (resolved.directTerminalId) {
    deps.navigate(`/canvas/${deps.canvas.id}/terminal/${resolved.directTerminalId}`);
    deps.toast.success(`Focused on terminal ${resolved.directTerminalId}`);
    return ok(command.action);
  }

  const terminalId = resolved.nodeId ? terminalMap[resolved.nodeId] : undefined;
  if (terminalId) {
    deps.navigate(`/canvas/${deps.canvas.id}/terminal/${terminalId}`);
    deps.toast.success(`Focused on terminal for ${target.value}`);
    return ok(command.action);
  }

  deps.toast.error(`Terminal for '${target.value}' not found in session.`);
  return fail(command.action, 'target_not_found');
}

async function executeStatus(
  command: RuntimeCommand,
  deps: CommandExecutorDeps,
): Promise<CommandExecutorResult> {
  if (!deps.canvas) {
    deps.toast.error('No active canvas open.');
    return fail(command.action, 'no_canvas');
  }
  const sessionName =
    deps.canvas.deploy_state?.session_name || deps.canvas.config?.session_name;
  if (!sessionName) {
    deps.toast.error('No active session associated with this canvas.');
    return fail(command.action, 'no_session');
  }

  try {
    const terminals = await deps.goCore.listTerminalsInSession(sessionName);
    if (terminals.length === 0) {
      const text = 'No active terminals found in this session.';
      deps.toast.info(text);
      deps.speak?.(text);
    } else {
      const summary = terminals
        .map((t) => `${t.display_name || t.id} is ${t.status}`)
        .join(', ');
      const text = `Session status: ${summary}`;
      deps.toast.success(text);
      deps.speak?.(text, 'en-US');
    }
    return ok(command.action);
  } catch (err) {
    const reason = String(err);
    deps.toast.error('Failed to retrieve session status: ' + reason);
    return fail(command.action, 'api_error', reason);
  }
}

async function executeDeploy(
  command: RuntimeCommand,
  deps: CommandExecutorDeps,
): Promise<CommandExecutorResult> {
  if (!deps.canvas) {
    deps.toast.error('No active canvas open.');
    return fail(command.action, 'no_canvas');
  }
  if (deps.canvas.deploy_state?.status !== 'draft') {
    deps.toast.error('Canvas is already deployed.');
    return fail(command.action, 'already_deployed');
  }

  const validation = deps.bus.validateForDeploy(deps.canvas);
  if (!validation.ok) {
    const reason = validation.reasons.join('; ');
    deps.toast.error(`Validation failed: ${reason}`);
    return fail(command.action, 'validation_failed', reason);
  }

  deps.toast.info('Starting deployment via voice...');
  try {
    const next = await deps.bus.reconcile(deps.canvas.id);
    deps.toast.success('Materialization completed successfully!');
    deps.onUpdateCanvas?.(() => next);
    return ok(command.action);
  } catch (err) {
    const reason = String(err);
    deps.toast.error('Deployment failed: ' + reason);
    return fail(command.action, 'api_error', reason);
  }
}

async function executePause(
  command: RuntimeCommand,
  deps: CommandExecutorDeps,
): Promise<CommandExecutorResult> {
  if (!deps.canvas) {
    deps.toast.error('No active canvas open.');
    return fail(command.action, 'no_canvas');
  }
  const terminalMap = deps.canvas.deploy_state?.terminal_map;
  if (!terminalMap) {
    deps.toast.error('Canvas is not deployed.');
    return fail(command.action, 'not_deployed');
  }
  const target = command.target;
  if (!target) {
    deps.toast.error('Target not specified for pause command.');
    return fail(command.action, 'no_target');
  }

  const resolved = resolveTargetNode(deps.canvas, target, terminalMap);
  const terminalId =
    resolved.directTerminalId ?? (resolved.nodeId ? terminalMap[resolved.nodeId] : undefined);
  if (!terminalId) {
    deps.toast.error(`Terminal for '${target.value}' not found in session.`);
    return fail(command.action, 'target_not_found');
  }

  try {
    // Ctrl+S — preserves the original VoicePanel signal byte for "pause".
    await deps.goCore.sendTerminalInput(terminalId, '\x13');
    deps.toast.info(`Sent pause signal to terminal for ${target.value}`);
    return ok(command.action);
  } catch (err) {
    const reason = String(err);
    deps.toast.error('Failed to send pause signal: ' + reason);
    return fail(command.action, 'api_error', reason);
  }
}

async function executeEmit(
  command: RuntimeCommand,
  event: 'add_node' | 'connect',
  deps: CommandExecutorDeps,
): Promise<CommandExecutorResult> {
  if (deps.emit) {
    deps.emit(event, command);
  } else if (typeof window !== 'undefined' && typeof CustomEvent !== 'undefined') {
    window.dispatchEvent(
      new CustomEvent(`voice-canvas-${event}`, { detail: command }),
    );
  }
  return ok(command.action);
}

async function executeKill(
  command: RuntimeCommand,
  deps: CommandExecutorDeps,
): Promise<CommandExecutorResult> {
  if (!deps.canvas) {
    deps.toast.error('No active canvas open.');
    return fail(command.action, 'no_canvas');
  }
  const terminalMap = deps.canvas.deploy_state?.terminal_map;
  if (!terminalMap) {
    deps.toast.error('Canvas is not deployed.');
    return fail(command.action, 'not_deployed');
  }
  const target = command.target;
  if (!target) {
    deps.toast.error('Target not specified for kill command.');
    return fail(command.action, 'no_target');
  }

  const resolved = resolveTargetNode(deps.canvas, target, terminalMap);
  const nodeId = resolved.nodeId;
  const terminalId =
    resolved.directTerminalId ?? (nodeId ? terminalMap[nodeId] : undefined);
  if (!terminalId || !nodeId) {
    deps.toast.error(`Terminal for '${target.value}' not found in session.`);
    return fail(command.action, 'target_not_found');
  }

  const nodeName =
    deps.canvas.nodes.find((n) => n.id === nodeId)?.data.display_name ?? target.value;

  const confirmed = await deps.confirm({
    title: 'Confirm Kill Terminal',
    message:
      `Are you sure you want to kill the terminal for agent '${nodeName}'? ` +
      `This will terminate the agent process immediately.`,
  });
  if (!confirmed) {
    return fail(command.action, 'cancelled');
  }

  deps.toast.info(`Killing terminal for ${nodeName}...`);
  try {
    await deps.goCore.deleteTerminal(terminalId);
    deps.toast.success(`Terminal for '${nodeName}' has been killed.`);
    deps.onUpdateCanvas?.((current) => {
      const nextMap = { ...(current.deploy_state.terminal_map ?? {}) };
      delete nextMap[nodeId];
      const nextStatus: CanvasDocument['deploy_state']['status'] =
        Object.keys(nextMap).length === 0 ? 'draft' : 'degraded';
      return {
        ...current,
        deploy_state: {
          ...current.deploy_state,
          status: nextStatus,
          terminal_map: nextMap,
        },
      };
    });
    return ok(command.action);
  } catch (err) {
    const reason = String(err);
    deps.toast.error('Failed to kill terminal: ' + reason);
    return fail(command.action, 'api_error', reason);
  }
}

async function executeStopAll(
  command: RuntimeCommand,
  deps: CommandExecutorDeps,
): Promise<CommandExecutorResult> {
  if (!deps.canvas) {
    deps.toast.error('No active canvas open.');
    return fail(command.action, 'no_canvas');
  }
  const sessionName =
    deps.canvas.deploy_state?.session_name || deps.canvas.config?.session_name;
  if (!sessionName) {
    deps.toast.error('No active session found.');
    return fail(command.action, 'no_session');
  }

  let activeCount = Object.keys(deps.canvas.deploy_state?.terminal_map ?? {}).length;
  if (activeCount === 0) {
    activeCount = deps.canvas.nodes.length;
  }

  const confirmed = await deps.confirm({
    title: 'Confirm Stop All',
    message:
      `Confirm stop all? This will kill ${activeCount} terminals and tear down ` +
      `the GO Core session '${sessionName}'.`,
  });
  if (!confirmed) {
    return fail(command.action, 'cancelled');
  }

  deps.toast.info(`Stopping all terminals for session '${sessionName}'...`);
  try {
    await deps.goCore.deleteSession(sessionName);
    deps.toast.success('Session torn down successfully.');
    deps.onUpdateCanvas?.((current) => ({
      ...current,
      deploy_state: { status: 'draft' },
    }));
    return ok(command.action);
  } catch (err) {
    const reason = String(err);
    deps.toast.error('Failed to tear down session: ' + reason);
    return fail(command.action, 'api_error', reason);
  }
}

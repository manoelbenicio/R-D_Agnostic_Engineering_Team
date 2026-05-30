/**
 * src/shared/validation-proxy.ts
 *
 * SPA-side proxy layer for the `validation-proxy` change (task §2). Wraps the
 * pure `topology-guard` so a single deployed `CanvasDocument` can be turned
 * into an enforcement seam that decides allow/deny for an agent→agent
 * `handoff` / `assign` / `send_message`, and emits an auditable record on
 * every violation.
 *
 * ─────────────────────────────────────────────────────────────────────────
 * HUMAN DECISION POINT — INSTALL POINT (UNRESOLVED, blocks task §2.1/§2.3)
 * ─────────────────────────────────────────────────────────────────────────
 * This module is the install-point-agnostic *decision core*. Where it is
 * actually invoked to *prevent* (not merely observe) a violation is a human
 * architecture decision that folds into `cloud-runtime-deployment`:
 *
 *   (a) CAO middleware — intercept orchestration inside CAO before the call
 *       executes. Strongest enforcement; BLOCKED: CAO source is not in this
 *       repo, so the actual interception patch cannot be written here.
 *   (b) Sidecar proxy — a process in front of CAO's orchestration endpoints
 *       that calls `guardOrchestration` and returns 4xx on deny. Needs the
 *       cloud-runtime auth-proxy to exist first.
 *   (c) SPA-side only — observe/audit. CANNOT prevent: agent→agent calls
 *       happen inside CAO, not through the SPA's HTTP surface (the SPA's
 *       `sendInboxMessage` / `sendTerminalInput` are user→terminal). SPA-side
 *       use is defense-in-depth + UI surfacing only.
 *
 * Do NOT invent CAO server code here. Until (a)/(b) is chosen and the cloud
 * runtime lands, this remains the unit-tested core that all three options
 * call unchanged.
 *
 * Pure + dependency-free (only `@/shared` types + `topology-guard`).
 */

import type { CanvasDocument } from './canvas-types';
import {
  buildCanvasTopology,
  validateOrchestrationCall,
  type CanvasTopology,
  type OrchestrationCall,
  type TopologyViolationCode,
} from './topology-guard';

/** Allow/deny decision returned to the caller. */
export interface OrchestrationDecision {
  allowed: boolean;
  reason?: string;
  code?: TopologyViolationCode;
}

/** Auditable record emitted whenever a call is denied. */
export interface ViolationRecord {
  timestamp: string;
  canvasId: string;
  code: TopologyViolationCode;
  reason: string;
  /** The original call as received (pre terminal_map resolution). */
  call: OrchestrationCall;
  /** Identities after terminal_map resolution (what the guard saw). */
  resolved: { source: string; target: string };
}

export interface ValidationProxyOptions {
  /** Audit sink fired on every denial (logs / health panel). */
  onViolation?: (record: ViolationRecord) => void;
  /** Injected clock for deterministic tests. */
  now?: () => string;
}

export interface ValidationProxy {
  guardOrchestration(call: OrchestrationCall): OrchestrationDecision;
}

/**
 * Build a proxy bound to one deployed canvas.
 *
 * `deploy_state.terminal_map` maps `node_id → terminal_id`; CAO emits calls
 * referencing terminal ids, so we invert it and resolve any terminal id back
 * to its node id before delegating to the topology guard (task §2.2).
 */
export function createValidationProxy(
  canvas: CanvasDocument,
  options: ValidationProxyOptions = {},
): ValidationProxy {
  const now = options.now ?? (() => new Date().toISOString());
  const topology: CanvasTopology = buildCanvasTopology(canvas);

  // terminal_id → node_id, so a call addressed by terminal resolves to a node.
  const terminalToNode = new Map<string, string>();
  for (const [nodeId, terminalId] of Object.entries(
    canvas.deploy_state.terminal_map ?? {},
  )) {
    terminalToNode.set(terminalId, nodeId);
  }

  const resolve = (identity: string): string =>
    terminalToNode.get(identity) ?? identity;

  return {
    guardOrchestration(call) {
      const resolved = {
        source: resolve(call.source),
        target: resolve(call.target),
      };
      const result = validateOrchestrationCall(topology, {
        action: call.action,
        ...resolved,
      });

      if (!result.ok) {
        options.onViolation?.({
          timestamp: now(),
          canvasId: topology.canvasId,
          code: result.code!,
          reason: result.reason!,
          call,
          resolved,
        });
      }

      return { allowed: result.ok, reason: result.reason, code: result.code };
    },
  };
}

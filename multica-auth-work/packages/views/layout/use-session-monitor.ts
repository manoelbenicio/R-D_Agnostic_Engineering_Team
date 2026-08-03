"use client";

import { useCallback, useRef } from "react";
import { useWSEvent } from "@multica/core/realtime";
import type {
  CredentialSessionAlertOutcome,
  CredentialSessionAlertPayload,
} from "@multica/core/types";
import { toast } from "sonner";

const OUTCOMES = new Set<CredentialSessionAlertOutcome>([
  "rotated",
  "no_account_available",
  "reassignment_failed",
]);

export function isExpiringSoon(
  expiresAt: string | undefined,
  withinMinutes = 30,
  nowMs = Date.now(),
): boolean {
  if (!expiresAt || !Number.isFinite(withinMinutes) || withinMinutes < 0) {
    return false;
  }
  const expiryMs = Date.parse(expiresAt);
  if (!Number.isFinite(expiryMs)) return false;
  return expiryMs <= nowMs + withinMinutes * 60_000;
}

function parseAlert(payload: unknown): CredentialSessionAlertPayload | null {
  if (!payload || typeof payload !== "object") return null;
  const value = payload as Record<string, unknown>;
  const uuidPattern =
    /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;
  if (
    typeof value.task_id !== "string" ||
    !uuidPattern.test(value.task_id) ||
    typeof value.agent_id !== "string" ||
    !uuidPattern.test(value.agent_id) ||
    typeof value.provider !== "string" ||
    !/^[a-z0-9_-]{1,32}$/.test(value.provider) ||
    typeof value.outcome !== "string" ||
    !OUTCOMES.has(value.outcome as CredentialSessionAlertOutcome)
  ) {
    return null;
  }
  if (
    value.reason !== undefined &&
    (typeof value.reason !== "string" ||
      !/^[a-z0-9_-]{1,64}$/.test(value.reason))
  ) {
    return null;
  }
  if (
    value.expires_at !== undefined &&
    (typeof value.expires_at !== "string" ||
      !Number.isFinite(Date.parse(value.expires_at)))
  ) {
    return null;
  }
  return value as unknown as CredentialSessionAlertPayload;
}

/**
 * Surfaces authenticated, workspace-scoped credential-session outcomes.
 * Malformed events fail closed and produce no user-visible success signal.
 */
export function useSessionMonitor(): void {
  const previousEvent = useRef<string | null>(null);
  const handleAlert = useCallback((raw: unknown) => {
    const alert = parseAlert(raw);
    if (!alert) return;

    const eventKey = [
      alert.task_id,
      alert.agent_id,
      alert.provider,
      alert.outcome,
      alert.reason ?? "",
      alert.expires_at ?? "",
    ].join("|");
    if (previousEvent.current === eventKey) return;
    previousEvent.current = eventKey;

    if (isExpiringSoon(alert.expires_at)) {
      toast.warning(`Credential session for ${alert.provider} is expired or expiring soon.`);
    }

    switch (alert.outcome) {
      case "rotated":
        toast.success(`Credential session rotated for ${alert.provider}.`);
        break;
      case "no_account_available":
        toast.warning(`No credential session is available for ${alert.provider}.`);
        break;
      case "reassignment_failed":
        toast.error(`Credential session reassignment failed for ${alert.provider}.`);
        break;
    }
  }, []);

  useWSEvent("credential:session_alert", handleAlert);
}

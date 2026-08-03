import { renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { toast } from "sonner";
import { useWSEvent } from "@multica/core/realtime";
import { isExpiringSoon, useSessionMonitor } from "./use-session-monitor";

vi.mock("@multica/core/realtime", () => ({ useWSEvent: vi.fn() }));
vi.mock("sonner", () => ({
  toast: { success: vi.fn(), warning: vi.fn(), error: vi.fn() },
}));

function subscribedHandler(): (payload: unknown) => void {
  const call = vi.mocked(useWSEvent).mock.calls.at(-1);
  if (!call) throw new Error("missing realtime subscription");
  return call[1] as (payload: unknown) => void;
}

describe("isExpiringSoon", () => {
  const now = Date.parse("2026-08-03T12:00:00Z");

  it("accepts stored timestamps inside the threshold", () => {
    expect(isExpiringSoon("2026-08-03T12:10:00Z", 30, now)).toBe(true);
    expect(isExpiringSoon("2026-08-03T13:00:00Z", 30, now)).toBe(false);
  });

  it("fails closed for missing, malformed, or invalid thresholds", () => {
    expect(isExpiringSoon(undefined, 30, now)).toBe(false);
    expect(isExpiringSoon("not-a-date", 30, now)).toBe(false);
    expect(isExpiringSoon("2026-08-03T12:10:00Z", -1, now)).toBe(false);
  });
});

describe("useSessionMonitor", () => {
  beforeEach(() => vi.clearAllMocks());

  it("emits success only for a valid rotated event", () => {
    renderHook(() => useSessionMonitor());
    subscribedHandler()({
      task_id: "11111111-1111-4111-8111-111111111111",
      agent_id: "22222222-2222-4222-8222-222222222222",
      provider: "codex",
      outcome: "rotated",
    });
    expect(toast.success).toHaveBeenCalledWith("Credential session rotated for codex.");
  });

  it("drops malformed and unknown events without a success signal", () => {
    renderHook(() => useSessionMonitor());
    const handler = subscribedHandler();
    handler({ provider: "codex", outcome: "rotated" });
    handler({
      task_id: "11111111-1111-4111-8111-111111111111",
      agent_id: "22222222-2222-4222-8222-222222222222",
      provider: "codex",
      outcome: "unknown",
    });
    handler({
      task_id: "11111111-1111-4111-8111-111111111111",
      agent_id: "22222222-2222-4222-8222-222222222222",
      provider: "codex",
      outcome: "rotated",
      expires_at: "invalid",
    });
    handler({
      task_id: "11111111-1111-4111-8111-111111111111",
      agent_id: "22222222-2222-4222-8222-222222222222",
      provider: "codex",
      outcome: "rotated",
      reason: { leaked: "unbounded" },
    });
    for (const [taskId, agentId] of [
      ["", "22222222-2222-4222-8222-222222222222"],
      ["11111111-1111-4111-8111-111111111111", ""],
      ["not-a-task-id", "22222222-2222-4222-8222-222222222222"],
      ["11111111-1111-4111-8111-111111111111", "not-an-agent-id"],
      ["t".repeat(129), "22222222-2222-4222-8222-222222222222"],
      ["11111111-1111-4111-8111-111111111111", "a".repeat(129)],
    ]) {
      handler({
        task_id: taskId,
        agent_id: agentId,
        provider: "codex",
        outcome: "rotated",
      });
    }
    expect(toast.success).not.toHaveBeenCalled();
    expect(toast.warning).not.toHaveBeenCalled();
    expect(toast.error).not.toHaveBeenCalled();
  });

  it("surfaces bounded failure and no-account outcomes and deduplicates", () => {
    renderHook(() => useSessionMonitor());
    const handler = subscribedHandler();
    const failure = {
      task_id: "11111111-1111-4111-8111-111111111111",
      agent_id: "22222222-2222-4222-8222-222222222222",
      provider: "kiro",
      outcome: "reassignment_failed",
    };
    handler(failure);
    handler(failure);
    handler({ ...failure, outcome: "no_account_available" });
    expect(toast.error).toHaveBeenCalledTimes(1);
    expect(toast.warning).toHaveBeenCalledWith(
      "No credential session is available for kiro.",
    );
    expect(toast.success).not.toHaveBeenCalled();
  });
});

// @vitest-environment jsdom

import type { ReactNode } from "react";
import { act, renderHook } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { agentTaskSnapshotKeys } from "../agents/queries";
import { workspaceKeys } from "../workspace/queries";
import { runtimeKeys } from "./queries";

const { deleteRuntime, archiveAgentsAndDeleteRuntime } = vi.hoisted(() => ({
  deleteRuntime: vi.fn(),
  archiveAgentsAndDeleteRuntime: vi.fn(),
}));

vi.mock("../api", () => ({
  api: {
    deleteRuntime,
    archiveAgentsAndDeleteRuntime,
  },
}));

import {
  useArchiveAgentsAndDeleteRuntime,
  useDeleteRuntime,
} from "./mutations";

const WS_ID = "ws-1";
const RUNTIME_ID = "runtime-1";

function testHarness() {
  const queryClient = new QueryClient({
    defaultOptions: {
      mutations: { retry: false },
      queries: { retry: false },
    },
  });
  const invalidate = vi.spyOn(queryClient, "invalidateQueries");
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
  return { queryClient, invalidate, wrapper };
}

describe("runtime deletion cache invalidation", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it.each([
    ["success", false],
    ["error", true],
  ] as const)(
    "invalidates the runtime tree after strict-delete %s",
    async (_outcome, rejects) => {
      if (rejects) {
        deleteRuntime.mockRejectedValueOnce(new Error("delete failed"));
      } else {
        deleteRuntime.mockResolvedValueOnce({ status: "ok" });
      }
      const { queryClient, invalidate, wrapper } = testHarness();
      const { result } = renderHook(() => useDeleteRuntime(WS_ID), { wrapper });

      await act(async () => {
        try {
          await result.current.mutateAsync(RUNTIME_ID);
        } catch {
          // Error is expected in the onSettled coverage case.
        }
      });

      expect(deleteRuntime).toHaveBeenCalledWith(RUNTIME_ID);
      expect(invalidate).toHaveBeenCalledOnce();
      expect(invalidate).toHaveBeenCalledWith({
        queryKey: runtimeKeys.all(WS_ID),
      });
      queryClient.clear();
    },
  );

  it.each([
    ["success", false],
    ["error", true],
  ] as const)(
    "invalidates runtime, agents, and task snapshot after cascade-delete %s",
    async (_outcome, rejects) => {
      if (rejects) {
        archiveAgentsAndDeleteRuntime.mockRejectedValueOnce(
          new Error("cascade failed"),
        );
      } else {
        archiveAgentsAndDeleteRuntime.mockResolvedValueOnce({
          status: "ok",
          agents_archived: 2,
          tasks_cancelled: 1,
        });
      }
      const { queryClient, invalidate, wrapper } = testHarness();
      const { result } = renderHook(
        () => useArchiveAgentsAndDeleteRuntime(WS_ID),
        { wrapper },
      );
      const variables = {
        runtimeId: RUNTIME_ID,
        expectedActiveAgentIds: ["agent-1", "agent-2"],
      };

      await act(async () => {
        try {
          await result.current.mutateAsync(variables);
        } catch {
          // Error is expected in the onSettled coverage case.
        }
      });

      expect(archiveAgentsAndDeleteRuntime).toHaveBeenCalledWith(
        RUNTIME_ID,
        variables.expectedActiveAgentIds,
      );
      expect(invalidate).toHaveBeenCalledTimes(3);
      expect(invalidate).toHaveBeenNthCalledWith(1, {
        queryKey: runtimeKeys.all(WS_ID),
      });
      expect(invalidate).toHaveBeenNthCalledWith(2, {
        queryKey: workspaceKeys.agents(WS_ID),
      });
      expect(invalidate).toHaveBeenNthCalledWith(3, {
        queryKey: agentTaskSnapshotKeys.all(WS_ID),
      });
      queryClient.clear();
    },
  );
});

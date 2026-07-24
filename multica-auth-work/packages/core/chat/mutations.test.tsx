/**
 * @vitest-environment jsdom
 */
import { afterEach, describe, expect, it, vi } from "vitest";
import { renderHook } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";

import { setApiInstance } from "../api";
import type { ApiClient } from "../api/client";
import type { ChatSession } from "../types";
import { useCreateChatSession } from "./mutations";

// useCreateChatSession invalidates chatKeys.sessions(wsId) onSettled; the
// workspace id is provided by ../hooks. Mock it so the hook is deterministic.
vi.mock("../hooks", () => ({
  useWorkspaceId: () => "ws-1",
}));

function createWrapper(qc: QueryClient) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={qc}>{children}</QueryClientProvider>;
  };
}

function makeSession(overrides: Partial<ChatSession> = {}): ChatSession {
  return {
    id: "sess-1",
    workspace_id: "ws-1",
    agent_id: "agent-1",
    creator_id: "user-1",
    title: "T",
    status: "active",
    has_unread: false,
    created_at: "2025-01-01T00:00:00Z",
    updated_at: "2025-01-01T00:00:00Z",
    ...overrides,
  };
}

afterEach(() => {
  vi.clearAllMocks();
});

describe("useCreateChatSession", () => {
  it("forwards an explicit agent_id (direct-to-agent escape hatch)", async () => {
    const createChatSession =
      vi.fn<(d: { agent_id?: string; title?: string }) => Promise<ChatSession>>()
        .mockResolvedValue(makeSession());
    setApiInstance({ createChatSession } as unknown as ApiClient);
    const qc = new QueryClient({
      defaultOptions: { mutations: { retry: false } },
    });

    const { result } = renderHook(() => useCreateChatSession(), {
      wrapper: createWrapper(qc),
    });
    await result.current.mutateAsync({ agent_id: "agent-1", title: "T" });

    expect(createChatSession).toHaveBeenCalledWith({
      agent_id: "agent-1",
      title: "T",
    });
  });

  it("omits agent_id for an untargeted chat (default TL routing)", async () => {
    const createChatSession =
      vi.fn<(d: { agent_id?: string; title?: string }) => Promise<ChatSession>>()
        .mockResolvedValue(makeSession({ id: "sess-2" }));
    setApiInstance({ createChatSession } as unknown as ApiClient);
    const qc = new QueryClient({
      defaultOptions: { mutations: { retry: false } },
    });

    const { result } = renderHook(() => useCreateChatSession(), {
      wrapper: createWrapper(qc),
    });
    await result.current.mutateAsync({ title: "Untargeted" });

    expect(createChatSession).toHaveBeenCalledWith({ title: "Untargeted" });
    const arg = createChatSession.mock.calls[0]![0];
    expect(arg.agent_id).toBeUndefined();
  });
});

import { describe, it, expect, vi } from "vitest";

// Mock the workspace id singleton — items() reads it imperatively.
vi.mock("@multica/core/platform", () => ({
  getCurrentWsId: () => "ws-1",
}));

const searchIssuesMock = vi.fn();
const searchProjectsMock = vi.fn();
vi.mock("@multica/core/api", () => ({
  api: {
    get searchIssues() {
      return searchIssuesMock;
    },
    get searchProjects() {
      return searchProjectsMock;
    },
  },
}));

const authState = { user: { id: "u1" } as { id: string } | null };
vi.mock("@multica/core/auth", () => ({
  useAuthStore: { getState: () => authState },
}));

import { workspaceKeys } from "@multica/core/workspace/queries";
import { issueKeys } from "@multica/core/issues/queries";
import type { QueryClient } from "@tanstack/react-query";
import { createMentionSuggestion, type MentionItem } from "./mention-suggestion";
import { BaseMentionExtension } from "./mention-extension";

// The chat direct-to-agent escape hatch is a two-sided contract: the composer
// has to emit `mention://agent/<id>` markup, and the server routes the turn on
// exactly that markup (server/internal/util.FirstAgentMention, verified in
// internal/util/mention_chat_route_test.go). This mirrors the server-side
// pattern so a change on either side breaks a test instead of silently
// disabling the hatch — the failure mode we are guarding against is a chat
// message that looks addressed to @codex but still runs on the squad TL.
const SERVER_AGENT_MENTION_RE = /\[@?(.+?)\]\(mention:\/\/agent\/([0-9a-fA-F-]+)\)/;

const CODEX_ID = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa";

const renderMentionMarkdown = BaseMentionExtension.config.renderMarkdown as (node: {
  attrs: Record<string, string>;
}) => string;

// Minimal QueryClient stand-in: items() only reads cached workspace data.
function fakeQc(agents: Array<{ id: string; name: string }>): QueryClient {
  const map = new Map<string, unknown>();
  // items() resolves the caller's workspace role from this cache before
  // deciding which agents they may address.
  map.set(JSON.stringify(workspaceKeys.members("ws-1")), [
    { user_id: "u1", name: "Alice", role: "member" },
  ]);
  map.set(
    JSON.stringify(workspaceKeys.agents("ws-1")),
    agents.map((a) => ({ ...a, archived_at: null, visibility: "workspace", owner_id: null })),
  );
  map.set(JSON.stringify(workspaceKeys.squads("ws-1")), []);
  return {
    getQueryData: (key: unknown) => map.get(JSON.stringify(key)),
    getQueriesData: ({ queryKey }: { queryKey: unknown }) =>
      JSON.stringify(queryKey) === JSON.stringify(issueKeys.list("ws-1")) ? [[queryKey, undefined]] : [],
  } as unknown as QueryClient;
}

describe("chat direct-to-agent escape hatch (composer side)", () => {
  it("offers agents in the chat mention list and emits markup the server routes on", () => {
    // mode:"context" is what chat-input passes; typing a query is what makes
    // the list fall through to workspace targets instead of chat context only.
    const config = createMentionSuggestion(fakeQc([{ id: CODEX_ID, name: "Codex" }]), {
      mode: "context",
      getContextItems: () => [
        { id: "i1", label: "ORQ-54", type: "issue", group: "current" } as MentionItem,
      ],
    });

    const items = config.items!({ query: "codex", editor: {} as never }) as MentionItem[];
    const codex = items.find((item) => item.type === "agent" && item.label === "Codex");
    expect(codex).toBeDefined();

    const markdown = renderMentionMarkdown({
      attrs: { id: codex!.id, label: codex!.label, type: codex!.type },
    });
    expect(markdown).toBe(`[@Codex](mention://agent/${CODEX_ID})`);

    const match = SERVER_AGENT_MENTION_RE.exec(`${markdown} please take this one`);
    expect(match?.[2]).toBe(CODEX_ID);
  });

  it("keeps an agent label with brackets routable after markdown escaping", () => {
    // renderMarkdown escapes brackets in labels ("Codex[5.6]" → "Codex\[5.6\]").
    // The server's non-greedy label capture has to survive that, otherwise
    // agents with bracketed names would silently lose the escape hatch.
    const markdown = renderMentionMarkdown({
      attrs: { id: CODEX_ID, label: "Codex[5.6]", type: "agent" },
    });
    expect(markdown).toBe(`[@Codex\\[5.6\\]](mention://agent/${CODEX_ID})`);
    expect(SERVER_AGENT_MENTION_RE.exec(markdown)?.[2]).toBe(CODEX_ID);
  });

  it("does not treat squad or member mentions as direct-agent targets", () => {
    // A squad mention would re-introduce the TL hop the hatch bypasses, so the
    // server ignores it — pin the markup difference the server relies on.
    const squad = renderMentionMarkdown({
      attrs: { id: CODEX_ID, label: "Workspace Team", type: "squad" },
    });
    const member = renderMentionMarkdown({
      attrs: { id: CODEX_ID, label: "Alice", type: "member" },
    });
    expect(SERVER_AGENT_MENTION_RE.test(squad)).toBe(false);
    expect(SERVER_AGENT_MENTION_RE.test(member)).toBe(false);
  });
});

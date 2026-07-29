// @vitest-environment jsdom

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import {
  act,
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { I18nProvider } from "@multica/core/i18n/react";
import { runtimeModelsKeys } from "@multica/core/runtimes";
import type {
  RuntimeModelListRequest,
  RuntimeModelsResult,
} from "@multica/core/types";
import enCommon from "../../locales/en/common.json";
import enAgents from "../../locales/en/agents.json";
import enIssues from "../../locales/en/issues.json";

const TEST_RESOURCES = {
  en: { common: enCommon, agents: enAgents, issues: enIssues },
};

const mockInitiateListModels = vi.hoisted(() => vi.fn());
const mockGetListModelsResult = vi.hoisted(() => vi.fn());

vi.mock("@multica/core/api", () => ({
  api: {
    initiateListModels: (...args: unknown[]) =>
      mockInitiateListModels(...args),
    getListModelsResult: (...args: unknown[]) =>
      mockGetListModelsResult(...args),
  },
}));

vi.mock("@multica/core/hooks", () => ({
  useWorkspaceId: () => "workspace-1",
}));

import { ModelDropdown } from "./model-dropdown";

const CACHED_MODELS: RuntimeModelsResult = {
  supported: true,
  models: [
    { id: "runtime-default", label: "Runtime Default" },
    { id: "gpt-5", label: "GPT 5", provider: "openai" },
  ],
};

function renderDropdown({
  runtimeId = "runtime-1",
  runtimeOnline = false,
  runtimeProvider,
  runtimeListProvider = "anthropic",
}: {
  runtimeId?: string;
  runtimeOnline?: boolean;
  runtimeProvider?: string;
  runtimeListProvider?: string | null;
} = {}) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  const now = Date.now();
  queryClient.setQueryData(
    runtimeModelsKeys.forRuntime(runtimeId),
    CACHED_MODELS,
    { updatedAt: now - 100 },
  );
  if (runtimeListProvider !== null) {
    queryClient.setQueryData(
      ["runtimes", "workspace-1", "list"],
      [{ id: runtimeId, provider: runtimeListProvider }],
      { updatedAt: now },
    );
  }
  queryClient.setQueryData(
    ["runtimes", "workspace-1", "list", "mine"],
    [{ id: runtimeId, provider: "conflicting-provider" }],
    { updatedAt: now + 1 },
  );

  const onChange = vi.fn();
  const result = render(
    <I18nProvider locale="en" resources={TEST_RESOURCES}>
      <QueryClientProvider client={queryClient}>
        <ModelDropdown
          runtimeId={runtimeId}
          runtimeOnline={runtimeOnline}
          runtimeProvider={runtimeProvider}
          value=""
          onChange={onChange}
        />
      </QueryClientProvider>
    </I18nProvider>,
  );
  return { ...result, onChange, queryClient };
}

function openDropdown() {
  const label = screen.getByText("Runtime offline — enter manually");
  const trigger = label.closest("button");
  if (!trigger) throw new Error("model dropdown trigger not found");
  fireEvent.click(trigger);
}

describe("ModelDropdown provider visibility", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    cleanup();
  });

  it("shows cached offline models with runtime fallback and explicit-provider precedence without discovery", async () => {
    renderDropdown();
    openDropdown();

    expect(await screen.findByText("anthropic")).toBeInTheDocument();
    expect(screen.getByText("openai")).toBeInTheDocument();
    expect(screen.getByText("Runtime Default")).toBeInTheDocument();
    expect(screen.getByText("GPT 5")).toBeInTheDocument();
    expect(mockInitiateListModels).not.toHaveBeenCalled();
    expect(mockGetListModelsResult).not.toHaveBeenCalled();
  });

  it("searches by provider and keeps only that provider group", async () => {
    renderDropdown();
    openDropdown();

    fireEvent.change(
      await screen.findByPlaceholderText("Search or type a model ID"),
      { target: { value: "OPENAI" } },
    );

    expect(screen.getByText("openai")).toBeInTheDocument();
    expect(screen.getByText("GPT 5")).toBeInTheDocument();
    expect(screen.queryByText("anthropic")).toBeNull();
    expect(screen.queryByText("Runtime Default")).toBeNull();
    expect(screen.queryByText('Use "OPENAI"')).toBeNull();
    expect(mockInitiateListModels).not.toHaveBeenCalled();
  });

  it("prefers an explicit runtime provider over the exact workspace cache", async () => {
    renderDropdown({ runtimeProvider: "explicit-runtime-provider" });
    openDropdown();

    expect(
      await screen.findByText("explicit-runtime-provider"),
    ).toBeInTheDocument();
    expect(screen.queryByText("anthropic")).toBeNull();
    expect(screen.queryByText("conflicting-provider")).toBeNull();
  });

  it("derives a known built-in identity when the exact runtime cache is absent", async () => {
    renderDropdown({ runtimeId: "kimi", runtimeListProvider: null });
    openDropdown();

    expect(await screen.findByText("kimi")).toBeInTheDocument();
    expect(screen.getByText("Runtime Default")).toBeInTheDocument();
    expect(screen.getByText("openai")).toBeInTheDocument();
    expect(mockInitiateListModels).not.toHaveBeenCalled();
    expect(mockGetListModelsResult).not.toHaveBeenCalled();
  });

  it("labels an unknown custom runtime explicitly instead of inferring a vendor", async () => {
    renderDropdown({
      runtimeId: "custom-claude-runtime",
      runtimeListProvider: null,
    });
    openDropdown();

    expect(await screen.findByText("Unknown/Custom")).toBeInTheDocument();
    expect(screen.queryByText("claude")).toBeNull();
    expect(screen.getByText("openai")).toBeInTheDocument();
    expect(mockInitiateListModels).not.toHaveBeenCalled();
    expect(mockGetListModelsResult).not.toHaveBeenCalled();
  });

  it("performs one discovery when mounted after a reconnect with a fresh stale catalog", async () => {
    const completed: RuntimeModelListRequest = {
      id: "request-1",
      runtime_id: "runtime-1",
      status: "completed",
      models: CACHED_MODELS.models,
      supported: true,
      created_at: "2026-07-18T00:00:00Z",
      updated_at: "2026-07-18T00:00:00Z",
    };
    mockInitiateListModels.mockResolvedValue(completed);

    renderDropdown({ runtimeOnline: true });

    await waitFor(() => {
      expect(mockInitiateListModels).toHaveBeenCalledTimes(1);
    });
    expect(mockInitiateListModels.mock.calls[0]?.[1]).toBeInstanceOf(
      AbortSignal,
    );
    await act(async () => {
      await Promise.resolve();
    });
    expect(mockInitiateListModels).toHaveBeenCalledTimes(1);
  });
});

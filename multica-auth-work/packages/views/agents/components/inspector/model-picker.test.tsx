// @vitest-environment jsdom

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { I18nProvider } from "@multica/core/i18n/react";
import { runtimeModelsKeys } from "@multica/core/runtimes";
import type { RuntimeModelsResult } from "@multica/core/types";
import enCommon from "../../../locales/en/common.json";
import enAgents from "../../../locales/en/agents.json";
import enIssues from "../../../locales/en/issues.json";

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

import { ModelPicker } from "./model-picker";

const CACHED_MODELS: RuntimeModelsResult = {
  supported: true,
  models: [
    { id: "runtime-default", label: "Runtime Default" },
    { id: "gpt-5", label: "GPT 5", provider: "openai" },
  ],
};

function renderPicker({
  runtimeId = "runtime-1",
  runtimeProvider,
  runtimeListProvider = "anthropic",
}: {
  runtimeId?: string;
  runtimeProvider?: string;
  runtimeListProvider?: string | null;
} = {}) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  queryClient.setQueryData(
    runtimeModelsKeys.forRuntime(runtimeId),
    CACHED_MODELS,
  );
  if (runtimeListProvider !== null) {
    queryClient.setQueryData(["runtimes", "workspace-1", "list"], [
      { id: runtimeId, provider: runtimeListProvider },
    ]);
  }

  const result = render(
    <I18nProvider locale="en" resources={TEST_RESOURCES}>
      <QueryClientProvider client={queryClient}>
        <ModelPicker
          runtimeId={runtimeId}
          runtimeOnline={false}
          runtimeProvider={runtimeProvider}
          value=""
          onChange={vi.fn()}
        />
      </QueryClientProvider>
    </I18nProvider>,
  );
  return { ...result, queryClient };
}

describe("inspector ModelPicker provider visibility", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    cleanup();
  });

  it("renders the same fallback and explicit provider headers from the cached offline catalog", async () => {
    renderPicker();
    fireEvent.click(screen.getByRole("button", { name: "Model · Default" }));

    expect(await screen.findByText("anthropic")).toBeInTheDocument();
    expect(screen.getByText("openai")).toBeInTheDocument();
    expect(screen.getByText("Runtime Default")).toBeInTheDocument();
    expect(screen.getByText("GPT 5")).toBeInTheDocument();
    expect(mockInitiateListModels).not.toHaveBeenCalled();
    expect(mockGetListModelsResult).not.toHaveBeenCalled();
  });

  it("uses a known runtime identity without an exact runtime-list cache", async () => {
    renderPicker({ runtimeId: "cline", runtimeListProvider: null });
    fireEvent.click(screen.getByRole("button", { name: "Model · Default" }));

    expect(await screen.findByText("cline")).toBeInTheDocument();
    expect(screen.getByText("Runtime Default")).toBeInTheDocument();
    expect(screen.getByText("openai")).toBeInTheDocument();
    expect(mockInitiateListModels).not.toHaveBeenCalled();
    expect(mockGetListModelsResult).not.toHaveBeenCalled();
  });

  it("keeps explicit runtime and model provider precedence", async () => {
    renderPicker({
      runtimeProvider: "explicit-runtime-provider",
      runtimeListProvider: "cached-runtime-provider",
    });
    fireEvent.click(screen.getByRole("button", { name: "Model · Default" }));

    expect(
      await screen.findByText("explicit-runtime-provider"),
    ).toBeInTheDocument();
    expect(screen.getByText("openai")).toBeInTheDocument();
    expect(screen.queryByText("cached-runtime-provider")).toBeNull();
  });
});

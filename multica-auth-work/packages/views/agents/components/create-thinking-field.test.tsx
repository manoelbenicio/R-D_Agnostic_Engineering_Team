// @vitest-environment jsdom

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import type {
  RuntimeModel,
  RuntimeModelListRequest,
} from "@multica/core/types";
import { I18nProvider } from "@multica/core/i18n/react";
import enAgents from "../../locales/en/agents.json";
import enCommon from "../../locales/en/common.json";
import enIssues from "../../locales/en/issues.json";

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

import { CreateThinkingField } from "./create-thinking-field";

const TEST_RESOURCES = {
  en: { agents: enAgents, common: enCommon, issues: enIssues },
};

const STRUCTURED_MODEL: RuntimeModel = {
  id: "claude-opus-4.8",
  label: "claude-opus-4.8",
  default: true,
  thinking: {
    supported_levels: [
      { value: "low", label: "Low" },
      { value: "medium", label: "Medium" },
      { value: "high", label: "High" },
      { value: "xhigh", label: "Extra high" },
      { value: "max", label: "Max" },
    ],
  },
};

const AGY_MODEL: RuntimeModel = {
  id: "gemini-3.6-flash-high",
  label: "gemini-3.6-flash-high",
  default: true,
};

function result(models: RuntimeModel[]): RuntimeModelListRequest {
  return {
    id: "request-1",
    runtime_id: "runtime-1",
    status: "completed",
    models,
    supported: true,
    created_at: "2026-07-27T00:00:00Z",
    updated_at: "2026-07-27T00:00:00Z",
  };
}

function renderField(
  models: RuntimeModel[],
  props: Partial<React.ComponentProps<typeof CreateThinkingField>> = {},
) {
  const onChange = vi.fn();
  mockInitiateListModels.mockResolvedValue(result(models));
  mockGetListModelsResult.mockResolvedValue(result(models));
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  render(
    <I18nProvider locale="en" resources={TEST_RESOURCES}>
      <QueryClientProvider client={queryClient}>
        <CreateThinkingField
          runtimeId="runtime-1"
          runtimeOnline
          model=""
          value=""
          onChange={onChange}
          {...props}
        />
      </QueryClientProvider>
    </I18nProvider>,
  );
  return { onChange };
}

describe("CreateThinkingField", () => {
  beforeEach(() => vi.clearAllMocks());
  afterEach(() => cleanup());

  it("renders structured levels separately and emits the selected token", async () => {
    const { onChange } = renderField([STRUCTURED_MODEL]);

    const trigger = await screen.findByRole("button", {
      name: /Thinking.*Follow CLI config/i,
    });
    fireEvent.click(trigger);
    fireEvent.click(await screen.findByText("Max"));

    expect(onChange).toHaveBeenCalledWith("max");
  });

  it("does not render a second tier picker for AGY model IDs", async () => {
    renderField([AGY_MODEL], {
      model: "gemini-3.6-flash-high",
      value: "",
    });

    await waitFor(() => {
      expect(mockInitiateListModels).toHaveBeenCalled();
    });
    expect(screen.queryByText("Thinking")).toBeNull();
  });

  it("clears a stale structured value after switching to AGY", async () => {
    const { onChange } = renderField([AGY_MODEL], {
      model: "gemini-3.6-flash-high",
      value: "max",
    });

    await waitFor(() => {
      expect(onChange).toHaveBeenCalledWith("");
    });
    expect(screen.queryByText("Thinking")).toBeNull();
  });
});

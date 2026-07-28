// @vitest-environment jsdom

import { describe, it, expect, vi, beforeEach } from "vitest";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen } from "@testing-library/react";
import type { AgentRuntime } from "@multica/core/types";
import { I18nProvider } from "@multica/core/i18n/react";
import enCommon from "../../locales/en/common.json";
import enRuntimes from "../../locales/en/runtimes.json";
import enAgents from "../../locales/en/agents.json";

const TEST_RESOURCES = {
  en: { common: enCommon, runtimes: enRuntimes, agents: enAgents },
};

const { toastSuccess } = vi.hoisted(() => ({
  toastSuccess: vi.fn(),
}));

// Stub the workspace queries the columns reach into. None of them feed the
// row menu directly, but `createRuntimeColumns` wires CliCell + CostCell
// against the same query client, so we still need useQuery to resolve.
vi.mock("@tanstack/react-query", async () => {
  const actual = await vi.importActual<typeof import("@tanstack/react-query")>(
    "@tanstack/react-query",
  );
  return {
    ...actual,
    useQuery: vi.fn(() => ({ data: [], isLoading: false })),
  };
});

vi.mock("@multica/core/runtimes/mutations", () => ({
  useDeleteRuntime: () => ({
    mutate: vi.fn(),
    isPending: false,
    mutateAsync: vi.fn(),
  }),
  useArchiveAgentsAndDeleteRuntime: () => ({
    mutate: vi.fn(),
    isPending: false,
    mutateAsync: vi.fn(),
  }),
}));

vi.mock("@multica/core/runtimes", () => ({
  deriveRuntimeHealth: () => "online",
  runtimeUsageOptions: () => ({ kind: "usage" }),
}));

vi.mock("@multica/core/agents", () => ({
  deriveWorkload: () => "idle",
  useWorkspacePresenceMap: () => ({ byAgent: new Map(), loading: false }),
}));

// This file owns the list affordance only. Keep the real portal-heavy dialog
// in delete-runtime-dialog.test.tsx so clicking the row action cannot leave
// this narrow suite waiting on dialog/tooltip lifecycle work.
vi.mock("./delete-runtime-dialog", () => ({
  DeleteRuntimeDialog: ({
    open,
    onDeleted,
  }: {
    open: boolean;
    onDeleted: () => void;
  }) =>
    open ? (
      <div role="dialog" aria-label="Delete runtime confirmation">
        <button type="button" onClick={onDeleted}>
          Complete mocked delete
        </button>
      </div>
    ) : null,
}));

vi.mock("@multica/core/auth", () => ({
  useAuthStore: (sel: (s: { user: { id: string } }) => unknown) =>
    sel({ user: { id: "user-me" } }),
}));

vi.mock("@multica/core/api", () => ({
  api: {
    deleteRuntime: vi.fn(),
    archiveAgentsAndDeleteRuntime: vi.fn(),
  },
  ApiError: class ApiError extends Error {},
}));

vi.mock("sonner", () => ({
  toast: { error: vi.fn(), success: toastSuccess },
}));

vi.mock("../../common/use-viewing-timezone", () => ({
  useViewingTimezone: () => "UTC",
}));

vi.mock("./provider-logo", () => ({ ProviderLogo: () => null }));
vi.mock("./shared", () => ({
  HealthIcon: () => null,
  useHealthLabel: () => () => "Online",
}));

import { CliCell, RuntimeDeleteButton, type RuntimeRow } from "./runtime-list";

function makeRuntime(overrides: Partial<AgentRuntime>): AgentRuntime {
  return {
    id: "rt-1",
    workspace_id: "ws-1",
    daemon_id: null,
    name: "rt",
    runtime_mode: "local",
    provider: "claude",
    launch_header: "",
    status: "online",
    device_info: "",
    metadata: {},
    owner_id: "user-1",
    visibility: "private",
    last_seen_at: null,
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-01-01T00:00:00Z",
    ...overrides,
  };
}

function makeRow(runtime: AgentRuntime, canDelete = true): RuntimeRow {
  return {
    runtime,
    ownerMember: null,
    workload: { agentIds: [], runningCount: 0, queuedCount: 0 },
    canDelete,
  };
}

// The delete button is a plain exported component on the ListGrid version of
// the list — render it directly with the row fields it reads.
function renderActionsCell(row: RuntimeRow) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });

  return render(
    <I18nProvider locale="en" resources={TEST_RESOURCES}>
      <QueryClientProvider client={qc}>
        <RuntimeDeleteButton
          runtime={row.runtime}
          wsId="ws-1"
          canDelete={row.canDelete}
        />
      </QueryClientProvider>
    </I18nProvider>,
  );
}

describe("runtime list delete button", () => {
  beforeEach(() => vi.clearAllMocks());

  it("renders a directly visible delete button and hands off to the confirmation dialog", () => {
    renderActionsCell(
      makeRow(makeRuntime({ runtime_mode: "local", status: "online" })),
    );

    const deleteButton = screen.getByRole("button", { name: "Delete" });
    expect(deleteButton).toBeVisible();

    fireEvent.click(deleteButton);

    expect(
      screen.getByRole("dialog", { name: "Delete runtime confirmation" }),
    ).toBeInTheDocument();
  });

  it("closes the dialog and reports success after the dialog completes deletion", () => {
    renderActionsCell(
      makeRow(makeRuntime({ runtime_mode: "local", status: "offline" })),
    );

    fireEvent.click(screen.getByRole("button", { name: "Delete" }));
    fireEvent.click(
      screen.getByRole("button", { name: "Complete mocked delete" }),
    );

    expect(
      screen.queryByRole("dialog", { name: "Delete runtime confirmation" }),
    ).not.toBeInTheDocument();
    expect(toastSuccess).toHaveBeenCalledOnce();
    expect(toastSuccess).toHaveBeenCalledWith("Runtime deleted");
  });

  it("renders the delete button for an offline local runtime", () => {
    renderActionsCell(
      makeRow(makeRuntime({ runtime_mode: "local", status: "offline" })),
    );
    expect(screen.getByRole("button", { name: "Delete" })).toBeVisible();
  });

  it("renders the delete button for a cloud runtime regardless of status", () => {
    renderActionsCell(
      makeRow(makeRuntime({ runtime_mode: "cloud", status: "online" })),
    );
    expect(screen.getByRole("button", { name: "Delete" })).toBeVisible();
  });

  it("hides the delete button when the caller lacks delete permission", () => {
    renderActionsCell(
      makeRow(
        makeRuntime({ runtime_mode: "local", status: "offline" }),
        /* canDelete */ false,
      ),
    );
    expect(
      screen.queryByRole("button", { name: "Delete" }),
    ).not.toBeInTheDocument();
  });
});

// The CLI cell is a plain exported component — render it in isolation,
// mirroring renderActionsCell.
function renderCliCell(row: RuntimeRow) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });

  return render(
    <I18nProvider locale="en" resources={TEST_RESOURCES}>
      <QueryClientProvider client={qc}>
        <CliCell runtime={row.runtime} />
      </QueryClientProvider>
    </I18nProvider>,
  );
}

describe("runtime list CLI column", () => {
  beforeEach(() => vi.clearAllMocks());

  // #3838: every agent showed the same number because the column rendered the
  // shared multica daemon `cli_version`. It must instead show the agent's own
  // tool version from `metadata.version`.
  it("shows the agent's own CLI tool version, not the shared daemon version", () => {
    renderCliCell(
      makeRow(
        makeRuntime({
          runtime_mode: "local",
          metadata: { version: "2.1.5 (Claude Code)", cli_version: "0.3.17" },
        }),
      ),
    );
    expect(screen.getByText("2.1.5 (Claude Code)")).toBeInTheDocument();
    expect(screen.queryByText("0.3.17")).not.toBeInTheDocument();
  });

  it("falls back to an em dash when the agent version is missing", () => {
    renderCliCell(
      makeRow(
        makeRuntime({
          runtime_mode: "local",
          metadata: { cli_version: "0.3.17" },
        }),
      ),
    );
    expect(screen.queryByText("0.3.17")).not.toBeInTheDocument();
    expect(screen.getByText("—")).toBeInTheDocument();
  });

  it("renders an em dash for cloud runtimes", () => {
    renderCliCell(
      makeRow(
        makeRuntime({
          runtime_mode: "cloud",
          metadata: { version: "2.1.5 (Claude Code)" },
        }),
      ),
    );
    expect(screen.getByText("—")).toBeInTheDocument();
  });
});

// @vitest-environment jsdom

import {
  cleanup,
  fireEvent,
  render,
  screen,
  within,
} from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { I18nProvider } from "@multica/core/i18n/react";
import type { RuntimeDevice } from "@multica/core/types";
import enAgents from "../../locales/en/agents.json";
import enCommon from "../../locales/en/common.json";

vi.mock("../../runtimes/components/provider-logo", () => ({
  ProviderLogo: ({ provider }: { provider: string }) => (
    <span data-testid={`provider-logo-${provider}`} />
  ),
}));

vi.mock("../../common/actor-avatar", () => ({
  ActorAvatar: () => <span data-testid="actor-avatar" />,
}));

import { RuntimePicker, runtimeProviderLabel } from "./runtime-picker";

const TEST_RESOURCES = { en: { common: enCommon, agents: enAgents } };
const USER_ID = "user-1";

const PROVIDERS = [
  ["claude", "Claude"],
  ["codebuddy", "CodeBuddy"],
  ["cline", "Cline"],
  ["codex", "Codex"],
  ["copilot", "GitHub Copilot"],
  ["nim", "NVIDIA NIM"],
  ["opencode", "OpenCode"],
  ["openclaw", "OpenClaw"],
  ["hermes", "Hermes"],
  ["gemini", "Gemini"],
  ["pi", "Pi"],
  ["cursor", "Cursor"],
  ["kimi", "Kimi"],
  ["kiro", "Kiro"],
  ["antigravity", "Antigravity"],
  ["qoder", "Qoder"],
] as const;

function runtime(provider: string, index: number): RuntimeDevice {
  return {
    id: `runtime-${provider}`,
    workspace_id: "workspace-1",
    daemon_id: null,
    name: `Runtime ${provider}`,
    runtime_mode: "local",
    provider,
    launch_header: "",
    status: index % 2 === 0 ? "online" : "offline",
    device_info: `host-${provider}`,
    metadata: {},
    owner_id: USER_ID,
    visibility: "private",
    last_seen_at: null,
    created_at: "2026-07-18T00:00:00Z",
    updated_at: "2026-07-18T00:00:00Z",
  };
}

function renderPicker() {
  const runtimes = PROVIDERS.map(([provider], index) => runtime(provider, index));
  const result = render(
    <I18nProvider locale="en" resources={TEST_RESOURCES}>
      <RuntimePicker
        runtimes={runtimes}
        members={[]}
        currentUserId={USER_ID}
        selectedRuntimeId="runtime-claude"
        onSelect={vi.fn()}
      />
    </I18nProvider>,
  );
  return { ...result, runtimes };
}

describe("RuntimePicker provider identity", () => {
  afterEach(() => {
    cleanup();
    document.body.innerHTML = "";
  });

  it("renders accessible runtime and provider identity for all supported runtime types", () => {
    renderPicker();
    fireEvent.click(screen.getByText("Runtime claude"));

    for (const [provider, label] of PROVIDERS) {
      const runtimeName = screen.getAllByText(`Runtime ${provider}`).at(-1);
      expect(runtimeName).toBeDefined();
      const row = runtimeName?.closest("button");
      expect(row).not.toBeNull();
      expect(
        within(row as HTMLButtonElement).getByText(label),
      ).toBeInTheDocument();
      expect(row).toHaveAccessibleName(
        expect.stringMatching(new RegExp(`Runtime ${provider}.*${label}`, "i")),
      );
    }
  });

  it("uses distinct local marks for cline and nim instead of the generic provider-logo fallback", () => {
    renderPicker();
    fireEvent.click(screen.getByText("Runtime claude"));

    const clineRow = screen.getByText("Runtime cline").closest("button");
    const nimRow = screen.getByText("Runtime nim").closest("button");
    expect(clineRow).not.toBeNull();
    expect(nimRow).not.toBeNull();

    expect(
      within(clineRow as HTMLButtonElement).getByText("CL"),
    ).toBeInTheDocument();
    expect(
      within(nimRow as HTMLButtonElement).getByText("NIM"),
    ).toBeInTheDocument();
    expect(
      within(clineRow as HTMLButtonElement).queryByTestId(
        "provider-logo-cline",
      ),
    ).toBeNull();
    expect(
      within(nimRow as HTMLButtonElement).queryByTestId("provider-logo-nim"),
    ).toBeNull();
  });

  it("keeps unknown future runtime providers identifiable", () => {
    expect(runtimeProviderLabel(" future-provider ")).toBe("future-provider");
    expect(runtimeProviderLabel("   ")).toBe("Runtime");
  });
});

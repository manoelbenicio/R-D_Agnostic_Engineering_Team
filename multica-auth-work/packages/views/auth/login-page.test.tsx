import { beforeEach, describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ReactElement, ReactNode } from "react";
import { I18nProvider } from "@multica/core/i18n/react";
import enCommon from "../locales/en/common.json";
import enAuth from "../locales/en/auth.json";
import enSettings from "../locales/en/settings.json";

const TEST_RESOURCES = {
  en: { common: enCommon, auth: enAuth, settings: enSettings },
};

function I18nWrapper({ children }: { children: ReactNode }) {
  return (
    <I18nProvider locale="en" resources={TEST_RESOURCES}>
      {children}
    </I18nProvider>
  );
}

function renderWithI18n(ui: ReactElement) {
  return render(ui, { wrapper: I18nWrapper });
}

const mockLogin = vi.hoisted(() => vi.fn());
const mockApiListWorkspaces = vi.hoisted(() => vi.fn());
const mockApiSetToken = vi.hoisted(() => vi.fn());
const mockApiGetMe = vi.hoisted(() => vi.fn());
const mockApiIssueCliToken = vi.hoisted(() => vi.fn());
const mockSetQueryData = vi.hoisted(() => vi.fn());

vi.mock("@tanstack/react-query", async () => {
  const actual = await vi.importActual<typeof import("@tanstack/react-query")>(
    "@tanstack/react-query",
  );
  return {
    ...actual,
    useQueryClient: () => ({ setQueryData: mockSetQueryData }),
  };
});

vi.mock("@multica/core/auth", () => ({
  useAuthStore: Object.assign(
    (selector?: (state: { login: typeof mockLogin }) => unknown) => {
      const state = { login: mockLogin };
      return selector ? selector(state) : state;
    },
    { getState: () => ({ login: mockLogin }) },
  ),
}));

vi.mock("@multica/core/api", () => ({
  api: {
    listWorkspaces: mockApiListWorkspaces,
    setToken: mockApiSetToken,
    getMe: mockApiGetMe,
    issueCliToken: mockApiIssueCliToken,
  },
}));

import { LoginPage, validateCliCallback } from "./login-page";

const loggedInUser = {
  id: "user-1",
  email: "user@example.com",
  name: "User",
};

describe("LoginPage", () => {
  const onSuccess = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
    localStorage.clear();
    mockApiGetMe.mockRejectedValue(new Error("unauthorized"));
    Object.defineProperty(window, "location", {
      configurable: true,
      writable: true,
      value: { href: "http://localhost:3000" },
    });
  });

  it("renders the password form without an email-code step", () => {
    renderWithI18n(<LoginPage onSuccess={onSuccess} />);

    expect(screen.getByText("Sign in to Multica")).toBeInTheDocument();
    expect(screen.getByLabelText("Email")).toHaveAttribute("autocomplete", "email");
    expect(screen.getByLabelText("Password")).toHaveAttribute(
      "autocomplete",
      "current-password",
    );
    expect(screen.queryByText(/code/i)).not.toBeInTheDocument();
  });

  it("validates required credentials before calling the auth service", () => {
    renderWithI18n(<LoginPage onSuccess={onSuccess} />);

    fireEvent.submit(document.querySelector("#login-form")!);

    expect(screen.getByRole("alert")).toHaveTextContent("Email is required");
    expect(mockLogin).not.toHaveBeenCalled();
  });

  it("logs in, seeds workspaces, and completes the web flow", async () => {
    const workspaces = [{ id: "workspace-1", slug: "acme" }];
    mockLogin.mockResolvedValue({ token: "jwt", user: loggedInUser });
    mockApiListWorkspaces.mockResolvedValue(workspaces);
    const onTokenObtained = vi.fn();
    const user = userEvent.setup();
    renderWithI18n(
      <LoginPage
        onSuccess={onSuccess}
        onTokenObtained={onTokenObtained}
      />,
    );

    await user.type(screen.getByLabelText("Email"), " user@example.com ");
    await user.type(screen.getByLabelText("Password"), "password");
    await user.click(screen.getByRole("button", { name: "Sign in" }));

    await waitFor(() => {
      expect(mockLogin).toHaveBeenCalledWith("user@example.com", "password");
      expect(mockSetQueryData).toHaveBeenCalledWith(
        expect.any(Array),
        workspaces,
      );
      expect(onTokenObtained).toHaveBeenCalledTimes(1);
      expect(onSuccess).toHaveBeenCalledTimes(1);
    });
  });

  it("shows an authentication error without entering a second step", async () => {
    mockLogin.mockRejectedValue(new Error("invalid email or password"));
    const user = userEvent.setup();
    renderWithI18n(<LoginPage onSuccess={onSuccess} />);

    await user.type(screen.getByLabelText("Email"), "user@example.com");
    await user.type(screen.getByLabelText("Password"), "wrong");
    await user.click(screen.getByRole("button", { name: "Sign in" }));

    expect(await screen.findByRole("alert")).toHaveTextContent(
      "invalid email or password",
    );
    expect(screen.getByLabelText("Password")).toBeInTheDocument();
  });

  it("returns the password-login token to a CLI callback", async () => {
    mockLogin.mockResolvedValue({ token: "cli-jwt", user: loggedInUser });
    const user = userEvent.setup();
    renderWithI18n(
      <LoginPage
        onSuccess={onSuccess}
        cliCallback={{ url: "http://localhost:9876/callback", state: "state-1" }}
      />,
    );

    await user.type(screen.getByLabelText("Email"), "user@example.com");
    await user.type(screen.getByLabelText("Password"), "password");
    await user.click(screen.getByRole("button", { name: "Sign in" }));

    await waitFor(() => {
      expect(window.location.href).toBe(
        "http://localhost:9876/callback?token=cli-jwt&state=state-1",
      );
    });
    expect(localStorage.getItem("multica_token")).toBe("cli-jwt");
    expect(onSuccess).not.toHaveBeenCalled();
  });

  it("keeps cookie-session CLI authorization through issueCliToken", async () => {
    mockApiGetMe.mockResolvedValue(loggedInUser);
    mockApiIssueCliToken.mockResolvedValue({ token: "cookie-cli-jwt" });
    const user = userEvent.setup();
    renderWithI18n(
      <LoginPage
        onSuccess={onSuccess}
        cliCallback={{ url: "http://127.0.0.1:9876/callback", state: "state-2" }}
      />,
    );

    expect(await screen.findByText("Authorize CLI")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Authorize" }));

    await waitFor(() => {
      expect(mockApiIssueCliToken).toHaveBeenCalledTimes(1);
      expect(window.location.href).toBe(
        "http://127.0.0.1:9876/callback?token=cookie-cli-jwt&state=state-2",
      );
    });
  });

  it("preserves Google OAuth state", async () => {
    const user = userEvent.setup();
    renderWithI18n(
      <LoginPage
        onSuccess={onSuccess}
        google={{
          clientId: "google-client",
          redirectUri: "http://localhost:3000/auth/callback",
          state: "platform:desktop",
        }}
      />,
    );

    await user.click(
      screen.getByRole("button", { name: "Continue with Google" }),
    );

    expect(window.location.href).toContain(
      "https://accounts.google.com/o/oauth2/v2/auth?",
    );
    expect(window.location.href).toContain("state=platform%3Adesktop");
  });
});

describe("validateCliCallback", () => {
  it.each([
    "http://localhost:9876/callback",
    "http://127.0.0.1:8080/cb",
    "http://10.0.0.5:9876/callback",
    "http://172.16.0.1:9876/callback",
    "http://172.31.255.255:1234/cb",
    "http://192.168.1.131:41117/callback",
  ])("accepts safe local callback %s", (url) => {
    expect(validateCliCallback(url)).toBe(true);
  });

  it.each([
    "https://localhost:9876/callback",
    "http://evil.com:9876/callback",
    "http://172.32.0.1:9876/callback",
    "http://192.169.1.1:9876/callback",
    "not-a-url",
  ])("rejects unsafe callback %s", (url) => {
    expect(validateCliCallback(url)).toBe(false);
  });
});

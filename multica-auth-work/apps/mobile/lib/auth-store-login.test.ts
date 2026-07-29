import { beforeEach, describe, expect, it, vi } from "vitest";
import { useAuthStore } from "../data/auth-store";

const mocks = vi.hoisted(() => ({
  login: vi.fn(),
  apiSetToken: vi.fn(),
  getToken: vi.fn(),
  setToken: vi.fn(),
  clearToken: vi.fn(),
}));

vi.mock("../data/api", () => ({
  api: {
    login: mocks.login,
    getMe: vi.fn(),
    setToken: mocks.apiSetToken,
  },
  ApiError: class ApiError extends Error {
    status = 500;
  },
}));
vi.mock("../data/secure-storage", () => ({
  getToken: mocks.getToken,
  setToken: mocks.setToken,
  clearToken: mocks.clearToken,
}));
vi.mock("../data/workspace-store", () => ({
  useWorkspaceStore: { getState: () => ({ restoreSlug: vi.fn() }) },
}));

describe("auth store password login", () => {
  const password = "sentinel-password-never-persist";
  const priorUser = { id: "prior-user", email: "prior@example.test" } as never;
  const nextUser = { id: "next-user", email: "next@example.test" } as never;

  beforeEach(() => {
    vi.clearAllMocks();
    useAuthStore.setState({ user: priorUser, isLoading: false });
  });

  it("persists and activates only the token returned by a successful response", async () => {
    mocks.login.mockResolvedValue({ token: "token-2", user: nextUser });
    mocks.setToken.mockResolvedValue(undefined);

    await expect(
      useAuthStore.getState().login("next@example.test", password),
    ).resolves.toBe(nextUser);

    expect(mocks.login).toHaveBeenCalledWith("next@example.test", password);
    expect(mocks.setToken).toHaveBeenCalledWith("token-2");
    expect(mocks.apiSetToken).toHaveBeenCalledWith("token-2");
    expect(mocks.setToken.mock.invocationCallOrder[0]).toBeLessThan(
      mocks.apiSetToken.mock.invocationCallOrder[0],
    );
    expect(useAuthStore.getState().user).toBe(nextUser);
    expect(JSON.stringify(useAuthStore.getState())).not.toContain(password);
    expect(JSON.stringify(mocks.setToken.mock.calls)).not.toContain(password);
  });

  it("preserves prior state and storage when authentication fails", async () => {
    mocks.login.mockRejectedValue(new Error("invalid credentials"));

    await expect(
      useAuthStore.getState().login("next@example.test", password),
    ).rejects.toThrow("invalid credentials");

    expect(mocks.setToken).not.toHaveBeenCalled();
    expect(mocks.apiSetToken).not.toHaveBeenCalled();
    expect(useAuthStore.getState().user).toBe(priorUser);
  });

  it("does not activate a token or replace state when secure persistence fails", async () => {
    mocks.login.mockResolvedValue({ token: "token-2", user: nextUser });
    mocks.setToken.mockRejectedValue(new Error("storage unavailable"));

    await expect(
      useAuthStore.getState().login("next@example.test", password),
    ).rejects.toThrow("storage unavailable");

    expect(mocks.apiSetToken).not.toHaveBeenCalled();
    expect(useAuthStore.getState().user).toBe(priorUser);
  });
});

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("../data/workspace-store", () => ({
  getCurrentSlug: () => null,
}));
vi.mock("./parse-response", () => ({
  parseWithFallback: (value: unknown) => value,
}));
vi.mock("./request-id", () => ({
  createRequestId: () => "request-test",
}));

describe("ApiClient.login", () => {
  const password = "sentinel-password-never-log";
  let logSpy: ReturnType<typeof vi.spyOn>;
  let errorSpy: ReturnType<typeof vi.spyOn>;
  let warnSpy: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    vi.resetModules();
    process.env.EXPO_PUBLIC_API_URL = "https://api.example.test";
    logSpy = vi.spyOn(console, "log").mockImplementation(() => undefined);
    errorSpy = vi.spyOn(console, "error").mockImplementation(() => undefined);
    warnSpy = vi.spyOn(console, "warn").mockImplementation(() => undefined);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
    delete process.env.EXPO_PUBLIC_API_URL;
  });

  it("posts email and password to the password-login endpoint without logging the password", async () => {
    const response = {
      token: "token-1",
      user: { id: "user-1", email: "person@example.test" },
    };
    const fetchMock = vi
      .fn()
      .mockResolvedValue(new Response(JSON.stringify(response), { status: 200 }));
    vi.stubGlobal("fetch", fetchMock);

    const { ApiClient } = await import("../data/api");
    const client = new ApiClient();
    await expect(client.login("person@example.test", password)).resolves.toEqual(
      response,
    );

    expect(fetchMock).toHaveBeenCalledOnce();
    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(url).toBe("https://api.example.test/auth/login");
    expect(init.method).toBe("POST");
    expect(JSON.parse(String(init.body))).toEqual({
      email: "person@example.test",
      password,
    });
    expect(
      JSON.stringify([logSpy.mock.calls, errorSpy.mock.calls, warnSpy.mock.calls]),
    ).not.toContain(password);
  });

  it("does not treat invalid login credentials as an expired existing session", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ message: `invalid ${password}` }), {
          status: 401,
          statusText: "Unauthorized",
        }),
      ),
    );
    const onUnauthorized = vi.fn();
    const { ApiClient } = await import("../data/api");
    const client = new ApiClient();
    client.setOptions({ onUnauthorized });

    await expect(client.login("person@example.test", password)).rejects.toThrow(
      "invalid credentials",
    );
    expect(onUnauthorized).not.toHaveBeenCalled();
    expect(
      JSON.stringify([logSpy.mock.calls, errorSpy.mock.calls, warnSpy.mock.calls]),
    ).not.toContain(password);
  });
});

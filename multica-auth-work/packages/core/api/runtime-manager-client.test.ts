import { afterEach, describe, expect, it, vi } from "vitest";
import { ApiClient } from "./client";

afterEach(() => vi.unstubAllGlobals());

function jsonResponse(body: unknown = {}) {
  return new Response(JSON.stringify(body), {
    status: 200,
    headers: { "Content-Type": "application/json" },
  });
}

// C3 serves exactly two scopes: `/api/runtime-standards...` and
// `/api/workspaces/{workspaceId}/runtime-bindings/{bindingId}/configuration-versions...`
// plus that binding's `/rollback`. Tests below are split accordingly so an
// unserved route is never mistaken for a verified server contract.
describe("Runtime Manager API client — routes served by frozen C3", () => {
  it("uses the frozen standard routes", async () => {
    const fetchMock = vi.fn().mockImplementation(async () => jsonResponse({ items: [], next_cursor: null }));
    vi.stubGlobal("fetch", fetchMock);
    const client = new ApiClient("https://api.example.test");

    await client.listRuntimeStandards({ limit: 20 });
    await client.getRuntimeStandard("standard/1");

    expect(fetchMock.mock.calls.map(([url]) => url)).toEqual([
      "https://api.example.test/api/runtime-standards?limit=20",
      "https://api.example.test/api/runtime-standards/standard%2F1",
    ]);
  });

  it("uses the frozen binding configuration-version routes", async () => {
    const fetchMock = vi.fn().mockImplementation(async () => jsonResponse({ id: "version-1" }));
    vi.stubGlobal("fetch", fetchMock);
    const client = new ApiClient("https://api.example.test");
    const base = "https://api.example.test/api/workspaces/ws-1/runtime-bindings/binding-1";

    await client.createRuntimeBindingConfigurationVersion(
      "ws-1",
      "binding-1",
      { configuration: { version: "v1", values: { model: "opus" } }, reason: "pin_model" },
    );
    await client.validateRuntimeBindingConfigurationVersion("ws-1", "binding-1", "version-1");

    expect(fetchMock.mock.calls.map(([url]) => url)).toEqual([
      `${base}/configuration-versions`,
      `${base}/configuration-versions/version-1/validate`,
    ]);
  });

  it("sends the document contract version as `version`, never `schema_version`", async () => {
    const fetchMock = vi.fn().mockImplementation(async () => jsonResponse({ id: "version-1" }));
    vi.stubGlobal("fetch", fetchMock);
    const client = new ApiClient("https://api.example.test");

    await client.createRuntimeBindingConfigurationVersion(
      "ws-1",
      "binding-1",
      { configuration: { version: "v1", values: { model: "opus" } }, reason: "pin_model" },
    );

    const [, init] = fetchMock.mock.calls[0]!;
    const body = JSON.parse(String(init.body));
    expect(body.configuration.version).toBe("v1");
    expect(body.configuration).not.toHaveProperty("schema_version");
    // C3 decodes request bodies with unknown fields rejected, so a stray key is
    // a 400 rather than a silently dropped constraint.
    expect(Object.keys(body).sort()).toEqual(["configuration", "reason"]);
  });

  it("uses the frozen activation contract and sends a symbolic reason code", async () => {
    const fetchMock = vi.fn().mockImplementation(async () => jsonResponse({ active_version_id: "version-2" }));
    vi.stubGlobal("fetch", fetchMock);
    const client = new ApiClient("https://api.example.test");

    await client.activateRuntimeBindingConfigurationVersion(
      "ws-1",
      "binding-1",
      "version-2",
      {
        expected_active_version_id: "version-1",
        expected_binding_generation: 4,
        // C3 requires lowercase symbolic codes; prose such as
        // "Use approved model" is rejected with 400 invalid_argument.
        reason: "use_approved_model",
      },
      "idem-2",
    );

    const [url, init] = fetchMock.mock.calls[0]!;
    expect(url).toBe(
      "https://api.example.test/api/workspaces/ws-1/runtime-bindings/binding-1/configuration-versions/version-2/activate",
    );
    expect(JSON.parse(String(init.body))).toEqual({
      expected_active_version_id: "version-1",
      expected_binding_generation: 4,
      reason: "use_approved_model",
    });
    expect(/^[a-z0-9_\-.]+$/.test(JSON.parse(String(init.body)).reason)).toBe(true);
  });

  it("sends explicit null for a first activation rather than omitting the fence", async () => {
    const fetchMock = vi.fn().mockImplementation(async () => jsonResponse({ active_version_id: "version-1" }));
    vi.stubGlobal("fetch", fetchMock);
    const client = new ApiClient("https://api.example.test");

    await client.activateRuntimeBindingConfigurationVersion(
      "ws-1",
      "binding-1",
      "version-1",
      {
        expected_active_version_id: null,
        expected_binding_generation: 1,
        reason: "first_activation",
      },
    );

    const [, init] = fetchMock.mock.calls[0]!;
    const body = String(init.body);
    // Omitted and null are different to C3: omitted is a 400, null is the
    // explicit assertion that no active version exists.
    expect(body).toContain('"expected_active_version_id":null');
    expect(JSON.parse(body)).toHaveProperty("expected_active_version_id", null);
  });

  it("carries the binding generation on rollback, which binding scope requires", async () => {
    const fetchMock = vi.fn().mockImplementation(async () => jsonResponse({ active_version_id: "version-1" }));
    vi.stubGlobal("fetch", fetchMock);
    const client = new ApiClient("https://api.example.test");

    await client.rollbackRuntimeBindingConfiguration("ws-1", "binding-1", {
      target_version_id: "version-1",
      expected_active_version_id: "version-3",
      expected_binding_generation: 6,
      reason: "revert_bad_activation",
    });

    const [url, init] = fetchMock.mock.calls[0]!;
    expect(url).toBe(
      "https://api.example.test/api/workspaces/ws-1/runtime-bindings/binding-1/rollback",
    );
    const body = JSON.parse(String(init.body));
    expect(body.expected_binding_generation).toBe(6);
    expect(body.target_version_id).toBe("version-1");
  });

  it("omits the binding generation on standard rollback, which forbids it", async () => {
    const fetchMock = vi.fn().mockImplementation(async () => jsonResponse({ active_version_id: "version-1" }));
    vi.stubGlobal("fetch", fetchMock);
    const client = new ApiClient("https://api.example.test");

    await client.rollbackRuntimeStandard("standard-1", {
      target_version_id: "version-1",
      expected_active_version_id: "version-3",
      reason: "revert_bad_activation",
    });

    const [, init] = fetchMock.mock.calls[0]!;
    // Standard scope has no generation fence and rejects unknown fields.
    expect(JSON.parse(String(init.body))).not.toHaveProperty("expected_binding_generation");
  });

  it("always sends an Idempotency-Key, generating one when the caller omits it", async () => {
    const fetchMock = vi.fn().mockImplementation(async () => jsonResponse({ id: "standard-1" }));
    vi.stubGlobal("fetch", fetchMock);
    const client = new ApiClient("https://api.example.test");

    await client.createRuntimeStandard({ name: "Approved" });
    await client.createRuntimeStandard({ name: "Approved" }, "idem-explicit");

    const generated = (fetchMock.mock.calls[0]![1].headers as Record<string, string>)["Idempotency-Key"];
    const explicit = (fetchMock.mock.calls[1]![1].headers as Record<string, string>)["Idempotency-Key"];
    // Narrow rather than coerce: coercing undefined to "undefined" would make
    // the length bound below pass even when the header is absent.
    if (typeof generated !== "string") {
      throw new Error("client did not send a generated Idempotency-Key header");
    }
    expect(generated.length).toBeGreaterThan(0);
    expect(generated.length).toBeLessThanOrEqual(200);
    expect(explicit).toBe("idem-explicit");
  });

  it("surfaces canonical nested Runtime Manager errors without leaking detail", async () => {
    vi.stubGlobal(
      "fetch",
      // A Response body can be read only once, and this case issues two calls,
      // so each invocation has to build its own response.
      vi.fn().mockImplementation(
        async () =>
          new Response(
            JSON.stringify({
              error: {
                code: "generation_conflict",
                message: "Credential failed at /home/private for person@example.test: token=secret",
                request_id: "request-1",
                retryable: true,
              },
            }),
            { status: 409, statusText: "Conflict", headers: { "Content-Type": "application/json" } },
          ),
      ),
    );
    const client = new ApiClient("https://api.example.test");

    await expect(
      client.validateRuntimeBindingConfigurationVersion("ws-1", "binding-1", "version-1"),
    ).rejects.toMatchObject({
      message: "Runtime Manager request failed (generation_conflict; HTTP 409).",
      status: 409,
    });

    try {
      await client.validateRuntimeBindingConfigurationVersion("ws-1", "binding-1", "version-1");
    } catch (error) {
      expect(String(error)).not.toMatch(/home\/private|example\.test|token=secret|credential/i);
    }
  });
});

// The routes below are constructed correctly by the client but have NO endpoint
// in the frozen C3 surface: no session list/create/enroll, no binding list, no
// binding detail, no credential-home list and no home assignment. These assert
// client URL construction only and must not be read as server-contract
// verification. Reported as the primary Lane D blocking gap.
describe("Runtime Manager API client — routes with no frozen C3 endpoint", () => {
  it("builds the proposed session, binding and credential-home paths", async () => {
    const fetchMock = vi.fn().mockImplementation(async () => jsonResponse({ items: [], next_cursor: null }));
    vi.stubGlobal("fetch", fetchMock);
    const client = new ApiClient("https://api.example.test");

    await client.listRuntimeSessions();
    await client.listRuntimeBindings("ws/1");
    await client.listRuntimeCredentialHomes("ws/1");
    await client.getRuntimeBinding("ws-1", "binding-1");

    expect(fetchMock.mock.calls.map(([url]) => url)).toEqual([
      "https://api.example.test/api/runtime-sessions",
      "https://api.example.test/api/workspaces/ws%2F1/runtime-bindings",
      "https://api.example.test/api/workspaces/ws%2F1/credential-homes",
      "https://api.example.test/api/workspaces/ws-1/runtime-bindings/binding-1",
    ]);
  });

  it("sends compare-and-swap attachment fields with no path, account or credential", async () => {
    const fetchMock = vi.fn().mockImplementation(async () => jsonResponse({ id: "binding-1" }));
    vi.stubGlobal("fetch", fetchMock);
    const client = new ApiClient("https://api.example.test");

    await client.assignRuntimeCredentialHome(
      "ws-1",
      "binding-1",
      {
        home_ref: "home_opaque",
        expected_binding_generation: 7,
        expected_catalog_generation: 11,
      },
      "idem-1",
    );

    const [, init] = fetchMock.mock.calls[0]!;
    expect(init).toMatchObject({
      method: "POST",
      body: JSON.stringify({
        home_ref: "home_opaque",
        expected_binding_generation: 7,
        expected_catalog_generation: 11,
      }),
    });
    expect((init.headers as Record<string, string>)["Idempotency-Key"]).toBe("idem-1");
    expect(String(init.body)).not.toMatch(/path|account|credential/i);
  });
});

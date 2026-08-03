import { afterEach, describe, expect, it, vi } from "vitest";
import { ApiClient } from "./client";

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("ApiClient Packet B model discovery", () => {
  it("forwards one AbortSignal through model-list initiation and polling", async () => {
    const pending = {
      id: "request-1",
      runtime_id: "runtime-1",
      status: "pending",
      supported: true,
      created_at: "2026-08-03T00:00:00Z",
      updated_at: "2026-08-03T00:00:00Z",
    };
    const completed = {
      ...pending,
      status: "completed",
      updated_at: "2026-08-03T00:00:01Z",
      models: [],
    };
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(
        new Response(JSON.stringify(pending), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        }),
      )
      .mockResolvedValueOnce(
        new Response(JSON.stringify(completed), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        }),
      );
    vi.stubGlobal("fetch", fetchMock);

    const client = new ApiClient("https://api.example.test");
    const controller = new AbortController();

    await client.initiateListModels("runtime-1", controller.signal);
    await client.getListModelsResult(
      "runtime-1",
      "request-1",
      controller.signal,
    );

    expect(fetchMock).toHaveBeenNthCalledWith(
      1,
      "https://api.example.test/api/runtimes/runtime-1/models",
      expect.objectContaining({
        method: "POST",
        signal: controller.signal,
        credentials: "include",
      }),
    );
    expect(fetchMock).toHaveBeenNthCalledWith(
      2,
      "https://api.example.test/api/runtimes/runtime-1/models/request-1",
      expect.objectContaining({
        signal: controller.signal,
        credentials: "include",
      }),
    );
  });
});

// @vitest-environment jsdom

import {
  QueryClient,
  QueryClientProvider,
  useQuery,
} from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import type { PropsWithChildren } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type {
  RuntimeModel,
  RuntimeModelListRequest,
  RuntimeModelsResult,
} from "../types";

const mockInitiateListModels = vi.hoisted(() => vi.fn());
const mockGetListModelsResult = vi.hoisted(() => vi.fn());

vi.mock("../api", () => ({
  api: {
    initiateListModels: (...args: unknown[]) =>
      mockInitiateListModels(...args),
    getListModelsResult: (...args: unknown[]) =>
      mockGetListModelsResult(...args),
  },
}));

import {
  filterRuntimeModelGroups,
  groupRuntimeModelsByProvider,
  MAX_SESSION_MODELS_PER_CATALOG,
  MAX_SESSION_RUNTIME_CATALOGS,
  resolveRuntimeModels,
  resolveRuntimeProviderIdentity,
  RUNTIME_MODEL_CATALOG_GC_TIME_MS,
  runtimeListSnapshotFromCache,
  runtimeModelSearchMatchesProvider,
  runtimeModelsKeys,
  runtimeModelsOptions,
  UNKNOWN_CUSTOM_RUNTIME_PROVIDER,
  useLastKnownRuntimeModels,
  useRuntimeModelsLifecycle,
  useRuntimeProviderIdentity,
} from "./models";

const MODELS: RuntimeModel[] = [
  { id: "claude-sonnet", label: "Claude Sonnet", provider: "anthropic" },
  { id: "gpt-5", label: "GPT 5", provider: "openai" },
  { id: "runtime-default", label: "Runtime default" },
];

function listRequest(
  status: RuntimeModelListRequest["status"],
  models: RuntimeModel[] = [],
): RuntimeModelListRequest {
  return {
    id: "request-1",
    runtime_id: "runtime-1",
    status,
    models,
    supported: true,
    created_at: "2026-07-18T00:00:00Z",
    updated_at: "2026-07-18T00:00:00Z",
  };
}

const CACHED_RESULT: RuntimeModelsResult = {
  models: MODELS,
  supported: true,
};

const BUILTIN_RUNTIME_PROVIDERS = [
  "claude",
  "codebuddy",
  "cline",
  "codex",
  "copilot",
  "nim",
  "opencode",
  "openclaw",
  "hermes",
  "gemini",
  "pi",
  "cursor",
  "kimi",
  "kiro",
  "antigravity",
  "qoder",
] as const;

function useOfflineCatalog(runtimeId: string) {
  const query = useQuery(runtimeModelsOptions(runtimeId, false));
  return useLastKnownRuntimeModels(
    runtimeId,
    query.data,
    query.dataUpdatedAt,
  );
}

describe("runtime model catalog options", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("preserves the runtime-specific cache key while offline and disables discovery", () => {
    const options = runtimeModelsOptions("runtime-1", false);

    expect(options.queryKey).toEqual(runtimeModelsKeys.forRuntime("runtime-1"));
    expect(options.enabled).toBe(false);
  });

  it("groups multiple providers and falls back only for missing providers", () => {
    const groups = groupRuntimeModelsByProvider(MODELS, "runtime-vendor");

    expect(groups.map((group) => group.provider)).toEqual([
      "anthropic",
      "openai",
      "runtime-vendor",
    ]);
    expect(groups[0]?.models[0]?.provider).toBe("anthropic");
    expect(groups[1]?.models[0]?.provider).toBe("openai");
    expect(groups[2]?.models[0]?.provider).toBe("runtime-vendor");
  });

  it("names provider-less rows for every supported built-in identity", () => {
    for (const provider of BUILTIN_RUNTIME_PROVIDERS) {
      const fallback = resolveRuntimeProviderIdentity(provider);
      const groups = groupRuntimeModelsByProvider(
        [{ id: `${provider}-default`, label: `${provider} default` }],
        fallback,
      );

      expect(groups).toHaveLength(1);
      expect(groups[0]?.provider).toBe(provider);
      expect(groups[0]?.models[0]?.provider).toBe(provider);
    }
  });

  it("keeps provider precedence and labels arbitrary custom identities without guessing", () => {
    expect(
      resolveRuntimeProviderIdentity(
        "custom-claude-runtime",
        "explicit-provider",
        "cached-provider",
        "remembered-provider",
      ),
    ).toBe("explicit-provider");
    expect(
      resolveRuntimeProviderIdentity(
        "custom-claude-runtime",
        undefined,
        "cached-provider",
        "remembered-provider",
      ),
    ).toBe("cached-provider");
    expect(resolveRuntimeProviderIdentity("custom-claude-runtime")).toBe(
      UNKNOWN_CUSTOM_RUNTIME_PROVIDER,
    );

    const groups = groupRuntimeModelsByProvider(
      [
        { id: "runtime-default", label: "Runtime default" },
        { id: "explicit", label: "Explicit", provider: "model-provider" },
      ],
      UNKNOWN_CUSTOM_RUNTIME_PROVIDER,
    );
    expect(groups.map((group) => group.provider)).toEqual([
      UNKNOWN_CUSTOM_RUNTIME_PROVIDER,
      "model-provider",
    ]);
  });

  it("retains an authoritative provider for an opaque runtime id within one session", () => {
    const queryClient = new QueryClient();
    const wrapper = ({ children }: PropsWithChildren) => (
      <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
    );
    const identity = renderHook(
      ({ cachedProvider }: { cachedProvider?: string }) =>
        useRuntimeProviderIdentity(
          "opaque-runtime-uuid",
          undefined,
          cachedProvider,
        ),
      {
        initialProps: { cachedProvider: "kimi" } as {
          cachedProvider?: string;
        },
        wrapper,
      },
    );

    expect(identity.result.current).toBe("kimi");
    identity.rerender({ cachedProvider: undefined });
    expect(identity.result.current).toBe("kimi");

    identity.unmount();
    queryClient.clear();
    const nextSession = renderHook(
      () =>
        useRuntimeProviderIdentity("opaque-runtime-uuid", undefined, undefined),
      { wrapper },
    );
    expect(nextSession.result.current).toBe(UNKNOWN_CUSTOM_RUNTIME_PROVIDER);
  });

  it("retains a bounded offline catalog after normal React Query gcTime without API calls", async () => {
    vi.useFakeTimers();
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    const runtimeId = "offline-gc-runtime";
    const oversizedCatalog: RuntimeModelsResult = {
      supported: true,
      models: Array.from(
        { length: MAX_SESSION_MODELS_PER_CATALOG + 10 },
        (_, index) => ({ id: `model-${index}`, label: `Model ${index}` }),
      ),
    };
    queryClient.setQueryData(
      runtimeModelsKeys.forRuntime(runtimeId),
      oversizedCatalog,
    );
    const wrapper = ({ children }: PropsWithChildren) => (
      <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
    );

    const mounted = renderHook(() => useOfflineCatalog(runtimeId), { wrapper });
    expect(mounted.result.current?.models).toHaveLength(
      MAX_SESSION_MODELS_PER_CATALOG + 10,
    );
    mounted.unmount();

    await act(async () => {
      await vi.advanceTimersByTimeAsync(RUNTIME_MODEL_CATALOG_GC_TIME_MS + 1);
    });
    expect(
      queryClient.getQueryData(runtimeModelsKeys.forRuntime(runtimeId)),
    ).toBeUndefined();

    const remounted = renderHook(() => useOfflineCatalog(runtimeId), { wrapper });
    expect(remounted.result.current?.models).toHaveLength(
      MAX_SESSION_MODELS_PER_CATALOG,
    );
    expect(remounted.result.current?.models[0]?.id).toBe("model-0");
    expect(mockInitiateListModels).not.toHaveBeenCalled();
    expect(mockGetListModelsResult).not.toHaveBeenCalled();
    remounted.unmount();
    queryClient.clear();
  });

  it("evicts the oldest retained runtime catalog beyond the session bound", () => {
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    for (let index = 0; index <= MAX_SESSION_RUNTIME_CATALOGS; index++) {
      queryClient.setQueryData(runtimeModelsKeys.forRuntime(`runtime-${index}`), {
        supported: true,
        models: [{ id: `model-${index}`, label: `Model ${index}` }],
      } satisfies RuntimeModelsResult);
    }
    const wrapper = ({ children }: PropsWithChildren) => (
      <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
    );
    const retained = renderHook(
      ({ runtimeId }) => useOfflineCatalog(runtimeId),
      { initialProps: { runtimeId: "runtime-0" }, wrapper },
    );
    for (let index = 1; index <= MAX_SESSION_RUNTIME_CATALOGS; index++) {
      retained.rerender({ runtimeId: `runtime-${index}` });
    }
    retained.unmount();
    queryClient.removeQueries({ queryKey: runtimeModelsKeys.all() });

    const oldest = renderHook(() => useOfflineCatalog("runtime-0"), {
      wrapper,
    });
    const newest = renderHook(
      () => useOfflineCatalog(`runtime-${MAX_SESSION_RUNTIME_CATALOGS}`),
      { wrapper },
    );
    expect(oldest.result.current).toBeUndefined();
    expect(newest.result.current?.models[0]?.id).toBe(
      `model-${MAX_SESSION_RUNTIME_CATALOGS}`,
    );
  });

  it("invalidates a retained catalog when a newer online lifecycle arrives", async () => {
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    const runtimeId = "retained-runtime";
    queryClient.setQueryData(
      runtimeModelsKeys.forRuntime(runtimeId),
      CACHED_RESULT,
      { updatedAt: 100 },
    );
    const wrapper = ({ children }: PropsWithChildren) => (
      <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
    );
    const seeded = renderHook(() => useOfflineCatalog(runtimeId), { wrapper });
    seeded.unmount();
    queryClient.removeQueries({
      queryKey: runtimeModelsKeys.forRuntime(runtimeId),
      exact: true,
    });

    const lifecycle = renderHook(
      () => ({
        enabled: useRuntimeModelsLifecycle(runtimeId, true, 200),
        catalog: useLastKnownRuntimeModels(runtimeId, undefined, 0),
      }),
      { wrapper },
    );

    await waitFor(() => {
      expect(lifecycle.result.current.enabled).toBe(true);
      expect(lifecycle.result.current.catalog).toBeUndefined();
    });
  });

  it("matches provider names during model search", () => {
    const groups = groupRuntimeModelsByProvider(MODELS, "runtime-vendor");

    expect(filterRuntimeModelGroups(groups, "OPENAI")).toEqual([groups[1]]);
    expect(filterRuntimeModelGroups(groups, "runtime-vendor")).toEqual([
      groups[2],
    ]);
    expect(runtimeModelSearchMatchesProvider(groups, "OPENAI")).toBe(true);
    expect(runtimeModelSearchMatchesProvider(groups, "Claude Sonnet")).toBe(
      false,
    );
  });

  it("uses only the exact authoritative runtime-list key when caches conflict", () => {
    const queryClient = new QueryClient();
    const workspaceKey = ["runtimes", "workspace-1", "list"] as const;
    const mineKey = ["runtimes", "workspace-1", "list", "mine"] as const;
    queryClient.setQueryData(workspaceKey, [
      { id: "runtime-1", provider: "workspace-provider" },
    ]);
    queryClient.setQueryData(mineKey, [
      { id: "runtime-1", provider: "mine-provider" },
    ]);

    expect(
      runtimeListSnapshotFromCache(
        queryClient,
        "runtime-1",
        workspaceKey,
      ).provider,
    ).toBe("workspace-provider");
    expect(
      runtimeListSnapshotFromCache(queryClient, "runtime-1", mineKey).provider,
    ).toBe("mine-provider");
  });

  it("passes one AbortSignal through initiation, polling sleep, and polling API", async () => {
    vi.useFakeTimers();
    const controller = new AbortController();
    mockInitiateListModels.mockResolvedValue(listRequest("pending"));
    mockGetListModelsResult.mockResolvedValue(
      listRequest("completed", MODELS),
    );

    const result = resolveRuntimeModels("runtime-1", controller.signal);
    await vi.advanceTimersByTimeAsync(0);
    expect(mockInitiateListModels).toHaveBeenCalledWith(
      "runtime-1",
      controller.signal,
    );

    await vi.advanceTimersByTimeAsync(500);
    await expect(result).resolves.toEqual(CACHED_RESULT);
    expect(mockGetListModelsResult).toHaveBeenCalledWith(
      "runtime-1",
      "request-1",
      controller.signal,
    );
  });

  it("aborts an in-flight polling sleep before another API call", async () => {
    vi.useFakeTimers();
    const controller = new AbortController();
    mockInitiateListModels.mockResolvedValue(listRequest("pending"));

    const result = resolveRuntimeModels("runtime-1", controller.signal);
    await vi.advanceTimersByTimeAsync(0);
    controller.abort();

    await expect(result).rejects.toMatchObject({ name: "AbortError" });
    expect(mockGetListModelsResult).not.toHaveBeenCalled();
  });

  it("cancels the exact runtime catalog on an online-to-offline transition", async () => {
    const queryClient = new QueryClient();
    queryClient.setQueryData(
      runtimeModelsKeys.forRuntime("runtime-1"),
      CACHED_RESULT,
      { updatedAt: 200 },
    );
    const cancel = vi.spyOn(queryClient, "cancelQueries");
    const wrapper = ({ children }: PropsWithChildren) => (
      <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
    );

    const { rerender } = renderHook(
      ({ online }) =>
        useRuntimeModelsLifecycle("runtime-1", online, 100),
      { initialProps: { online: true }, wrapper },
    );

    expect(cancel).not.toHaveBeenCalled();
    rerender({ online: false });

    await waitFor(() => {
      expect(cancel).toHaveBeenCalledWith({
        queryKey: runtimeModelsKeys.forRuntime("runtime-1"),
        exact: true,
      });
    });
  });

  it("invalidates a fresh stale catalog when mounted after reconnect", async () => {
    const queryClient = new QueryClient();
    queryClient.setQueryData(
      runtimeModelsKeys.forRuntime("runtime-1"),
      CACHED_RESULT,
      { updatedAt: 100 },
    );
    const invalidate = vi.spyOn(queryClient, "invalidateQueries");
    const wrapper = ({ children }: PropsWithChildren) => (
      <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
    );

    const { result } = renderHook(
      () => useRuntimeModelsLifecycle("runtime-1", true, 200),
      { wrapper },
    );

    await waitFor(() => {
      expect(invalidate).toHaveBeenCalledWith({
        queryKey: runtimeModelsKeys.forRuntime("runtime-1"),
        exact: true,
        refetchType: "none",
      });
      expect(result.current).toBe(true);
    });
  });
});

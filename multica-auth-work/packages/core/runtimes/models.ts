import { useEffect, useMemo, useState } from "react";
import {
  queryOptions,
  type QueryClient,
  useQueryClient,
} from "@tanstack/react-query";
import { api } from "../api";
import type {
  RuntimeDevice,
  RuntimeModel,
  RuntimeModelsResult,
} from "../types/agent";

export const runtimeModelsKeys = {
  all: () => ["runtimes", "models"] as const,
  forRuntime: (runtimeId: string) =>
    [...runtimeModelsKeys.all(), runtimeId] as const,
};

const POLL_INTERVAL_MS = 500;
// The daemon allows slow provider CLIs up to 40s, so polling must outlive that
// bound plus heartbeat/report latency. The server still enforces its own 60s
// running timeout as the final escape hatch.
const POLL_TIMEOUT_MS = 50_000;

// React Query intentionally releases inactive catalogs after ten minutes.
// Keep a smaller QueryClient-scoped session fallback so an offline picker can
// remount after that GC without a request. At most 32 recently used runtimes
// and 256 rows per runtime are retained (8,192 model rows worst case). The
// non-fetching QueryClient sentinel survives normal catalog GC but is
// removed by queryClient.clear() at logout. Bounded recency eviction prevents
// the sentinel from becoming an unbounded process-global catalog cache.
export const RUNTIME_MODEL_CATALOG_GC_TIME_MS = 10 * 60 * 1000;
export const MAX_SESSION_RUNTIME_CATALOGS = 32;
export const MAX_SESSION_MODELS_PER_CATALOG = 256;
const MAX_SESSION_RUNTIME_IDENTITIES = 64;
export const UNKNOWN_CUSTOM_RUNTIME_PROVIDER = "Unknown/Custom";

const KNOWN_RUNTIME_PROVIDERS = new Set([
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
]);

interface SessionRuntimeCatalog {
  result: RuntimeModelsResult;
  updatedAt: number;
}

interface RuntimeModelSessionState {
  catalogs: Map<string, SessionRuntimeCatalog>;
  providers: Map<string, string>;
}

const runtimeModelSessionKey = ["runtime-model-session-cache"] as const;

function readRuntimeModelSession(
  queryClient: QueryClient,
): RuntimeModelSessionState | undefined {
  return queryClient.getQueryData<RuntimeModelSessionState>(
    runtimeModelSessionKey,
  );
}

function updateRuntimeModelSession(
  queryClient: QueryClient,
  update: (state: RuntimeModelSessionState) => RuntimeModelSessionState,
): void {
  queryClient.setQueryDefaults(runtimeModelSessionKey, {
    gcTime: Infinity,
    staleTime: Infinity,
  });
  queryClient.setQueryData<RuntimeModelSessionState>(
    runtimeModelSessionKey,
    (current) =>
      update(current ?? { catalogs: new Map(), providers: new Map() }),
  );
}

function setBoundedRecent<K, V>(
  map: Map<K, V>,
  key: K,
  value: V,
  maxSize: number,
): void {
  map.delete(key);
  map.set(key, value);
  while (map.size > maxSize) {
    const oldest = map.keys().next().value as K | undefined;
    if (oldest === undefined) break;
    map.delete(oldest);
  }
}

function retainedRuntimeCatalog(
  queryClient: QueryClient,
  runtimeId: string,
): SessionRuntimeCatalog | undefined {
  return readRuntimeModelSession(queryClient)?.catalogs.get(runtimeId);
}

function rememberRuntimeCatalog(
  queryClient: QueryClient,
  runtimeId: string,
  result: RuntimeModelsResult,
  updatedAt: number,
): void {
  const boundedResult = {
    ...result,
    models: result.models.slice(0, MAX_SESSION_MODELS_PER_CATALOG),
  };
  updateRuntimeModelSession(queryClient, (state) => {
    const catalogs = new Map(state.catalogs);
    setBoundedRecent(
      catalogs,
      runtimeId,
      { result: boundedResult, updatedAt },
      MAX_SESSION_RUNTIME_CATALOGS,
    );
    return { ...state, catalogs };
  });
}

function forgetRuntimeCatalog(queryClient: QueryClient, runtimeId: string): void {
  const current = readRuntimeModelSession(queryClient);
  if (!current?.catalogs.has(runtimeId)) return;
  updateRuntimeModelSession(queryClient, (state) => {
    const catalogs = new Map(state.catalogs);
    catalogs.delete(runtimeId);
    return { ...state, catalogs };
  });
}

// resolveRuntimeModels initiates a list-models request against the daemon
// (via heartbeat piggyback) and polls until the daemon reports back or
// the request times out. Returns both the models list and a
// `supported` flag: `supported=false` means the provider ignores
// per-agent model selection entirely (hermes today) — the UI uses
// this to disable its dropdown instead of accepting a value that
// wouldn't be honoured at runtime.
export async function resolveRuntimeModels(
  runtimeId: string,
  signal?: AbortSignal,
): Promise<RuntimeModelsResult> {
  throwIfAborted(signal);
  const initial = await api.initiateListModels(runtimeId, signal);
  const start = Date.now();
  let current = initial;
  while (current.status === "pending" || current.status === "running") {
    if (Date.now() - start > POLL_TIMEOUT_MS) {
      throw new Error("model discovery timed out");
    }
    await abortableDelay(POLL_INTERVAL_MS, signal);
    current = await api.getListModelsResult(runtimeId, initial.id, signal);
  }
  if (current.status === "failed" || current.status === "timeout") {
    throw new Error(current.error || "model discovery failed");
  }
  return { models: current.models ?? [], supported: current.supported };
}

export function runtimeModelsOptions(
  runtimeId: string | null | undefined,
  enabled = true,
) {
  return queryOptions({
    queryKey: runtimeId
      ? runtimeModelsKeys.forRuntime(runtimeId)
      : runtimeModelsKeys.all(),
    queryFn: ({ signal }) =>
      resolveRuntimeModels(runtimeId as string, signal),
    // Keep the runtime-specific key even while discovery is disabled. This
    // lets an offline runtime continue to expose its cached catalog without
    // making a request.
    enabled: Boolean(runtimeId) && enabled,
    // Models rarely change; cache for 60s to match the server-side
    // cache in agent.ListModels.
    staleTime: 60_000,
    gcTime: RUNTIME_MODEL_CATALOG_GC_TIME_MS,
    retry: false,
  });
}

/**
 * Return live React Query data when present, otherwise the bounded catalog
 * retained for this QueryClient session. Remembering happens after commit so
 * render stays side-effect free. This hook never initiates discovery.
 */
export function useLastKnownRuntimeModels(
  runtimeId: string | null | undefined,
  liveResult: RuntimeModelsResult | undefined,
  liveUpdatedAt: number,
): RuntimeModelsResult | undefined {
  const queryClient = useQueryClient();
  const retained = runtimeId
    ? retainedRuntimeCatalog(queryClient, runtimeId)?.result
    : undefined;

  useEffect(() => {
    if (!runtimeId || !liveResult) return;
    rememberRuntimeCatalog(
      queryClient,
      runtimeId,
      liveResult,
      liveUpdatedAt || Date.now(),
    );
  }, [liveResult, liveUpdatedAt, queryClient, runtimeId]);

  return liveResult ?? retained;
}

/** Resolve only identities that cannot be confused with arbitrary custom IDs. */
export function knownRuntimeProviderFromIdentity(
  runtimeId: string | null | undefined,
): string | undefined {
  const normalized = runtimeId?.trim().toLowerCase() ?? "";
  if (KNOWN_RUNTIME_PROVIDERS.has(normalized)) return normalized;

  const marked = /^(?:builtin|provider|runtime):([^:]+)(?::.*)?$/.exec(
    normalized,
  )?.[1];
  return marked && KNOWN_RUNTIME_PROVIDERS.has(marked) ? marked : undefined;
}

export function resolveRuntimeProviderIdentity(
  runtimeId: string | null | undefined,
  explicitProvider?: string,
  cachedProvider?: string,
  rememberedProvider?: string,
): string {
  return (
    explicitProvider?.trim() ||
    cachedProvider?.trim() ||
    rememberedProvider?.trim() ||
    knownRuntimeProviderFromIdentity(runtimeId) ||
    UNKNOWN_CUSTOM_RUNTIME_PROVIDER
  );
}

/**
 * Resolve a model-group fallback without guessing from arbitrary custom IDs.
 * Exact props/cache win; their runtime-id mapping is retained only for this
 * QueryClient session so an offline remount can still name an opaque UUID.
 */
export function useRuntimeProviderIdentity(
  runtimeId: string | null | undefined,
  explicitProvider?: string,
  cachedProvider?: string,
): string {
  const queryClient = useQueryClient();
  const rememberedProvider = runtimeId
    ? readRuntimeModelSession(queryClient)?.providers.get(runtimeId)
    : undefined;
  const authoritativeProvider =
    explicitProvider?.trim() || cachedProvider?.trim() || "";

  useEffect(() => {
    if (!runtimeId || !authoritativeProvider) return;
    updateRuntimeModelSession(queryClient, (state) => {
      const providers = new Map(state.providers);
      setBoundedRecent(
        providers,
        runtimeId,
        authoritativeProvider,
        MAX_SESSION_RUNTIME_IDENTITIES,
      );
      return { ...state, providers };
    });
  }, [authoritativeProvider, queryClient, runtimeId]);

  return resolveRuntimeProviderIdentity(
    runtimeId,
    explicitProvider,
    cachedProvider,
    rememberedProvider,
  );
}

function abortReason(signal: AbortSignal): unknown {
  return signal.reason ?? new DOMException("The operation was aborted", "AbortError");
}

function throwIfAborted(signal?: AbortSignal): void {
  if (signal?.aborted) throw abortReason(signal);
}

function abortableDelay(ms: number, signal?: AbortSignal): Promise<void> {
  throwIfAborted(signal);
  if (!signal) {
    return new Promise((resolve) => setTimeout(resolve, ms));
  }

  return new Promise((resolve, reject) => {
    const timeout = setTimeout(() => {
      signal.removeEventListener("abort", onAbort);
      resolve();
    }, ms);
    const onAbort = () => {
      clearTimeout(timeout);
      reject(abortReason(signal));
    };
    signal.addEventListener("abort", onAbort, { once: true });
  });
}

export interface RuntimeModelGroup {
  provider: string;
  models: RuntimeModel[];
}

/**
 * Group models by their explicit provider, falling back to the selected
 * runtime's provider only when a catalog row omitted it. Explicit provider
 * metadata always wins.
 */
export function groupRuntimeModelsByProvider(
  models: RuntimeModel[],
  fallbackProvider = "",
): RuntimeModelGroup[] {
  const groups = new Map<string, RuntimeModel[]>();
  const fallback = fallbackProvider.trim();

  for (const model of models) {
    const explicitProvider = model.provider?.trim() ?? "";
    const provider = explicitProvider || fallback;
    const normalized =
      !explicitProvider && provider ? { ...model, provider } : model;
    const group = groups.get(provider);
    if (group) group.push(normalized);
    else groups.set(provider, [normalized]);
  }

  return Array.from(groups, ([provider, groupedModels]) => ({
    provider,
    models: groupedModels,
  }));
}

/** Search model identifiers, labels, and provider group names. */
export function filterRuntimeModelGroups(
  groups: RuntimeModelGroup[],
  search: string,
): RuntimeModelGroup[] {
  const needle = search.trim().toLowerCase();
  if (!needle) return groups;

  return groups.flatMap((group) => {
    if (group.provider.toLowerCase().includes(needle)) return [group];

    const models = group.models.filter(
      (model) =>
        model.id.toLowerCase().includes(needle) ||
        model.label.toLowerCase().includes(needle) ||
        model.provider?.toLowerCase().includes(needle),
    );
    return models.length > 0 ? [{ ...group, models }] : [];
  });
}

/** Whether the current search is a provider-group search. */
export function runtimeModelSearchMatchesProvider(
  groups: RuntimeModelGroup[],
  search: string,
): boolean {
  const needle = search.trim().toLowerCase();
  return (
    needle.length > 0 &&
    groups.some((group) => group.provider.toLowerCase().includes(needle))
  );
}

export type RuntimeListQueryKey =
  | readonly ["runtimes", string, "list"]
  | readonly ["runtimes", string, "list", "mine"];

export interface RuntimeListCacheSnapshot {
  provider: string;
  generation: number;
}

/**
 * Resolve the provider from already-cached runtime-list data. The picker
 * parents already subscribe to this server-state query; consulting the cache
 * avoids a second source of truth and never initiates a request.
 */
export function runtimeListSnapshotFromCache(
  queryClient: QueryClient,
  runtimeId: string | null | undefined,
  runtimeListKey: RuntimeListQueryKey,
): RuntimeListCacheSnapshot {
  const runtimes = queryClient.getQueryData<RuntimeDevice[]>(runtimeListKey);
  const runtime = runtimeId
    ? runtimes?.find((candidate) => candidate.id === runtimeId)
    : undefined;
  return {
    provider: runtime?.provider?.trim() ?? "",
    // React Query advances dataUpdatedAt whenever the authoritative runtime
    // list is refreshed after lifecycle/reconnect invalidation. Comparing it
    // with the catalog timestamp also covers pickers mounted after reconnect.
    generation: queryClient.getQueryState(runtimeListKey)?.dataUpdatedAt ?? 0,
  };
}

/**
 * Prepare one exact catalog for runtime lifecycle changes. The hook disables
 * discovery until a newer runtime-list generation has marked the catalog
 * stale. That sequencing prevents an enable-triggered request followed by a
 * second invalidate-triggered request.
 */
export function useRuntimeModelsLifecycle(
  runtimeId: string | null | undefined,
  runtimeOnline: boolean,
  runtimeListGeneration: number,
): boolean {
  const queryClient = useQueryClient();
  const queryKey = useMemo(
    () =>
      runtimeId
        ? runtimeModelsKeys.forRuntime(runtimeId)
        : runtimeModelsKeys.all(),
    [runtimeId],
  );
  const queryState = queryClient.getQueryState(queryKey);
  const retainedUpdatedAt = runtimeId
    ? retainedRuntimeCatalog(queryClient, runtimeId)?.updatedAt ?? 0
    : 0;
  const [preparedLifecycle, setPreparedLifecycle] = useState<{
    runtimeId: string;
    generation: number;
  } | null>(null);
  const lifecyclePrepared =
    preparedLifecycle !== null &&
    preparedLifecycle.runtimeId === runtimeId &&
    preparedLifecycle.generation >= runtimeListGeneration;
  const catalogPredatesRuntime =
    Boolean(runtimeId) &&
    runtimeListGeneration > 0 &&
    runtimeListGeneration >
      Math.max(queryState?.dataUpdatedAt ?? 0, retainedUpdatedAt);
  const waitingForInvalidation =
    runtimeOnline &&
    catalogPredatesRuntime &&
    !queryState?.isInvalidated &&
    !lifecyclePrepared;

  useEffect(() => {
    if (!runtimeId) return;

    if (!runtimeOnline) {
      // Disabling an observer does not abort an already-running React Query.
      // Explicit cancellation propagates through the query AbortSignal into
      // the HTTP request, polling request and abortable delay.
      void queryClient.cancelQueries({ queryKey, exact: true });
      return;
    }

    if (catalogPredatesRuntime && !lifecyclePrepared) {
      forgetRuntimeCatalog(queryClient, runtimeId);
      // Re-read live state so two pickers observing the same runtime cannot
      // both prepare it. Cancellation and invalidation update synchronously;
      // refetchType:none keeps discovery disabled until local preparation
      // triggers the next render.
      if (!queryClient.getQueryState(queryKey)?.isInvalidated) {
        void queryClient.cancelQueries(
          { queryKey, exact: true },
          { revert: false },
        );
        void queryClient.invalidateQueries({
          queryKey,
          exact: true,
          refetchType: "none",
        });
      }
      setPreparedLifecycle({
        runtimeId,
        generation: runtimeListGeneration,
      });
    }
  }, [
    catalogPredatesRuntime,
    lifecyclePrepared,
    queryClient,
    queryKey,
    runtimeId,
    runtimeListGeneration,
    runtimeOnline,
  ]);

  return Boolean(runtimeId) && runtimeOnline && !waitingForInvalidation;
}

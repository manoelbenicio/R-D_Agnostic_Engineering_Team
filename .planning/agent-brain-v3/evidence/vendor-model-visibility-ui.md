# Vendor/Model Visibility UI Repair

- Recorded: 2026-07-18
- Scope: bounded runtime/model picker UI, model-catalog query options, and local catalog-cache/process cleanup hardening
- Status: PRODUCED — p9 R24/R25 remediation pending independent re-review
- State discipline: no GSD state or OpenSpec task checkbox was modified

## Root cause

Backend model rows already carried optional provider metadata through the typed model-catalog response. Visibility was lost or made inconsistent in the UI for four independent reasons:

1. Both model pickers converted an offline runtime ID to `null` before building query options. That changed the React Query key from the runtime-specific catalog key to the generic model key while also disabling the query, so a valid cached catalog for that runtime was no longer visible offline.
2. The create picker grouped a model with a missing `model.provider` under an empty string and omitted the group header. It did not fall back to the selected runtime's known provider.
3. The inspector picker rendered one flat list with no provider grouping, while the create picker grouped rows. The two user flows therefore exposed the same catalog differently.
4. Search matched only model ID and label, so searching by vendor/provider hid otherwise valid catalog rows.
5. Provider-less rows still had an unnamed-group edge case when both an explicit provider and the exact authoritative runtime-list cache were absent. A conservative built-in runtime identity map was needed; arbitrary custom IDs could not safely be guessed from substrings.
6. Normal React Query `gcTime` removed an inactive runtime catalog after ten minutes, so an offline remount could lose a previously observed catalog even though no fetch was allowed.

The existing runtime-list queries are the authoritative cached source for a selected runtime's provider. No additional runtime fetch or UI-owned server-state store was necessary.

## Files and behavior

### Implementation

- `multica-auth-work/packages/core/runtimes/models.ts`
  - `runtimeModelsOptions(runtimeId, enabled)` retains `runtimeModelsKeys.forRuntime(runtimeId)` while offline and sets only `enabled=false`.
  - The React Query `AbortSignal` is passed through initiation, abortable polling sleeps and every polling API request.
  - Provider grouping falls back to the runtime provider only when a model row has no non-empty provider. An explicit model provider always wins.
  - Model filtering matches ID, label and provider/group name.
  - Provider-group search results suppress the unrelated custom-model creation action.
  - Runtime provider fallback requires one exact authoritative workspace list/listMine query key and never scans the query cache.
  - Offline state cancels the exact runtime catalog query. Reconnect preparation cancels any obsolete in-flight work, invalidates with `refetchType: "none"`, and only then enables discovery, preventing a duplicate reconnect request.
  - The existing runtime-list query's `dataUpdatedAt` is the lifecycle generation. Comparing it with catalog `dataUpdatedAt` covers a picker mounted after the central reconnect invalidation/refetch has already completed.
  - Provider-less rows use the exact explicit/cache provider first, then a QueryClient-session remembered authoritative identity, then one of the 16 exact built-in runtime identities. Unknown custom IDs render as `Unknown/Custom`; names containing a built-in token are not guessed.
  - A non-fetching QueryClient-scoped session sentinel retains at most 32 recent catalogs and 256 rows per catalog (8,192 rows maximum). The ordinary per-runtime query still has a ten-minute `gcTime`; the sentinel survives that normal GC for offline remounts and is removed by `queryClient.clear()` on logout/session reset. Runtime lifecycle invalidation removes the retained catalog before an online refresh.
- `multica-auth-work/packages/core/api/client.ts`
  - The two public list-model methods accept an optional `AbortSignal` and pass it to the underlying HTTP request. No route, payload or response behavior changed.
- `multica-auth-work/packages/views/agents/components/model-dropdown.tsx`
  - Uses the stable offline query key and cached catalog.
  - Renders provider groups with fallback/explicit-provider precedence.
  - Includes provider names in search.
- `multica-auth-work/packages/views/agents/components/inspector/model-picker.tsx`
  - Uses the same stable offline catalog, fallback, grouping and provider-search behavior as the create picker.
  - Renders provider section headers consistently with the create flow.
- `multica-auth-work/packages/views/agents/components/runtime-picker.tsx`
  - The bounded prior correction exposes accessible provider/runtime identity for every supported runtime profile and gives `cline`/`nim` distinct identities; `qoder` remains built-in-only. This catalog round did not change its behavior further.

### Local catalog hardening

- `multica-auth-work/server/pkg/agent/models.go`
  - Dynamic discovery is single-flight per exact cache key and cancellation-safe.
  - Every executable-backed provider now keys positive, negative, stale and in-flight state by provider type plus a normalized executable identity. PATH names are resolved with `LookPath`; paths are made absolute/clean and existing symlinks are resolved. Cursor, Copilot, Hermes, Kimi, Kiro, Qoder, OpenCode, Pi, OpenClaw, Cline, Antigravity and CodeBuddy all use this key for custom executables.
  - Positive catalogs refresh after 60 seconds but retain a last-known fallback for at most ten minutes. A failed or blank refresh serves that last-known value without extending its hard deadline and applies a five-second retry backoff.
  - Empty catalogs without a last-known value are negative-cached for five seconds. Hard-expired entries are deleted eagerly; deterministic recency eviction caps the cache at 64 keys and each catalog at 2,048 rows.
  - ACP discovery sends only the defined `initialize` and `session/new` requests. Unix cleanup closes stdin as the supported EOF signal, waits 500 ms, then terminates the entire process group and reaps the parent. No session/destroy/shutdown RPC was added.
- `multica-auth-work/server/pkg/agent/thinking.go`
  - Thinking catalogs are capped at 64 keys and 256 model entries per key with eager ten-minute expiry deletion; executable-backed Claude, Codex and CodeBuddy entries use normalized executable identities.
  - CodeBuddy help discovery is keyed by provider plus normalized executable, cancellation-safe single-flight, and bounded by a 32-key cap, 256 KiB retained-output cap, 60-second positive TTL and five-second empty-result TTL.
- `multica-auth-work/server/pkg/agent/proc_other.go`, `proc_windows.go`
  - Unix discovery preserves isolated process-group containment and whole-group cleanup.
  - Windows discovery now fails closed before `cmd.Start`. The removed post-start Job attachment could not guarantee containment because a descendant could escape before assignment. No whole-tree containment is claimed on Windows until an atomic pre-execution Job launcher exists; there is no parent-only or `taskkill` fallback.

### Direct tests

- `multica-auth-work/packages/core/runtimes/models.test.tsx`
  - stable runtime-specific offline key with `enabled=false`;
  - multi-provider grouping;
  - missing-provider fallback;
  - explicit-provider precedence;
  - provider search;
  - provider-group custom-action suppression;
  - one signal through initiation and polling;
  - cancellation during polling sleep before another API request;
  - exact online-to-offline cancellation;
  - post-reconnect mount invalidation of an otherwise fresh catalog;
  - conflicting workspace/listMine cache precedence using the exact supplied key.
  - all 16 supported built-in provider-less runtime identities;
  - explicit/cache/remembered provider precedence and conservative `Unknown/Custom` behavior;
  - opaque authoritative identity retention only within one QueryClient session;
  - offline remount after more than normal `gcTime`, zero API calls, 256-row catalog cap and 32-catalog recency eviction;
  - newer lifecycle invalidation removes the retained offline catalog.
- `multica-auth-work/packages/views/agents/components/model-dropdown.test.tsx`
  - cached offline visibility with no discovery call;
  - multi-provider headers and runtime fallback;
  - explicit-provider precedence;
  - provider-name search without custom creation;
  - explicit runtime-provider precedence over cache;
  - exactly one discovery after a post-reconnect mount with a fresh stale catalog.
- `multica-auth-work/packages/views/agents/components/inspector/model-picker.test.tsx`
  - cached offline visibility with no discovery call;
  - matching inspector provider headers and fallback/precedence behavior.

The backend cache tests additionally cover positive soft/hard retention, blank/error stale fallback, hard expiry deletion, short negative caching, deterministic key/row caps, concurrent single-flight, and a cancelled leader that cannot poison a live waiter. Two validated synthetic executable paths prove that fresh catalogs, negative entries, stale catalogs and refresh errors cannot cross-contaminate; a direct Cursor two-executable test and a CodeBuddy help-cache test cover the previously provider-only cases. Synthetic Unix ACP scripts cover graceful stdin EOF, orphan-child cleanup and context-timeout cleanup. Windows-only tests assert deterministic refusal before process start and are cross-compiled into the Windows test binary; they were not executed on a Windows host.

## Deterministic regression review

The create and inspector flows were inspected from runtime selection through catalog subscription, offline display, search, grouping and reconnect.

The initial review found a broad `runtimes` cache scan. The independent p8 correction tightened this further: the resolver no longer scans even filtered runtime-list entries. Its caller supplies exactly one authoritative `runtimeKeys.list(workspaceId)` or `runtimeKeys.listMine(workspaceId)` key, and an explicit `runtimeProvider` remains higher precedence. A conflicting-cache test proves that the non-supplied key cannot win.

The correction review also found that observer disablement alone does not abort React Query work, and that a transition-only reconnect hook misses pickers mounted after reconnect. The corrected flow uses React Query cancellation plus the existing runtime-list `dataUpdatedAt` lifecycle generation. It prepares invalidation while discovery is disabled, then enables the query on the next render. A live `isInvalidated` check prevents two observers from preparing the same catalog twice.

No other deterministic UI regression attributable to this diff was found. Pre-existing consumers outside the bounded create/inspector picker scope were not edited.

## OpenCode/Pi offline-contract audit

The checked-in command contracts and direct tests define OpenCode discovery as `opencode models --verbose` with a fallback to `opencode models`, and Pi discovery as `pi --list-models`. The source explicitly notes that both commands may consult configured/hosted providers and applies a 15-second process timeout. No checked-in CLI/help contract documents a local-only, bundled-only or offline flag for either command, so none was invented.

When a runtime is offline, the picker keeps the exact runtime query key with `enabled=false`, cancels that exact in-flight query, and serves only bounded last-known data; it initiates no daemon discovery request. There is no backend offline-state parameter in the owned `agent.ListModels` API, so this evidence does not claim that an unrelated direct caller can communicate offline state to `models.go`. No central API/daemon entrypoint was changed to manufacture such a signal.

## Mocked verification

All catalog values were synthetic or reference-only. The view tests mock `@multica/core/api`, and the offline cases explicitly assert that model discovery is never invoked.

Passed:

```text
pnpm --filter @multica/core exec vitest run runtimes/models.test.tsx --pool=threads --maxWorkers=1
Test Files  1 passed (1)
Tests       14 passed (14)

pnpm --filter @multica/views exec vitest run agents/components/model-dropdown.test.tsx --pool=threads --maxWorkers=1
Test Files  1 passed (1)
Tests       6 passed (6)

pnpm --filter @multica/views exec vitest run agents/components/inspector/model-picker.test.tsx --pool=threads --maxWorkers=1
Test Files  1 passed (1)
Tests       3 passed (3)
```

Local Go verification used the existing Go 1.26 container with `--network none`:

```text
go test ./pkg/agent -count=1
ok github.com/multica-ai/multica/server/pkg/agent 6.683s

go test -race ./pkg/agent -count=1
ok github.com/multica-ai/multica/server/pkg/agent 10.257s

go vet ./pkg/agent
PASS (no output)

GOOS=windows GOARCH=amd64 go test ./pkg/agent -run '^$' -c
PASS (compile-only; no execution)
```

Before the correction round, related focused regression coverage also passed:

```text
pnpm --filter @multica/views exec vitest run agents/components/model-dropdown.test.tsx agents/components/inspector/model-picker.test.tsx agents/components/inspector/thinking-prop-row.test.tsx
Test Files  3 passed (3)
Tests       9 passed (9)
```

Typecheck passed:

```text
pnpm --filter @multica/core typecheck
pnpm --filter @multica/views typecheck
```

File-scoped lint and diff validation passed:

```text
pnpm --filter @multica/core exec eslint api/client.ts runtimes/models.ts runtimes/models.test.tsx
pnpm --filter @multica/views exec eslint agents/components/model-dropdown.tsx agents/components/model-dropdown.test.tsx agents/components/inspector/model-picker.tsx agents/components/inspector/model-picker.test.tsx
git diff --check -- <bounded implementation and test files>
```

Several combined-suite worker launches failed before loading any test file because the constrained host worker did not respond within Vitest's startup timeout. The same focused files then completed individually under the recorded single threads-worker commands. These startup failures are not counted as test executions or passes.

## p9 R24/R25 remediation verification

All values and executables used by this correction were synthetic and local. Status remains **PRODUCED — pending independent re-review**.

Passed from `multica-auth-work/server` with the pinned local Go toolchain:

```text
go test ./pkg/agent -run 'Test(ExecutableDiscoveryKeys|EveryExecutableBackedProvider|CachedDiscovery|ListModels(Antigravity|Cursor)CachesPerExecutable|CodebuddyHelpCacheIsolatedByExecutable|CodebuddyHelpOutput|ThinkingCache|DiscoverACPModels)' -count=1
ok github.com/multica-ai/multica/server/pkg/agent 1.233s

go test -race ./pkg/agent -run 'Test(ExecutableDiscoveryKeys|EveryExecutableBackedProvider|CachedDiscovery|ListModels(Antigravity|Cursor)CachesPerExecutable|CodebuddyHelpCacheIsolatedByExecutable|CodebuddyHelpOutput|ThinkingCache|DiscoverACPModels)' -count=1
ok github.com/multica-ai/multica/server/pkg/agent 2.281s

go test ./pkg/agent -count=1
ok github.com/multica-ai/multica/server/pkg/agent 8.911s

go test -race ./pkg/agent -count=1
ok github.com/multica-ai/multica/server/pkg/agent 10.343s

go vet ./pkg/agent
PASS (no output)

GOOS=windows GOARCH=amd64 go test ./pkg/agent -run 'TestWindows(ACPDiscoveryFailsClosedBeforeStart|DynamicModelDiscoveryFailsClosed)$' -c
PASS (compile-only; Windows tests were not executed on this Linux host)

gofmt -d <p9-owned Go files>
PASS (no output)
```

An earlier full-suite run hit the unrelated timing-sensitive `TestCodexExecuteLegacyFirstTurnMessageSatisfiesProgress` timeout. Both final full normal and race reruns passed, so no residual package-test failure is claimed for this snapshot.

## Explicit non-claims

- No vendor or provider login/logout was performed.
- No credential, token, cookie, auth file or secret was accessed, read, copied, printed, rewritten or mutated.
- No live provider, OmniRoute, daemon model-discovery endpoint or backend provider API was called.
- No live model catalog was requested or validated.
- No Windows whole-tree containment is claimed. Windows executable-backed model discovery is disabled before process start pending an atomic Job-assignment launcher.
- No OpenCode/Pi offline flag is claimed; no unsupported CLI flag or ACP cleanup RPC was added.
- No production, cutover, tier activation or native-adapter behavior is claimed.
- Passing mocked UI tests proves deterministic cache/grouping/search behavior only; it does not prove live provider catalog completeness or availability.
- No GSD state, OpenSpec task checkbox, central daemon/config/health entrypoint, gateway, deploy or observability file was changed.

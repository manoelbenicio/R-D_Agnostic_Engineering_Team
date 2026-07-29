# RES — Gateway readiness fetch path trace + single-flight/backoff/bounded-fresh-wait injection points

- agent `Opus48#B` · lane `RES-TRACE` · pane `w6:p2` · task `RES-READINESS-TRACE`
- check-in: `.deploy-control/p0/checkins/Opus48-B__RES-READINESS-TRACE__20260723T021402Z.json`
- posture: **READ-ONLY** (no edits). HEAD `a6d50986…`. Paths under `server/internal/daemon/`.

## Fetch path (admission → readiness → registry → HTTP)

`brain_integration.go admitTask`:
1. `:210` `registry, _ := gateway.NewRegistry(ModelsFetchFunc(client.FetchModels …), time.Second)`
   — **created per admitTask**, ttl=1s; the fetch closure applies the `OMNIROUTE_DEV_MODELS_COMPAT` projection.
2. `:232` `checker, _ := gateway.NewReadinessChecker(client, registry, cfg.Neutral.Gateway.Readiness, correlation)`.
3. `:~238` `admission, _ := brain.NewGatewayAdmissionController(checker, readinessPolicy)` →
   `decision, err := admission.Admit(ctx, task)` → on err `readiness_cancelled` (fail closed).
4. `checker.CheckGatewayReadiness` (`health_models.go`): `CheckLiveness` (unauth) → `CheckReadiness`
   (auth) → `registry.Snapshot(ctx)` → selected-model `Available` → selected-protocol match →
   `policy.Evaluate`.
5. `registry.ValidateCapability` + `LookupModelCapability` (protocol/streaming/tools gate).

`client.go` (`FetchModels`/probes/`do`): single `httpClient.Do` per op (no internal retry);
`classifyTransportError` (cancel→cancelled; timeout→`ErrorTimeout` retryable; else `ErrorTransport`
retryable); non-2xx → `classifyStatus`.

## Classification + Retry-After (readiness/models path = `errors.go`)

`errors.go classifyStatus` (used by `client.do`): `401→authentication`, `403→authorization`,
`408→timeout retryable`, **`429→rate_limited retryable, RetryAfter=parseRetryAfter(Retry-After)`**,
**`503→overloaded retryable, RetryAfter=parseRetryAfter`**, **`5xx→upstream retryable`**, other→protocol.
`parseRetryAfter` accepts integer seconds **or** HTTP-date, rejects ≤0 / >24h → `time.Duration` on
`GatewayError.RetryAfter`. `GatewayError` is body/credential/header-free.

> Note: `classification.go ClassifyFailure` (the richer account/model/provider 429 taxonomy + circuit
> scope) is the **executor/inference-path** decision, **not** the readiness/models fetch. The readiness
> path uses `classifyStatus`/`classifyTransportError` only. (Flagged so injection targets the right layer.)

## Existing single-flight + backoff (already in `registry.go Snapshot`, `:89`)

- **TTL positive cache:** `if snapshot.Version != "" && now.Before(expiresAt) → return clone` (`:93`).
- **Single-flight:** `r.refreshing` channel — one refresher; others `select{<-ctx.Done(); <-wait}` then
  retry (`:104-115`); `generation` fence via `Invalidate()` prevents a stale refresh from populating cache.
- **Negative-cache backoff:** on gateway/registry error (ctx not cancelled)
  `r.refreshErr = err; r.retryAt = failedAt.Add(RegistryRefreshFailureBackoff)` (`:~130`); callers within
  `retryAt` get the cached error immediately.

## KEY GAP — the Registry/ReadinessChecker are per-call ephemeral

`NewRegistry`/`NewReadinessChecker` are constructed **inside `admitTask`** (fresh each admission,
ttl=1s) and discarded after. Therefore the Registry's single-flight, 1s TTL, and negative-cache backoff
**do NOT span concurrent tasks**: N concurrent admissions build N registries → N `CheckLiveness` +
N `CheckReadiness` + N `FetchModels` (stampede); backoff/TTL never protect the next task.

## Injection points (exact; advisory — no edits)

1. **Single-flight across tasks — `brain_integration.go:210-236`.** Hoist `NewRegistry` +
   `NewReadinessChecker` (and the admission controller) into a **long-lived instance on
   `agentBrainRuntime`**, built once at runtime init (where `client`/config are wired), and reuse it in
   `admitTask`. This makes `registry.go`'s existing `r.refreshing` single-flight + TTL + negative cache
   actually dedupe concurrent admissions. (Minimal correctness change; no new mechanism needed — the
   single-flight already exists, it's just thrown away per call.)
2. **Backoff honoring Retry-After / exponential — `registry.go` failure branch (`r.retryAt =
   failedAt.Add(RegistryRefreshFailureBackoff)`).** Replace the constant with: honor
   `GatewayError.RetryAfter` when the classified `err` carries one (429/503 already set it via
   `errors.go parseRetryAfter`), else exponential backoff keyed on consecutive-failure count (add a
   `consecutiveFailures`/`lastBackoff` field). The RetryAfter value is already available on the returned
   `*GatewayError`; extract via `errors.As`.
3. **Bounded fresh-wait — two candidate sites:**
   - `registry.go Snapshot` in-flight wait (`select{<-ctx.Done(); <-wait}`, `:108-113`): add a bounded
     timer so a caller waits at most a fresh-wait budget for an in-flight refresh, then returns
     stale-usable (if a prior snapshot exists) or a deterministic timeout — instead of blocking to the
     full request ctx.
   - `brain_integration.go admission.Admit(ctx, …)` call: wrap `ctx` with
     `context.WithTimeout(ctx, readinessFreshWaitBudget)` so a single admission can't block a task on
     readiness beyond a bounded budget (probes are already bounded by `client.requestTimeout`).
   Prefer the registry-level bound (covers both probe-driven and fetch-driven waits) plus a shared
   long-lived registry from injection #1.

## Non-claims
Static read-only trace; no edit; no DB/network/live run; advisory recommendations only. Executor-path
`ClassifyFailure` circuit/retry semantics are out of scope (distinct from the readiness/models fetch).
Reported to Kiro via this evidence file.

# RES-design — /v1/models poll resilience + fresh-admission wait (design only)

- agent: `Codex56#A` · lane: `resilience-design` · task: `RES-DESIGN` · pane: `w7:p3`
- lock: `.deploy-control/p0/evidence/RES-design.md`
- MODE: **DESIGN ONLY** (read-only; no edits, no deploy). Hard invariants: never reuse stale; never weaken `StrictReadinessPolicy`.

> SUMMARY: Most primitives already exist. **Single-flight** poll = present (`Registry.refreshing`
> channel). **Retry-After parsing** = present (`errors.go` → `GatewayError.RetryAfter`). **Bounded
> exp-backoff-honoring-Retry-After** = present as `Coordinator.backoff` (`dispatch.go`) — the pattern to
> reuse. The real gaps: the registry's failure backoff is a **fixed 2s** const (no exp/jitter/Retry-
> After/class-awareness), there is **no configurable poll cadence policy**, and admission does a
> **single, no-wait** readiness check that can serve a **stale-within-TTL** snapshot. This spec adds a
> `RegistryPollPolicy`, class-aware jittered backoff on the registry, a stale-free `SnapshotFresh`, and a
> bounded `AwaitFreshReady` admission wait — all fail-closed and additive.

## 1. Current state (verified by reading)

| Concern | Current | File |
|---|---|---|
| Single-flight poll | **Present** — `refreshing chan struct{}` coalesces concurrent refreshes; waiters block; `generation` fences `Invalidate`. | `gateway/registry.go` `Snapshot` |
| Failure backoff | **Fixed 2s** const `RegistryRefreshFailureBackoff`; same for every error/attempt; ignores Retry-After. | `registry.go` (`retryAt = failedAt.Add(RegistryRefreshFailureBackoff)`) |
| Retry-After | **Parsed already** into `GatewayError.RetryAfter` on 429/503. | `gateway/errors.go:78,82` `parseRetryAfter`; test `gateway_test.go:117-126` |
| Bounded exp backoff + Retry-After | **Exists for request dispatch** (`Coordinator.backoff(attempt, err)`; honors `retryAfter(err)`, caps at `MaximumBackoff`). Jitter field exists in `RetryPolicy` but that backoff does not apply it. | `gateway/dispatch.go:313`, `policy.go:10 RetryPolicy` |
| Positive cache | TTL `expiresAt`; `Snapshot` returns cached if `now.Before(expiresAt)`. | `registry.go` |
| Admission | `GatewayAdmissionController.Admit` calls `CheckGatewayReadiness` **once**; on error → `unavailableDecision()` (fail-closed, **no wait**). Uses `registry.Snapshot` (may be TTL-stale). | `brain/admission.go:82`; `gateway/health_models.go:73` |
| Strict policy | `CheckGatewayReadiness` sets Live/Authenticated/ModelRegistryReady/SelectedModelReady/SelectedProtocolReady then `policy.Evaluate` (strict, fail-closed; constructor rejects non-strict). | `health_models.go`, `admission.go:53` |

## 2. RegistryPollPolicy (new struct + validation) — gateway package

Mirrors `RetryPolicy`/`CircuitPolicy` conventions (`policy.go`).
```go
type RegistryPollPolicy struct {
    Cadence          time.Duration // steady positive-cache TTL / refresh interval (== Registry ttl)
    MinimumBackoff   time.Duration // base failure backoff (retryable classes)
    MaximumBackoff   time.Duration // per-attempt cap
    Jitter           bool          // apply full jitter to each computed backoff
    HonorRetryAfter  bool          // use upstream Retry-After when present (429/503)
    MaxRetryAfter    time.Duration // cap on an honored Retry-After hint
    NonRetryBackoff  time.Duration // conservative backoff for non-retryable/protocol failures (still fail-closed)
    FreshnessWindow  time.Duration // admission: max snapshot age that counts as FRESH
    MaxAdmissionWait time.Duration // admission: bounded total wait for fresh-ready
    MaxAttempts      int           // 0 = unbounded within MaxAdmissionWait; else cap refresh attempts per admission
}

func DefaultRegistryPollPolicy(cadence time.Duration) RegistryPollPolicy {
    return RegistryPollPolicy{
        Cadence: cadence, MinimumBackoff: 500 * time.Millisecond, MaximumBackoff: 15 * time.Second,
        Jitter: true, HonorRetryAfter: true, MaxRetryAfter: 30 * time.Second,
        NonRetryBackoff: 15 * time.Second, FreshnessWindow: cadence, MaxAdmissionWait: 20 * time.Second,
        MaxAttempts: 0,
    }
}

func (p RegistryPollPolicy) Validate() error // bounds:
//  Cadence in [1s, 24h] (matches NewRegistry ttl bounds);
//  0 < MinimumBackoff <= MaximumBackoff <= 5m;
//  HonorRetryAfter ⇒ MaxRetryAfter >= MaximumBackoff and <= 5m;
//  NonRetryBackoff in [MinimumBackoff, 5m];
//  0 < FreshnessWindow <= Cadence;
//  0 < MaxAdmissionWait <= 60s;  MaxAttempts >= 0.
// Returns &GatewayError{Operation:"registry", Class: ErrorInvalidConfiguration} on violation.
```

## 3. Registry changes (`gateway/registry.go`) — single-flight preserved, class-aware jittered backoff

**3.1 struct fields (add):**
```go
type Registry struct {
    // ... existing ...
    policy              RegistryPollPolicy
    consecutiveFailures int             // guarded by mu
    jitter              func(time.Duration) time.Duration // injectable; nil ⇒ default full jitter
}
```
`ttl` becomes `policy.Cadence` (keep the field or read `policy.Cadence`). Keep `now func() time.Time`.

**3.2 constructors:**
```go
func NewRegistryWithPolicy(fetcher ModelsFetcher, policy RegistryPollPolicy) (*Registry, error) {
    if fetcher == nil { return nil, &GatewayError{Operation:"registry", Class: ErrorInvalidConfiguration} }
    if err := policy.Validate(); err != nil { return nil, err }
    return &Registry{fetcher: fetcher, policy: policy, now: time.Now}, nil
}
// Back-compat: existing NewRegistry(fetcher, ttl) delegates with DefaultRegistryPollPolicy(ttl).
func NewRegistry(fetcher ModelsFetcher, ttl time.Duration) (*Registry, error) {
    return NewRegistryWithPolicy(fetcher, DefaultRegistryPollPolicy(ttl))
}
```

**3.3 backoff computation (replace the fixed const at the failure branch of `Snapshot`):**
- OLD (in `Snapshot`, failure branch): `r.refreshErr = err; r.retryAt = failedAt.Add(RegistryRefreshFailureBackoff)`
- NEW intent:
  ```go
  r.refreshErr = err
  r.consecutiveFailures++
  r.retryAt = failedAt.Add(r.nextRefreshBackoff(r.consecutiveFailures, err))
  ```
- On success branch (existing): also `r.consecutiveFailures = 0`.
- New method:
  ```go
  func (r *Registry) nextRefreshBackoff(attempt int, err error) time.Duration {
      // 1) Honor upstream Retry-After first (429/503), capped.
      if r.policy.HonorRetryAfter {
          if hint := retryAfter(err); hint > 0 {                     // reuse dispatch.go retryAfter(err)
              if hint > r.policy.MaxRetryAfter { hint = r.policy.MaxRetryAfter }
              return r.applyJitter(hint)
          }
      }
      // 2) Non-retryable / protocol errors: conservative fixed backoff, still fail-closed.
      if !isRetryableRefreshErr(err) { return r.applyJitter(r.policy.NonRetryBackoff) }
      // 3) Retryable (429 w/o hint, 5xx, timeout): bounded exponential.
      delay := r.policy.MinimumBackoff
      for i := 1; i < attempt && delay < r.policy.MaximumBackoff; i++ { delay *= 2 }
      if delay > r.policy.MaximumBackoff { delay = r.policy.MaximumBackoff }
      return r.applyJitter(delay)
  }
  func (r *Registry) applyJitter(d time.Duration) time.Duration {
      if !r.policy.Jitter || d <= 0 { return d }
      if r.jitter != nil { return r.jitter(d) }
      // full jitter in [d/2, d] to avoid thundering herd while keeping progress
      return d/2 + time.Duration(rand.Int63n(int64(d/2)+1))
  }
  // isRetryableRefreshErr classifies via the existing GatewayError.Class:
  // ErrorRateLimited, ErrorUpstream(5xx), ErrorTimeout ⇒ true; ErrorProtocol/ErrorInvalid* ⇒ false.
  ```
- `retryAfter(err)` (dispatch.go:367) currently reads `*GatewayError`; ensure it unwraps with `errors.As` so a wrapped fetch error still yields the hint.

**3.4 Single-flight:** unchanged — the `refreshing chan struct{}` coalescing + `generation` fence already satisfy "single-flight/coalesced poll." Documented as such; no behavioral change.

**3.5 Stale-free fresh read for admission (new):**
```go
// SnapshotFresh returns a cached snapshot ONLY if it is younger than maxAge;
// otherwise it forces a coalesced (single-flight) refresh. It NEVER returns a
// snapshot older than maxAge. maxAge<=0 always refreshes.
func (r *Registry) SnapshotFresh(ctx context.Context, maxAge time.Duration) (RegistrySnapshot, error)
```
Implementation mirrors `Snapshot` but the positive-cache hit condition becomes
`r.snapshot.Version != "" && now.Sub(r.snapshot.FetchedAt) <= maxAge` (instead of `now.Before(expiresAt)`),
reusing the same `refreshing`/`generation`/negative-cache machinery. `Snapshot` (TTL) is unchanged for
non-admission callers.

## 4. Retry-After surfacing — NO client change required

`gateway/errors.go` already sets `GatewayError.RetryAfter = parseRetryAfter(resp.Header.Get("Retry-After"), now)`
on 429/503, verified by `gateway_test.go:117-126` (CheckReadiness 429 → `RetryAfter==3s`, `Retryable`).
`Client.FetchModels` (`client.go:170`) returns that `*GatewayError`. Requirement: the `ModelsFetcher`
closure wiring the registry to `FetchModels` must return the error **unwrapped enough** for
`errors.As(err, *GatewayError)` in `retryAfter`. Add an `errors.As` in `retryAfter` if any wrapper is introduced.

## 5. Bounded admission wait for FRESH ready — never stale, never weaken strict

**5.1 Gateway-side awaiter (new; `gateway/health_models.go`):**
```go
// AwaitFreshReady performs strict, fresh readiness evaluation and waits up to
// deadline for the gateway to become freshly ready. It NEVER returns a stale
// (older than policy.FreshnessWindow) registry, NEVER weakens StrictReadinessPolicy,
// and fails closed when the deadline elapses.
func (c *ReadinessChecker) AwaitFreshReady(ctx context.Context, req brain.ReadinessRequest, deadline time.Duration) (brain.ReadinessSnapshot, error)
```
Loop:
1. Compute a bounded sub-context: `waitCtx, cancel := context.WithTimeout(ctx, min(deadline, policy.MaxAdmissionWait))`.
2. Evaluate readiness with the **fresh** registry: same five-field strict sequence as
   `CheckGatewayReadiness`, but `ModelRegistryReady`/`SelectedModel*` are computed from
   `registry.SnapshotFresh(waitCtx, policy.FreshnessWindow)` (NOT `Snapshot`). Refactor
   `CheckGatewayReadiness` to a private `evaluate(ctx, req, snapFn)` shared by both; the public
   `CheckGatewayReadiness` keeps using `Snapshot` (back-compat), `AwaitFreshReady` uses `SnapshotFresh`.
3. Apply `c.policy.Evaluate(snapshot)` — **unchanged strict fail-closed policy**.
4. If ready → return snapshot,nil.
5. If not ready: if `waitCtx` expired → return the (not-ready) snapshot + fail-closed error. Else, if the
   underlying error is retryable (429/5xx/timeout) or readiness simply not-yet-ready, sleep
   `min(registry.nextRefreshBackoff(attempt,err), remaining(waitCtx))` via `sleepContext(waitCtx, …)`
   (reuse `dispatch.go sleepContext`), increment attempt (respect `policy.MaxAttempts` if >0), retry.
6. Never admit on a stale snapshot; never downgrade any of the five fields.

**5.2 Admission wiring (`brain/admission.go`) — additive, non-breaking:**
- Add optional interface (brain package):
  ```go
  type FreshReadinessAwaiter interface {
      AwaitFreshReady(context.Context, ReadinessRequest, time.Duration) (ReadinessSnapshot, error)
  }
  ```
- Add field `FreshWait time.Duration` to `GatewayAdmissionController`.
- In `Admit`, replace the single `a.Checker.CheckGatewayReadiness(ctx, req)` with:
  ```go
  var snapshot ReadinessSnapshot; var err error
  if aw, ok := a.Checker.(FreshReadinessAwaiter); ok && a.FreshWait > 0 {
      snapshot, err = aw.AwaitFreshReady(ctx, req, a.FreshWait) // bounded, fresh, strict
  } else {
      snapshot, err = a.Checker.CheckGatewayReadiness(ctx, req)  // unchanged fallback
  }
  ```
  The rest of `Admit` (Live/Authenticated/... → `unavailableDecision()` on any gap) is unchanged →
  strict fail-closed preserved. If the wait elapses without fresh-ready, `err`/`!snapshot.Live...` →
  `unavailableDecision()` (rejected), never admitted on stale.

## 6. Focused tests (author with the owning edit)

- `registry_test.go`: `nextRefreshBackoff` — 429 with Retry-After honored+capped; 429 w/o hint & 5xx & timeout → exp growth capped at `MaximumBackoff` with jitter in [d/2,d] (inject deterministic `jitter`); protocol error → `NonRetryBackoff`; `consecutiveFailures` resets on success. Single-flight: N concurrent `Snapshot` → exactly one `FetchModels` (existing coalescing test extended).
- `registry_test.go`: `SnapshotFresh` returns cached only within `maxAge`; forces refresh when older; never returns stale; coalesces.
- `health_models`/`registry_test.go`: `AwaitFreshReady` returns ready once fresh; fails closed at deadline; never uses stale snapshot; preserves all five strict fields (drop any one → not ready).
- `brain/admission` (`g2a_test.go`): controller uses `AwaitFreshReady` when checker implements it + `FreshWait>0`; falls back otherwise; deadline elapse → `unavailableDecision()`.
Run focused: `go test ./internal/daemon/gateway ./internal/daemon/brain -count=1` + `go vet` (/tmp caches).

## 7. Invariants preserved (self-check)
- **Single-flight**: existing `refreshing` channel retained; `SnapshotFresh`/`AwaitFreshReady` reuse it.
- **Retry-After honored**: via existing `GatewayError.RetryAfter` + `retryAfter(err)`, capped by `MaxRetryAfter`.
- **Bounded exp backoff + jitter**: `nextRefreshBackoff` (×2 to `MaximumBackoff`, full jitter) on 429/5xx/timeout.
- **Configurable cadence**: `RegistryPollPolicy.Cadence` + all backoff/wait knobs, validated bounds.
- **Bounded admission wait**: `MaxAdmissionWait`/`FreshWait` cap total wait; `sleepContext` is context-aware.
- **Never reuse stale**: admission uses `SnapshotFresh(FreshnessWindow)`; deadline elapse ⇒ fail closed, never admit stale.
- **Never weaken StrictReadinessPolicy**: `policy.Evaluate` and the five-field sequence unchanged; constructor still rejects non-strict/non-fail-closed; changes are additive (`AwaitFreshReady` shares the same `evaluate`).
- **Fail-closed on all non-retryable/protocol errors** (unchanged negative-cache semantics, conservative `NonRetryBackoff`).

## 8. Non-claims / limitations
- Design only — no source edited, no tests run, no deploy, no secret, no OmniRoute-internal change.
- `retryAfter(err)` may need an `errors.As` unwrap if the fetcher wraps the client error.
- Whether `ttl` is renamed to `policy.Cadence` or kept as a mirror is an implementation detail; behavior identical.
- Default policy values (§2) are proposed tier-20-safe starting points for the owner/Principal to ratify.

# RES — deterministic fake-fetch readiness/registry tests

- agent: **Opus48#C** · lane **tests** · task **RES-READINESS-TESTS** · pane `w8:p1`
- as-of (UTC): `2026-07-23T02:18Z` · HEAD `a6d50986aee7f1981a323b29c9d90f175b4b6f8b`
- lock: `multica-auth-work/server/internal/daemon/gateway/res_readiness_test.go` (NEW) + this evidence file.
- check-in: `.deploy-control/p0/checkins/Opus48-C__RES-READINESS-TESTS__20260723T021711Z.json`
- go: `/home/ec2-user/goroot/go/bin/go` (go1.26.1); GOCACHE/GOTMPDIR under `/tmp`.

## 0. Result

**DONE — 5/5 deterministic tests added and green.** New file `internal/daemon/gateway/res_readiness_test.go` (`package gateway`) drives the model-registry readiness poller with a **fake fetch** (`ModelsFetchFunc`) and an **injected clock** (`Registry.now`) — no wall-clock dependence for correctness, no network/OmniRoute. Reuses the existing `validModelsDocument()`/`boolPointer` helpers (no duplication).

## 1. Coverage → test → mechanism

| Lane requirement | Test | Mechanism asserted |
|---|---|---|
| transient 429/5xx/timeout then success → **FRESH ready admitted** | `TestRegistryTransientThenSuccessAdmitsFreshReady` (subtests: rate_limited/overloaded/upstream/timeout) | fake fetch returns a retryable `GatewayError` (class per subtest), then a valid doc after the negative-cache backoff; asserts (a) first poll fails-closed, (b) within-backoff poll returns the cached error **without re-fetching**, (c) after `RegistryRefreshFailureBackoff` a retry admits a FRESH snapshot (`Version=="fresh-v1"`, exactly 2 fetches) and `LookupModel` is `Available` |
| persistent failure → **fail closed** | `TestRegistryPersistentFailureFailsClosed` | fake fetch always errors; across 5 polls (advancing the clock past each backoff window) `Snapshot` and `LookupModel` always fail closed — never admit a usable snapshot |
| **single-flight** coalesces concurrent polls | `TestRegistrySingleFlightCoalescesConcurrentPolls` | fetch blocks until 16 concurrent pollers pile onto the in-flight refresh, then returns; asserts **exactly 1 fetch** and all 16 receive the same fresh snapshot |
| **Retry-After + backoff honored** | `TestGatewayRetryAfterAndBackoffHonored` | `parseRetryAfter` (delta-seconds, HTTP-date, reject empty/0/negative/non-numeric/out-of-range→0); `classifyStatus` marks 429→`ErrorRateLimited`+Retryable+RetryAfter, 503→`ErrorOverloaded`+Retryable+RetryAfter, 500→`ErrorUpstream`+Retryable, 408→`ErrorTimeout`+Retryable, 401→`ErrorAuthentication` terminal; `retryAfter(err)` extracts exactly the hint the pre-commit backoff honors; and the poller's bounded negative-cache backoff `RegistryRefreshFailureBackoff` is a small positive bound |
| **stale-ready never reused** | `TestRegistryStaleReadyNeverReused` | after `Invalidate()` (generation fence) the next poll **re-fetches** and returns v2 (never stale v1, +1 fetch); after **TTL expiry** the next poll re-fetches v3 (never the expired snapshot); within-TTL reads serve the positive cache only (no extra fetch) |

## 2. Evidence — exact commands + exit codes

Run from `multica-auth-work/server`, `GOROOT=/home/ec2-user/goroot/go`, `GOCACHE=/tmp/l5-gocache`, `GOTMPDIR=/tmp`:
```text
gofmt -l internal/daemon/gateway/res_readiness_test.go            -> (empty; clean) ; exit 0
go vet ./internal/daemon/gateway/                                 -> clean ; exit 0
go test ./internal/daemon/gateway/ -run '<the 5 RES tests>' -count=1 -v
    --- PASS: TestRegistryTransientThenSuccessAdmitsFreshReady (+4 subtests)
    --- PASS: TestRegistryPersistentFailureFailsClosed
    --- PASS: TestRegistrySingleFlightCoalescesConcurrentPolls (0.05s)
    --- PASS: TestGatewayRetryAfterAndBackoffHonored
    --- PASS: TestRegistryStaleReadyNeverReused
    -> ok 0.061s ; exit 0
go test ./internal/daemon/gateway/ -count=1  (full package regression) -> ok 0.254s ; exit 0
git diff --check -- .../res_readiness_test.go                     -> PASS ; exit 0
```

## 3. Scope / non-claims
- ONE new test file added (`res_readiness_test.go`, untracked); **no product code edited**; full gateway package regression green (no name collision / compile break).
- `-race` on the single-flight test is **NOT_AVAILABLE** (cgo requires `gcc`, absent in env): `CGO_ENABLED=1 go test -race` → `C compiler "gcc" not found` (build failed). The single-flight test passes under the normal race-free run; determinism does not depend on the race detector.
- The registry uses a fixed bounded negative-cache backoff (`RegistryRefreshFailureBackoff`), not Retry-After; the upstream **Retry-After** hint is parsed/classified/extracted by `parseRetryAfter`/`classifyStatus`/`retryAfter` (the value the pre-commit dispatch backoff honors). The `Coordinator.backoff` exponential/cap math is already covered by existing dispatch tests and is intentionally not duplicated here.
- No install; `go.mod`/`go.sum` unchanged; no OmniRoute internals probed (fake `ModelsFetchFunc`, no network); no live run/inference/secret.

## 4. Status
- STATUS: DONE. DELIVERED: 5 deterministic fake-fetch readiness/registry tests (all green) + full-package regression green. FILES: created `res_readiness_test.go` + this evidence. No product change.

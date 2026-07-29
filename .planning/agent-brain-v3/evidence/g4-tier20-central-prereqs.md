# G4 Tier-20 Central Technical Prerequisites

**Status: PRODUCED / AWAITING pB INDEPENDENT ACCEPTANCE**

**Classification: TECHNICAL PREREQUISITE EVIDENCE ONLY — NOT ACCEPTANCE, READINESS, TASK 9.2 AUTHORIZATION, OR TIER ACTIVATION**

Date: 2026-07-18

Owner scope: Codex1 central brain/daemon/config integration only

Frozen project state for this handoff: OpenSpec remains **46/85**; task **8.1 is OPEN**; task **9.1 is STOPPED**. This document does not alter any of those states.

## Inputs inspected

- `REVIEW-G3-02` is accepted for all three findings at `.planning/agent-brain-v3/evidence/g3-independent-security-rereview.md:3`–`:22`.
- The preparation checklist remains non-accepting at `.planning/agent-brain-v3/evidence/g4-tier20-acceptance-checklist.md:1`–`:3` and retains numeric-threshold and host-resource gates at `:30`, `:153`–`:155`, and `:195`–`:200`.
- The current capacity evidence remains `Partial` at `.planning/agent-brain-v3/evidence/g4-synthetic-capacity-phase1.md:3`, `:86`, and `:101`.
- The consolidated task-9.2 row remains `Partial / B-CAP-HOST` at `.planning/agent-brain-v3/evidence/g4-consolidated-matrix.md:137`; that matrix explicitly withholds task-9.2 authorization at `:221`.

## Gap determination

A concrete central gap existed. The owned path already had:

- fail-closed readiness tests at `multica-auth-work/server/internal/daemon/brain/g2a_test.go:63` and `multica-auth-work/server/internal/daemon/brain_integration_test.go:160`;
- slot-before-claim overload containment at `multica-auth-work/server/internal/daemon/runtime_isolation_test.go:164`;
- cancellation and terminal-result tests at `multica-auth-work/server/internal/daemon/brain/g2a_test.go:181` and `multica-auth-work/server/internal/daemon/daemon_test.go:2052`; and
- a hard one-task development cap in central configuration.

However, no single central lifecycle ledger reconciled admission, overload, start, terminal cancellation/failure, and exactly-once capacity release. The tier-20 schema-to-one-task fail-closed invariant also lacked a direct central regression test. Codex4's deterministic harness reconciles its modeled workload, but it is not the active central daemon lifecycle counter.

The subsequent pB Medium review identified a narrower observable-lifecycle gap: the lease serialized duplicate `Start`/`Finish` transitions, but the central callers ignored the returned winning boolean. A duplicate start could therefore emit another launch diagnostic, and duplicate or contradictory terminal calls could emit cancellation/terminal diagnostics before or after losing the terminal transition. Counter reconciliation alone did not prove diagnostic reconciliation.

## Minimal central correction

1. Added a neutral, content-free lifecycle capacity ledger at `multica-auth-work/server/internal/daemon/brain/capacity.go:9` onward.
   - `TryBegin` at `:151` reserves bounded capacity without queueing.
   - Overload is `TaskStatusOverloaded`, retryable, and machine-classified as `local_capacity_overloaded`; the state is frozen at `brain/admission.go:17`.
   - Full point-in-time reconciliation at `capacity.go:91` checks offered/admitted/rejected/pending admission, admitted/start/pre-start terminal/pending start, started/terminal/active, failure, cancellation, in-use, and acquisition/release equations; the content-free I1 reconciliation is at `:62`.
   - `Start` and `Finish` at `capacity.go:253` and `:271` enforce exactly-once transition and release.
2. Wired the ledger only into the existing default-off development runtime at `multica-auth-work/server/internal/daemon/brain_integration.go:157`, `:252`, `:422`, `:440`, and `:500`.
   - Capacity is reserved before readiness or the synthetic credential callback.
   - Rejected readiness/capability attempts release their reservation.
   - A successful admission owns one lease through its terminal result.
3. Moved the central launch count to the final execution boundary, after fail-closed environment/config/model validation, at `multica-auth-work/server/internal/daemon/daemon.go:3805`.
4. Preserved fail-closed tier gating through `effectiveTaskAdmissionLimit` at `multica-auth-work/server/internal/daemon/config.go:566` and `:724`.
   - `CapacityTier=20` remains schema metadata only.
   - Gateway-required development execution remains capped at `agentBrainDevelopmentMaxTasks=1`.
   - No acceptance flag, threshold, broad-admission flag, or tier-enablement path was added.
5. Exposed only content-free counter values in neutral local health diagnostics at `multica-auth-work/server/internal/daemon/health.go:56` and `:169`.
6. Corrected the central observable lifecycle at `multica-auth-work/server/internal/daemon/brain_integration.go:422`–`:470`.
   - `recordLaunch` returns immediately when `CapacityLease.Start()` loses; route-selection and launch diagnostics are emitted only by the winning transition.
   - `recordTerminal` classifies the candidate outcome, commits `CapacityLease.Finish()` first, and returns immediately if it loses. Cancellation and terminal diagnostics are emitted only after the winning terminal commit.
   - This does not add a new event type or counter. It narrows the guarantee to one internally committed lifecycle winner and prevents duplicate/racing result, error, and cancellation callers from producing contradictory central diagnostics.

## Synthetic regression coverage

- `multica-auth-work/server/internal/daemon/brain/capacity_test.go:5` proves one admitted/start/completion, one retryable overload, a post-release admission, a pre-start cancellation, duplicate-terminal suppression, zero residual capacity, peak bound 1, and full counter reconciliation.
- `multica-auth-work/server/internal/daemon/brain/capacity_test.go:52` proves a rejected admission releases a reservation exactly once and reconciles.
- `multica-auth-work/server/internal/daemon/brain_integration_test.go:177` proves the active central development runtime rejects overload before another credential callback, then reconciles admitted/completed/cancelled counters and releases capacity for reuse.
- `multica-auth-work/server/internal/daemon/brain_integration_test.go:231` races 64 duplicate starts and 64 duplicate cancellation finishes, proving exactly one route-selection event, one launch log, one cancellation event, one terminal log, one start/cancellation counter transition, and one capacity release.
- `multica-auth-work/server/internal/daemon/brain_integration_test.go:271` races completed, failed, and cancelled terminal candidates and proves the single winning terminal log, optional cancellation event, terminal counter, and release agree; losing candidates emit no contradictory event or log.
- `multica-auth-work/server/internal/daemon/brain_integration_test.go:313` proves tier-20 schema selection still yields an effective central admission limit of one; disabling the development slice leaves the legacy limit unchanged.
- Existing one-router/readiness/legacy-isolation checks remain anchored at `brain_integration_test.go:143`, `:160`, `:330`, and `:344`.

## I1/I2 corrections — produced, awaiting pB

### I1 `LedgerCounters`

Frozen implementation anchors:

- shape: `multica-auth-work/server/internal/daemon/brain/capacity.go:39`;
- quiescent reconciliation: `capacity.go:62`;
- invalid and valid pre-start regression tests: `capacity_test.go:211` and `:225`.

Exact content-free JSON shape, all `uint64`:

```text
offered
admitted
rejected
overloaded
started
completed
failed
failed_injected
cancelled
```

The frozen equations and guards are:

```text
offered = admitted + rejected + overloaded
pre_start_terminal = admitted - started
completed <= started
pre_start_terminal <= failed + cancelled
admitted = completed + failed + cancelled
failed_injected <= failed
```

The pB correction rejects the previously possible `Started=0, Completed=1` snapshot. Valid terminal-before-start failure and cancellation snapshots remain accepted. `failed` is the total failure count; a consumer derives non-injected/real failures as `failed - failed_injected` only after reconciliation succeeds.

### I2 `InjectedFailureMarker`

Frozen implementation anchors:

- boolean marker: `multica-auth-work/server/internal/daemon/brain/contracts.go:94`;
- strict JSON decoder: `contracts.go:104`;
- JSON/classification regression tests: `capacity_test.go:136`.

Exact content-free shape: one boolean carried by `TaskResult.injected_failure`; there is no reason, identifier, message, payload, model, route, account, or caller-content field. Omission remains zero-value/false-compatible, explicit `false` remains non-injected, and explicit `true` marks an injected synthetic failure. Explicit `null`, string, number, object, and array values return a decoding error and reset the receiver to the non-injected value. A true marker validates only for terminal failure statuses and increments `failed_injected` within the total `failed` count.

## Final I3–I5 freeze — produced, awaiting pB

All three contracts are defined only in `internal/daemon/brain`; gateway producers and observability consumers import `brain`, never each other. The `brain` package imports only the Go standard library and does not import `gateway` or `observability`.

### I3 `GatewaySyntheticTelemetry`

Frozen anchors:

- shape: `multica-auth-work/server/internal/daemon/brain/measurability.go:13`;
- validation: `measurability.go:23`;
- deterministic validation/shape tests: `measurability_test.go:10`.

Exact content-free JSON shape, all `uint64`:

```text
retry_count
fallback_same_model
fallback_cross_model
```

`Validate(maxCount uint64, crossModelFallbackApproved bool)` checks every counter against the synthetic-run bound supplied by the caller. The `crossModelFallbackApproved` input is a method argument only; it is not serialized or retained in telemetry. A nonzero `fallback_cross_model` fails validation when that input is false. The method does not replace the separate p8 fallback-cycle/ordered-policy proof: missing approval or cycle validation remains a STOP before pA integration.

No request, task, session, model, route, provider, account, correlation, error text, timestamp, prompt, response, tool, or repository field exists in I3.

### I4 `QueueDepthAccessor`

Frozen anchors:

- `QueueDepthSample`: `multica-auth-work/server/internal/daemon/brain/measurability.go:43`;
- `QueueDepthAccessor`: `measurability.go:60`;
- deterministic accessor/bound/shape tests: `measurability_test.go:66`.

Exact content-free JSON shape, both `uint64`:

```text
depth
bound
```

`QueueDepth() (QueueDepthSample, error)` is read-only. Validation requires `bound > 0` and `depth <= bound`. The sample is instantaneous and deliberately carries no timestamp, phase, cadence, queue item, identifier, or content. The pA consumer owns the monotonic observation timestamp, sampling cadence, phase association, peak/growth calculations, and missing-sample STOP behavior. p8 owns only the bounded numeric producer result when a real gateway queue exists.

### I5 `SteadyStatePredicate`

Frozen anchors:

- `SteadyStateFacts`: `multica-auth-work/server/internal/daemon/brain/measurability.go:67`;
- fact validation: `measurability.go:74`;
- `SteadyStatePredicate`: `measurability.go:84`;
- deterministic predicate/fact/shape tests: `measurability_test.go:104`.

Exact content-free JSON shape, all booleans:

```text
ready
circuits_closed
accounts_reentered
steady
```

`SteadyState() (SteadyStateFacts, error)` reports neutral facts without gateway or observability imports. Validation enforces exactly:

```text
steady = ready AND circuits_closed AND accounts_reentered
```

p8 owns production of the facts. pA owns the bounded recovery clock, deadline, observation cadence, and STOP decision when steady state is absent or late. No timing, identity, account detail, circuit detail, route/model, content, or acceptance threshold is carried by I5.

## Verification executed

Executed on 2026-07-18 with the local `golang:1.26` image, networking disabled, product source mounted read-only, and pre-existing module/build cache volumes. No daemon dispatch, credential source, auth file, provider account, live provider, or OmniRoute service was used.

Focused I1–I5 contract repetition:

```text
docker run --rm --network none -v "$PWD":/src:ro -v agent-brain-g4-gomod:/go/pkg/mod -v agent-brain-g4-gobuild:/root/.cache/go-build -w /src golang:1.26 go test ./internal/daemon/brain -run 'TestInjectedFailureMarkerRejectsCallerContentAndNonFailureStatus|TestLedgerCountersRejectsUnreconciledSnapshots|TestLedgerCountersAcceptsPreStartFailureAndCancellation|TestGatewaySyntheticTelemetryValidationAndFrozenJSONShape|TestQueueDepthAccessorBoundedSampleContract|TestSteadyStatePredicateExplicitFactsContract' -count=20
PASS: internal/daemon/brain, 20 repetitions
```

Focused I1–I5 race repetition, including exactly-once classified finish/release:

```text
docker run --rm --network none -v "$PWD":/src:ro -v agent-brain-g4-gomod:/go/pkg/mod -v agent-brain-g4-gobuild:/root/.cache/go-build -w /src golang:1.26 go test -race ./internal/daemon/brain -run 'TestInjectedFailureMarkerRejectsCallerContentAndNonFailureStatus|TestLedgerCountersRejectsUnreconciledSnapshots|TestLedgerCountersAcceptsPreStartFailureAndCancellation|TestLifecycleCapacityConcurrentClassifiedFinishReleasesOnce|TestGatewaySyntheticTelemetryValidationAndFrozenJSONShape|TestQueueDepthAccessorBoundedSampleContract|TestSteadyStatePredicateExplicitFactsContract' -count=10
PASS: internal/daemon/brain, race detector, 10 repetitions
```

Full applicable daemon tests and vet were invoked as:

```text
docker run --rm --network none -v "$PWD":/src:ro -v agent-brain-g4-gomod:/go/pkg/mod -v agent-brain-g4-gobuild:/root/.cache/go-build -w /src golang:1.26 go test ./internal/daemon/... -count=1

docker run --rm --network none -v "$PWD":/src:ro -v agent-brain-g4-gomod:/go/pkg/mod -v agent-brain-g4-gobuild:/root/.cache/go-build -w /src golang:1.26 go vet ./internal/daemon/...
```

Current results:

- `go vet ./internal/daemon/...`: **PASS**.
- First concurrent full-test invocation: **FAIL**, with the unrelated timing-sensitive `TestWatchTaskCancellation_RunningTaskNotInterrupted` reporting two polls where at least five were expected in 150 ms.
- Contained follow-up `go test ./internal/daemon -run '^TestWatchTaskCancellation_RunningTaskNotInterrupted$' -count=20`: **PASS**, 20 repetitions.
- Serial full-test rerun: `internal/daemon`, `brain`, `deploy`, `execenv`, `gateway`, `repocache`, and `runtimeenv` passed; the overall command **FAILS CURRENTLY** because Codex4-owned `internal/daemon/observability/aggregate_test.go:58` and `:95` still provide the old primitive ledger callback while the current `ReconciliationHooks.Ledger` expects `func() (brain.LedgerCounters, error)`.

The observability compile mismatch was not modified because it is outside Codex1 ownership. Therefore this document does **not** claim a green full daemon suite, readiness, or acceptance. The focused I1–I5 contract/race results and current vet pass are produced evidence awaiting pB review.

## Current contract file hashes

Captured after the validation above with:

```text
sha256sum internal/daemon/brain/capacity.go internal/daemon/brain/capacity_test.go internal/daemon/brain/contracts.go internal/daemon/brain/measurability.go internal/daemon/brain/measurability_test.go
```

```text
4a1077455c0f121bdfb710ac3fcc7cb6e7f51868c008741ac3da62de7c23b4f2  internal/daemon/brain/capacity.go
10cf143cbbcc9ea9ed3106672b5a4ef7ddca9c4f3f065decbfd64bbcf15dc576  internal/daemon/brain/capacity_test.go
c61f298083ad8c9cad228681b88a56809692a95a9a3a473b87fce543b534d5fd  internal/daemon/brain/contracts.go
370d37351108c857738843d01400cd47b497d97c061af0c9e10aa242a7d9ff0f  internal/daemon/brain/measurability.go
7034173b5783175948882ec17115fb807789cd5faa01b4de1ea4ca604a811708  internal/daemon/brain/measurability_test.go
```

The evidence document itself is intentionally excluded from its own hash table because embedding its digest would be self-referential.

## Produced guarantees awaiting independent acceptance

- Local central admission is bounded and non-queuing for the development slice.
- A full local slot produces a deterministic retryable overload before readiness/credential/process work.
- Offered/admitted/rejected/overloaded and started/completed/failed/cancelled counters have explicit reconciliation equations.
- Pre-start and post-start cancellation are distinguished; capacity is released once even if terminal recording is repeated.
- A launch diagnostic is emitted only after the lease wins `Start`. A cancellation/terminal diagnostic is emitted only after the lease wins `Finish`; duplicate and contradictory racing terminal candidates are silent losers.
- One router, readiness failure, legacy startup suppression, and legacy Go-rotation suppression remain covered by existing central tests.
- Tier 20 is not enabled. The effective gateway-required development admission limit remains one.

## Remaining external blockers and stop conditions

This correction closes only the central code/test prerequisite. Task 9.2 remains stopped until all external acceptance inputs are independently complete:

- OpenSpec task 8.1 remains OPEN; this document supplies no live-auth or model-route acceptance.
- OpenSpec task 9.1 remains STOPPED; `EV-G4-CAP` must eventually be produced and independently accepted before task 9.2 can be considered.
- The current full daemon test command must return green; the external observability test callback mismatch recorded above is a present STOP.
- A capacity/SLO owner must approve numeric thresholds; none were invented here.
- Host-sampled latency, CPU, memory, sockets, queue, recovery, cancellation deadline, and rollback evidence must replace or supplement modeled values.
- The responsible external queue must prove its approved bound and overload/retry semantics. This central non-queuing gate does not claim to measure or accept the server-side queue.
- Codex4's evidence/provenance/consolidation artifacts must reconcile against the exact tested build and accepted thresholds.
- Any counter/diagnostic mismatch, leaked capacity, duplicate or contradictory terminal diagnostic, launch diagnostic from a losing start, launch on failed readiness, second router, direct-provider fallback, real credential requirement, live-provider dependency, or request to activate a tier is an immediate STOP.

This artifact is **PRODUCED / AWAITING pB INDEPENDENT ACCEPTANCE**. No OpenSpec task, `EVIDENCE_INDEX`, `STATE.md`, `AGENT_LEDGER.md`, product-code file, observability/deploy file, credential/auth material, tier setting, or product route was changed by this documentation handoff.

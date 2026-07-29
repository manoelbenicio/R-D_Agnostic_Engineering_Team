# G4 Gateway Protocol and Failure Validation

- Owner: Codex 2
- Scope: gateway portions of OpenSpec 8.1, 8.4, 8.5, 8.6, and 8.7; deterministic hardening; gateway I3-I5 neutral producers
- Latest evidence-only coverage checkpoint: 2026-07-18T13:09:54Z
- Repository revision: `b6571299b00c8e388abefe7ef9dcbcf8ac715d7f`
- Toolchain: `go1.26.5 linux/amd64`
- Classification: redacted, synthetic, offline

## Result

Test result: PASS. I3-I5 evidence disposition: **PRODUCED, not independently accepted**. Sixteen G4 tests inside `internal/daemon/gateway/**` cover the requested gateway protocol families, concurrent routing and continuation behavior, deterministic failure handling, replay boundaries, cancellation cleanup, circuit transitions, and synthetic account lifecycle. Six property-style tests use explicit fixed seeds, bounded iterations, and bounded goroutine counts so failures are reproducible and race-detector friendly. The complete current gateway package has 59 top-level tests, including deterministic registry invalidation/size/depth hardening and five bounded I3-I5 producer tests.

The hardening audit found and fixed one gateway-package defect in `protocol.go`: an empty SSE sequence for a known protocol was incorrectly classified as an unsupported capability, and a terminal event could appear before the final position without rejection. The validator now returns a protocol error for known empty streams and rejects early/duplicate terminal placement while preserving capability errors for unknown protocols. The audit also strengthened the synthetic retry harness so simultaneous duplicate request IDs single-flight instead of independently executing.

No live provider, live OmniRoute endpoint, daemon dispatch, credential store, secret file, or production account was accessed. Shared OpenSpec items remain open because Codex 2 owns only their gateway portions; Codex 4 owns cross-owner consolidation and `EVIDENCE_INDEX.md`.

## Deterministic stress profile

| Property | Fixed seed | Bounded workload |
| --- | ---: | --- |
| Strict independent-request round-robin | `0x440401` | 24 iterations; 2–8 accounts and 96–160 concurrent requests per iteration |
| Continuation affinity | `0x440402` | 48 keys × 32 shuffled continuations; 128 simultaneous first requests; ineligible-owner rebind for all three affinity kinds |
| Cancellation capacity release | `0x440603` | 384 acquired requests cancelled in a fixed shuffled order |
| Retry pre-commit boundary and dedup | `0x440604` | 512 concurrent randomized bounded attempt plans; 128 simultaneous duplicates behind one in-flight leader |
| Circuit transitions | `0x440505` | 64 synthetic-clock timelines; thresholds 2–6; 1–8 bounded half-open probes plus 32 rejected contenders |
| Account lifecycle | `0x440706` | 128 fixed operations; 32 concurrent selections after each add/remove/quarantine/re-entry/restart/rollback transition |

## Evidence map

### EV-G4-01 — Protocol conformance

- `TestG4ProtocolFamiliesNonStreamingAndStreaming` validates synthetic non-streaming and SSE shapes for Anthropic Messages, OpenAI Responses, OpenAI Chat Completions, and the documented Antigravity-compatible contract family through an in-memory mock HTTP transport.
- The Anthropic Messages table includes exact route `agy/claude-opus-4-6-thinking` with `CLIKind=claude-code`; both request modes assert exact model propagation, the trusted `omniroute-anthropic-messages` profile and `/v1/messages` endpoint, then validate the synthetic JSON response shape and SSE sequence.
- `TestG4InitialModelSetRouteConformanceDriftGuard` reads the frozen public `brain.InitialModelSet()` contract directly and fails if any `(RouteModel, CLIKind, ProtocolFamily)` tuple lacks a route-conformance row. This existing one-way gateway-test-to-brain dependency does not create an import cycle.
- `TestG4AuthenticatedSyntheticModelsAndCapabilities` verifies separated liveness/readiness behavior and authenticated synthetic `/v1/models` capability loading without reading a secret file.
- `TestG4SSEContractRejectsEmptyAndEarlyTerminal` is the regression test for the discovered SSE classification/order defect.
- Antigravity is validated only against its evidence-gated compatible test contract; no native or live wire acceptance is claimed.

### EV-G4-04 — Rotation and continuation affinity

- The original 96-request concurrent round-robin test and three affinity-family cases remain.
- `TestG4PropertyStrictRoundRobinFixedSeed` proves an uninterrupted atomic sequence across randomized, concurrent account/request populations.
- `TestG4PropertyContinuationAffinityFixedSeed` proves concurrent affinity stability, first-request single ownership, no cursor advance from continuations, owner-ineligibility rebind, and protection from stale-owner reclamation for `previous_response_id`, prompt-cache, and tool-turn keys.

### EV-G4-05 — Failure injection and circuit scope

- Existing synthetic failure classification covers expired access, revoked refresh, forbidden access, quota exhaustion, account-scoped 429, provider-global 429, 5xx, timeout, and malformed upstream behavior through pure decisions and mock transports.
- `TestG4PropertyCircuitTransitionsFixedSeed` uses a synthetic clock to prove closed → open → half-open → closed and half-open → open behavior, exact failure thresholds, open-duration boundaries, and atomic half-open probe limits under concurrency.
- Circuit behavior remains a synthetic supplier-contract model; it does not claim execution against a deployed OmniRoute circuit implementation.

### EV-G4-06 — Replay safety, deduplication, and cancellation

- Existing replay tests continue to prove pre-output retry and no replay after output or a committed non-idempotent tool action.
- `TestG4PropertyRetryPreCommitBoundaryFixedSeed` checks 512 fixed randomized attempt plans concurrently and proves 128 simultaneous duplicates join one in-flight execution.
- `TestG4PropertyCancellationReleasesCapacityFixedSeed` proves all 384 acquired slots return exactly to zero after fixed-order concurrent cancellation.

### EV-G4-07 — Synthetic lifecycle and rollback

- The original continuously loaded lifecycle/restart/rollback test remains.
- `TestG4PropertyAccountLifecycleFixedSeed` applies 128 fixed add/remove/quarantine/re-entry/stop/restart/rollback operations. Each transition is followed by a concurrent batch checked against an independent eligibility and round-robin reference model; stopped and empty pools fail closed.

### Gateway robustness corrections — synthetic scope

- `TestRegistryRefreshFailureBackoffIsConcurrentAndClockBounded` proves 64 concurrent and repeated `Snapshot` callers share one failed refresh, receive the preserved failure during the named two-second negative-cache window, and perform one recovery refresh exactly at the deterministic clock boundary.
- `TestRegistryInvalidateFencesInFlightSuccess` and `TestRegistryInvalidateFencesInFlightFailureWithoutPoisoningNegativeCache` prove a generation fence prevents any older in-flight completion from committing after invalidation, then deterministically refreshes the current generation. `TestRegistryInvalidatePreservesCancellationWithoutPoisoningCache` proves a cancelled stale caller preserves cancellation classification without suppressing the next healthy refresh.
- `TestRegistryRejectsEveryFallbackCycleIndependentOfDocumentOrder` covers self, two-node, and deep cycles in forward and reversed document order. `TestRegistryAcceptsFallbackDAGIndependentOfDocumentOrder` proves a branching/merging DAG remains valid in both orders.
- `TestRegistryBoundsDocumentSizeAndFallbackDepth` rejects more than 1,024 models and more than 32 fallback edges, including an oversized linear chain, while accepting both exact boundaries. The limits are named gateway policy constants; validation occurs before unbounded graph work.
- `TestCircuitPolicyDurationBounds` accepts positive and exact documented maxima, then rejects one-nanosecond-over and maximum-duration timer values. The policy maxima are 15 minutes for observation and one hour for open duration, matching bounded development recovery and the documented approximate one-hour quota cooldown.

### I3-I5 gateway producers — PRODUCED, not accepted

- `TestGatewaySyntheticTelemetryProducesDeterministicCounters` maps one terminal sanitized gateway record to exact retry, same-model fallback, and cross-model fallback counters, then calls the frozen `brain.GatewaySyntheticTelemetry.Validate(bound, crossModelFallbackApproved)` contract.
- `TestGatewaySyntheticTelemetryFailsClosedOnBoundApprovalAndCycleProof` rejects a counter above the declared bound, an unapproved cross-model fallback, a missing cycle proof, an unsanitized negative retry count, and an unreported model change. A `FallbackCycleProof` is minted only after the existing deterministic bounded registry graph validator succeeds and is copied with immutable snapshots; failed or zero snapshots cannot expose I3 values.
- `TestQueueDepthFuncValidatesBoundsAndReadsExactlyOnce` proves I4 emits only `Depth` plus `Bound`, invokes an existing sampler once, rejects a zero bound or depth over bound, and preserves cancellation without a partial sample. It does not introduce a gateway queue, timestamp, or cadence.
- `TestSteadyStateProducerUsesFullConjunction` exhaustively checks all eight readiness/circuit/account combinations and sets `Steady` if and only if all three facts are true. `TestSteadyStateProducerPreservesCancellationAndSamplesExactlyOnce` proves cancellation propagation, zero output on error, nil-source fail-closed behavior, and exactly one atomic source read for each of 64 concurrent calls.
- Dependency direction remains `gateway -> brain`; gateway does not import `observability`, and no duplicate neutral contract types were introduced.

## Verification

The task-8.8 provenance checkpoint used the complete current gateway source and
test set pinned in `g4-provenance-manifest.md`. In the cached immutable
`golang:1.26` image at digest
`sha256:ae5a2316d12f3e78fd99177dad452e6ad4f240af2d71d57b480c3477f250fec6`,
with Go 1.26.5 linux/amd64, `--network none`, `--pull=never`, read-only
source/root, and ephemeral build caches, this exact in-container command passed:

```text
/usr/local/go/bin/go test ./internal/daemon/gateway -count=1 -coverprofile=/build/gateway.cover
/usr/local/go/bin/go tool cover -func=/build/gateway.cover | tail -n 1
PASS; statement coverage 82.5%
```

Current deterministic counts are 16 G4 tests, 6 property-style tests, and 59
top-level tests. The transcript below records earlier successful hardening
checkpoints and is retained as historical evidence only; its timing and
coverage figures are superseded by the pinned task-8.8 checkpoint above.

```text
go test ./internal/daemon/gateway -run '^TestG4Property' -count=20
PASS (0.518s)

go test -race ./internal/daemon/gateway -run '^TestG4(Property|SSEContract)' -count=3
PASS (1.449s)

go test ./internal/daemon/gateway -count=20
PASS (1.701s)

go test -race ./internal/daemon/gateway -count=1
PASS (1.247s)

go vet ./internal/daemon/gateway
PASS

go test ./internal/daemon/gateway -run '^(TestRegistryRefreshFailureBackoffIsConcurrentAndClockBounded|TestRegistryRejectsEveryFallbackCycleIndependentOfDocumentOrder|TestRegistryAcceptsFallbackDAGIndependentOfDocumentOrder|TestCircuitPolicyDurationBounds)$' -count=20
PASS (0.029s)

go test ./internal/daemon/gateway -count=1
PASS (0.141s)

go test -race ./internal/daemon/gateway -count=1
PASS (1.318s)

go vet ./internal/daemon/gateway
PASS

go test ./internal/daemon/gateway -count=1 -cover
HISTORICAL/SUPERSEDED checkpoint: PASS (0.118s); statement coverage 79.1%

go test ./internal/daemon/gateway -run '^(TestG4ProtocolFamiliesNonStreamingAndStreaming|TestG4InitialModelSetRouteConformanceDriftGuard|TestG4AuthenticatedSyntheticModelsAndCapabilities)$' -count=1
PASS (0.010s)

go test ./internal/daemon/gateway -count=1
PASS (0.105s)

go test -race ./internal/daemon/gateway -count=1
PASS (1.259s)

go vet ./internal/daemon/gateway
PASS

go test ./internal/daemon/gateway -run TestGatewaySyntheticTelemetry -count=20
PASS (0.017s)

go test ./internal/daemon/gateway -run TestQueueDepth -count=20
PASS (0.012s)

go test ./internal/daemon/gateway -run TestSteadyState -count=20
PASS (0.039s)

go test ./internal/daemon/gateway -count=20
PASS (2.250s)

go test -race ./internal/daemon/gateway -count=1
PASS (1.353s)

go vet ./internal/daemon/gateway
PASS
```

Historical subset checksums (the complete current 25-file gateway set and its
canonical set digest are pinned in `g4-provenance-manifest.md`):

```text
4ee0bb641765e1e1c2fb9235a7598a29c556a903bebe0d091c34b941cc33f122  protocol.go
f81222127404efa29362f552fe113c726095b5f5588f0e0645e418d9e7d7abf5  g4_protocol_conformance_test.go
ab741967e43164869f0808bd07ce7b39253ec3503b142ab467492e788604e515  g4_routing_failure_test.go
700662a0c69f5ba1bb383dbaa943657628261c2e9499c78cb8d37d38fa5cada6  g4_property_stress_test.go
546518fed76677cf63364acb776437c987178d7b01132b5c1bfe604b0e4e6e07  registry.go
c6a0e2bad8b6cfcfc2de629127cc2ee5efd8ac14bf18768f377b3ada3a09e230  registry_test.go
d8a87ac586c26f8efa97b51b0fca53798e9ef641858661949219e0b78ef5e7cc  policy.go
87be2c953dfc9b20f83219ebc6d26fd2fb2cc3061bbe5690f6db38d21b07aa8d  measurability.go
88780fc76674a3b7068cc79f8bcec10b0a6209e8f755bdcba8b8b1bf49cdaff0  measurability_test.go
f965b75cd3d18aae335ef6525b8f72769fde0117bb81e4d50483456766f924ec  contracts_test.go
```

## Safety boundary and residuals

- Synthetic identities use non-production labels and `.invalid` endpoints. Authentication is supplied only by an in-memory test source with a synthetic value; no `SecretFileRef` is opened.
- No credential/auth/secret value was read, copied, printed, rewritten, rotated, quarantined, or mutated. Synthetic account and circuit state exists only inside the test process.
- No live provider or OmniRoute validation, Multica-daemon dispatch, production/cutover operation, Prodex removal, tier activation, or native 5.6–5.8 acceptance was performed.
- Exact `agy/claude-opus-4-6-thinking` coverage is a synthetic contract check only. It does not grant a new route approval, prove live/native Antigravity behavior, or close shared OpenSpec task 8.1.
- Registry backoff and graph tests use deterministic clocks/documents only; they do not prove a live OmniRoute registry, approve any fallback edge, or change route admission.
- I3 is one bounded terminal-record measurement scope. The cross-model approval boolean remains the caller's separate assertion of an accepted ordered policy; the gateway-produced proof covers only the accepted bounded cycle-free registry graph.
- I4 validates a supplied content-free queue sample but does not claim a production queue exists. I5 reports only the three current facts and their conjunction; it does not measure or claim recovery time, cadence, or a threshold.
- I3-I5 artifacts are PRODUCED only. No independent acceptance, 9.1 execution, task closure, or production measurability claim is made.
- Circuit upper-bound validation is a gateway contract check only; it does not prove that a deployed OmniRoute breaker consumes or enforces these values.
- No central daemon, config, health, cmd, module, adapters, execenv, models, brain, Prodex, runtimeenv, deploy, or observability file was edited by this hardening.
- This evidence does not close shared OpenSpec tasks or modify `EVIDENCE_INDEX.md`.

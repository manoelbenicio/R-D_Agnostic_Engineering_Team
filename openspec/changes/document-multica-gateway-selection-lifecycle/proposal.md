# Change: Document preserved Multica Gateway selection and lifecycle primitives

## Authority

The project owner has directed that the current Multica implementation remain exactly
AS-IS and that OmniRouter is excluded from this solution.

This change documents existing candidate behavior only. It authorizes no code change,
runtime wiring, credential work, deployment, production acceptance, or publication.

## Provenance

Candidate: `a5aa53e8e89d2845cacfbc82ca851fbd18f9a505`

Covered commits and exact behavior/path mapping:

- `64fef29deeb320f52878529cf6a54ef58f19c31f`: initial round-robin selection,
  failure classification, and dispatch coordinator.
  - Paths: `classification.go`, `classification_test.go`, `dispatch.go`,
    `dispatch_test.go`, `selection.go`, `selection_test.go`.
- `5471eda25682ef12f32fa2c3c3c5f241b548d887`: account lifecycle and re-entry
  tests on the production Selector type.
  - Path: `selection_lifecycle_test.go`.
- `b608539afe37dbb174ebdcce5170eddce69d015e`: selection, classification, and
  dispatch remediation.
  - Paths: `classification.go`, `classification_test.go`, `dispatch.go`,
    `dispatch_test.go`, `selection.go`, `selection_test.go`.
- `93d4a255710e0531c1b576033b1beb59b67ffeab`: second remediation plus Executor
  composition.
  - Paths: `classification.go`, `classification_test.go`, `dispatch.go`,
    `dispatch_test.go`, `executor.go`, `executor_test.go`, `selection.go`,
    `selection_lifecycle_test.go`, `selection_test.go`.
- `1fd7f94b9bdfb4d8524d705b9e2179ca8a55f937`: concurrent round-robin,
  affinity, and pseudonymous-record acceptance evidence.
  - Paths: `executor.go`, `g4_task84_acceptance_test.go`.
- `653b5846b02eb07936b6b0d3f8311d52520ef238`: Task84 test-name correction.
  - Path: `g4_task84_acceptance_test.go`.

Every mapped path is relative to
`multica-auth-work/server/internal/daemon/gateway/`. The exact ten-file changed-path
allowlist is:

- `multica-auth-work/server/internal/daemon/gateway/classification.go`
- `multica-auth-work/server/internal/daemon/gateway/classification_test.go`
- `multica-auth-work/server/internal/daemon/gateway/dispatch.go`
- `multica-auth-work/server/internal/daemon/gateway/dispatch_test.go`
- `multica-auth-work/server/internal/daemon/gateway/selection.go`
- `multica-auth-work/server/internal/daemon/gateway/selection_test.go`
- `multica-auth-work/server/internal/daemon/gateway/selection_lifecycle_test.go`
- `multica-auth-work/server/internal/daemon/gateway/executor.go`
- `multica-auth-work/server/internal/daemon/gateway/executor_test.go`
- `multica-auth-work/server/internal/daemon/gateway/g4_task84_acceptance_test.go`

## AS-IS

Multica contains Gateway selection, lifecycle, classification, retry/deduplication, and
execution primitives with deterministic in-process tests. Production Gateway
client/readiness wiring exists, but no production constructor or caller of Selector,
Executor, or the Gateway dispatch Coordinator is evidenced in the candidate.

Existing live and archived OpenSpec assigns account selection elsewhere and therefore
contradicts the preserved implementation.

## TO-BE documentation state

The repository records the exact behavior and provenance of the preserved Multica
primitives, explicitly distinguishes implemented library behavior from evidenced live
wiring, and leaves contradictory external-router authority for a separately authorized
documentation reconciliation.

## Non-claims

- No production selection/lifecycle constructor is claimed.
- No credential or account source is defined.
- No runtime behavior beyond the candidate implementation is authorized.
- No code, test, deployment, production, publication, or integration acceptance is granted.

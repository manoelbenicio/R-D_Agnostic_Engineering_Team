# ORQ-78 owner-decision application — exact evidence delta

- Timestamp: 2026-08-03 (America/Sao_Paulo)
- Scope: apply Root Owner decisions `G01-B`, `G02-A`, `G03-A`, `G06-B`, and `G08-A` without repeating the accepted 140-commit audit or technical test suites.
- Candidate before this artifact: `b4535a8f95cea396f2fcc20e89554b2b0e2f88e4`.
- Evidence source commit for historical CREDISO bytes: `da42282372d42f61c24c3b8b67bc79e86dc85473`.

## GAP-01 — G01-B

Root Owner waived only the irrecoverable historical producer/pre-edit attribution. No producer identity or check-in is inferred or fabricated.

The exact historical CREDISO-4.1 byte identities are independently reproducible from Git objects:

| Path at `da422823...` | Git blob | SHA-256 |
|---|---|---|
| `multica-auth-work/server/internal/rotation/detector_discovery.go` | `47d730f45e23bb206f6ffce013a4ecc960ecf77e` | `bc61a46c0c700010001a49c9075b87f97e89f802b138e63f90b53193a4b45b55` |
| `multica-auth-work/server/internal/rotation/detector_discovery_test.go` | `d3a7fdf225d80ed9388aa9819402c416ca203c50` | `4e8092ff6b0f37a8416b961c89040433682e2d0d654293e192992dd3e1d55a4f` |

These hashes match the preserved independent reproduction in `.planning/agent-brain-v3/evidence/credential-isolation-4.1-push-eligibility-independent-review.md` and the provenance reconciliation in `.planning/agent-brain-v3/evidence/credential-isolation-4.1-provenance-reconciliation.md`.

Decision effect: missing attribution is waived; the exact historical artifact bytes are pinned. This does not assert that the detector files are present in the current candidate, authorize their reintroduction, or upgrade historical evidence beyond its recorded classification.

## GAP-02 — G02-A

The governing policy is same-session single-flight mutation with atomic commit or fail-closed rejection and no partial state. ORQ-96 credential-folder recycler locking is not authority for handler-session concurrency.

Decision effect: owner policy is fixed. Repository conformance remains a bounded technical delta and must be evidenced separately without repeating unrelated tests.

## GAP-03 — G03-A

The named frontend hooks `useSessionMonitor` and `isExpiringSoon` remain in scope alongside daemon diagnostics. Fail-closed failure-path validation remains mandatory.

Decision effect: scope is fixed without daemon-only re-scoping. Delivery/conformance remains a bounded technical delta.

## GAP-06 — G06-B

Root Owner waived only the irrecoverable historical producer attribution. No producer identity or pre-edit check-in is inferred or fabricated.

Exact historical redaction-core byte identities:

| Artifact/path | Git blob | SHA-256 |
|---|---|---|
| `da422823...:multica-auth-work/server/pkg/redact/redact.go` | `b508efd782497ed3029fa889a4158c22bcb4ae02` | `f409ba8a9f3e63618d59c5a8692296f8f7c019c9e558576b8786a058fbf68a5c` |
| `da422823...:multica-auth-work/server/pkg/redact/redact_test.go` | `f30fb8cd4c7dac1dfede4649557a4d96b31eb153` | `5a37941a1c7f1bd7263368a6479104c81300f7981e9f47bfb6b0cd17a602fec9` |
| Current `.planning/agent-brain-v3/evidence/credential-isolation-redact-core-review.md` | `e30d767789f659b21f07974586a07cf835dcc787` | `521cef3196ca3c8b0c98b1ecdb120407c217d33c060660350783ac09e7fa8c12` |

The exact core pin is established. A distinct independent security review remains required by `G06-B`; prior technical reproduction should be reused rather than rerun.

## GAP-08 — G08-A

Historical setter/pre-edit attribution for Chat 1.1/1.4 is waived. Their technical evidence remains `CORROBORATED`, not `DIRECT`, and ORQ-88 remains Done. No evidence is upgraded and no accepted card is reopened.

## Preserved boundaries

- No credential home, credential value, authentication material, runtime, service, timer, production resource, or remote ref was accessed or mutated to create this artifact.
- No historical producer, reviewer, check-in, hash, or test result was invented.
- No broad scan, 140-commit rescan, technical test rerun, or product-code change was performed.
- This artifact applies decisions and exact pins only. GAP-02/GAP-03 conformance and GAP-06 distinct security review remain explicit bounded deltas.

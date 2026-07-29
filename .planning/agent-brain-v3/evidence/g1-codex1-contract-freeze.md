# G1 Codex 1 — frozen contracts and preserved-baseline audit

status: FROZEN · contract: `agent-brain.v1` · scope: OpenSpec 1.1, 1.2, 1.6, 2.1–2.4

## EV-G1-01 — terminology and cold/hot boundary

- `Agent Brain` is the approved working name until the G8 debrand gate.
- Agent Brain owns the cold plane: admission, task/workspace/session lifecycle, CLI process
  lifecycle, cancellation, watchdog, recovery, result streaming, context and local skills.
- OmniRoute owns the hot plane: provider credentials/accounts, model routing, selection,
  rotation, affinity, quota, refresh, retry/fallback, circuit behavior and hot-path evidence.
- A gateway-required request has exactly one `RouterOwner`: `omniroute`. It cannot use
  Prodex/Rust L2, native-provider selection or legacy Go rotation as fallback.
- The frozen Go contract is `internal/daemon/brain` and is not wired into the active daemon.

## EV-G1-02 — neutral identifiers and initial release decision

- `CLIKind` identifies only the executable frontend.
- `RouteModel` is the exact gateway model ID and never selects a credential owner.
- `RouterOwner` is a single value; the target value is `omniroute`. Legacy values exist only
  for bounded translation and drain evidence.
- Common task/session/request correlation is mandatory; continuation correlation is optional.
- The initial build/canary set contains one route:
  `CLIKind=claude-code`, `RouteModel=agy/claude-opus-4-6-thinking`, Anthropic Messages.
- The route is a build-only canary candidate and is not admissible until its protocol and
  capability evidence passes. Same-model account fallback is owned by OmniRoute, pre-commit
  only. Cross-model fallback is empty until capability equivalence is proven and approved.
- Stable-key scope is inference-only, approved-routes-only, management-denied and
  provider-native-denied.
- Host/WSL topology uses the loopback OmniRoute root; container topology uses configured
  reachable container DNS. The endpoint is topology-specific, never universal.
- Capacity target is tier 20 only. Tier 50 and 100 schema entries are frozen but remain
  evidence-gated.

## EV-G1-03 — neutral configuration freeze

Neutral environment names:

- `AGENT_BRAIN_CONTROL_URL`
- `AGENT_BRAIN_GATEWAY_REQUIRED`
- `AGENT_BRAIN_GATEWAY_BASE_URL`
- `AGENT_BRAIN_GATEWAY_SECRET_FILE`
- `AGENT_BRAIN_GATEWAY_READINESS_POLICY`
- `AGENT_BRAIN_TASK_CAPACITY_TIER`
- `AGENT_BRAIN_LEGACY_EXECUTION_ENABLED`

The dedicated child-only Codex custom-provider `env_key` is frozen as
`AGENT_BRAIN_OMNIROUTE_API_KEY`. It is applied last only to authorized inference children;
the service continues to accept only the restricted secret-file reference.

Precedence is frozen as: neutral CLI, legacy CLI, neutral environment, legacy environment,
neutral stored config, legacy stored config, default. Conflicts at the same source fail.
Prodex/L2 settings are not semantically valid OmniRoute aliases; they are inventoried only
to prevent an unsafe automatic translation.

Gateway-required mode requires an absolute secret-file reference and strict fail-closed
checks for liveness, authentication, model registry, selected model and selected protocol.
The contract stores only the reference; it does not read or expose secret values.

Tier schema: 20 = canary-authorized; 50 and 100 = evidence-required. The schema does not
conflate task admission with OmniRoute route/account concurrency.

## EV-G1-04 — bounded compatibility facade

Frozen legacy surfaces are the daemon API `provider/model`, task `auth_token` and child
token alias, `runtime_router_owner`, legacy environment/stored configuration, daemon CLI
command and runtime brief. The facade translates to neutral contracts, records alias use,
redacts the opaque task token and rejects a legacy router owner that conflicts with
gateway-required mode. No facade is wired into the active path in G1.

Removal of any alias requires zero-use telemetry, named consumer migration, release notes
and rollback independence.

## EV-G1-05 — preserved dirty baseline and 16-task reconciliation

The `persist-prodex-runtime-integration` worktree was preserved without reset, stash,
revert or discard. Audit scope contained 19 dirty entries at check-in: 10 modified and
9 added/untracked, with no deletions. Whitespace validation passed.

| Baseline task | Reconciled state | Audit disposition |
|---|---|---|
| 1.1 | IMPLEMENTED | Adapter path and pinned Prodex path are separate and validated. |
| 1.2 | IMPLEMENTED | Go launches the adapter and passes the pinned gateway path through its environment. |
| 1.3 | IMPLEMENTED | Required mode fails closed instead of silently downgrading. |
| 2.1 | INCOMPLETE | No persistent launcher/service is present in the preserved diff. |
| 2.2 | INCOMPLETE | No auditable persistent environment-file update is present in the repository diff. |
| 2.3 | IMPLEMENTED | Health reports configuration source, runtime authority and profile count. |
| 3.1 | IMPLEMENTED | Validated account inventory drives Codex profile mapping. |
| 3.2 | IMPLEMENTED | Registration uses reference-only profile enrollment. |
| 3.3 | IMPLEMENTED | Root containment, filesystem, modes, credential presence and duplicate identity are checked. |
| 3.4 | IMPLEMENTED | L2 sessions select only reconciled profile references and fail closed in required mode. |
| 3.5 | INCOMPLETE | Inventory-scoped credential/account purge is not implemented by the preserved product diff. |
| 4.1 | PARTIAL | Existing helper/compile coverage passes; required-mode/profile tests remain absent. |
| 4.2 | INCOMPLETE | Dedicated reconciliation cases are absent. |
| 4.3 | PARTIAL | Rust tests pass and Go hotspots compile; no signed build attestation exists. |
| 4.4 | INCOMPLETE | No authorized live restart/readiness evidence exists. |
| 4.5 | INCOMPLETE | No complete Prodex-continuity operational evidence set exists. |

Because the change is incomplete and superseded as an active routing plan, its OpenSpec
checkboxes remain unchanged. Its implemented security guarantees remain part of the dirty,
auditable baseline until the later signed replacement/removal gate.

Verification recorded at G1:

- Rust sidecar: 4 tests passed.
- Credential-isolation harness: passed.
- Full Go suite, including neutral package and daemon/CLI packages: passed after approved
  local-listener access.
- Existing Prodex helper tests: passed.
- The first sandboxed full-suite attempt correctly exposed a local-listener restriction;
  the approved rerun removed that environment limitation and passed.

## Consumer publication and ownership

- Codex 2 consumes only `internal/daemon/brain` contracts and owns new `gateway/**` files.
- Codex 3 consumes only the frozen contracts and owns new `runtimeenv/**` plus assigned CLI
  adapter files. Codex 3 must not edit Codex 1 hotspots.
- Codex 4 consumes the frozen config/readiness/tier/cutover identifiers for operations and
  evidence assets; it does not edit daemon, gateway or runtime implementation.
- Only Codex 1 integrates shared daemon/config/health/command entrypoints and the enumerated
  Prodex, execenv and model hotspots.

Mandatory cutover blockers remain protocol/model conformance, failure injection,
single-flight refresh, readiness fail-closed proof, Smart Context parity or waiver,
continuation-affinity proof, state-topology decision, pinned image digest and formal sign-off.

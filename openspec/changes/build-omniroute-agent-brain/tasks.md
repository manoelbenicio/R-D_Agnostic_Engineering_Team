## 1. Single-router runtime removal

- [x] 1.1 Delete dedicated alternate-router binaries, sidecars, runtime packages and legacy account-rotation package.
- [x] 1.2 Delete dedicated daemon startup/facade/profile/session consumers and tests.
- [x] 1.3 Delete dedicated smoke scripts, prompts, operational documents and obsolete OpenSpec changes.
- [x] 1.4 Remove obsolete ambiguous alternate startup, shutdown, retry/rotation and task-time
  routing branches from `daemon.go`; preserve the approved R3 `native_credential_home` contract.
- [x] 1.5 Remove alternate config structs, environment aliases and CLI migration flag.

## 2. Mutually exclusive transport-binding contract

- [x] 2.1 Accept exactly `omniroute` or `native_credential_home`; reject missing, ambiguous, or
  unknown bindings without translation.
- [x] 2.2 Require one complete binding-specific admission plan before CLI launch.
- [x] 2.3 Preserve workspace/repository/worktree, context/skills, process lifecycle, watchdog,
  cancellation, stream, commit-ledger and terminal-result paths.
- [x] 2.4 Keep controlled `runtimeenv`/`execenv`: credentialless and home-free for `omniroute`;
  daemon-local isolated-home references only for `native_credential_home`.
- [x] 2.5 Keep provider-account lookup inside OmniRoute for `omniroute`; for native mode use only
  R3's approved exclusive opaque assignment and never perform task-time account/home rotation.

## 3. Recovery and health

- [x] 3.1 Implement binding-local `NORMAL(<pinned binding>)` and `DEGRADED(none)` states.
- [x] 3.2 Require a session boundary and strict readiness of the same binding for restore.
- [x] 3.3 Add tests proving an outage never promotes, translates, rotates, or retries the other
  binding or another native home.
- [x] 3.4 Report pinned binding authority and Main Brain diagnostics without raw path, account
  identity, credential data, or obsolete migration-mode fields.

## 4. Product and operations preservation

- [x] 4.1 Preserve backend/web, projects, squads, Kanban issues/tasks, Postgres and terminal/session persistence.
- [x] 4.2 Update README and self-host Compose comments for the binding-specific Main Brain topology.
- [x] 4.3 Rewrite rollout and rollback runbooks to keep admission closed instead of switching
  binding, selecting another native home, or mutating source credential homes.
- [x] 4.4 Update typed deployment catalog wording and secret-reference failure policy.
- [x] 4.5 Document and exercise Kanban-only dispatch with exactly one product task per
  activation; prohibit parallel Herdr execution.

## 5. Verification

- [x] 5.1 Run `gofmt` on every modified Go file using an available pinned Go toolchain.
- [x] 5.2 Run targeted tests for `internal/daemon/brain`, `internal/daemon/gateway`, `internal/daemon/runtimeenv`, `internal/daemon/execenv`, `internal/daemon/deploy` and `internal/daemon`.
- [x] 5.3 Run a server-wide build/test check and resolve compile fallout from deleted symbols.
- [x] 5.4 Run `openspec validate build-omniroute-agent-brain --strict`.
- [x] 5.5 Run residual scans proving active authority permits only the two exact bindings, keeps
  their ownership non-overlapping, forbids cross-mode fallback, and never deletes or mutates a
  source credential home.

## 6. Operational work still requiring external readiness/evidence

- [x] 6.1 For `omniroute` only, consume the owner/operator OmniRoute immutable-revision and
  selected-route/protocol readiness declaration, and verify Main Brain's strict fail-closed
  reaction; do not inspect, test or validate OmniRoute provider/model mappings, accounts,
  credentials, sessions or rotation internals. Do not apply this gateway check to
  `native_credential_home`.
- [x] 6.2 Complete metadata-only end-to-end correlation for ingress, queue, daemon, CLI,
  selected transport, terminal persistence and UI delivery.
- [ ] 6.3 Validate and approve the 20-task bounded capacity profile; keep the current lower limit until evidence passes.
- [ ] 6.4 Validate 50- and 100-task profiles only after the lower tier and observability gates pass.
- [x] 6.5 Perform an owner-approved Kanban → Main Brain → OmniRoute → terminal acceptance run for
  the `omniroute` binding in the controlled environment; do not perform it as part of basic
  source validation and do not treat it as native-mode evidence.

## 7. SPE-5 documentation authority reconciliation

- [x] 7.1 Reconcile proposal, design, tasks and affected specs with
  `credential-account-home-restoration`; this checkmark records documentation only.
- [x] 7.2 Replace automatic 24-hour source-slot destruction with metadata-only retirement and
  active-reference-safe cleanup of task-local non-source material only.
- [x] 7.3 Freeze the future pre-handoff/deploy synchronization gate: both OpenSpecs strict-valid,
  cross-authority scan clean, zero pending owned files, exact commit published to a non-main
  upstream, matching local/upstream SHA, 0/0 divergence, and deployment evidence pinned to it.


## 8. Additive contract reconciliation candidate

- [x] Retain accepted mutually exclusive `omniroute` and `native_credential_home` semantics.
- [x] Reject later OmniRoute-only reductions as non-superseding historical divergence.
- [x] Preserve later stable physical cardinality/tombstone semantics in credential REQ-34/35.
- [x] Preserve the non-fabricated `proc_id` limitation while retaining recorder wiring.
- [x] Record current local state as composed, uncommitted, unpushed and externally action-gated.
- [x] Strict-validate this change together with the reconciled credential change.
- [x] Require exact SPE-7 and cross-authority validation before candidate acceptance.

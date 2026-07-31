# Tasks - SPE-5 Runtime Manager Contract Freeze

> SPE-5 is documentation/OpenSpec only. Checkmarks below certify contract documentation, not
> executable implementation or production readiness. K1 is the sole canonical writer; K3 is an
> independent non-editing reviewer.

## 1. Canonical authority freeze

- [x] 1.1 Declare this change canonical for Runtime Standards, Runtime Sessions, workspace
  bindings, runtime configuration, native account-home selection and task snapshots.
- [x] 1.2 Supersede incompatible OmniRoute-only account ownership for those subjects without
  editing the conflicting change or weakening OmniRoute gateway routing ownership.
- [x] 1.3 Freeze fail-closed behavior and immutable ORQ2-dev reservations.

## 2. Runtime and configuration contracts

- [x] 2.1 Freeze owner-global versioned Runtime Standards and owner-global reusable accountless
  Runtime Sessions.
- [x] 2.2 Freeze workspace enrollment onto existing agent/runtime rows and the existing daemon,
  without recreation.
- [x] 2.3 Freeze one agent/runtime/exclusive home and subagent inheritance semantics.
- [x] 2.4 Freeze platform > standard > runtime > explicit delegable task precedence.
- [x] 2.5 Enumerate model, reasoning, limits, concurrency, retry, flags, env, skills, MCP/tools,
  permissions, eligibility, health and fallback fields.
- [x] 2.6 Freeze per-field capability validation, delegability, hot/restart apply class,
  activation, audit, redaction, drift and rollback.
- [x] 2.7 Freeze claim-time generation/version/digest task snapshots and reclaim invariants.

## 3. Dynamic credential-home catalog

- [x] 3.1 Replace slot allowlists/raw paths with opaque arbitrary-child controlled-root discovery.
- [x] 3.2 Freeze hint plus periodic full reconciliation, overflow handling and monotonic atomic
  catalog generations.
- [x] 3.3 Freeze filesystem-identity deduplication, TTL, quarantine, active references,
  watermarks, tombstones and retention.
- [x] 3.4 Prohibit copy/move/delete/truncate/sanitize/overwrite of source credential homes.
- [x] 3.5 Make configured concurrency independent of account inventory.

## 4. Shared interface freeze

- [x] 4.1 Reserve migration names `130_runtime_standards`, `131_runtime_sessions`,
  `132_credential_home_catalog`, `133_runtime_bindings`, and
  `134_runtime_configuration_snapshots`, subject to registrar recheck.
- [x] 4.2 Freeze pathless REST resources, exact methods/routes/body fields, pagination,
  idempotency and status codes.
- [x] 4.3 Freeze event envelope/types, at-least-once deduplication and generation-gap recovery.
- [x] 4.4 Freeze owner/admin/daemon/workspace authorization, canonical errors, audit and secrecy.
- [x] 4.5 Create a deferred, unsent SPE-5 Kanban payload with no invented UUID and a UUID lookup
  prerequisite; final in-review transition is conditional on K3 review.

## 5. Validation and handoff

- [x] 5.1 Run strict OpenSpec validation.
- [x] 5.2 Parse the deferred JSON payload and run `git diff --check`.
- [x] 5.3 Verify topic coverage, documentation-only scope, exactly five owned paths and no
  path/account/credential disclosure.
- [ ] 5.4 K3 independently reviews and signs the frozen interfaces.

## 6. Downstream implementation - not authorized by SPE-5

- [ ] 6.1 C2 rechecks migration identities immediately before implementation; any collision
  blocks work and requires a canonical amendment.
- [ ] 6.2 C1/C2/C3/C4/K2 implement registry, schema, Runtime Manager, UI/CLI and catalog only in
  their assigned worktrees after approval.
- [ ] 6.3 K1 integrates reviewed commits and shared daemon/protocol wiring in SPE-9.
- [ ] 6.4 C5/K3 execute independent migration, race, privacy, capability, recovery, drift,
  rollback and end-to-end evidence gates.
- [ ] 6.5 Council accepts, K3 pins a release recommendation, and owner separately authorizes
  production rollout. SharePoint remains prohibited.

# Design - Runtime Manager and Opaque Account-Home Contracts

## 1. Authority and system boundary

This document is canonical for Runtime Standards, Runtime Sessions, workspace bindings,
runtime configuration, native ORQ2 account-home selection, and task snapshots. The only current
solution and transport binding is `native_credential_home`. OmniRouter/OmniRoute is not a current
solution, fallback, credential authority, dependency, or acceptance source. R3 resolves exactly
one approved exclusive opaque home for one existing logical runtime/agent before launch, and the
daemon supplies only daemon-local isolated-home references required by that native CLI. There is
no router, global HOME, cross-workspace, cross-home, stale-generation, or ORQ2-dev fallback.
Missing, ambiguous, stale, unauthorized, unhealthy, unsupported, or conflicting state fails closed.

The design reuses existing agent rows, runtime rows, workspace, and
`orq2-credential-runtime-v1`. It creates no daemon, container, runtime installation, agent,
workspace, project, squad, or credential home. Existing ORQ2-dev bindings are immutable.

## 2. Canonical concepts and identities

All externally visible IDs are UUIDs or opaque non-path references.

- **Runtime Standard**: owner-global named policy; immutable versions, one active version.
- **Runtime Session**: owner-global reusable accountless logical runtime identity. It names a
  provider adapter/runtime family and configuration lineage, never an account or home.
- **Workspace enrollment**: authorization for a workspace to project an owner-global session
  onto an existing runtime row and existing daemon.
- **Runtime binding**: `(workspace_id, agent_id, runtime_id, session_id)` plus active
  configuration and optional exclusive opaque home assignment.
- **Catalog entry**: daemon-local metadata for a discovered home. Product surfaces receive
  `home_ref`, generation, provider, state and health only; never path, source identifier,
  account identity, inode/device, credential filename or credential data.
- **Configuration version**: immutable full runtime configuration document and digest.
- **Task snapshot**: immutable claim-time IDs, versions, generations and digests.

A Runtime Session is deliberately accountless. Subscription/home enrollment is a separate
exclusive binding operation and can change without changing session, agent, runtime, or daemon
identity.

## 3. Implemented additive migration chain

The accepted local source contains the byte-verified additive chain:

1. `128_task_usage_account_id` - immutable producer-account attribution.
2. `129_task_usage_price_snapshot` - paired `price_version` and `computed_cost_usd`.
3. `130_runtime_standards` - standards and immutable versions.
4. `131_runtime_sessions` - accountless sessions; down intentionally refuses with SQLSTATE `55000`.
5. `132_credential_home_catalog` - pathless catalog metadata and generations.
6. `133_runtime_bindings` - existing-row binding and exclusive assignment.
7. `134_runtime_configuration_snapshots` - immutable versions, activation and task snapshots.
8. `135_migration_checksums` - self-contained `schema_migrations` checksum support.

Rollback is forward-safe: reversible verification covers 134 through 132 down/up, migration 131
refusal is asserted separately, and migrations 130/129/128 are not destructively crossed by the
Runtime Manager rollback gate. Operational rollback activates prior immutable versions and never
mutates source credential homes.

## 4. Frozen configuration document

Every standard version and runtime version carries these typed groups. Unknown fields are
rejected, not ignored.

| Group | Required contract |
|---|---|
| routing | `provider`, optional opaque `subscription_ref`, `transport_binding` fixed to `native_credential_home` |
| model | literal `model_id`, provider catalog/version, capability digest |
| reasoning | mode/level/budget exactly as provider capability advertises; no ID normalization |
| limits | context, input, output and total token ceilings; wall/idle timeout |
| concurrency | session/task/subagent maxima and queue bounds, independent of account count |
| retry | max attempts, backoff, jitter, retryable classes and deadline budget |
| flags | ordered validated CLI flags; forbidden credential/path flags rejected |
| env | key allowlist and non-secret values/references; no inherited ambient credential vars |
| skills | allowed skill IDs and immutable versions/digests |
| mcp_tools | allowed server/tool IDs, scopes, timeouts and versions; no inline secrets |
| permissions | filesystem roots by symbolic policy, network egress policy, process/tool grants |
| eligibility | provider/runtime/daemon/workspace predicates and required capabilities |
| health | freshness TTL, readiness thresholds, circuit state and probe class |
| fallback | ordered provider/model attempts, bounds and failure classes within the pinned native home; never router, another home, global HOME or cross-workspace |

Each leaf is registered with: JSON type/schema, source, effective value, `delegable: true|false`,
`apply_class: hot|restart`, capability predicate, redaction class, and audit representation.
Unregistered leaves, unsupported values, and task overrides of non-delegable leaves fail before
activation or launch.

## 5. Precedence and inheritance

Effective configuration is deterministic per leaf:

`platform constraint > active Runtime Standard version > active per-runtime version > explicit delegable task override`.

Higher policy constrains lower layers; a lower layer cannot widen a maximum, permission,
eligibility set, network/filesystem scope, fallback set, or secret exposure. Omission inherits;
explicit `null` is accepted only where the leaf schema defines clearing. The resolver emits a
canonical JSON document and SHA-256 digest.

Subagents inherit the parent's pinned task snapshot. They may receive only explicit task
overrides marked delegable, within parent and platform bounds. They share the parent binding
and home assignment, do not create persistent runtime/session rows, and count against configured
concurrency, not against account inventory.

## 6. Capability validation, activation and rollback

Creation stores an inactive immutable version. Validation is pure/control-plane and checks:
provider/runtime version, model/reasoning combinations, all limits, flags/env, skills/MCP,
permissions, eligibility, health and fallback compatibility. It returns field-addressed issues
without credential reads or inference.

Activation is compare-and-swap on expected active version and catalog/binding generation. It
records actor, request/correlation ID, previous/new versions and digests, validation capability
digest, apply class, reason, and timestamp. Secret/reference fields are redacted. Hot fields
apply only after daemon acknowledgement. Restart-class changes remain pending until the existing
runtime process restarts and acknowledges the version. Running tasks remain pinned unless a
separately authorized hard revoke terminates them.

Rollback creates an auditable activation of a prior immutable version; it does not mutate
history. Drift compares desired version/digest with daemon-applied version/digest. Drift,
missing acknowledgement or unsupported capability blocks new claims for that binding.

## 7. Dynamic controlled-root catalog

A daemon-local registry contains configured controlled roots, each validated as absolute,
owner-controlled and non-overlapping. Roots and raw child paths never leave the daemon. Every
arbitrary immediate child is a candidate; there is no slot number grammar, fixed slot range, or
per-slot allowlist. Provider-specific layout validators inspect metadata only and MUST NOT open
credential contents.

Reconciliation combines filesystem notifications as hints with periodic full scans. Hint loss,
overflow, watcher restart, startup and interval expiry trigger a full scan. A new immutable
snapshot is swapped atomically with a strictly increasing `catalog_generation`; partial scans
never publish. Full scans discover overflow beyond prior capacity assumptions.

Entries deduplicate by stable filesystem identity inside a root. Aliases, hard-link/layout
collisions, root escape, symlink traversal, wrong ownership/mode, duplicate provider identity,
invalid layout and ambiguous bindings are quarantined. Quarantine is metadata-only.

Lifecycle states are `candidate`, `healthy`, `degraded`, `quarantined`, `missing`, `draining`,
and `retired`. Metadata has `first_seen_at`, `last_seen_at`, `last_full_scan_at`, TTL, health
watermark, missing watermark and retention deadline. Active task references prevent retirement.
Missing entries remain tombstoned through TTL and retention; generation never decreases and
opaque refs are never reused. Low/high/critical watermarks emit state changes and can close
admission. Catalog lifecycle never copies, moves, deletes, truncates, sanitizes, overwrites,
chmods, changes ownership of, or authenticates a source home; retention expiry retires metadata
only. Active, draining, referenced, reserved, admissible, unproven, and quarantined homes remain
physically preserved. Physical whole-home deletion belongs only to the separately accepted cleanup
executor after the guarded state progression below.

`PROTECTED_ACTIVE -> DRAINING_PROTECTED -> INACTIVE_PROVEN -> RETENTION_HOLD ->
POLICY_NOT_RETAINED -> DELETION_ELIGIBLE` is the sole cleanup progression. `INACTIVE_PROVEN`
requires released assignment, zero reservations and active references, non-admissibility, current
unambiguous authority mapping, and successful authoritative reads/fences. Age greater than 24 hours
is required to leave `RETENTION_HOLD`, but age or mtime is never sufficient. `DELETION_ELIGIBLE`
also requires independently accepted ORQ-96 database/admission/fencing/no-bypass gates and a
separate destructive cutover authorization. Any ambiguity or failed guard transitions to or remains
`QUARANTINED_FAIL_CLOSED`; force semantics cannot bypass a guard. Durable tombstones survive any
eventual physical deletion and remain non-reusable.

## 8. Exclusive assignment and admission

An active home assignment is unique by opaque `home_ref` and unique by binding. One persistent
agent maps to one existing runtime row and one Runtime Session projection. A
`native_credential_home` launch must have exactly one healthy exclusive assignment resolved by
R3 and persisted before claim for that existing logical runtime/agent. The daemon resolves the
opaque reference internally and supplies only isolated task-local home references required by
the native CLI; no global HOME, source raw path, account identity, or credential value crosses a
product/API/event/log boundary. Enrollment attaches a newly legitimate subscription/home to the
existing binding without recreation. Deterministic initial assignment operates only among
healthy, eligible, unassigned entries. It is not task-time rotation. Stale generations,
conflicts and ORQ2-dev reservations fail closed.

Concurrency limits are configuration policy and are independent of the number of catalog homes.
A home is never shared merely to satisfy concurrency. Every launch uses `native_credential_home`
and the pinned exclusive `home_ref`. A bounded provider/model retry may run only when it preserves
that exact native binding and home. It MUST NOT switch, translate, rotate, or retry through a
router or another home. Global HOME, cross-workspace, stale-generation, and ORQ2-dev fallback are
forbidden.

## 9. Task snapshot

At atomic claim, persist:

- `runtime_session_id`, `runtime_id`, `agent_id`, `workspace_id`;
- `runtime_standard_version_id`, `runtime_configuration_version_id`;
- canonical `effective_configuration_digest`;
- `runtime_binding_id`, `binding_generation`;
- optional opaque `home_ref` and `catalog_generation` for native execution;
- provider capability/catalog digest and snapshot creation time.

Reuse/reclaim preserves the snapshot. No live lookup rewrites history. A stale/missing/revoked
snapshot is refused or drained according to explicit revoke policy, never silently remapped.

## 10. Frozen REST contracts

All endpoints are JSON under `/api`, require authenticated human owner/admin unless explicitly
marked daemon, apply workspace authorization where a workspace is present, and return only
opaque IDs. Task actors cannot administer these resources.

| Method and path | Purpose / body |
|---|---|
| `GET /api/runtime-standards` | list owner-global standards, redacted summaries |
| `POST /api/runtime-standards` | create standard `{name, description}` |
| `GET /api/runtime-standards/{standard_id}` | standard and active version |
| `POST /api/runtime-standards/{standard_id}/versions` | create inactive `{configuration, reason}` |
| `POST /api/runtime-standards/{standard_id}/versions/{version_id}/validate` | capability validation, no inference |
| `POST /api/runtime-standards/{standard_id}/versions/{version_id}/activate` | `{expected_active_version_id, reason}` |
| `POST /api/runtime-standards/{standard_id}/rollback` | `{target_version_id, expected_active_version_id, reason}` |
| `GET /api/runtime-sessions` | list reusable accountless owner-global sessions |
| `POST /api/runtime-sessions` | `{name, provider, runtime_kind, standard_id}` |
| `GET /api/runtime-sessions/{session_id}` | redacted session detail |
| `POST /api/workspaces/{workspace_id}/runtime-sessions/{session_id}/enroll` | `{runtime_id, agent_id}` existing rows only |
| `DELETE /api/workspaces/{workspace_id}/runtime-sessions/{session_id}/enrollment` | drain/deactivate projection; no source mutation |
| `GET /api/workspaces/{workspace_id}/runtime-bindings` | pathless binding/session/config/home-state list |
| `GET /api/workspaces/{workspace_id}/runtime-bindings/{binding_id}` | redacted binding detail |
| `POST /api/workspaces/{workspace_id}/runtime-bindings/{binding_id}/home-assignments` | `{home_ref, expected_binding_generation, expected_catalog_generation}` |
| `DELETE /api/workspaces/{workspace_id}/runtime-bindings/{binding_id}/home-assignment` | `{expected_binding_generation, drain}` |
| `POST /api/workspaces/{workspace_id}/runtime-bindings/{binding_id}/configuration-versions` | `{configuration, reason}` |
| `POST /api/workspaces/{workspace_id}/runtime-bindings/{binding_id}/configuration-versions/{version_id}/validate` | effective validation |
| `POST /api/workspaces/{workspace_id}/runtime-bindings/{binding_id}/configuration-versions/{version_id}/activate` | `{expected_active_version_id, expected_binding_generation, reason}` |
| `POST /api/workspaces/{workspace_id}/runtime-bindings/{binding_id}/rollback` | `{target_version_id, expected_active_version_id, reason}` |
| `GET /api/workspaces/{workspace_id}/credential-homes` | opaque catalog projection only |
| `POST /api/workspaces/{workspace_id}/credential-homes/reconcile` | request full reconciliation; returns accepted operation ID |
| `GET /api/workspaces/{workspace_id}/credential-homes/reconciliation/{operation_id}` | generation/result counters only |
| `POST /api/daemon/credential-catalog/reconciliation` | daemon report `{daemon_id, previous_generation, generation, scan_kind, counters, digest}` |
| `POST /api/daemon/runtime-bindings/{binding_id}/acknowledgements` | applied/restart/drift acknowledgement with versions/digests |

Create returns 201, reads/updates 200, reconciliation request 202, and successful drain/delete
204. Mutations require `Idempotency-Key`; replay returns the original result. Pagination uses
`cursor`/`limit`; list responses are `{items, next_cursor}`.

Canonical errors are `{error:{code,message,field?,request_id,retryable}}` with status:
`400 invalid_argument`, `401 unauthenticated`, `403 forbidden`, `404 not_found`,
`409 version_conflict|generation_conflict|exclusive_assignment_conflict|active_reference`,
`412 capability_unsupported|health_stale|drift_detected`, `422 invalid_configuration`,
`429 capacity_exhausted`, and `503 catalog_unavailable|daemon_not_ready`. Messages MUST NOT
contain paths, account identities, credentials, environment values or provider response bodies.

## 11. Frozen event contracts

Events use the existing authenticated workspace stream. Envelope:
`{type,event_id,occurred_at,workspace_id?,resource_type,resource_id,generation?,version_id?,digest?,data}`.
Delivery is at-least-once; consumers deduplicate by `event_id`. Per-resource generations are
monotonic; a gap triggers GET/full reconciliation, never inferred state.

Frozen types: `runtime_standard.version_created`, `runtime_standard.activated`,
`runtime_standard.rolled_back`, `runtime_session.created`, `runtime_session.enrolled`,
`runtime_session.unenrolled`, `credential_catalog.reconciliation_started`,
`credential_catalog.generation_published`, `credential_catalog.entry_state_changed`,
`credential_catalog.watermark_changed`, `runtime_binding.home_assigned`,
`runtime_binding.home_draining`, `runtime_binding.home_released`,
`runtime_configuration.version_created`, `runtime_configuration.validation_failed`,
`runtime_configuration.activated`, `runtime_configuration.restart_required`,
`runtime_configuration.applied`, `runtime_configuration.drift_detected`,
`runtime_configuration.rolled_back`, and `task.runtime_snapshot_pinned`.

Event `data` contains only opaque refs, state, reason code, counters, apply class, versions and
digests. No raw path, filesystem identity, account identity, credential name/value, CLI argv,
environment value, prompt or provider response is permitted.

## 12. Authorization and audit

Owner may administer owner-global standards/sessions and all owned workspaces. Workspace admin
may read owner-global reusable choices and administer bindings/configuration only in that
workspace; it cannot alter global standards/sessions. Daemon credentials may report only for
the daemon and workspace bindings explicitly authorized to them. Members and task actors are
read-only only where separately granted; cross-workspace access is always denied.

Every mutation, denial, activation, rollback, assignment, reconciliation, quarantine, drift and
hard revoke is audited with actor type/opaque ID, workspace/resource IDs, request/event IDs,
old/new versions or generations, redacted diff, reason code and UTC time. Audit and task
snapshots are retained at least as long as task/usage evidence; catalog tombstones outlive the
maximum task retention plus reconciliation TTL. Retention or age alone never authorizes source-home
deletion; only the complete guarded cleanup progression in section 7 can establish eligibility.

## 13. Repository/OpenSpec synchronization gate

Before any future handoff or deployment, all affected OpenSpecs MUST strict-validate and a
cross-authority scan MUST prove the active transport, credential ownership, lifecycle, rollback,
and secrecy texts agree. The Git index and worktree MUST contain zero staged, modified, or
untracked owned files. The exact local commit MUST be published to its configured non-main
remote branch; local HEAD and upstream SHA MUST be identical, and ahead/behind MUST be `0/0`.
Deployment evidence MUST pin that exact SHA. Pending owned files, a missing/non-main upstream,
unpublished commits, SHA mismatch, or any divergence fails closed. This is a future handoff and
deployment gate only; documenting it performs no publish or deployment.

## 14. Stable physical allocator and T2 reconciliation

Accepted REQ-05 dynamic opaque catalog discovery has precedence; static slot lists and slot-number authority are forbidden. The physical allocator validates a canonical non-zero `AGENT_CRED_ISOLATION_AGENT_ID` and a
64-lowercase-hex `AGENT_CRED_ISOLATION_SUBSCRIPTION_FINGERPRINT` before allocation. It removes
Herdr/pane/TTY/PID/random fallback identity and persists one physical home per stable active binding.
Source v2 is a future separately authorized rollout, not installed state. Production reconciliation requires a privileged all-process `/proc` audit, preserves every live
reference, and fails closed when audit is unavailable. Age greater than 24 hours is a necessary
post-inactivity policy guard, never a sufficient deletion authority.

The catalog remains metadata-only. `home_ref` is a canonical UUID, `name_ref` is separate, immutable
generations and tombstones survive restart, and no catalog action creates, copies or deletes a
physical home. T2 keeps AGY/Antigravity, Codex and Kiro operational while preserving each runtime's
reasoning wire format. `task_usage` receives only the claim-time producer-account snapshot plus the
paired pricing snapshot; the daemon cannot select or rewrite financial attribution.

## 15. Current state and action boundary

Historical source-composition and validation evidence predates this authority correction and does
not validate these amended bytes. ORQ-107/108/109 remain pending independent acceptance and do not
authorize cleanup. Installed registry v1 is not changed by this documentation candidate. This
candidate performs no push, deployment, restart, credential access, credential-home mutation, or
destructive cutover. External actions remain separately owner-gated.

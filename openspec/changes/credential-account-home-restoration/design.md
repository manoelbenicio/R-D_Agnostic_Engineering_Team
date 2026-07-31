# Design - Runtime Manager and Opaque Account-Home Contracts

## 1. Authority and system boundary

This document is canonical for Runtime Standards, Runtime Sessions, workspace bindings,
runtime configuration, native ORQ2 account-home selection, and task snapshots. In a conflict,
this contract supersedes OmniRoute-only provider-account ownership language for these subjects
without changing OmniRoute's exclusive ownership of gateway routing and gateway inference
credentials. Native execution and gateway execution are distinct: gateway launches are
credentialless and receive no home reference; approved native launches resolve an opaque,
exclusive ORQ2 account-home reference. Ambiguity fails closed.

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

## 3. Frozen migration reservation

C2 MUST recheck the migration registrar immediately before implementation. If any identity is
occupied, implementation stops and K1 amends this canonical contract before files are created.
Subject to that recheck, names and order are frozen:

1. `130_runtime_standards` - standards, immutable standard versions, activation pointer/audit.
2. `131_runtime_sessions` - owner-global accountless sessions and workspace enrollments.
3. `132_credential_home_catalog` - opaque catalog metadata, generations, health/quarantine.
4. `133_runtime_bindings` - existing-row projection and exclusive home assignment.
5. `134_runtime_configuration_snapshots` - immutable configuration versions, activation,
   task snapshots, drift/rollback audit.

Migration 128 remains independent historical work. Migrations are additive. Operational
rollback deactivates/returns pointers to prior versions; it never deletes source homes or rows
still referenced by active tasks. Destructive down migration is prohibited while active
references exist.

## 4. Frozen configuration document

Every standard version and runtime version carries these typed groups. Unknown fields are
rejected, not ignored.

| Group | Required contract |
|---|---|
| routing | `provider`, optional opaque `subscription_ref`, `transport` (`gateway` or `native`) |
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
| fallback | ordered eligible routes, bounds and failure classes; never global HOME/cross-workspace |

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
admission. No lifecycle action copies, moves, deletes, truncates, sanitizes, overwrites, chmods,
or authenticates a source home.

## 8. Exclusive assignment and admission

An active home assignment is unique by opaque `home_ref` and unique by binding. One persistent
agent maps to one existing runtime row and one Runtime Session projection; native work requiring
an account must have exactly one healthy exclusive assignment. Enrollment attaches a newly
legitimate subscription/home to the existing binding without recreation. Deterministic selection
operates only among healthy, eligible, unassigned entries and persists the assignment before
claim. Stale generations, conflicts and ORQ2-dev reservations fail closed.

Concurrency limits are configuration policy and are independent of the number of catalog homes.
A home is never shared merely to satisfy concurrency. Gateway transport has no home assignment.
Fallback may choose only prevalidated routes in the pinned configuration; native-to-global,
cross-workspace, unauthorized native-to-gateway, and ORQ2-dev fallback are forbidden.

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
maximum task retention plus reconciliation TTL. Retention never authorizes source-home deletion.

# Spec - credential-account-home

## ADDED Requirements

### Requirement: REQ-01 Canonical authority and fail-closed boundary

This change and the reconciled `build-omniroute-agent-brain` change MUST jointly govern Runtime
Standards, Runtime Sessions, workspace bindings, runtime configuration, account-home selection,
and task snapshots. Every launch MUST pin exactly one transport binding:
`omniroute` or `native_credential_home`.

#### Scenario: OmniRoute binding
- **WHEN** the pinned binding is `omniroute`
- **THEN** OmniRoute MUST be the sole inference router and account/credential owner
- **AND** the child MUST be provider-credentialless with no native home reference
- **AND** missing gateway readiness MUST block launch

#### Scenario: Native credential-home binding
- **WHEN** the pinned binding is `native_credential_home`
- **THEN** R3 MUST resolve exactly one approved exclusive opaque home for one existing logical
  runtime/agent before launch
- **AND** the daemon MUST supply only daemon-local isolated-home references required by the CLI
- **AND** OmniRoute MUST NOT be probed, contacted, or used
- **AND** global HOME, raw path, account identity, and credential data MUST NOT enter product
  APIs, events, logs, or evidence

#### Scenario: Binding ambiguity or failure
- **WHEN** the binding is missing, ambiguous, stale, unauthorized, unhealthy, unsupported, or
  conflicts with an active assignment
- **THEN** admission MUST fail closed
- **AND** the system MUST NOT fallback, translate, rotate, or retry between bindings or homes

### Requirement: REQ-02 Owner-global versioned Runtime Standards

The platform MUST provide owner-global Runtime Standards with immutable versions, validation,
activation, audit and rollback.

#### Scenario: Activate a standard version
- **WHEN** an owner activates a validated version using the expected active version
- **THEN** activation MUST be compare-and-swap and auditable
- **AND** existing versions MUST remain immutable
- **AND** rollback MUST activate a prior version without rewriting history

### Requirement: REQ-03 Accountless reusable Runtime Sessions

Runtime Sessions MUST be owner-global, reusable and accountless.

#### Scenario: Reuse a session in another authorized workspace
- **WHEN** an owner enrolls a session into an authorized workspace
- **THEN** the platform MUST project it onto an existing runtime row, agent and daemon
- **AND** MUST NOT create or recreate an agent, runtime, daemon, container or credential home
- **AND** subscription/home attachment MUST remain a separate operation

### Requirement: REQ-04 Existing-row workspace binding

Each active workspace binding MUST identify exactly one existing agent, runtime row and Runtime
Session, and MUST preserve existing ORQ2-dev reservations.

#### Scenario: Enrollment references a missing or reserved row
- **WHEN** enrollment references a nonexistent row, another workspace, or protected ORQ2-dev binding
- **THEN** enrollment MUST fail before mutation
- **AND** MUST NOT move, share, borrow or steal any existing assignment

### Requirement: REQ-05 Opaque dynamic controlled-root registry

The daemon MUST discover arbitrary child homes under validated controlled roots without slot
allowlists or product-visible raw paths.

#### Scenario: New valid child appears beyond prior capacity
- **WHEN** a full reconciliation sees a valid arbitrary child not present in the prior generation
- **THEN** it MUST issue a new opaque home reference and publish a higher atomic generation
- **AND** no slot number grammar or fixed capacity allowlist may reject it
- **AND** APIs, events, logs and evidence MUST NOT expose the raw path

### Requirement: REQ-06 Reconciliation completeness and monotonicity

Filesystem notifications MUST be hints; startup, overflow, hint loss, watcher restart and
interval expiry MUST trigger a complete scan.

#### Scenario: Watcher overflows
- **WHEN** the filesystem watcher reports overflow
- **THEN** admission MUST use the last complete generation or fail closed
- **AND** a full scan MUST publish only after completion
- **AND** catalog generation MUST increase monotonically and never be reused

### Requirement: REQ-07 Catalog lifecycle safety

Catalog lifecycle MUST implement filesystem-identity deduplication, TTL, quarantine, active
reference protection, watermarks, tombstones and retention.

#### Scenario: Home disappears while referenced
- **WHEN** a home is absent from a full scan but an active task references its generation
- **THEN** it MUST become missing/draining and remain tombstoned
- **AND** it MUST NOT be reassigned, retired or have its opaque reference reused
- **AND** source data MUST NOT be copied, moved, deleted, truncated, sanitized, overwritten,
  chmodded, or have ownership changed
- **AND** cleanup MUST be limited to task-local non-source material after active-reference checks

#### Scenario: Duplicate or invalid identity
- **WHEN** aliases, symlink escape, wrong ownership/mode, invalid layout or duplicate filesystem
  identity is detected
- **THEN** every ambiguous candidate MUST be quarantined metadata-only
- **AND** new admission MUST fail closed

### Requirement: REQ-08 Exclusive account-home assignment

A healthy native account-home MUST be exclusive to one active persistent agent/runtime binding,
and one binding MUST have at most one active home assignment.

#### Scenario: Concurrent assignment
- **WHEN** two bindings race to assign the same opaque home reference
- **THEN** exactly one compare-and-swap MUST succeed
- **AND** the loser MUST receive `exclusive_assignment_conflict`

#### Scenario: Attach a new subscription
- **WHEN** a legitimate newly enrolled home becomes healthy and unassigned
- **THEN** it MAY attach to the existing binding
- **AND** agent, runtime, session and daemon IDs MUST remain unchanged

### Requirement: REQ-09 Configuration completeness

Every versioned runtime configuration MUST cover `transport_binding`, provider and opaque
subscription reference, literal model,
reasoning, context/input/output/total token limits, concurrency, timeout/retry, CLI flags,
environment allowlist, skills, tools/MCP, filesystem/network permissions, eligibility, health
and fallback.

#### Scenario: Unknown or missing field contract
- **WHEN** a configuration contains an unregistered field or a required group lacks a schema
- **THEN** validation MUST fail with `invalid_configuration`
- **AND** the field MUST NOT be ignored or passed to a runtime

### Requirement: REQ-10 Deterministic precedence

Effective configuration MUST resolve each field using
`platform > standard > runtime > explicitly delegable task`.

#### Scenario: Lower layer widens policy
- **WHEN** runtime or task configuration attempts to widen a platform/standard limit, permission,
eligibility, fallback or exposure boundary
- **THEN** resolution MUST reject the version or task before launch
- **AND** omission MUST inherit rather than erase higher policy

### Requirement: REQ-11 Capability, delegability and apply class

Every configurable leaf MUST declare type, capability predicate, redaction, delegability and
`hot` or `restart` apply class.

#### Scenario: Unsupported reasoning or model
- **WHEN** a model/reasoning/limit combination is absent from the pinned capability catalog
- **THEN** activation MUST fail with a field-addressed `capability_unsupported`
- **AND** validation MUST NOT make an inference request

#### Scenario: Non-delegable task override
- **WHEN** a task overrides a non-delegable field
- **THEN** claim or launch MUST fail before process creation

#### Scenario: Restart-class activation
- **WHEN** a restart-required version is activated
- **THEN** it MUST remain pending until the existing runtime process restarts and acknowledges it
- **AND** running tasks MUST retain their pinned prior version

### Requirement: REQ-12 Activation, audit, redaction, drift and rollback

Configuration activation and rollback MUST be compare-and-swap, redacted and auditable; daemon
application MUST be acknowledged and continuously compared for drift.

#### Scenario: Digest drift
- **WHEN** daemon-applied version/digest differs from desired state
- **THEN** `runtime_configuration.drift_detected` MUST be emitted
- **AND** new claims on that binding MUST fail closed until reconciled
- **AND** audit/events MUST contain no secret, raw path, argv/env value or provider response

### Requirement: REQ-13 Immutable task snapshots

Atomic claim MUST pin session/runtime/agent/workspace IDs, standard and runtime configuration
version IDs, effective digest, binding ID/generation, capability digest, and for native tasks the
opaque home reference/catalog generation.

#### Scenario: Configuration or assignment changes after claim
- **WHEN** active configuration or home assignment changes after a task is claimed
- **THEN** that attempt and its subagents MUST retain the original snapshot
- **AND** reclaim/reuse MUST NOT perform a live lookup that rewrites history

### Requirement: REQ-14 Subagent inheritance

Subagents MUST inherit their parent's task snapshot and MAY receive only explicitly delegable,
bounded task overrides.

#### Scenario: Spawn subagent
- **WHEN** a task spawns a subagent
- **THEN** no persistent runtime/session/account-home row MUST be created
- **AND** it MUST share the parent binding/home and count against configured concurrency

### Requirement: REQ-15 Concurrency independent of accounts

Session, task and subagent concurrency MUST be explicit configuration and MUST NOT derive from
the count of catalog homes.

#### Scenario: Concurrency exceeds home inventory
- **WHEN** configured native concurrency cannot be served without sharing an exclusive home
- **THEN** excess work MUST queue or fail with `capacity_exhausted`
- **AND** no home or ORQ2-dev assignment may be shared

### Requirement: REQ-16 Controlled fallback

Fallback MUST be ordered, bounded, capability-validated, health-gated and frozen in the task's
effective configuration. Every fallback route MUST preserve the pinned transport binding and,
for native execution, the pinned exclusive `home_ref`.

#### Scenario: Preferred route is unhealthy
- **WHEN** the preferred route is unhealthy
- **THEN** only a prevalidated eligible route within the same pinned binding MAY run
- **AND** fallback, translation, rotation, or retry between `omniroute` and
  `native_credential_home`, or between native homes, MUST be forbidden
- **AND** global HOME, cross-workspace, stale-generation, and ORQ2-dev fallback MUST be forbidden

### Requirement: REQ-17 Frozen migration identities

Implementation MUST reserve migrations 130-134 as named in the design, subject to an immediate
registrar recheck.

#### Scenario: Registrar detects collision
- **WHEN** any identity 130-134 is occupied before implementation
- **THEN** schema work MUST stop
- **AND** K1 MUST amend the canonical contract before any replacement name is used

### Requirement: REQ-18 Frozen REST contracts

Implementations MUST use the methods, routes, body fields, pagination, idempotency, response
status and error envelope frozen in the design.

#### Scenario: Administrative mutation
- **WHEN** a Runtime Standard, session, enrollment, binding, assignment or configuration is mutated
- **THEN** an authenticated authorized human owner/admin and `Idempotency-Key` MUST be required
- **AND** expected version/generation MUST be checked where specified
- **AND** task actors and cross-workspace actors MUST receive `forbidden`

#### Scenario: Error response
- **WHEN** a request fails
- **THEN** it MUST return `{error:{code,message,field?,request_id,retryable}}`
- **AND** message/field data MUST NOT reveal a path, account identity, credential, env value or
  provider response body

### Requirement: REQ-19 Frozen event contracts

Runtime-manager events MUST use the frozen envelope and type names, at-least-once delivery and
per-resource monotonic generations.

#### Scenario: Duplicate or gap
- **WHEN** a consumer receives a duplicate event
- **THEN** it MUST deduplicate by `event_id`
- **WHEN** it observes a generation gap
- **THEN** it MUST GET/reconcile complete state rather than infer missing changes

### Requirement: REQ-20 Authorization and path secrecy

Owner-global administration MUST be owner-only; workspace administrators MUST be confined to
their workspace; daemons MUST report only authorized daemon/binding state.

#### Scenario: Read catalog or evidence
- **WHEN** an authorized actor reads catalog, event, audit or task evidence
- **THEN** only opaque refs, states, versions, generations, digests, reason codes and counters MAY
  be returned
- **AND** raw paths, filesystem identity, account identity, credential names/data, prompts and
  provider response bodies MUST be absent

### Requirement: REQ-21 Health, revoke and retention

Eligibility MUST require fresh health and catalog data. Revocation MUST drain by default; hard
revoke requires explicit policy and audit. Audit/task snapshots and catalog tombstones MUST be
retained through all active references and evidence windows.

#### Scenario: Stale health or active reference
- **WHEN** health TTL expires or retirement is requested for a referenced home
- **THEN** new claims MUST fail with `health_stale` or `active_reference`
- **AND** every source credential home MUST remain untouched regardless of TTL or retention age
- **AND** only task-local non-source material MAY be cleaned after active-reference checks

### Requirement: REQ-22 Existing ORQ2 topology and release boundary

The solution MUST reuse the existing ORQ2 daemon and existing rows, preserve all ORQ2-dev
bindings, and remain fail closed until approved implementation and assurance gates pass.

#### Scenario: Documentation completion
- **WHEN** SPE-5 documentation validates
- **THEN** it MUST NOT imply executable implementation, production authorization or SharePoint
  authorization
- **AND** K3 review, integrated gates, Council acceptance and separate rollout authorization MUST
  remain mandatory

### Requirement: REQ-23 Repository/OpenSpec synchronization gate

Before any future handoff or deployment, every affected OpenSpec MUST pass strict validation,
the cross-authority residual scan MUST pass, and the Git index/worktree MUST have zero staged,
modified, or untracked owned files. The exact local commit MUST be published to its configured
non-main remote branch; local HEAD and upstream SHA MUST match and ahead/behind MUST be `0/0`.
Deployment evidence MUST pin that exact SHA. This requirement documents a future gate and does
not itself authorize publishing or deployment.

#### Scenario: Pending file or repository divergence
- **WHEN** an owned file is staged, modified, or untracked, an affected OpenSpec or authority
  scan fails, the exact commit is unpublished, upstream is absent or points to main, local and
  upstream SHAs differ, or ahead/behind is not `0/0`
- **THEN** handoff and deployment MUST fail closed
- **AND** no deployment evidence may claim a different or unresolved SHA

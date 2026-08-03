# Proposal - Runtime Manager and Credential Account-Home Restoration

## Authority amendment

This canonical change freezes the owner-approved Multica native credential-home authority.
Every launch uses `native_credential_home`; OmniRouter/OmniRoute is not a current solution,
fallback, credential authority, dependency, or acceptance source. R3 resolves one approved
exclusive opaque home for one existing logical runtime/agent before launch, and the daemon
supplies only daemon-local isolated-home references required by the native CLI. Missing or
invalid native authority fails closed. It never translates, rotates, or falls back to a router,
another home, global HOME, another workspace, or an ORQ2-dev reservation, and it exposes no raw
path, account identity, or credential data through product APIs, events, logs, or evidence.

## Why

The current slot allowlist and raw-path design cannot safely support reusable owner-global
sessions, per-runtime configuration, dynamic capacity, or durable task attribution. The
platform needs one frozen contract before schema, API, daemon, UI, and assurance streams can
work independently without leaking paths or changing existing infrastructure.

## What changes

- Add owner-global, immutable-versioned **Runtime Standards**.
- Add owner-global, reusable, accountless **Runtime Sessions**. A session is logical
  runtime/configuration identity, not a provider account, credential home, process, daemon,
  container, workspace, project, squad, or agent.
- Bind a workspace to existing runtime rows and the existing
  `orq2-credential-runtime-v1` daemon; enrollment or subscription attachment never recreates
  the agent, runtime row, or daemon.
- Replace slot allowlists and raw paths with an opaque, dynamic controlled-root catalog.
- Enforce one persistent agent, one runtime row, and one exclusive account-home binding;
  subagents inherit and do not create persistent runtime rows.
- Version each runtime configuration and resolve fields with strict precedence:
  `platform > standard > runtime > explicitly delegable task`.
- Cover provider/subscription selection, model, reasoning, context/output/token limits,
  concurrency, timeout/retry, CLI flags, environment allowlist, skills, tools/MCP,
  filesystem/network permissions, eligibility, health, and fallback.
- Validate provider capabilities before activation; classify every field as delegable or not
  and as hot-apply or restart-required.
- Add activation, audit, redaction, drift, rollback, immutable task generation/version/digest
  snapshots, lifecycle reconciliation, and retention contracts.
- Freeze REST, event, authorization, error, and path-secrecy contracts plus migration names
  130-134, subject to the mandatory registrar recheck immediately before implementation.

## Additive T2 and stable-allocator amendment

Runtime Manager REQ-01..23 retain their numbering and unaffected contracts; ORQ-114 narrows their
transport and cleanup clauses to current owner authority. Later ORQ2 requirements are incorporated
as REQ-24..35 under accepted-REQ-05 precedence: provider-specific layout after dynamic opaque discovery, fail-closed path validation, Codex AccountHome, equal
Prepare/Reuse coverage, stable agent/subscription identity, mandatory AGY/Codex/Kiro families, T2
topology and durability, reasoning-wire preservation, immutable producer-account pricing snapshots,
one admissible home per active binding (never one global slot), guarded 24-hour cleanup convergence,
and durable metadata tombstones. REQ-28 is future source-v2 rollout and REQ-30 is a non-authorizing
owner-gated cutover condition. `native_credential_home` is the only current solution; these additions
do not authorize router fallback or source-home mutation outside the gated cleanup state machine.

## Invariants

1. One account-home is exclusively bound to at most one active agent/runtime binding.
2. Runtime/session concurrency is policy-driven and independent of physical account count.
3. No process copies, moves, truncates, sanitizes, overwrites, chmods, or exposes a source
   credential home or credential artifact. Active, draining, referenced, reserved, admissible,
   unproven, or quarantined homes are preserved. A whole source home may be deleted only after
   authoritative `INACTIVE_PROVEN`, age greater than 24 hours, independently accepted ORQ-96
   database/admission/fencing/no-bypass gates, and separate destructive cutover authorization.
   Age is necessary and never sufficient.
4. Discovery uses hints plus periodic full reconciliation, supports overflow, deduplicates by
   filesystem identity, and publishes monotonic generations.
5. Missing, stale, ambiguous, unhealthy, unsupported, unauthorized, or conflicting state
   fails closed; it never falls back to a global HOME, static slot list, another workspace, or
   an ORQ2-dev reservation.
6. Running tasks pin their runtime/session, configuration version and digest, binding/catalog
   generation and opaque account-home reference.

## Scope

Canonical documentation and contracts for AGY/Antigravity, Codex, and Kiro on the existing
ORQ2 daemon and existing Multica rows. The contracts are provider-extensible, but no other
provider becomes eligible merely by being discovered.

## Non-goals

- No production deployment, restart, credential inspection, credential-home mutation, push or
  publication is authorized by this contract reconciliation.
- No OmniRouter/OmniRoute or global HOME fallback, and no physical-home deletion by catalog
  metadata lifecycle alone. The separately gated cleanup executor is not authorized here.
- No claim that installed legacy registry v1 already enforces source v2 stable identity.

## Current local release boundary

The historical local RC evidence covers migrations 128-135, Runtime Manager API/UI, PostgreSQL
behavior, N0v2 task-usage pricing, registry/catalog behavior, and allocator-v2 source. It predates
this authority correction and is not validation of these amended bytes. ORQ-107, ORQ-108, and
ORQ-109 remain pending independent acceptance and do not authorize cleanup. Installed registry v1
and owner-confirmed live homes are outside this documentation-only change. Push, deployment,
restart, and destructive cutover each require their own current authorization.

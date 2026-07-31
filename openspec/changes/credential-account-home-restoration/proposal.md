# Proposal - Runtime Manager and Credential Account-Home Restoration

## Authority amendment

This canonical change and the reconciled `build-omniroute-agent-brain` change jointly freeze
one transport authority. Every launch pins exactly one binding: `omniroute` or
`native_credential_home`. With `omniroute`, OmniRoute is the sole inference router and the sole
account/credential owner, and native homes are forbidden. With `native_credential_home`, R3
resolves one approved exclusive opaque home for one existing logical runtime/agent before
launch, the daemon supplies only daemon-local isolated-home references required by the native
CLI, and OmniRoute is not used. Neither binding may fall back, translate, or rotate to the other.
Both fail closed, preserve existing ORQ2-dev reservations, and expose no global HOME, raw path,
account identity, or credential data through product APIs, events, logs, or evidence.

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

## Invariants

1. One account-home is exclusively bound to at most one active agent/runtime binding.
2. Runtime/session concurrency is policy-driven and independent of physical account count.
3. No process copies, moves, deletes, truncates, sanitizes, overwrites, chmods, or exposes a
   source credential home or credential artifact. Cleanup is limited to task-local non-source
   material after active-reference checks.
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

- Executable code, schema, migrations, APIs, authentication, daemon/runtime/container changes,
  credential provisioning, production rollout, or SharePoint work.
- Creating or recreating infrastructure or changing ORQ2-dev assignments.
- Reading credential contents or making raw paths/account identities product-visible.
- Making model inference health requests as a control-plane check.

## Impact and release boundary

Downstream SPE-6/SPE-8/SPE-9/SPE-10 implementations and SPE-18/SPE-19/SPE-20/SPE-21/SPE-22
assurance must conform to this freeze. Migration identities are reserved, not implemented.
Production remains NO-GO until independent K3 acceptance, integrated gates, Council acceptance,
and separate written rollout authorization. SharePoint SPE-11 through SPE-17 remains prohibited.

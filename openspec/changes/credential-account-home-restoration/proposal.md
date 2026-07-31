# Proposal - Runtime Manager and Credential Account-Home Restoration

## Authority amendment

This canonical change is the authority for owner-global Runtime Standards, Runtime Sessions,
workspace runtime bindings, native credential account-home isolation, and task snapshots. For
those subjects it supersedes incompatible active text that assigns all provider-account or
credential ownership exclusively to OmniRoute. That other change is not edited here.
OmniRoute remains the sole model router for gateway-routed inference; native approved coding
CLI execution on ORQ2 may use only the opaque, exclusive account-home contract defined here.
All conflicts fail closed and preserve the existing ORQ2-dev reservations.

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
3. No process copies, moves, deletes, truncates, sanitizes, overwrites, or exposes a source
   credential home or credential artifact.
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

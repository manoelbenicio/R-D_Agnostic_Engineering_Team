## Context

The approved architecture has two distinct Rust executables:

- the pinned upstream Prodex `0.246.0`, which owns profile routing, the gateway, Smart Context, affinity, and pre-commit behavior;
- the Multica `prodex-sidecar` adapter, which exposes the local `rpp.l2.v1` control contract and launches the pinned Prodex gateway.

The current daemon configuration conflates those executables. `MULTICA_PRODEX_PATH` is used both as the agent runtime and as the process expected to expose `rpp.l2.v1`, while `normalizeL2SidecarArgs` rejects the actual adapter path. Tests and historical evidence started the adapter manually, so they did not prove restart continuity.

The host already has a secure ext4 configuration file at `/home/dataops-lab/runtime/prodex.env`, but no daemon service or launcher imports it. The active application `.env` was regenerated without Prodex/L2 keys. Prodex state is therefore empty even though isolated Multica credential slots exist.

## Goals / Non-Goals

**Goals:**

- Preserve Prodex and L2 activation across daemon and host restarts.
- Model the Prodex binary and the `rpp.l2.v1` adapter as separate executables.
- Project explicitly approved Codex credential slots into Prodex using `prodex profile add --codex-home`, which registers a reference and does not copy credentials.
- Fail closed on unsafe filesystem, permissions, duplicate credential identity, missing profile, adapter failure, or state-backend failure.
- Make effective runtime authority and profile reconciliation visible to operators.
- Keep the implementation compatible with the existing `rpp.l2.v1` client and single-router invariant.

**Non-Goals:**

- Re-authenticate OAuth accounts or rotate API keys.
- Copy `auth.json`, provider databases, cookies, or token stores into `PRODEX_HOME`.
- Force non-Codex providers into the Codex profile format. Their isolated Multica slots remain authoritative until their native Prodex enrollment path is explicitly supported.
- Upgrade the pinned Prodex version in this change.
- Enable Caveman, unsafe child environment forwarding, auto-redeem, or live Smart Context by default.

## Decisions

### Separate executable configuration

Add `MULTICA_L2_SIDECAR_PATH` for the Multica adapter. `MULTICA_PRODEX_PATH` continues to identify only the pinned upstream Prodex binary. The Go daemon launches the adapter, and the adapter inherits `MULTICA_PRODEX_PATH` to launch the real Prodex gateway.

This replaces the current ambiguous `MULTICA_L2_SIDECAR_ARGS` executable normalization. Arguments remain arguments; executable identity is configured and validated independently.

Alternative considered: teach upstream Prodex `app-server-broker` to expose `rpp.l2.v1`. Rejected because the pinned command only reports an experimental JSON-RPC capability and does not provide the required HTTP control surface.

### Durable environment source

Use one mode-0600 EnvironmentFile on a POSIX filesystem as the operational source for Prodex/L2 settings. The service/launcher imports it before constructing daemon configuration. The file contains executable paths, versions, loopback endpoints, state roots, and secret references; secret values remain protected and are never logged.

The daemon records the configuration source and fails startup when `MULTICA_PRODEX_REQUIRED=1` but Prodex or L2 resolves disabled.

Alternative considered: rely on shell exports or the application `.env`. Rejected because both have already been lost or regenerated during restart.

### Reference-only profile reconciliation

The current Multica `accounts` rows for the configured tenant are the sole credential inventory. Each validated Codex account maps a stable Prodex profile name to its isolated slot `CODEX_HOME`. Reconciliation invokes the official reference-only form:

`prodex profile add <profile> --codex-home <slot>/codex`

Existing matching registrations are idempotent. A name pointing to another home is an error. Homes not referenced by the current account inventory are ignored and purged as legacy credential sources.

Before registration, reconciliation verifies:

- root and credential homes are on an approved POSIX filesystem;
- directories are mode 0700 and credential files are mode 0600;
- the resolved home stays under the approved slot root;
- every selected slot has a valid Codex auth store;
- credential identities are unique across selected profiles;
- no profile falls back to a global or another slot's credential home.

Alternative considered: copy each auth store into `$PRODEX_HOME/profiles`. Rejected because copies drift after refresh and recreate the credential-clobber problem.

### Legacy credential purge

The newly validated Multica `accounts` inventory is the only credential authority. Credential files outside homes referenced by that inventory are obsolete and are removed rather than retained as fallback candidates. Purging is scoped to provider credential material; agent skills, sessions, prompts, caches, and workspaces are not deleted.

For Codex slots, only homes referenced by the current Codex account inventory retain `auth.json`. Unreferenced slot-local `auth.json` files are removed. Obsolete account rows are removed only after confirming that no current assignment references them.

### Provider boundary

The Multica credential-slot registry remains the universal source of isolation. Codex slots are projected into native Prodex profiles now. Other providers are passed to L2 as opaque approved profile references only when the adapter/provider capability supports them; otherwise they remain native isolated runtimes and fail closed instead of being coerced into a Codex profile.

### Restart and readiness contract

Startup order is:

1. load the durable environment;
2. validate the pinned Prodex binary and adapter binary;
3. reconcile approved profiles;
4. launch the adapter;
5. require authenticated `/healthz` and `/readyz`;
6. apply policy and register profile references;
7. mark the daemon ready with `runtime_router_owner=rust_l2`.

Any failure prevents new L2-owned sessions. Health output exposes only counts, paths reduced to configuration-source labels, version/commit, and opaque profile names.

## Risks / Trade-offs

- [Existing duplicate slot credentials] → Reconciliation rejects duplicates among approved profiles and reports slot names only; it never prints credential material.
- [Pinned Prodex and adapter versions drift] → Validate each executable independently and record hashes in deployment evidence.
- [Adapter becomes a second process to supervise] → Go owns lifecycle, readiness, restart backoff, and shutdown for the adapter; the adapter owns its Prodex gateway child.
- [EnvironmentFile contains sensitive settings] → Keep it on ext4, owned by the daemon user, mode 0600, and redact secret-bearing values from diagnostics.
- [Non-Codex provider enrollment differs] → Preserve provider-native slot isolation and add provider-specific projection only behind declared capability support.
- [Fallback to raw runtime hides regression] → `MULTICA_PRODEX_REQUIRED=1` converts the silent downgrade into an explicit startup failure.

## Migration Plan

1. Build and attest the tracked `prodex-sidecar` adapter.
2. Extend the existing secure `prodex.env` with separate adapter and L2 settings.
3. Select only the current validated Codex account rows for the configured tenant.
4. Purge unreferenced legacy credential material and obsolete unassigned account records.
5. Run reconciliation in audit mode, then register reference-only profiles.
6. Install/reload the persistent daemon service or launcher.
7. Restart and verify Prodex profile count, adapter readiness, Postgres readiness, and `rust_l2` ownership.
8. Keep the existing rollback command available; rollback disables L2 explicitly and returns to raw Codex only as a declared operator action.

## Open Questions

None for the Codex reference-only migration. Provider-native projection for Cline, Kiro, Antigravity, Gemini, and NVIDIA remains capability-gated follow-up work.

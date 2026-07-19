> **DISPOSITION (Wave A, 2026-07-19, owner decision D-V3-16): DEFERRED — COLD-RECOVERY-ONLY.**
> This change is re-scoped from "Prodex required and restart-durable" to a **default-OFF,
> mutually-exclusive, operator-gated cold platform recovery mode** in the final Kanban lane.
> Prodex is never a per-request router, never an automatic fallback, and never simultaneously
> hot with OmniRoute. `MULTICA_PRODEX_REQUIRED` becomes an explicit operator-only recovery
> toggle that defaults to `0` (OFF); it is NOT a fail-closed startup requirement in the target
> path. This supersedes the PROGRAM HOLD and resolves the reconciliation audit
> (`persist-prodex-vs-omniroute-reconciliation-audit.md`) as an Option-C variant. Existing
> checkbox states are preserved (0/16); no product code is changed by this Wave A edit.

## Why

The Multica daemon can restart without reloading the already-provisioned Prodex configuration, silently falling back to raw runtime execution while the isolated agent credential slots remain invisible to Prodex. This breaks the approved Go L4 / Rust L2 architecture after a routine restart and forces operators to reconstruct state manually.

## What Changes

- Add a durable, single-source startup configuration for Prodex and the `rpp.l2.v1` adapter that survives daemon and host restarts **when recovery mode is explicitly enabled by an operator**; the default posture is OFF and OmniRoute-primary.
- When (and only when) an operator explicitly enables cold recovery mode, make startup fail closed if the required Prodex/L2 binary, adapter, state backend, or configuration is unavailable. `MULTICA_PRODEX_REQUIRED` defaults to `0` and is an operator recovery toggle, not a target-path startup requirement.
- Reconcile the newly validated Multica `accounts` inventory into the Prodex profile registry by reference, without copying or sharing credential files.
- Purge credential material and account records outside the newly validated Multica inventory so obsolete tokens cannot be selected again.
- Enforce one agent slot to one credential home and reject unsafe permissions, non-POSIX filesystems, duplicate credential identities, missing profiles, and cross-slot fallback.
- Expose runtime authority and reconciliation health so operators can verify `rust_l2`, profile count, configuration source, and readiness after restart.
- Add restart, profile-isolation, and Go↔Rust integration tests.

## Capabilities

### New Capabilities

- `prodex-runtime-continuity`: Durable Prodex/L2 startup, fail-closed readiness, and isolated Multica slot-to-Prodex profile reconciliation across restarts.

### Modified Capabilities

None.

## Impact

- Multica daemon configuration and startup lifecycle under `multica-auth-work/server/internal/daemon`.
- Prodex profile enrollment/reconciliation and the local `prodex-sidecar` adapter.
- Runtime environment/configuration under the approved Linux filesystem.
- Health/readiness and operational documentation.
- No raw OAuth token, API key, cookie, or `auth.json` content crosses the Go↔Rust control contract.

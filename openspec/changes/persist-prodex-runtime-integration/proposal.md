## Why

The Multica daemon can restart without reloading the already-provisioned Prodex configuration, silently falling back to raw runtime execution while the isolated agent credential slots remain invisible to Prodex. This breaks the approved Go L4 / Rust L2 architecture after a routine restart and forces operators to reconstruct state manually.

## What Changes

- Add a durable, single-source startup configuration for Prodex and the `rpp.l2.v1` adapter that survives daemon and host restarts.
- Make startup fail closed when Prodex/L2 is required but its binary, adapter, state backend, or configuration is unavailable.
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

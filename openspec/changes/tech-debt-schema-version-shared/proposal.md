## Why

During v1 implementation, `SCHEMA_VERSION` was placed in `src/canvas-document/schema.ts` instead of `src/shared/` as prescribed by design decision D9 (§14.2 of the master spec). This was accepted during implementation to avoid import cycles between `canvas-document` and `shared`, but it violates the directory ownership model where cross-capability constants live in `src/shared/`.

## What Changes

Move the `SCHEMA_VERSION` constant (and associated `CURRENT_SCHEMA_VERSION`) from `src/canvas-document/schema.ts` to `src/shared/constants.ts` (or a new `src/shared/schema-version.ts`). Update all import sites across `canvas-document`, `canvas-builder`, and `canvas-reconciler`.

### Scope

- Move: `SCHEMA_VERSION` constant → `src/shared/`
- Update: all import paths in `canvas-document/`, `canvas-builder/`, `canvas-reconciler/`
- Verify: no circular dependencies introduced

## Why Post-v1

This is a low-risk, mechanical refactor. During v1, the team accepted the D9 violation to avoid blocking parallel development. The constant is only read in two places (schema migration and reconciler version check), so the blast radius is minimal. Moving it now would touch locked modules during the v1 close sprint, introducing unnecessary coordination risk.

## Impact

- **Code**: 3-5 files modified (move + update imports)
- **Risk**: Minimal — purely a namespace reorganization
- **Dependencies**: None

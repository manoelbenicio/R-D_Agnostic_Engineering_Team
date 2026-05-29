# tech-debt-schema-version-shared — Implementation Tasks

> Owner: **SUP** (cross-capability shared infra) with **CV** review (canvas-document is a locked module).
> Parallel-safe: only touches `src/shared/schema-version.ts` (new) and the four import sites listed below. No file overlap with the other tech-debt changes.

## 1. Move the constant (SUP)

- [x] 1.1 Create `src/shared/schema-version.ts` with:
  ```ts
  export const SCHEMA_VERSION = 1;

  export function isCompatible(doc: { schema_version: number }): boolean {
    return doc.schema_version <= SCHEMA_VERSION;
  }
  ```
- [x] 1.2 Delete the body of `src/canvas-document/schema-version.ts` and replace with a re-export:
  ```ts
  // Backwards-compatible re-export. New code should import from '@/shared/schema-version'.
  export { SCHEMA_VERSION, isCompatible } from '@/shared/schema-version';
  ```
  (Keeping the file as a thin re-export avoids touching every import site at once and gives downstream changes a deprecation window.)

## 2. Update import sites (SUP)

Each of the following files currently imports from `@/canvas-document/schema-version`. Update them to import from `@/shared/schema-version`:

- [x] 2.1 `src/canvas-document/store.ts` (3 references: `SCHEMA_VERSION`, `isCompatible`)
- [x] 2.2 `src/canvas-templates/templates.ts` (1 reference: `SCHEMA_VERSION`)
- [x] 2.3 `src/voice/voice-to-canvas.ts` (1 reference: `SCHEMA_VERSION`)
- [x] 2.4 Sweep with `grep -rn "@/canvas-document/schema-version" src/` and update any remaining hits

## 3. Verify (SUP)

- [x] 3.1 `npm run typecheck` clean
- [x] 3.2 `npm test -- src/canvas-document` and `npm test -- src/canvas-templates` and `npm test -- src/voice` all green
- [x] 3.3 No new circular-dependency warnings (run `madge --circular src/` if available; otherwise verify by checking that `src/shared/` does not import from any capability module)

## 4. Optional cleanup (defer if it widens blast radius)

- [ ] 4.1 If all import sites have been migrated cleanly, delete `src/canvas-document/schema-version.ts`. Skip this step if any external consumer (eslint rules, scripts) still references the old path; leave the re-export shim in place and reopen as a follow-up.

## Out of Scope

- Touching `CURRENT_SCHEMA_VERSION` in `src/shared/storage/migrations.ts` — that is a *separate* IDB-migration constant and is correctly placed
- Bumping `SCHEMA_VERSION` itself (still `1`)
- Refactoring the canvas-document Zod schema

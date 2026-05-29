# tech-debt-react-refresh-cleanups — Implementation Tasks

> Owners: **CV** (canvas-builder) + **DB** (finops). Two capabilities touched, but no file overlap with other tech-debt changes — fully parallel-safe.
> Risk: cosmetic dev-experience only. No production behavior change.

## 1. Canvas barrel re-exports (CV)

The current pattern of dual named + default re-exports in a tiny barrel file confuses the `react-refresh` plugin's heuristic for component boundaries.

Current state of `src/canvas-builder/CanvasBuilder.tsx`:
```ts
export { CanvasBuilderPage as CanvasBuilder } from './CanvasBuilderPage';
export { default } from './CanvasBuilderPage';
```

Same shape in `src/canvas-builder/CanvasList.tsx`.

Pick one of the two fixes (Fix A is preferred):

### Fix A — collapse the barrel by inlining at import sites (preferred)

- [x] 1.1 Update `src/canvas-builder/index.ts` so it re-exports both pages directly:
  ```ts
  export { CanvasBuilderPage as CanvasBuilder } from './CanvasBuilderPage';
  export { CanvasListPage as CanvasList } from './CanvasListPage';
  ```
- [x] 1.2 Update every import site that used `@/canvas-builder/CanvasBuilder` or `@/canvas-builder/CanvasList` to import from `@/canvas-builder` instead (typically `src/shell/router.tsx` and similar)
- [x] 1.3 Delete the now-unused `src/canvas-builder/CanvasBuilder.tsx` and `src/canvas-builder/CanvasList.tsx` barrel files

### Fix B — make the barrel a single named re-export (fallback if Fix A widens blast radius)

- [ ] 1.4 Replace the contents of `CanvasBuilder.tsx` and `CanvasList.tsx` with a single named export only (drop the `export { default }` line). Update import sites accordingly.

## 2. cost-warning component pattern (DB)

`src/finops/cost-warning.tsx` currently uses a `const` arrow function with `React.FC`, which the `react-refresh` plugin doesn't always recognize as a refresh boundary.

- [x] 2.1 Convert `CostWarning` to a function declaration:
  ```ts
  export function CostWarning({ showText = false, className = '', ...props }: CostWarningProps) {
    return (
      <span /* … */ />
    );
  }
  ```
- [x] 2.2 Drop the `React.FC` typing (use a plain props interface as above); keep the named + default export
- [x] 2.3 Verify `CostWarning.displayName` is unnecessary (a function declaration carries the name natively)

## 3. Verify (CV / DB)

- [x] 3.1 `npm run dev` boots cleanly with **zero** `react-refresh` warnings in the dev console
- [x] 3.2 Editing `CostWarning` while the dev server is running produces an HMR update (not a full reload)
- [x] 3.3 Editing `CanvasBuilderPage` while the dev server is running produces an HMR update
- [x] 3.4 `npm run lint` + `npm run typecheck` clean
- [x] 3.5 `npm test` green (no test relied on the deleted barrel paths)

## Out of Scope

- Any non-cosmetic refactor of `CanvasBuilderPage` or `FinopsPage`
- Touching the `eslint-plugin-react-refresh` config itself

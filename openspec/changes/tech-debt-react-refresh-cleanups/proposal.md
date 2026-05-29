## Why

The development build (`npm run dev`) emits `react-refresh` warnings from several components:

- `src/canvas-builder/CanvasBuilder.tsx` — barrel re-export triggers refresh boundary warning
- `src/canvas-builder/CanvasList.tsx` — barrel re-export triggers refresh boundary warning
- `src/finops/cost-warning.tsx` — component definition pattern not recognized by react-refresh plugin

These warnings are harmless (they don't affect production builds or functionality), but they pollute the dev console and can mask real warnings during development.

## What Changes

1. **Fix barrel re-exports** in `CanvasBuilder.tsx` and `CanvasList.tsx`:
   - Currently: `export { CanvasBuilderPage } from './CanvasBuilderPage';` (single-line re-exports)
   - Fix: Convert to proper barrel files that re-export via `export *` or remove the intermediary files and import directly from the page components.

2. **Fix `cost-warning.tsx`** component pattern:
   - Ensure the component is a named function declaration (not an arrow function assigned to a const), or add a `displayName` property.
   - Alternatively, restructure to satisfy the react-refresh plugin's heuristics.

3. **Verify no new warnings** appear in a clean `npm run dev` start.

### Scope

- Modify: `src/canvas-builder/CanvasBuilder.tsx`
- Modify: `src/canvas-builder/CanvasList.tsx`
- Modify: `src/finops/cost-warning.tsx`

## Why Post-v1

These are cosmetic dev-experience issues. They don't affect production builds, test suites, or end-user functionality. Fixing them during the close sprint risks touching locked modules (`canvas-builder` is locked) for zero functional benefit.

## Impact

- **Code**: 3 files modified (trivial changes)
- **Risk**: Minimal — purely cosmetic, no functional change
- **Dependencies**: Requires canvas-builder module to be unlocked

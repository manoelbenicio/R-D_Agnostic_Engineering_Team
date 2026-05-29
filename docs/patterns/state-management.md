# State Management (D3)

Two complementary stores per design D3:

- **Zustand** — UI / cross-component application state (current canvas,
  deploy progress, voice transcript, toasts, validated providers).
- **TanStack Query** — every CAO REST `GET` (caching, polling, invalidation).
- **`useState` / `useReducer`** — component-local form state.

## Owner-only stores

Each capability owns at most one Zustand store. Cross-capability reads
go through a typed selector exported from the owner. Cross-capability
writes go through a documented action exported from the owner.

```ts
// src/api/key-store/use-validated-providers.ts (IF owns this slice)
export const useValidatedProviders = () => useKeyStoreState((s) => s.validated);
```

Other capabilities import the selector — they do NOT subscribe to internal
slices directly.

## TanStack Query keys

Capability-owned namespacing:

```ts
['cao', 'health']
['cao', 'sessions']
['cao', 'session', name]
['cao', 'terminal', id]
['cao', 'terminal', id, 'inbox']
['cao', 'flows']
['cao', 'profiles']
['cao', 'providers']
```

Defined in `src/api/query-keys.ts`. New keys are added there, never inline.

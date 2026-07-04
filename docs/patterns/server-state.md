# Server-state fetching (D11, task 4.7)

Every `GET` against CAO flows through TanStack Query keyed in
`src/api/query-keys.ts`. Capability owners do NOT call `caoClient`
directly inside components — they use a `useXyz()` hook that wraps
`useQuery`.

## Polling cadences (master spec §9)

| Endpoint                                  | Poll  | Where                  |
| ----------------------------------------- | ----- | ---------------------- |
| `GET /health`                             | 10 s  | `useHealthStore` (4.8) |
| `GET /sessions`                           | 5 s   | Dashboard fleet KPI    |
| `GET /sessions/:name/terminals`           | 3 s   | Terminal tab bar       |
| `GET /flows`                              | 15 s  | Flows list             |

Polls SHALL pause on `document.hidden` and resume on visibility return —
TanStack Query's `refetchOnWindowFocus` plus `refetchInterval`-respecting
visibility provides this for free.

## Mutations

`useMutation` for every `POST/DELETE`. Successful mutations invalidate
the relevant query key. Optimistic updates are allowed (e.g., 19.5 enable
toggle) but MUST roll back on error.

## Errors

Every method on `CaoClient` throws `CaoApiError` or `CaoNetworkError`.
React Query consumers surface the message via the `error.message` of the
typed error.

## Fetch Boundary & Linting (Task 2.5)

1. **Direct fetches to CAO endpoints are strictly forbidden** in components or other modules. You MUST use the `CaoClient` defined in `src/api/cao-client.ts`. The custom ESLint rule `agentverse/no-direct-cao-fetch` enforces this at build time.
2. **Outbound calls to Firebase/AgentVerse platform endpoints** (which will require authentication tokens in later phases) MUST use the `appFetch` wrapper in `@/shell/app-fetch`. This serves as our single, auth-aware network boundary. Direct calls to platform routes via `fetch` are discouraged.


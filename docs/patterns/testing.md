# Testing Patterns (D10)

| Layer       | Tool                          | Owner       | Coverage  |
| ----------- | ----------------------------- | ----------- | --------- |
| Unit        | Vitest                        | each owner  | ≥70% logic |
| Component   | Vitest + RTL                  | each owner  | n/a        |
| Integration | Vitest + MSW                  | each + IF   | n/a        |
| E2E smoke   | Playwright                    | SUP         | critical path |
| Contract    | Vitest + `CAO_LIVE=1`         | IF          | response shape on every endpoint |
| A11y        | axe-core (CI script)          | SUP         | critical/serious zero |

## Patterns

- **Tests live next to code**: `src/<capability>/__tests__/<file>.test.tsx`.
- **No global mocks**: MSW handlers in `src/api/__tests__/msw/handlers.ts`
  are the source of truth. Tests use `server.use(...)` to override per-test.
- **Render via wrapper**: `renderWithProviders()` (in `src/__tests__/utils.tsx`)
  installs the QueryClient + Router providers.
- **Keep tests deterministic**: no `setTimeout` waits; use `findByText` or
  `waitFor`.

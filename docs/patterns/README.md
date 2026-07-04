# AgentVerse Established Patterns

Per design D8/D9 (parallel-from-day-zero, supervisor-gated `src/shared/`),
the supervisor reviews every PR. The first PR from each owner that
introduces a pattern that subsequent PRs will copy MUST also write that
pattern down here. This file is the index — individual pattern docs live
alongside.

## Index

| Pattern                              | Owner | Doc                              | Status      |
| ------------------------------------ | ----- | -------------------------------- | ----------- |
| State management (Zustand vs RQ)     | SUP   | `state-management.md`            | seeded      |
| Server-state fetching                | IF    | `server-state.md`                | seeded      |
| Component file layout                | SUP   | `component-layout.md`            | seeded      |
| Testing (unit / integration / e2e)   | SUP   | `testing.md`                     | seeded      |
| Error handling                       | IF    | `error-handling.md`              | seeded      |
| IndexedDB persistence                | SUP   | `indexeddb.md`                   | seeded      |

The first PRs from CV / TM / DB / ST / VX expand each entry above with
concrete examples drawn from their actual capability code.

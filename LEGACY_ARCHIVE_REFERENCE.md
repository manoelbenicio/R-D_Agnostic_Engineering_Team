# Legacy Archive Reference — GO Core Migration

**Created:** 2026-06-21
**Sprint:** 3 — Pre-Production | GO Core Migration

## Purpose
Original CAO files archived as `.old` (NOT deleted) for consultation during initial transition period.

## Archived Files

| Original Path | Archive Path | Replaced By |
|---------------|-------------|-------------|
| `src/api/base-url.ts` | `src/api/base-url.ts.old` | `src/api/go-core-base-url.ts` (port 8080) |
| `src/api/cao-client.ts` | `src/api/cao-client.ts.old` | `src/api/go-core-client.ts` (GoCoreClient) |
| `src/api/__tests__/cao-client.test.ts` | `src/api/__tests__/cao-client.test.ts.old` | `src/api/__tests__/go-core-client.test.ts` |
| `src/api/__tests__/contract/cao-contract.test.ts` | `src/api/__tests__/contract/cao-contract.test.ts.old` | `src/api/__tests__/contract/go-core-contract.test.ts` |

## Why Keep?
If GO Core shows unexpected crashes, team can compare implementations and cross-reference error handling.

## When to Delete
- [ ] GO Core running 2+ weeks without critical issues
- [ ] All contract tests pass against live GO Core
- [ ] Team consensus CAO rollback not needed

## Technical Impact
Files with `.old` are ignored by TypeScript, ESLint, Vite, and test runner. Zero bundle impact.
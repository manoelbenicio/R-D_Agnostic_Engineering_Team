# Component File Layout

Each capability is a directory under `src/<capability>/`:

```
src/<capability>/
├── components/        # React components, one file per component.
├── hooks/             # capability-owned hooks (use* functions)
├── lib/               # pure logic, no React (preferred unit-test target)
├── routes/            # React Router route components for /<route>
├── __tests__/         # vitest tests mirroring component/lib structure
└── index.ts           # public API: re-exports the things other capabilities consume
```

Cross-capability imports MUST go through `@/shared/` or `@/design-system/`.
The lint rule `agentverse/no-sideways-capability-imports` enforces this.

## Naming

- Components: `PascalCase.tsx`, default export.
- Hooks: `use-something.ts`, named export.
- Pure modules: `kebab-case.ts`.
- Tests: same path under `__tests__/`, suffix `.test.ts(x)`.

## Public API

A capability's `index.ts` re-exports only:

- types other capabilities consume,
- selectors / hooks designed for cross-capability use,
- the route component the shell mounts.

Internal modules SHALL NOT be re-exported.

## Design System Locked-Files Policy (Task 3.7)

After bootstrap, all files under `src/design-system/` are locked. 
Any Pull Request that modifies files in `src/design-system/` must carry the **`design-system-approved`** label. 
Pull Requests containing design system edits without this label will automatically fail the CI gate check triggered by `scripts/check-design-system-policy.sh`.


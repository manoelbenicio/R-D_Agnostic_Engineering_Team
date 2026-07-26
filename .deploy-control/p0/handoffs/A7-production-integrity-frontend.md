# A7 — Production Integrity Frontend/Mobile/Desktop Audit

agent: Agy-P0-A7
lane: A7
task: P0-PROD-INTEGRITY-FE
timestamp: 2026-07-21T22:45:00Z
status: DONE
verdict: **NO_REACHABLE_RESIDUAL**

## Preflight

| Tool     | Version              | Path/Source                                       |
|----------|----------------------|---------------------------------------------------|
| git      | 2.50.1               | system                                            |
| python3  | 3.9.25               | system                                            |
| rg       | 15.2.0               | system                                            |
| Node     | v22.23.1             | `/home/ec2-user/.nvm/versions/node/v22.23.1/bin/` |
| pnpm     | 10.28.2              | Corepack via Node 22.23.1                         |

Check-in: `.deploy-control/p0/checkins/Agy-P0-A7__P0-PROD-INTEGRITY-FE__20260721T224233Z.json`

## Scope

Audit all frontend source in `multica-auth-work/` for production-reachable:
- Mock data / fake responses / fake-success
- QA-only routes / dev-only routes
- Placeholder data returning synthetic success
- Demo persistence / synthetic defaults
- Hardcoded model IDs / sample task data

**Focus**: Kanban→terminal flow (squad → project → task creation → assignment → agent launch → result display).

**Exclusions per contract**: test fixtures (`*.test.ts`, `*.test.tsx`, `*.spec.*`, `__tests__/`, `__mocks__/`), Storybook (none found), guardrails, and isolated dev tooling behind build-time guards.

## Surfaces Audited

### 1. Web App (`apps/web/`)

**Route structure**: Next.js App Router with 25+ routes under `app/`.

| Search Pattern | Production Hits | Assessment |
|---|---|---|
| `mock`, `fake`, `stub`, `dummy` | 0 non-test | All in `*.test.tsx` files |
| `placeholder` | Input attrs only (`placeholder="Issue title"`) | Standard HTML form UX |
| `demo`, `sample data`, `seed data` | 0 | — |
| `QA_ONLY`, `qa-only`, `dev-only` | 0 routes | — |
| `TODO`, `FIXME`, `HACK` | Comments only | No behavioral impact |
| `hardcoded`, `hardcode` | Comments only | Design notes, not data |
| `__DEV__`, `NODE_ENV.*development` | 1 (react-grab inspector) | Gated by `process.env.NODE_ENV === "development"`, tree-shaken in production |
| `localStorage` with demo data | 0 | Auth token (`multica_token`) and UI dismiss flags only |
| `storybook`, `.stories.` | 0 | Not present in codebase |

**Kanban→terminal pipeline**:
- Issue creation: `createIssue()` → `POST /api/issues` (real HTTP)
- Assignment: `updateIssue()` → `PATCH /api/issues/{id}` (real HTTP)
- Agent launch: Backend-driven, status via WebSocket (`ws-client.ts`)
- Results: `execution-log-section.tsx` → real WS + `GET /api/tasks/{taskId}/logs`

**Verdict**: ✅ No reachable residual.

### 2. Mobile App (`apps/mobile/`)

**Route structure**: Expo Router v4 with 35+ file-based routes.

| Search Pattern | Production Hits | Assessment |
|---|---|---|
| `mock`, `fake` | 0 non-test | `hasFakeCaret` in OTP input is CSS animation, not data |
| `stub` | 2 (MoreStub, Pin optimistic) | **MoreStub**: Expo Router requirement, redirects to inbox on deep-link. **Pin stub**: Standard React Query optimistic update, replaced by server data on `onSuccess` |
| `placeholder` | Input attrs only | Standard React Native `placeholderTextColor` |
| `demo`, `sample data` | 0 | — |
| `QA_ONLY`, `dev-only`, `__DEV__` | 0 | — |
| `localStorage`, `AsyncStorage` | Auth tokens, dismiss flags | No demo data |

**Kanban→terminal pipeline**:
- Issue mutations: `useUpdateIssue()` → `api.updateIssue()` → real HTTP PATCH
- Task cancellation: `useCancelTask()` → `POST /api/tasks/{id}/cancel`
- Active/past tasks: `GET /api/issues/{id}/active-tasks`, `GET /api/issues/{id}/tasks`
- API defense: `parseWithFallback()` throws `ApiContractError` on schema mismatch — **never returns synthetic fallback data**

**Verdict**: ✅ No reachable residual.

### 3. Desktop App (`apps/desktop/`)

**Route structure**: Electron + Vite with 20+ routes in `routes.tsx`.

| Search Pattern | Production Hits | Assessment |
|---|---|---|
| `mock`, `fake`, `stub`, `dummy` | 0 non-test | — |
| `placeholder` | UI log panel collapse + input attrs | Log deduplication UX, not mock data |
| `hasLocalMachine` | 2 (agents-page, runtimes-page) | **Intentional UX**: Desktop passes `hasLocalMachine={true}` so users see Start/Stop daemon controls even before it registers. Not fake data — it's a capability flag for the desktop platform. Shared views render a local machine row only on desktop, correctly. |
| `dev-only` | 1 comment (react-grab) | Gated by `import.meta.env.DEV` |
| `is.dev` | Dev app name/userData path | `@electron-toolkit/utils` guard, false in packaged builds |

**Verdict**: ✅ No reachable residual.

### 4. Shared Packages (`packages/core`, `packages/ui`, `packages/views`)

| Search Pattern | Production Hits | Assessment |
|---|---|---|
| `EMPTY_LIST_AUTOPILOTS_RESPONSE` | Zod schema fallback param | **Safe**: `parseWithFallback()` ignores `_legacyFallback` param and throws `ApiContractError` on validation failure. Synthetic records are never returned. |
| `syntheticTask` | 1 (autopilot-detail-page.tsx) | **Safe**: Adapts real `AutopilotRun` fields (`task_id`, timestamps, failure_reason`) to `AgentTask` interface for `TranscriptButton`. Not fake data — type adapter from real server data. |
| `TEMPLATE_DEFAULTS` | 1 (step-agent.tsx) | **Safe**: Onboarding agent template defaults (name/emoji suggestions). Submitting calls real `api.createAgent()`. |
| `HELPER_INSTRUCTIONS` | 1 (helper-instructions.ts) | **Safe**: Localized system prompt for auto-created Multica Helper agent. Production content. |
| `DEFAULT_*` constants | 40+ (view stores) | **Safe**: UI sort direction, hidden columns, chat dimensions. Configuration defaults, not data. |
| `mock`, `fake`, `stub` | 0 non-test | — |

**API contract defense** (`packages/core/api/schema.ts:32–57`):
```ts
export function parseWithFallback<T>(
  data: unknown, schema: ZodType, _legacyFallback: T, opts: ParseOptions
): T {
  const result = schema.safeParse(data);
  if (result.success) return result.data as T;
  throw new ApiContractError(opts.endpoint, result.error.issues);
}
```
The `_legacyFallback` parameter is deliberately ignored — malformed data **always** throws, never silently returns empty/synthetic records.

**Verdict**: ✅ No reachable residual.

## Search Evidence Summary

Total distinct search patterns executed: 15+  
Total grep scans across 4 surfaces: ~60  
Methodology: `rg` with file-type includes, test/node_modules exclusions, case-insensitive, regex and literal modes.

Key search terms with zero production-reachable findings:
- `fake.*success|fakeSuccess|fake_success` → 0 results anywhere
- `QA_ONLY|qa-only|qaOnly` → 0 results
- `DEV_ONLY|devOnly` → 0 routes (comment mentions only)
- `storybook|\.stories\.` → 0 results in entire codebase
- `seedData|seed_data|sampleIssue|sampleTask` → 0 non-test results
- `lorem` → 0 results
- `__DEV__` in packages → 0 results

## Conclusion

### `NO_REACHABLE_RESIDUAL`

No mocks, fake-success handlers, QA-only routes, placeholder data returning synthetic success, demo persistence, or synthetic defaults are reachable in production across web, mobile, or desktop frontend paths relevant to the Kanban→terminal flow.

**Structural defenses in place**:
1. `parseWithFallback()` fails closed with `ApiContractError` — never returns synthetic fallback records
2. Dev tooling universally gated by `import.meta.env.DEV` / `process.env.NODE_ENV` / `is.dev`
3. All mock/fake/stub patterns isolated to `*.test.ts` / `*.test.tsx` files
4. No Storybook, no seed data, no demo modes exist in the codebase
5. Optimistic updates use standard React Query patterns with proper rollback

### Corrections Required: **NONE**

### W1 Handoff

No file changes needed. This audit is informational — the frontend production integrity is already clean. W1 may integrate this as evidence for P0 completion criteria item: "produção não expõe mocks, placeholders, fake-success, QA routes ou demo persistence nos caminhos alterados."

### Files Examined (non-exhaustive key paths)

- `apps/web/app/` — all 25+ routes
- `apps/mobile/app/` — all 35+ routes
- `apps/desktop/src/` — main, preload, renderer, routes
- `packages/core/api/client.ts` — API client (2000+ lines)
- `packages/core/api/schema.ts` — parseWithFallback defense
- `packages/core/api/schemas.ts` — Zod schemas and empty response constants
- `packages/views/issues/` — board-view, list-view, swimlane, gantt, issue-detail
- `packages/views/agents/` — agent overview, model picker
- `packages/views/autopilots/` — autopilot detail, syntheticTask adapter
- `packages/views/onboarding/` — templates, step-workspace, step-agent
- `packages/views/runtimes/` — runtime profiles, runtime detail
- `packages/views/layout/` — app sidebar, discord card

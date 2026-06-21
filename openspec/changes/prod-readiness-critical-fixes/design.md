# PROD Readiness: Critical Fixes & GO Core Migration — Design Document

## Overview

This change addresses two critical workstreams:
1. **Critical Fixes** — TypeScript build failures, ESLint plugin resolution, test reliability
2. **GO Core Migration** — Replace CAO backend references with new GO Core Server as the main brain

---

## Part 1: Critical Fixes (Unblock Build & Deploy)

### 1.1 TypeScript Type Definitions Fix

**Problem**: `CreateSessionInput` and `AddTerminalInput` types missing `provider` field, causing 6 TS errors.

**Files to Fix**:
- `src/api/types.ts` — Add `provider?: ProviderType` to `CreateSessionInput`
- `src/api/cao-client.ts` — Lines 59, 83 (already pass provider, will type-check after fix)
- `src/canvas-reconciler/reconciler.ts` — Lines 333, 406, 529, 602 (already pass provider)

**Solution**:
```typescript
// src/api/types.ts
export interface CreateSessionInput {
  profile: string;
  working_directory: string;
  provider?: ProviderType;  // ADD THIS
  env_vars?: Record<string, string>;
}
export type AddTerminalInput = CreateSessionInput;
```

### 1.2 ESLint Plugin Resolution Fix

**Problem**: `eslint-plugin-agentverse` fails to load due to UNC path resolution issues.

**Root Cause**: `package.json` references `file:./eslint-rules` but Node.js resolves UNC path `//21LAPGLMVPJ4/...` to incorrect Windows path.

**Solutions** (choose one):
- **Option A**: Publish plugin to local npm registry / GitHub Packages
- **Option B**: Use absolute file path in `package.json`: `"eslint-plugin-agentverse": "file:///M:/Automonous_Agentic/eslint-rules"`
- **Option C**: Copy plugin to `node_modules` via `prepare` script

**Recommended**: Option B — minimal change, works on Windows mapped drives.

### 1.3 Test Suite Reliability

**Problem**: Vitest times out (>4 min), many files show `0 test` during discovery.

**Suspected Causes**:
- UNC path + `fake-indexeddb` + `jsdom` performance
- MSW server startup overhead per test file
- Shared Zustand stores causing test pollution

**Solutions**:
- Run tests from mapped drive (M:) not UNC path
- Add `testTimeout: 60000` to `vitest.config.ts`
- Use `--pool=forks` for better isolation
- Add `setupFiles` to reset Zustand stores between tests

---

## Part 2: GO Core Migration (Replace CAO with GO Core Server)

### 2.1 Architecture Change

```
BEFORE (CAO):
+-------------+     HTTP/WS      +----------------------+
|   Frontend  | <--------------> |      CAO Server      |
|  (React)    |   :9889          | (Python/uvicorn)     |
+-------------+                  +----------------------+
                                     ^
                             bind-mount creds

AFTER (GO Core):
+-------------+     HTTP/WS      +----------------------+
|   Frontend  | <--------------> |   GO Core Server     |
|  (React)    |   :PORT          |      (Go)            |
+-------------+                  +----------------------+
                                     ^
                             Secret Manager / Config
```

### 2.2 API Compatibility Requirements

The GO Core Server **must implement** the following endpoints (CAO-compatible):

| Category | Endpoints | Notes |
|----------|-----------|-------|
| Health | `GET /health` | Returns `{ "status": "ok" }` |
| Profiles | `GET /agents/profiles`<br>`GET /agents/profiles/:name`<br>`POST /agents/profiles/install` | Markdown frontmatter |
| Providers | `GET /agents/providers` | Returns `ProviderAvailability[]` |
| Sessions | `POST /sessions`<br>`GET /sessions`<br>`GET /sessions/:name`<br>`DELETE /sessions/:name` | Query params: `provider`, `agent_profile`, `working_directory`, `env_vars` |
| Terminals | `POST /sessions/:name/terminals`<br>`GET /sessions/:name/terminals`<br>`GET /terminals/:id`<br>`GET /terminals/:id/output`<br>`GET /terminals/:id/working-directory`<br>`GET /terminals/:id/memory-context`<br>`POST /terminals/:id/input`<br>`POST /terminals/:id/exit`<br>`DELETE /terminals/:id` | WebSocket for streaming |
| Flows | `GET /flows`<br>`GET /flows/:name`<br>`POST /flows`<br>`DELETE /flows/:name`<br>`POST /flows/:name/enable`<br>`POST /flows/:name/disable`<br>`POST /flows/:name/run` | Cron scheduling |
| Auth/Sessions | `GET /auth/sessions`<br>`POST /auth/login`<br>`DELETE /auth/sessions/:id` | CLI session discovery |
| Settings | `GET /settings/agent-dirs`<br>`POST /settings/agent-dirs` | Agent directory config |
| Skills | `GET /skills/:name` | Skill definitions |

### 2.3 Frontend Changes Required

#### 2.3.1 Rename & Refactor API Client

| Old | New | File |
|-----|-----|------|
| `CaoClient` | `GoCoreClient` | `src/api/go-core-client.ts` (new) |
| `caoClient` | `goCoreClient` | Export singleton |
| `CAO_BASE_URL` | `GO_CORE_BASE_URL` | `src/api/base-url.ts` |
| `VITE_CAO_BASE_URL` | `VITE_GO_CORE_BASE_URL` | `.env.*`, `vite-env.d.ts` |
| `caoBaseUrl` setting | `goCoreBaseUrl` | `src/settings/settings-store.ts` |

#### 2.3.2 Update Type Definitions

**File**: `src/api/types.ts`
- Rename `ProviderType` if GO server uses different provider names
- Keep `CreateSessionInput` with `provider` field (fix from 1.1)
- Verify all response types match GO server contracts

#### 2.3.3 Update Reconciler

**File**: `src/canvas-reconciler/reconciler.ts`
- Import `goCoreClient` instead of `caoClient`
- Verify `resolveSessionEnv()` works with GO server's expected env vars
- Update any CAO-specific error handling

#### 2.3.4 Update Session Discovery

**Files**: `src/api/session-discovery.ts`, `src/api/session-store.ts`
- GO server's `/auth/sessions` should return same `DiscoveredSession` shape
- `resolveSessionEnv()` mapping should work unchanged

#### 2.3.5 Update Key Store Validators

**Files**: `src/api/key-store/validators/*.ts`
- Validators call external APIs (OpenAI, Anthropic, etc.) — **unchanged**
- Only the storage/retrieval of validated keys changes

#### 2.3.6 Update Terminal WebSocket

**File**: `src/api/connect-terminal-socket.ts`
- Update WebSocket URL construction for GO server
- Verify WebSocket protocol compatibility

#### 2.3.7 Update Settings Store

**File**: `src/settings/settings-store.ts`
- Rename `caoBaseUrl` → `goCoreBaseUrl`
- Update IndexedDB key from `caoBaseUrl` to `goCoreBaseUrl`
- Migrate existing setting on init

### 2.4 Environment Configuration

**Files to Update**:
- `.env.local` — `VITE_GO_CORE_BASE_URL=http://localhost:8080` (or GO server port)
- `.env.production` — Production GO Core URL
- `.env.example` — Template
- `vite-env.d.ts` — Type declaration for new env var
- `vite.config.ts` — Update proxy if needed

### 2.5 IndexedDB Migration

**File**: `src/shared/storage/migrations.ts`
- Add migration for `schema_version: 4`
- Rename `caoBaseUrl` setting key to `goCoreBaseUrl`
- Migrate any CAO-specific cached data

```typescript
if (oldVersion < 4) {
  // Migrate caoBaseUrl to goCoreBaseUrl
  const settingsStore = transaction.objectStore('settings');
  const oldSetting = await settingsStore.get('caoBaseUrl');
  if (oldSetting) {
    await settingsStore.put({ key: 'goCoreBaseUrl', value: oldSetting.value });
    await settingsStore.delete('caoBaseUrl');
  }
}
```

---

## Part 3: Implementation Plan

### Phase 1: Critical Fixes (Day 1)
- [ ] Fix `CreateSessionInput` type definition
- [ ] Fix ESLint plugin resolution
- [ ] Verify `npm run build` passes
- [ ] Verify `npm run lint` passes
- [ ] Run test suite, document baseline

### Phase 2: GO Core Client (Day 2)
- [ ] Create `src/api/go-core-client.ts` (copy + rename from `cao-client.ts`)
- [ ] Update `src/api/base-url.ts` to `go-core-base-url.ts`
- [ ] Update `src/api/types.ts` (verify GO server contracts)
- [ ] Update `src/api/index.ts` exports
- [ ] Update `vite-env.d.ts` for `VITE_GO_CORE_BASE_URL`

### Phase 3: Consumer Updates (Day 3)
- [ ] Update `src/canvas-reconciler/reconciler.ts` imports
- [ ] Update `src/api/session-discovery.ts` imports
- [ ] Update `src/api/session-store.ts` imports
- [ ] Update `src/api/connect-terminal-socket.ts`
- [ ] Update `src/settings/settings-store.ts`
- [ ] Update `src/shell/app-fetch.ts` if needed

### Phase 4: Configuration & Migration (Day 4)
- [ ] Update `.env.*` files
- [ ] Add IndexedDB migration (v4)
- [ ] Update `src/shared/storage/migrations.ts`
- [ ] Test migration from existing user data

### Phase 5: Verification (Day 5)
- [ ] `npm run build` passes
- [ ] `npm run lint` passes
- [ ] `npm run test` passes
- [ ] Manual E2E test with running GO Core Server
- [ ] Verify all CAO references removed (grep for "cao" / "CAO")

---

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| GO server API mismatch | Medium | High | Contract tests against live GO server |
| WebSocket protocol diff | Medium | High | Test terminal streaming early |
| Env var injection diff | Low | Medium | Verify `resolveSessionEnv` output |
| Session discovery diff | Low | Medium | Test `/auth/sessions` contract |
| User data migration loss | Low | High | Backup IndexedDB before migration |

---

## Rollback Plan

If GO Core migration introduces regressions:
1. Revert `src/api/` to CAO client (git checkout)
2. Revert `.env` to `VITE_CAO_BASE_URL`
3. Document GO server API gaps for future iteration

---

## Success Criteria

- [ ] `npm run build` exits 0
- [ ] `npm run lint` exits 0
- [ ] `npm run test` completes < 120s, >80% pass
- [ ] Frontend connects to GO Core Server (manual verify)
- [ ] Terminal streaming works
- [ ] Session discovery works
- [ ] Canvas deploy state transitions work (draft to deploying to deployed/degraded)
- [ ] No "cao" / "CAO" references in `src/` (except comments)
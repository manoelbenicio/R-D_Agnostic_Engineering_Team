# PROD Readiness: Critical Fixes & GO Core Migration — Task Breakdown

**Change**: `prod-readiness-critical-fixes`
**Status**: In Progress
**Owner**: IF (Infra) + SUP (Supervisor) + CV (Canvas) + TM (Terminal) + DB (Dashboard) + ST (Studio) + VX (Voice)

---

## Issue Classification Legend

| Priority | SLA | Description |
|----------|-----|-------------|
| 🔴 **CRITICAL** | 0-24h | Blocks build, deploy, or causes data loss |
| 🟠 **HIGH** | 1-3 days | Major functionality broken, security risk, or CI/CD broken |
| 🟡 **MEDIUM** | 3-7 days | Degraded functionality, tech debt, missing tests |
| 🟢 **LOW** | 7-14 days | Polish, documentation, nice-to-have |

---

## 🔴 CRITICAL ISSUES (Must Fix Before Any Deploy)

### CRIT-001: TypeScript Build Failure — Missing `provider` in `CreateSessionInput`
- **Severity**: 🔴 CRITICAL
- **Component**: `src/api/types.ts`, `src/api/cao-client.ts`, `src/canvas-reconciler/reconciler.ts`
- **Error Count**: 6 TypeScript errors blocking `tsc -b`
- **Root Cause**: Type definition missing `provider?: ProviderType` field that callers pass
- **Files & Lines**:
  - `src/api/types.ts:84-90` — Type definition
  - `src/api/cao-client.ts:59,83` — Uses `input.provider`
  - `src/canvas-reconciler/reconciler.ts:333,406,529,602` — Passes `provider` in object literals
- **Mitigation**:
  1. Add `provider?: ProviderType` to `CreateSessionInput` in `src/api/types.ts`
  2. Verify `AddTerminalInput = CreateSessionInput` inherits it
  3. Run `npm run typecheck` — must exit 0
- **Owner**: IF
- **Estimate**: 30 min
- **Verification**: `npm run build` passes

---

### CRIT-002: ESLint Plugin Resolution Failure
- **Severity**: 🔴 CRITICAL
- **Component**: `.eslintrc.cjs`, `package.json`, `eslint-rules/`
- **Error**: `ENOENT: no such file or directory, lstat 'C:\VMs\Projetos\Automonous_Agentic'`
- **Root Cause**: UNC path `//21LAPGLMVPJ4/Projetos/...` resolves incorrectly in Node.js
- **Mitigation Options** (pick one):
  - **Option A** (Recommended): Update `package.json` to absolute file URL:
    ```json
    "eslint-plugin-agentverse": "file:///M:/Automonous_Agentic/eslint-rules"
    ```
  - **Option B**: Publish to local verdaccio/GitHub Packages
  - **Option C**: Add `prepare` script to copy plugin to `node_modules`
- **Owner**: SUP
- **Estimate**: 1 hour
- **Verification**: `npm run lint` exits 0, custom rules enforced

---

### CRIT-003: GO Core Migration — Complete CAO → GO Core Replacement
- **Severity**: 🔴 CRITICAL
- **Component**: Entire `src/api/`, `src/canvas-reconciler/`, `src/settings/`, `src/shell/`, config files
- **Scope**: Replace ALL CAO references with GO Core Server as main brain
- **Sub-tasks**:

| Sub-task | File | Action |
|----------|------|--------|
| CRIT-003.1 | `src/api/types.ts` | Verify GO server contracts match types; add `provider` field (CRIT-001) |
| CRIT-003.2 | `src/api/base-url.ts` | Rename to `go-core-base-url.ts`; export `GO_CORE_BASE_URL` |
| CRIT-003.3 | `src/api/go-core-client.ts` | **NEW FILE** — Copy `cao-client.ts`, rename class to `GoCoreClient`, update imports |
| CRIT-003.4 | `src/api/index.ts` | Re-export `GoCoreClient`, `GO_CORE_BASE_URL`, types |
| CRIT-003.5 | `src/api/connect-terminal-socket.ts` | Import `GO_CORE_BASE_URL`; verify WS URL construction |
| CRIT-003.6 | `src/canvas-reconciler/reconciler.ts` | Import `goCoreClient`; verify `resolveSessionEnv()` compatibility |
| CRIT-003.7 | `src/api/session-discovery.ts` | Import `goCoreClient`; verify `/auth/sessions` contract |
| CRIT-003.8 | `src/api/session-store.ts` | Import `goCoreClient`; no logic changes expected |
| CRIT-003.9 | `src/settings/settings-store.ts` | Rename `caoBaseUrl` → `goCoreBaseUrl`; migrate IndexedDB key |
| CRIT-003.10 | `src/shell/app-fetch.ts` | Verify auth token attachment works with GO server |
| CRIT-003.11 | `.env.local` | `VITE_GO_CORE_BASE_URL=http://localhost:8080` (or GO server port) |
| CRIT-003.12 | `.env.production` | Production GO Core URL |
| CRIT-003.13 | `.env.example` | Template with `VITE_GO_CORE_BASE_URL` |
| CRIT-003.14 | `vite-env.d.ts` | Add `VITE_GO_CORE_BASE_URL?: string` |
| CRIT-003.15 | `src/shared/storage/migrations.ts` | Add v4 migration: `caoBaseUrl` → `goCoreBaseUrl` |
| CRIT-003.16 | `vite.config.ts` | Update proxy target if needed |

- **API Contract Requirements** (GO Server MUST implement):
  - Health: `GET /health`
  - Profiles: `GET/POST /agents/profiles*`
  - Providers: `GET /agents/providers`
  - Sessions: `POST/GET/DELETE /sessions*` with query params `provider`, `agent_profile`, `working_directory`, `env_vars`
  - Terminals: `POST/GET/DELETE /terminals*`, WebSocket streaming
  - Flows: `GET/POST/DELETE /flows*`, enable/disable/run
  - Auth: `GET/POST/DELETE /auth/sessions*`
  - Settings: `GET/POST /settings/agent-dirs`
  - Skills: `GET /skills/:name`
- **Owner**: IF (lead), CV, TM, DB
- **Estimate**: 3-4 days
- **Verification**: 
  - `npm run build` passes
  - `npm run lint` passes
  - Manual E2E with running GO Core Server
  - `grep -r "cao\|CAO" src/` returns only comments

---

### CRIT-004: Test Suite Unreliable / Timeout
- **Severity**: 🔴 CRITICAL
- **Component**: `vitest.config.ts`, test files, CI pipeline
- **Symptoms**: >4 min runtime, timeouts, many files show `0 test` during discovery
- **Root Causes**:
  1. UNC path + `fake-indexeddb` + `jsdom` performance issues
  2. MSW server startup overhead per test file
  3. Shared Zustand stores causing test pollution
  4. No test timeout configured
- **Mitigation**:
  1. Run tests from mapped drive (M:) not UNC
  2. Add to `vitest.config.ts`:
     ```typescript
     test: {
       testTimeout: 60000,
       pool: 'forks',
       poolOptions: { forks: { singleFork: true } },
       setupFiles: ['src/__tests__/setup.ts'],
       clearMocks: true,
       restoreMocks: true,
     }
     ```
  3. Add `beforeEach` in `setup.ts` to reset Zustand stores:
     ```typescript
     import { useSessionStore } from '@/api/session-store';
     import { useKeyStore } from '@/api/key-store/store';
     import { useDeployStore } from '@/canvas-reconciler/deploy-store';
     import { useSettingsStore } from '@/settings/settings-store';
     
     beforeEach(() => {
       useSessionStore.getState().clearError();
       useKeyStore.setState({ validated: [], initialized: false });
       useDeployStore.getState().clearDeploy();
       useSettingsStore.setState({ initialized: false });
     });
     ```
  4. Split large test files if needed
- **Owner**: SUP (lead), ALL
- **Estimate**: 1-2 days
- **Verification**: `npm run test` completes < 120s, >80% pass rate

---

## 🟠 HIGH ISSUES (Must Fix Before PROD)

### HIGH-001: Partial Deploy Orphan Cleanup (Reconciler)
- **Severity**: 🟠 HIGH
- **Component**: `src/canvas-reconciler/reconciler.ts`
- **Risk**: CAO resources (profiles, terminals) leaked on partial failure
- **Affected Code Paths**:
  - Lines 300-308: Profile install succeeds → terminal creation fails → profile orphaned
  - Lines 419-427: Terminal add fails → no cleanup
  - Lines 551-559: Update fails → old terminal deleted but new not created
  - Lines 624-632: Add terminal fails → profile installed but terminal not created
- **Mitigation**: Implement compensation transactions
  ```typescript
  // Pattern: try { A; B; } catch { cleanup(A); throw; }
  // For each CAO call sequence, add rollback on failure
  ```
- **Owner**: CV
- **Estimate**: 1 day
- **Verification**: Integration test simulating each failure path; verify no orphaned CAO resources

---

### HIGH-002: CAO/GO Contract Tests Not Running in CI
- **Severity**: 🟠 HIGH
- **Component**: CI pipeline, `src/api/__tests__/contract/`, `src/api/key-store/__tests__/contract/`
- **Risk**: API drift breaks production silently
- **Current State**: `npm run test:contract` requires `CAO_LIVE=1` and live CAO; not in CI
- **Mitigation**:
  1. Provision GO Core test instance in CI (container/VM)
  2. Add CI job: `contract-tests` runs on every PR
  3. Use `GO_CORE_LIVE=1` env var (rename from `CAO_LIVE=1`)
  4. Fail PR if contract tests fail
- **Owner**: IF
- **Estimate**: 1 day (infrastructure) + 0.5 day (config)
- **Verification**: Contract tests run on PR, catch API mismatches

---

### HIGH-003: Session Discovery Integration Gaps
- **Severity**: 🟠 HIGH
- **Component**: `src/api/session-discovery.ts`, `src/api/session-store.ts`, GO Core `/auth/sessions`
- **Risk**: Wrong env vars injected → terminal auth failures → silent deploy degradation
- **Gaps**:
  1. No integration test with real GO server + multiple CLI providers
  2. `resolveSessionEnv()` duplicated in 4 places in reconciler (DRY violation)
  3. Fallback logic in `discoverSessions()` returns mock sessions if `/auth/sessions` fails — may hide real issues
- **Mitigation**:
  1. Extract `resolveSessionEnv` to shared utility (single source of truth)
  2. Add integration test: real GO server + `claude_code`, `codex`, `gemini_cli`, `kiro_cli` sessions
  3. Remove silent fallback or add explicit warning when fallback triggers
- **Owner**: TM (lead), IF
- **Estimate**: 1 day
- **Verification**: Integration test passes with 4 CLI providers

---

### HIGH-004: WebGL Mandatory Terminal — Hardware Compatibility
- **Severity**: 🟠 HIGH
- **Component**: `src/terminal/TerminalView.tsx`, `src/terminal/xterm-theme.ts`, CI
- **Risk**: Users without WebGL2 (older GPUs, VMs, Chromebooks) cannot use terminal
- **Current**: Production refuses Canvas2D fallback (D7); CI runs WebGL-forced Chrome
- **Mitigation**:
  1. Test matrix: Chrome/FF/Safari on Windows/Mac/Linux, VMs, older GPUs
  2. Document minimum GPU requirements
  3. Improve error UX: clear message with troubleshooting link when WebGL fails
  4. Consider graceful degradation for read-only terminal view
- **Owner**: TM
- **Estimate**: 2 days (testing + UX)
- **Verification**: Compatibility matrix documented; error UX tested on non-WebGL machine

---

### HIGH-005: IndexedDB Migration for GO Core Migration
- **Severity**: 🟠 HIGH
- **Component**: `src/shared/storage/migrations.ts`, `src/settings/settings-store.ts`
- **Risk**: User data loss on schema migration (v3 → v4)
- **Migration Required**:
  - Rename `caoBaseUrl` setting key → `goCoreBaseUrl`
  - Preserve user's custom base URL setting
- **Mitigation**:
  1. Add migration v4 in `migrations.ts`
  2. Test migration with real user data (backup first)
  3. Add migration test in `src/shared/storage/__tests__/migrations.test.ts`
- **Owner**: IF
- **Estimate**: 4 hours
- **Verification**: Migration test passes; manual test with seeded IndexedDB

---

### HIGH-006: Key Store Validators — Contract Coverage Gap
- **Severity**: 🟠 HIGH
- **Component**: `src/api/key-store/__tests__/contract/`, validators
- **Risk**: Provider validation fails silently; user thinks key is valid when it's not
- **Current**: `tech-debt-keystore-validator-coverage` change exists but coverage incomplete
- **Mitigation**:
  1. Audit each validator: `openai`, `anthropic`, `google`, `aws`, `azure`, `moonshot`, `copilot`, `opencode`
  2. Ensure each has contract test against live API
  3. Run `npm run test:keystore-contract` in CI (requires valid keys in secrets)
- **Owner**: IF
- **Estimate**: 1 day
- **Verification**: All 8 validators have passing contract tests

---

## 🟡 MEDIUM ISSUES (Fix Before or Soon After PROD)

### MED-001: Voice NLU BYOK Fallback UX
- **Severity**: 🟡 MEDIUM
- **Component**: `src/voice/nlu.ts`, `src/voice/store.ts`, UI
- **Risk**: Voice disabled silently if no validated LLM key; user doesn't know why
- **Current**: Falls back Gemini Flash → GPT-4o-mini → Haiku; disables if none
- **Mitigation**:
  1. Show inline message in Voice Panel: "Voice requires a validated LLM key. Add one in Settings → Providers."
  2. Link directly to provider setup
  3. Log fallback chain for debugging
- **Owner**: VX
- **Estimate**: 4 hours
- **Verification**: Voice panel shows helpful message when no keys validated

---

### MED-002: Allowed Tools Free-Text Validation
- **Severity**: 🟡 MEDIUM
- **Component**: `src/canvas-builder/BlockConfigurationPanel.tsx`, `src/shared/canvas-types.ts`
- **Risk**: Typo in `allowedTools` (e.g., `read-file` vs `read_file`) silently yields unrecognized tool
- **Reference**: `agent-configuration-blueprint.md` §6 flags this
- **Mitigation**:
  1. Define `AllowedTool` type as union of known tools
  2. Change BlockConfigurationPanel to multi-select with validation
  3. Add validator in reconciler that warns on unknown tools
- **Owner**: CV
- **Estimate**: 1 day
- **Verification**: Unknown tool warning appears; multi-select UI works

---

### MED-003: Validation Proxy Absence (D6, R3)
- **Severity**: 🟡 MEDIUM (Accepted for v1, tracked)
- **Component**: `src/shared/validation-proxy.ts`, supervisor prompts
- **Risk**: Supervisor LLM can call `handoff("anything")` — CAO/GO obeys
- **Current**: Topology baked into prompt only; no enforcement
- **Mitigation** (Post-launch):
  1. Implement server-side Validation Proxy in GO Core
  2. Intercept `handoff`/`assign`/`send_message` calls
  3. Validate against deployed canvas topology
  4. Return error if target not in allowed list
- **Owner**: CV (post-launch change: `validation-proxy`)
- **Estimate**: Post-launch
- **Tracking**: Existing change `validation-proxy` in openspec

---

### MED-004: FinOps Tier 2 — Token Parsing
- **Severity**: 🟡 MEDIUM (Planned post-launch)
- **Component**: `src/finops/token-cost.ts`, `src/finops/usage-capture.ts`
- **Risk**: Cost estimates based on approximations, not actual token counts
- **Current**: Tier 1 (estimates only); Tier 2 (parsing) deferred
- **Mitigation** (Post-launch):
  1. GO server exposes per-turn token usage in terminal output
  2. Parse usage from terminal stream
  3. Store in `usage_events` (IndexedDB, migration v2)
  4. Replace estimates with actuals in dashboard
- **Owner**: DB (post-launch change: `finops-tier2-token-parsing`)
- **Estimate**: Post-launch
- **Tracking**: Existing change `finops-tier2-token-parsing` in openspec

---

### MED-005: Encrypted Key Storage (IndexedDB → Firestore/Secret Manager)
- **Severity**: 🟡 MEDIUM (Security)
- **Component**: `src/api/key-store/store.ts`, `src/shared/storage/idb.ts`
- **Risk**: Plaintext API keys in browser IndexedDB leak via devtools demos
- **Current**: Documented threat model (`docs/key-storage-v1.md`); UI masks values
- **Mitigation**:
  1. Encrypt keys before IndexedDB storage (Web Crypto API)
  2. Cloud mode: store in Firestore with user-scoped rules
  3. Add key rotation UI
- **Owner**: IF (new change needed)
- **Estimate**: 2-3 days
- **Verification**: Keys encrypted at rest; devtools shows only ciphertext

---

### MED-006: Bundle Size Budget Enforcement
- **Severity**: 🟡 MEDIUM
- **Component**: `vite.config.ts`, `scripts/check-bundle-size.mjs`, CI
- **Risk**: Bundle exceeds 1.5 MB gzipped budget
- **Current**: Script exists but not in CI
- **Mitigation**:
  1. Add `check-bundle-size` to CI pipeline
  2. Fail build if main chunk > 500 KB gzipped or total > 1.5 MB
  3. Monitor chunk sizes in `vite.config.ts` manualChunks
- **Owner**: SUP
- **Estimate**: 2 hours
- **Verification**: CI fails on bundle size regression

---

### MED-007: ESLint Rule `no-direct-cao-fetch` Obsolete After Migration
- **Severity**: 🟡 MEDIUM
- **Component**: `eslint-rules/index.cjs`, `.eslintrc.cjs`
- **Issue**: Rule name references "cao" — should be `no-direct-go-core-fetch` or similar
- **Mitigation**:
  1. Rename rule in `eslint-rules/index.cjs`
  2. Update `.eslintrc.cjs` rule reference
  3. Update logic to check for `goCoreClient` / `fetch` to GO server URLs
- **Owner**: SUP
- **Estimate**: 1 hour
- **Verification**: Rule catches direct fetch to GO server URLs

---

## 🟢 LOW ISSUES (Polish & Documentation)

### LOW-001: Remove Dead CAO References in Comments/Strings
- **Severity**: 🟢 LOW
- **Component**: All `src/` files
- **Action**: After migration, grep for `cao\|CAO\|Cao` and clean up comments, variable names, log messages
- **Owner**: ALL (final sweep)
- **Estimate**: 2 hours
- **Verification**: `grep -ri "cao" src/` returns empty

---

### LOW-002: Update Documentation for GO Core
- **Severity**: 🟢 LOW
- **Component**: `docs/`, `README.md`, `ARCHITECTURE.md`
- **Files to Update**:
  - `ARCHITECTURE.md` — Update architecture diagram, decisions
  - `docs/ARCHITECTURE_LOCAL_VS_CLOUD.md` — Replace CAO with GO Core
  - `docs/session-management.md` — Verify API endpoints
  - `docs/agent-configuration-blueprint.md` — No changes needed (platform-agnostic)
  - `README.md` — Update quick start, architecture
- **Owner**: SUP (docs owner)
- **Estimate**: 1 day
- **Verification**: All docs reflect GO Core architecture

---

### LOW-003: Add GO Core Health Check Endpoint to Dashboard
- **Severity**: 🟢 LOW
- **Component**: `src/dashboard/DashboardPage.tsx`, `src/api/go-core-client.ts`
- **Action**: Add "Runtime Status" card showing GO server health, version, uptime
- **Owner**: DB
- **Estimate**: 4 hours
- **Verification**: Dashboard shows GO Core health status

---

### LOW-004: Settings UI — Rename "CAO Base URL" to "GO Core Base URL"
- **Severity**: 🟢 LOW
- **Component**: `src/settings/`, any settings UI components
- **Action**: Update label, help text, placeholder
- **Owner**: ST
- **Estimate**: 30 min
- **Verification**: Settings page shows "GO Core Base URL"

---

### LOW-005: Add Migration Changelog Entry
- **Severity**: 🟢 LOW
- **Component**: `CHANGELOG.md`
- **Action**: Document CAO → GO Core migration, breaking changes, migration steps
- **Owner**: SUP
- **Estimate**: 30 min
- **Verification**: Changelog updated

---

## Issue Dependency Graph

```
CRIT-001 (TS Types)
    │
    ├──→ CRIT-003.1 (GO Core types) ──→ CRIT-003.3-3.16 (Full migration)
    │
CRIT-002 (ESLint) ──→ Enables CI quality gates
    │
CRIT-004 (Tests) ──→ Enables reliable CI verification
    │
HIGH-001 (Orphan cleanup) ← Depends on CRIT-003.6 (reconciler updated)
    │
HIGH-002 (Contract tests) ← Depends on CRIT-003 (GO server running in CI)
    │
HIGH-003 (Session discovery) ← Depends on CRIT-003.7-3.8
    │
HIGH-005 (IDB migration) ← Depends on CRIT-003.9, 3.15
    │
HIGH-006 (Keystore contracts) ← Independent, can parallelize
    │
MED-001, MED-002, MED-007 ← Can parallelize after CRIT-003
    │
LOW-001, LOW-002, LOW-004, LOW-005 ← Final cleanup after CRIT-003
```

---

## Resource Allocation (Parallelizable Workstreams)

| Workstream | Owner | Tasks | Days |
|------------|-------|-------|------|
| **Core Migration** | IF (lead) | CRIT-001, CRIT-003.1-3.16, HIGH-005 | 4 |
| **CI/CD & Quality** | SUP | CRIT-002, CRIT-004, MED-006, MED-007 | 2 |
| **Reconciler & Canvas** | CV | CRIT-003.6, HIGH-001, MED-002 | 2 |
| **Terminal & Session** | TM | CRIT-003.5, 3.7-3.8, HIGH-003, HIGH-004 | 2 |
| **Dashboard & FinOps** | DB | CRIT-003.10, MED-004, LOW-003 | 1 |
| **Studio & Settings** | ST | CRIT-003.9, LOW-004 | 1 |
| **Voice & Auth** | VX | MED-001 | 0.5 |
| **Documentation** | SUP | LOW-002, LOW-005 | 1 |

**Total Calendar Time**: 5 days (with parallelization)
**Critical Path**: CRIT-001 → CRIT-003 (Full Migration) → HIGH-001, HIGH-005 → Verification

---

## Definition of Done (Per Task)

Every task must meet:
- [ ] Code compiles (`npm run typecheck` exits 0)
- [ ] Lint passes (`npm run lint` exits 0)
- [ ] Unit tests pass for modified code
- [ ] Integration test added/updated where applicable
- [ ] Documentation updated (inline comments, README, or docs/)
- [ ] Peer reviewed (PR approved by supervisor)
- [ ] Merged to main with green CI

---

## Go/No-Go Criteria for PROD Deploy

| Gate | Criteria | Status |
|------|----------|--------|
| **Build** | `npm run build` exits 0 | ☐ |
| **Lint** | `npm run lint` exits 0, custom rules enforced | ☐ |
| **Types** | `npm run typecheck` exits 0 | ☐ |
| **Tests** | `npm run test` < 120s, >80% pass | ☐ |
| **Contract** | `GO_CORE_LIVE=1` contract tests pass in CI | ☐ |
| **Bundle** | `check-bundle-size` passes (< 1.5 MB gzipped) | ☐ |
| **E2E** | Manual: deploy canvas → terminal streams → session discovery works | ☐ |
| **Migration** | IndexedDB v3→v4 migration tested with user data | ☐ |
| **Cleanup** | No "cao"/"CAO" references in `src/` (except comments) | ☐ |
| **Docs** | Architecture docs reflect GO Core | ☐ |

**ALL GATES MUST BE GREEN FOR PROD DEPLOY**

---

## Rollback Procedures

### Rollback: TypeScript Fix Only (CRIT-001)
```bash
git checkout HEAD -- src/api/types.ts
# Revert provider field addition
# Document as known limitation
```

### Rollback: Full GO Core Migration (CRIT-003)
```bash
# 1. Revert API client
git checkout HEAD~1 -- src/api/

# 2. Revert consumers
git checkout HEAD~1 -- src/canvas-reconciler/reconciler.ts
git checkout HEAD~1 -- src/api/session-discovery.ts
git checkout HEAD~1 -- src/api/session-store.ts
git checkout HEAD~1 -- src/api/connect-terminal-socket.ts
git checkout HEAD~1 -- src/settings/settings-store.ts

# 3. Revert config
git checkout HEAD~1 -- .env.local .env.production .env.example vite-env.d.ts

# 4. Document GO server API gaps for re-attempt
```

### Rollback: IndexedDB Migration (HIGH-005)
```bash
# If migration corrupts user data:
# 1. User clears browser data (Settings → Clear Data)
# 2. Or: implement migration rollback in migrations.ts (down migration)
# 3. Document in troubleshooting guide
```

---

## Communication Plan

| Audience | Channel | Frequency | Content |
|----------|---------|-----------|---------|
| **Engineering Team** | Daily standup | Daily | Progress on assigned tasks, blockers |
| **Supervisor** | PR reviews | Per PR | Architecture decisions, code quality |
| **Stakeholders** | Weekly sync | Weekly | PROD readiness %, risk status, timeline |
| **On-call** | Runbook update | Pre-deploy | GO Core endpoints, troubleshooting, rollback |

---

## Appendix: File Change Summary

### New Files
- `src/api/go-core-client.ts`
- `src/api/go-core-base-url.ts`

### Modified Files (Critical)
- `src/api/types.ts`
- `src/api/index.ts`
- `src/canvas-reconciler/reconciler.ts`
- `src/api/session-discovery.ts`
- `src/api/session-store.ts`
- `src/api/connect-terminal-socket.ts`
- `src/settings/settings-store.ts`
- `src/shared/storage/migrations.ts`
- `.env.local`, `.env.production`, `.env.example`
- `vite-env.d.ts`
- `vitest.config.ts`
- `package.json` (ESLint plugin path)

### Deleted/Deprecated Files (After Migration)
- `src/api/cao-client.ts` → replaced by `go-core-client.ts`
- `src/api/base-url.ts` → replaced by `go-core-base-url.ts`
- `src/api/__tests__/contract/` → updated for GO Core contracts

### Test Files to Update
- All `__tests__` files importing from `src/api/` — update imports
- Contract tests: update endpoint expectations
- Integration tests: point to GO Core test instance

---

**Document Version**: 1.0
**Last Updated**: 2026-06-19
**Next Review**: Daily during implementation
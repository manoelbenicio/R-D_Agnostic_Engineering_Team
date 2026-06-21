# PROD Readiness: Critical Fixes & Risk Mitigation

## Executive Summary

This change addresses **showstopper blockers** preventing AgentVerse v1 from deploying to production. A comprehensive audit of the codebase (post-CORE merge) revealed:

| Blocker | Severity | Impact |
|---------|----------|--------|
| **TypeScript Build Failure** | 🔴 CRITICAL | `tsc -b` fails — cannot produce production bundle |
| **ESLint Plugin Load Failure** | 🔴 CRITICAL | CI quality gates bypassed — custom rules not enforced |
| **Test Suite Timeout/Unreliable** | 🟠 HIGH | Cannot verify regression status in CI |

Additionally, **architectural risks** were identified that require mitigation before PROD:
- CAO API contract drift (no nightly contract tests running)
- Partial deploy orphan cleanup (reconciler leaves CAO resources on failure)
- Session discovery integration gaps
- WebGL mandatory terminal compatibility

## Problem Statement

### 1. TypeScript Compilation Errors (BLOCKER)

**Files affected**: 6 locations across 3 files

```
src/api/cao-client.ts(59,23): error TS2339: Property 'provider' does not exist on type 'CreateSessionInput'.
src/api/cao-client.ts(83,23): error TS2339: Property 'provider' does not exist on type 'CreateSessionInput'.
src/canvas-reconciler/reconciler.ts(333,13): error TS2353: Object literal may only specify known properties, and 'provider' does not exist in type 'CreateSessionInput'.
src/canvas-reconciler/reconciler.ts(406,13): error TS2353: Object literal may only specify known properties, and 'provider' does not exist in type 'CreateSessionInput'.
src/canvas-reconciler/reconciler.ts(529,13): error TS2353: Object literal may only specify known properties, and 'provider' does not exist in type 'CreateSessionInput'.
src/canvas-reconciler/reconciler.ts(602,13): error TS2353: Object literal may only specify known properties, and 'provider' does not exist in type 'CreateSessionInput'.
```

**Root cause**: `CreateSessionInput` and `AddTerminalInput` types in `src/api/types.ts` are missing the `provider` field that the CAO API requires and that callers pass.

**Evidence**: CAO client constructs `URLSearchParams` with `provider: input.provider` but the type definition only includes `profile`, `working_directory`, and `env_vars`.

### 2. ESLint Custom Plugin Resolution Failure

**Error**:
```
Failed to load plugin 'agentverse' declared in '.eslintrc.cjs': 
ENOENT: no such file or directory, lstat 'C:\VMs\Projetos\Automonous_Agentic'
```

**Root cause**: The `eslint-plugin-agentverse` local plugin (in `eslint-rules/`) resolves via `file:./eslint-rules` in package.json, but the UNC network path (`//21LAPGLMVPJ4/Projetos/...`) causes Node.js to resolve to an incorrect Windows path (`C:\VMs\...`).

**Impact**: 
- `npm run lint` fails completely
- Custom rules `agentverse/no-sideways-capability-imports` and `agentverse/no-direct-cao-fetch` not enforced
- Architecture boundary violations (D9) can slip into main branch

### 3. Test Execution Reliability

**Observed**: `vitest run` takes >4 minutes, times out in CI, many test files show `0 test` during discovery.

**Suspected causes**:
- UNC path + `fake-indexeddb` + `jsdom` environment issues
- MSW server startup overhead
- Missing test isolation (shared Zustand stores across tests)

**Impact**: Cannot trust CI test results; unknown coverage/regression status.

---

## Architectural Risks Requiring Mitigation

### R1: CAO API Contract Drift (from ARCHITECTURE.md R8)
- **Current**: Nightly contract tests exist (`npm run test:contract`) but require `CAO_LIVE=1` and live CAO instance
- **Gap**: No evidence these run in CI/CD pipeline
- **Risk**: CAO backend changes break frontend silently
- **Mitigation**: Provision CAO test instance in CI; run contract suite on every PR

### R2: Partial Deploy Orphan Cleanup (from ARCHITECTURE.md R5)
- **Current**: Reconciler marks `degraded` state on partial failure but does NOT clean up CAO resources
- **Example**: Profile installed → terminal creation fails → profile remains in CAO (orphaned)
- **Locations**: `reconciler.ts` lines 300-308, 419-427, 551-559, 624-632
- **Risk**: Resource leaks, naming conflicts on retry, CAO state drift
- **Mitigation**: Implement compensation transactions (delete profile on terminal failure, etc.)

### R3: Session Discovery Integration Gaps
- **Current**: `resolveSessionEnv()` maps 4 CLI providers to env vars, duplicated in 4 places in reconciler
- **Gap**: No integration test with real CAO + multiple authenticated CLI sessions
- **Risk**: Wrong env vars → terminal auth failures → silent deploy degradation

### R4: WebGL Mandatory Terminal (D7, R7)
- **Current**: Production refuses Canvas2D fallback; CI runs WebGL-forced Chrome
- **Gap**: No compatibility testing on diverse hardware (older GPUs, VMs, Chromebooks)
- **Risk**: Segment of users cannot use terminal at all
- **Mitigation**: Test matrix + clear error UX + documentation

---

## Proposed Solution

### Phase 1: Critical Fixes (Days 1-2) — UNBLOCK DEPLOY

| Task | Owner | Files |
|------|-------|-------|
| Fix `CreateSessionInput` / `AddTerminalInput` types | IF | `src/api/types.ts` |
| Update all 6 call sites to match corrected types | IF | `src/api/cao-client.ts`, `src/canvas-reconciler/reconciler.ts` |
| Fix ESLint plugin resolution (local path or publish) | SUP | `package.json`, `.eslintrc.cjs`, `eslint-rules/` |
| Verify `npm run build` passes | IF | — |
| Verify `npm run lint` passes | SUP | — |
| Run full test suite locally, document failures | ALL | — |

### Phase 2: CI/CD Hardening (Days 3-5) — QUALITY GATES

| Task | Owner | Description |
|------|-------|-------------|
| Provision CAO test instance for CI | IF | Container/VM with CAO for contract tests |
| Add `test:contract` to CI pipeline | IF | Run on every PR when CAO available |
| Fix test timeout/flakiness | SUP | Optimize vitest config, isolate stores |
| Add bundle size check to CI | SUP | Enforce ≤ 1.5 MB gzipped budget |

### Phase 3: Reconciler Resilience (Days 5-7) — OPERATIONAL SAFETY

| Task | Owner | Description |
|------|-------|-------------|
| Add compensation logic for partial failures | CV | Cleanup CAO profiles/terminals on error |
| Extract `resolveSessionEnv` usage to single helper | CV | DRY: 4 duplicate calls in reconciler |
| Add integration test: session discovery → terminal creation | TM | Real CAO + multiple CLI providers |
| Document WebGL compatibility matrix | TM | Tested browsers/GPUs + error UX |

### Phase 4: Post-Launch Tracking (Separate Changes)

| Risk | Tracking Change |
|------|-----------------|
| Validation Proxy (server-side enforcement) | `validation-proxy` (existing) |
| FinOps Tier 2 (token parsing) | `finops-tier2-token-parsing` (existing) |
| Encrypted key storage | New change needed |
| Voice NLU fallback UX | Part of `tech-debt-voice-coverage-gap` |

---

## Success Criteria

### Phase 1 Complete When:
- [ ] `npm run build` exits 0 (TypeScript + Vite)
- [ ] `npm run lint` exits 0 (all custom rules enforced)
- [ ] `npm run test` completes < 120s with >80% pass rate
- [ ] `npm run typecheck` exits 0

### Phase 2 Complete When:
- [ ] CI pipeline runs `lint` + `typecheck` + `test` + `build` on every PR
- [ ] Contract tests run nightly against live CAO (or on PR if CAO available)
- [ ] Bundle size check fails build if > 1.5 MB gzipped

### Phase 3 Complete When:
- [ ] Reconciler cleans up CAO resources on any partial failure path
- [ ] Session discovery integration test passes in CI
- [ ] WebGL compatibility matrix documented + error UX verified

---

## Rollback Plan

If critical fixes introduce regressions:
1. Revert `src/api/types.ts` and call sites to pre-fix state
2. Document known build failure as technical debt
3. Schedule dedicated fix sprint

**Note**: Current state is already broken (cannot build), so rollback returns to same broken state. Fix is mandatory.

---

## Dependencies

- **CAO backend**: Must be available for contract tests (version compatibility matrix needed)
- **CI infrastructure**: Needs CAO test instance provisioned
- **Team capacity**: 2-3 engineers for 1 week (parallelizable across phases)

---

## Approval

| Role | Name | Approval |
|------|------|----------|
| Supervisor (SUP) | — | ☐ |
| Infra (IF) | — | ☐ |
| Canvas (CV) | — | ☐ |
| Terminal (TM) | — | ☐ |

---

## References

- `ARCHITECTURE.md` — Decisions D1-D15, Risks R1-R9
- `openspec/changes/milestone-1-canvas-deploy-run/design.md` — Authoritative design
- `openspec/changes/design-system-indra-alignment/sev0/` — SEV0 evidence for design system alignment
- `src/api/cao-client.ts` — CAO client surface (30+ endpoints)
- `src/canvas-reconciler/reconciler.ts` — Deploy/reconcile logic (700 lines)
- `src/api/session-discovery.ts` — Session discovery & env resolution
- `src/api/key-store/` — Provider key validation & storage
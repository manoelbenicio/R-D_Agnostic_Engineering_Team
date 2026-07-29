# Packet B — Independent Technical-vs-Governance Review (vendor model visibility)

- reviewer: Kiro (principal, independent)
- date: 2026-07-18T21:16:00Z
- mode: READ-ONLY. Offline deterministic checks only. No product/test/spec/checkbox/git/index edits; no credentials/env/network/services.
- producer evidence reviewed: `.planning/agent-brain-v3/evidence/vendor-model-visibility-ui.md` (SHA-256 `67cdb00d8a930a9c6ca02991e8b431b22f00a5fdde2dad74712b23af175b9106`)

## Check-in / Check-out (recorded here per ownership directive; no shared ledger/state/OpenSpec/index touched)
- CHECK-IN 2026-07-18T21:12:00Z — Kiro — stream PACKETB-ACCEPTANCE-REVIEW — READ-ONLY.
- CHECK-OUT 2026-07-18T21:16:00Z — DONE. Verdict: TECHNICAL PASS (bounded) / GOVERNANCE PENDING. client.ts AbortSignal cleanly separable. Kiro TL adjudicates; not self-accepted.

## VERDICT

- **Technical: PASS (bounded).** The implementation is correct and coherent by static review; core + views typechecks pass offline; producer's mocked unit tests are recorded green. Bound: jsdom UI tests could not be independently re-executed here (environment blocker, below).
- **Governance: PENDING.** No traceable OpenSpec task ID or EV contract authorizes acceptance; the producer evidence itself is "PRODUCED — pending independent re-review" and changed no checkbox. Not REJECT (technically sound), not ACCEPT (no authorizing contract). **Kiro TL adjudicates; I do not self-accept.**
- **client.ts AbortSignal: cleanly separable — YES.**

## Scope

In scope: 7 producer-covered files — `packages/core/runtimes/models.ts`; `packages/views/agents/components/model-dropdown.tsx`; `.../inspector/model-picker.tsx`; `.../runtime-picker.tsx`; and tests `models.test.tsx`, `model-dropdown.test.tsx`, `inspector/model-picker.test.tsx`. Plus `packages/core/api/client.ts` AbortSignal portion (separability only). Out of scope (as instructed): `packages/core/types/agent.ts`, `agent.test.ts`.

Source blob hashes (git, current staged content): models.ts `33e57e9e`, models.test.tsx `56d4485f`, model-dropdown.tsx `8ccdeb18`, model-dropdown.test.tsx `1acfa8e8`, model-picker.tsx `ae23dad9`, model-picker.test.tsx `4d204012`, runtime-picker.tsx `2b2a6674`, client.ts `9d6fe950`.

## Independent offline verification (exact commands)

Tooling: node v24.17.0, pnpm 10.28.2, `node_modules` present (no install performed → no network).

```
# PASS (deterministic, offline)
pnpm --filter @multica/core  exec tsc --noEmit   → clean (exit 0, no errors)
pnpm --filter @multica/views exec tsc --noEmit   → clean (VIEWS_TYPECHECK_EXIT=0)

# NOT REPRODUCIBLE HERE (environment blocker, not a test failure):
pnpm --filter @multica/core  exec vitest run runtimes/models.test.tsx --pool=threads --maxWorkers=1
  → "Failed to start threads worker … Timeout waiting for worker to respond"; "Test Files no tests / Errors 1"
pnpm --filter @multica/views exec vitest run model-dropdown.test.tsx inspector/model-picker.test.tsx runtime-picker.test.tsx
  → same worker-start timeout; no test executed
```

**Explicit non-claim / limitation:** the repo lives on a `/mnt/c` (drvfs) mount; the vitest jsdom worker pool times out on worker startup here (I/O-locality blocker; the producer documented the same intermittent failure). I therefore did **not** independently re-execute the jsdom UI tests. The recorded green UI results (core 14, views 6+3) are the producer's, not independently reproduced by me. Typechecks start deterministically and did pass.

## Technical assessment (static review of the 7 files + client.ts)

Correct and well-structured:
- **Cancellation:** `resolveRuntimeModels` threads `AbortSignal` into both API calls and `abortableDelay`; `abortableDelay` cleans up its listener (`removeEventListener`, `{once:true}`) and rejects with a proper `abortReason` (`signal.reason ?? AbortError`). No leak.
- **Offline visibility:** `runtimeModelsOptions` retains the runtime-specific query key while `enabled=false`, so a cached catalog stays visible without a request — matches the stated root cause fix.
- **Bounded session cache:** `setBoundedRecent` gives recency eviction with hard caps (32 catalogs, 256 rows/catalog, 64 identities); sentinel is `gcTime:Infinity` but bounded and cleared on `queryClient.clear()`. No unbounded growth.
- **Provider grouping/fallback:** explicit `model.provider` wins; runtime fallback only fills empty; search matches id/label/provider. Consistent across create (`model-dropdown`) and inspector (`model-picker`).
- **Conservative identity:** `knownRuntimeProviderFromIdentity` only resolves the 16 built-in providers or a `builtin|provider|runtime:` prefix; arbitrary custom IDs stay `Unknown/Custom` (no guessing).
- **Lifecycle:** `useRuntimeModelsLifecycle` cancels on offline, and on a newer runtime-list generation forgets/invalidates with `refetchType:"none"` guarded by `isInvalidated` + a prepared-generation latch, preventing duplicate reconnect requests.

Minor, non-blocking observations: `KNOWN_RUNTIME_PROVIDERS` is a hardcoded 16-entry set that must track backend runtimes (maintenance coupling, not a bug); complexity of the lifecycle hook is high but justified and covered by the producer's recorded tests.

**Coverage gap (independent finding):** the staged set includes `packages/views/agents/components/runtime-picker.test.tsx`, but the producer evidence's "Direct tests" list does **not** include it (it lists only models/model-dropdown/model-picker tests and states runtime-picker "did not change its behavior further"). That staged test is therefore **untraced** by the producer evidence and needs provenance before inclusion.

## client.ts AbortSignal separability

The entire `client.ts` change (staged, and worktree == index — no unstaged delta) is exactly:
- `initiateListModels(runtimeId, signal?)` → forwards `signal` to `fetch`.
- `getListModelsResult(runtimeId, requestId, signal?)` → forwards `signal` to `fetch`.

There are **zero** auth-login lines (no login/password/`/auth/`/verify-code/AuthProvider/credential) in the diff. **Conclusion: the AbortSignal change is cleanly and fully separable from the reopened auth-login work** — the auth-login change is not present in this file's current diff at all. The earlier "Bucket C (Native Runtimes web, REOPENED)" tag was over-broad for the *current* content.
- Dependency note: this AbortSignal passthrough is consumed by `models.ts` (`resolveRuntimeModels` passes the signal). It should therefore land **with Packet B**, not with the auth lane. Committed alone it is inert but harmless (optional param); committed with auth it would wrongly couple lanes.

## Governance analysis

- Producer evidence status: **"PRODUCED — pending independent re-review"** (stated twice) and "no GSD state or OpenSpec task checkbox was modified."
- No OpenSpec task in `native-runtimes-onboarding` owns the picker UI: 1.4 (model discovery) is backend and `[x]`; 1.5/1.6 are `[ ]` and their check-ins are BLOCKED; none is the picker UI.
- No EV id is registered for this packet (the `EV-*` names elsewhere are proposals).
- Therefore **no traceable task/EV contract authorizes acceptance** → governance = **PENDING**, not ACCEPT. It is not REJECT because the implementation is technically sound and typechecks pass.

## Recommendation (Kiro TL adjudicates; not executed here)

1. To move PENDING→ACCEPT: register/point to a concrete owning task ID + EV, and complete the independent re-review the producer evidence itself requests (ideally with jsdom tests executed on a POSIX-local checkout, not `/mnt/c`).
2. Provide provenance or exclude `runtime-picker.test.tsx` (untraced by producer evidence).
3. If any atomic commit proceeds, the `client.ts` AbortSignal hunk may travel with Packet B and must **not** be bundled with auth-login changes.
4. `types/agent.*` remain out of this scope and unaddressed here.

## Explicit non-claims

- Changed no product/test/spec/checkbox/git/index/shared-ledger/state; created only this artifact (and my uniquely-named check-in file). Ran no `git add/restore/commit/push`.
- Read no credential/env values; made no network/DB/live/service calls; ran no vendor login.
- Did **not** independently re-execute jsdom UI tests (recorded `/mnt/c` worker-start blocker); relied on typechecks + static review + the producer's recorded results.
- This is decision support: **technical PASS ≠ governance ACCEPT.** No acceptance/EV/checkbox is granted; Kiro TL adjudicates.

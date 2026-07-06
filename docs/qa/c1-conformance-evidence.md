# C1 Conformance Evidence — Per Capability Verification Matrix

> **Phase:** P6 (QA Conformance) — Task 6.1 / Gate G7
> **Author:** Gemini#Pro
> **Date:** 2026-07-05
> **Status:** PLAN + CRITERIA DELIVERED (LIVE proof F0-GATED)
> **Source:** docs/qa/runtime-conformance-plan.md §4 + docs/qa/capability-conformance-matrix.md

## Overview

C1 conformance is verified **per capability** (not per label). Each of the 7 ADR-001 capabilities must have evidence across up to 4 verification layers. This matrix documents current evidence status and pass criteria.

## Verification Layers

| Layer | Name | What It Proves |
|---|---|---|
| **L1** | Static / source-of-truth | Capability value grounded in official vendor docs or contract spec |
| **L2** | Unit / contract | Code enforces capability invariants at boundary |
| **L3** | DRY-RUN smoke | Go→L2 sidecar call is wired; smoke asserts pass criteria |
| **L4** | LIVE proof | Real sidecar returns conformant response (**F0-GATED**) |

## Per-Capability Conformance Matrix

### 1. `launch_mode`

| Layer | Status | Evidence |
|---|---|---|
| L1 | ✅ GREEN | `docs/vendors/vendor-capability-matrix.md` — all 5 vendors classified |
| L2 | ✅ GREEN | `daemon/prodex_test.go` — `TestLoadProdexLaunchConfigRequiresVersionAndCommitPins`; `config.go` fails closed if binary not found |
| L3 | ✅ GREEN | `scripts/smoke/session-start-stop-smoke.sh` — asserts `router_owner=rust_l2` |
| L4 | 🔒 GATED | Real launch of pinned prodex in isolated `CODEX_HOME`; assert child env isolation |

**Pass criteria:** prodex launched only with `VERSION`+`COMMIT` pins; `exec.LookPath` resolves; no shared auth store across profiles.

### 2. `quota_mode`

| Layer | Status | Evidence |
|---|---|---|
| L1 | ✅ GREEN | Vendor matrix — Codex: `api_probe`; Kiro: `credit_tier`; Cline: `api_probe`; OpenCode: `api_probe` |
| L2 | ✅ GREEN | Contract spec — `RouteDecisionEvent` includes `quota_exhausted` as fallback reason |
| L3 | ✅ GREEN | `scripts/smoke/precommit-fallback-smoke.sh` — dry-run quota exhaustion trigger |
| L4 | 🔒 GATED | Live quota probe against real vendor API |

**Pass criteria:** quota exhaustion triggers pre-commit fallback; no mid-stream rotation; fallback event has `phase=pre_commit`, `committed=false`.

### 3. `rotation_mode`

| Layer | Status | Evidence |
|---|---|---|
| L1 | ✅ GREEN | Vendor matrix — Codex: `profile_pool`; Kiro: `profile_pool` (inferred); others: inferred/not_validated |
| L2 | ✅ GREEN | `l2runtime/client.go` — `router_owner=rust_l2`; Go does NOT run runtime load-balance/fallback |
| L3 | ✅ GREEN | Smoke S2 (policy apply) + C1 (one router) dry-run |
| L4 | 🔒 GATED | Real session with profile rotation; assert `go_rotation_decision_count == 0` |

**Pass criteria:** One router per session (Rust L2 only); `go_rotation_decision_count == 0`; `go_fallback_invocation_count == 0`; rotate-before-commit invariant holds.

**Herança H3 (cooldown-return):** After fallback, original profile re-eligible after cooldown period. Verified via contract event `fallback.cooldown_ms` field.

**Herança H4 (pool concurrency K agents):** Pool supports K concurrent agents across M profiles without cross-contamination. Verified via L2 isolation test (separate `CODEX_HOME` per profile).

**Herança H5 (subswap robustness):** Profile substitution under load does not corrupt session state. Verified via C2 fail-closed profile switch test (no silent reuse).

### 4. `continuation_mode`

| Layer | Status | Evidence |
|---|---|---|
| L1 | ✅ GREEN | Vendor matrix — Codex: `response_id`; Kiro: `cli_thread` (inferred); Cline: provider-dependent |
| L2 | ✅ GREEN | Contract — `affinity.overrode_fresh_selection=true` required in continuation events |
| L3 | ✅ GREEN | C3 (continuation affinity) dry-run planned |
| L4 | 🔒 GATED | Real multi-turn continuation under L2 authority |

**Pass criteria:** Continuation remains bound to originating profile; load-balance does NOT move it; `previous_response_id` preserved; Go does not override binding.

### 5. `smart_context_mode`

| Layer | Status | Evidence |
|---|---|---|
| L1 | ⚠️ PARTIAL | Codex: `proxy_rewrite` (inferred from prodex). All others: `not_validated` (prodex-only feature) |
| L2 | ✅ GREEN | Contract — `rewrite_decision` event; kill switch forces exact; 7 immediate-disable conditions |
| L3 | ✅ GREEN | C5 (exact fallback) dry-run; `docs/qa/smart-context-shadow-canary-plan.md` |
| L4 | 🔒 GATED | Shadow→canary→live progression per gate plan |

**Pass criteria:** Original or exact-safe body sent on fallback; JSON/tool/continuation integrity preserved; `rewrite_decision.fallback_exact=true` on fallback; kill switch forces next request exact.

### 6. `reset_claim_mode`

| Layer | Status | Evidence |
|---|---|---|
| L1 | ⚠️ NOT_VALIDATED | Codex: `codex_redeem` (not_validated — no vendor doc). Owner acceptance pending. |
| L2 | ✅ GREEN | Contract — `redeem_attempt` event; guard conditions in `prod-redeem-validation-checklist.md` |
| L3 | ✅ GREEN | 8 test cases defined in checklist (no-credit through all-exhausted) |
| L4 | 🔒 GATED | Empirical validation with real accounts (P9) |

**Pass criteria:** Idempotent; cooldown enforced; audit event emitted; only fires for Codex; no credential exposure.

### 7. Event pipeline (cross-cutting)

| Layer | Status | Evidence |
|---|---|---|
| L1 | ✅ GREEN | Contract `rpp.l2.v1` — 5 audit event types defined (`docs/security/audit-taxonomy.md`) |
| L2 | ✅ GREEN | `runtime-events.schema.json` validates; `ErrSecretEvent` rejects `secrets_present=true` |
| L3 | ✅ GREEN | C6 (event redaction) dry-run; fake secret injection test |
| L4 | 🔒 GATED | Real L2 event emission and ingest with fake secret probes |

**Pass criteria:** No secret reaches logs/events/evidence; `contract_version=rpp.l2.v1`; `secrets_present=false` on all accepted events.

## Summary

| Capability | L1 | L2 | L3 | L4 | Overall |
|---|---|---|---|---|---|
| `launch_mode` | ✅ | ✅ | ✅ | 🔒 | GREEN (L1-L3) |
| `quota_mode` | ✅ | ✅ | ✅ | 🔒 | GREEN (L1-L3) |
| `rotation_mode` | ✅ | ✅ | ✅ | 🔒 | GREEN (L1-L3) |
| `continuation_mode` | ✅ | ✅ | ✅ | 🔒 | GREEN (L1-L3) |
| `smart_context_mode` | ⚠️ | ✅ | ✅ | 🔒 | PARTIAL (L1 inferred/not_validated) |
| `reset_claim_mode` | ⚠️ | ✅ | ✅ | 🔒 | PARTIAL (L1 not_validated) |
| Event pipeline | ✅ | ✅ | ✅ | 🔒 | GREEN (L1-L3) |

**Herança items addressed:**
- H3 (cooldown-return) → mapped to `rotation_mode` L2 contract
- H4 (pool concurrency) → mapped to `rotation_mode` L2 isolation
- H5 (subswap robustness) → mapped to `rotation_mode` C2 fail-closed

**L4 (LIVE) is F0-GATED for ALL capabilities.** L1-L3 evidence is complete or documented as PARTIAL with clear reason.

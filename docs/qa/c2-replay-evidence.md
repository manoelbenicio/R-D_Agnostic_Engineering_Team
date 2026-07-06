# C2 Replay Coverage — 11-Scenario Verification Matrix

> **Phase:** P6 (QA Conformance) — Task 6.2
> **Author:** Gemini#Pro
> **Date:** 2026-07-05
> **Status:** PLAN + CRITERIA DELIVERED (LIVE proof F0-GATED)
> **Source:** docs/qa/runtime-conformance-plan.md §5

## Overview

C2 replay tests verify runtime stability across 11 real-world conversation patterns. Each scenario exercises different stress points of the prodex/L2 runtime, Smart Context, and continuation mechanisms.

## 11 Minimum Replay Scenarios

### Scenario 1: 30+ Turn Continuation

| Field | Value |
|---|---|
| **Description** | Long multi-turn conversation exceeding 30 turns with continuous context |
| **Capabilities exercised** | `continuation_mode`, `smart_context_mode`, event pipeline |
| **Risk profile** | 🔴 HIGH — context window pressure, affinity drift, state accumulation |
| **Procedure** | Start session; issue 30+ sequential coding tasks building on prior output; verify `previous_response_id` chain is unbroken |
| **Pass criteria** | Continuation affinity maintained across all turns; no profile switch; `affinity.overrode_fresh_selection=true` on turns 2-30+; no context corruption |
| **Smart Context interaction** | Shadow mode: measure token reduction %; Canary: verify no information loss across turns |
| **Context windows** | 32k (tight), 128k (comfortable), 200k (max) |
| **Evidence** | Turn count, affinity events, token usage per turn, any fallback events |

### Scenario 2: Repeated Build/Test Output

| Field | Value |
|---|---|
| **Description** | Multiple iterations of build→test→fix cycle with repetitive compiler/test output |
| **Capabilities exercised** | `smart_context_mode`, `continuation_mode` |
| **Risk profile** | 🟡 MED — repetitive content triggers aggressive Smart Context compression |
| **Procedure** | Run build, collect errors, fix, rebuild × 5+ cycles; verify no critical error is dropped by Smart Context |
| **Pass criteria** | All unique error messages preserved; Smart Context does not deduplicate distinct errors; build output integrity maintained |
| **Smart Context interaction** | High compression opportunity; must NOT drop unique errors |
| **Context windows** | 16k (stress), 32k |
| **Evidence** | Unique errors in vs out; compression ratio; any dropped content |

### Scenario 3: Compiler/Runtime Errors

| Field | Value |
|---|---|
| **Description** | Session with cascading compiler errors and runtime stack traces |
| **Capabilities exercised** | `smart_context_mode`, `continuation_mode` |
| **Risk profile** | 🟡 MED — stack traces contain file paths and line numbers critical for debugging |
| **Procedure** | Trigger compilation errors with deep stack traces; verify Smart Context preserves file:line references |
| **Pass criteria** | All file:line references preserved; stack trace order maintained; no truncation of error context |
| **Smart Context interaction** | Must preserve structural integrity of stack traces |
| **Context windows** | 16k (stress), 32k |
| **Evidence** | Stack trace integrity, file:line accuracy |

### Scenario 4: Large Diffs

| Field | Value |
|---|---|
| **Description** | Session generating or reviewing diffs exceeding 5000 lines |
| **Capabilities exercised** | `smart_context_mode`, `continuation_mode` |
| **Risk profile** | 🔴 HIGH — large diffs can exceed context window; Smart Context must summarize without losing changes |
| **Procedure** | Generate refactoring diff > 5000 lines across multiple files; verify all file changes are accounted for |
| **Pass criteria** | All modified files listed; no silent drops of changed files; diff headers preserved; line counts accurate |
| **Smart Context interaction** | May compress diff bodies but MUST preserve all file headers and change summaries |
| **Context windows** | 32k, 128k, 200k |
| **Evidence** | Files in diff vs files in Smart Context output; line count comparison |

### Scenario 5: Multi-File Refactor

| Field | Value |
|---|---|
| **Description** | Coordinated refactoring across 10+ files with cross-file dependencies |
| **Capabilities exercised** | `continuation_mode`, `smart_context_mode`, event pipeline |
| **Risk profile** | 🔴 HIGH — cross-file consistency; interrupted refactor leaves inconsistent state |
| **Procedure** | Rename a type/function used across 10+ files; verify all references updated; verify no partial application |
| **Pass criteria** | All references updated atomically; no orphaned references; compilation succeeds after refactor |
| **Smart Context interaction** | Must maintain cross-file reference graph in context |
| **Context windows** | 128k, 200k |
| **Evidence** | File list, reference count before/after, compilation result |

### Scenario 6: Changing Static Instructions

| Field | Value |
|---|---|
| **Description** | Mid-session change of system prompt / static instructions |
| **Capabilities exercised** | `continuation_mode`, `smart_context_mode` |
| **Risk profile** | 🟡 MED — instruction change must take effect without stale context bleeding through |
| **Procedure** | Start session with instruction A; after 5 turns, change to instruction B; verify behavior reflects B |
| **Pass criteria** | Post-change behavior follows instruction B; no stale instruction A behavior; continuation binding preserved |
| **Smart Context interaction** | Must NOT cache/compress away the instruction change |
| **Context windows** | 32k |
| **Evidence** | Behavioral diff before/after instruction change |

### Scenario 7: Missing/Corrupted Artifacts

| Field | Value |
|---|---|
| **Description** | Session references artifacts (files, tools) that are missing or corrupted |
| **Capabilities exercised** | `smart_context_mode`, `rotation_mode` (fallback) |
| **Risk profile** | 🟡 MED — graceful degradation required |
| **Procedure** | Reference a file that was deleted mid-session; verify error handling and no crash |
| **Pass criteria** | Graceful error reported; session does NOT crash; Smart Context exact fallback if artifact reference is malformed; no silent data fabrication |
| **Smart Context interaction** | `rewrite_decision.fallback_exact=true` if artifact reference cannot be safely rewritten |
| **Context windows** | 32k |
| **Evidence** | Error handling behavior, Smart Context fallback event |

### Scenario 8: Duplicate Tool Calls

| Field | Value |
|---|---|
| **Description** | Session with duplicate or near-duplicate tool call requests |
| **Capabilities exercised** | `continuation_mode`, `smart_context_mode`, event pipeline |
| **Risk profile** | 🟡 MED — deduplication must not drop intentional retries |
| **Procedure** | Issue same tool call twice in succession; verify both are executed (not silently deduplicated) |
| **Pass criteria** | Both tool calls executed; tool_call_ids are unique; Smart Context does not merge them; continuation state tracks both |
| **Smart Context interaction** | Must preserve distinct tool_call_ids even if payload is identical |
| **Context windows** | 32k |
| **Evidence** | tool_call_id pairs, execution results |

### Scenario 9: Noisy Binary Output

| Field | Value |
|---|---|
| **Description** | Commands producing binary-like or heavily encoded output (base64, hex dumps, minified JS) |
| **Capabilities exercised** | `smart_context_mode`, `continuation_mode` |
| **Risk profile** | 🟡 MED — binary content can confuse Smart Context tokenizer |
| **Procedure** | Run command producing base64-encoded data or hex dump > 1KB; verify context handling |
| **Pass criteria** | Binary content either preserved verbatim or clearly marked as truncated; no corruption of surrounding text context; no tokenizer crash |
| **Smart Context interaction** | May truncate binary blocks but must NOT corrupt adjacent content |
| **Context windows** | 16k (stress), 32k |
| **Evidence** | Output integrity check, adjacent content verification |

### Scenario 10: Failure Recovery

| Field | Value |
|---|---|
| **Description** | Session recovery after provider failure, timeout, or rate limit mid-response |
| **Capabilities exercised** | `rotation_mode` (fallback), `continuation_mode`, event pipeline |
| **Risk profile** | 🔴 HIGH — recovery must not lose context or produce duplicate output |
| **Procedure** | Simulate provider timeout after partial response; verify fallback and recovery |
| **Pass criteria** | Pre-commit fallback triggered (`phase=pre_commit`, `committed=false`); no duplicate output; continuation binding preserved or cleanly re-established; event `fallback_triggered` emitted |
| **Smart Context interaction** | Partial response handling — must not rewrite partial content |
| **Context windows** | 32k, 128k |
| **Evidence** | Fallback event, recovery behavior, output continuity |

### Scenario 11: Context Window Variants (16k/32k/128k/200k)

| Field | Value |
|---|---|
| **Description** | Same task executed across 4 different context window sizes |
| **Capabilities exercised** | ALL — stress test at each window size |
| **Risk profile** | 🔴 HIGH — behavior must be consistent across window sizes; Smart Context compression varies |
| **Procedure** | Execute a standardized coding task (e.g., implement a REST API) at each context window size; compare outputs |
| **Pass criteria** | Functional equivalence across all 4 sizes; Smart Context compression increases at smaller windows without information loss; no crash at 16k |
| **Smart Context interaction** | Critical — compression ratio should increase for smaller windows; exact fallback at any sign of information loss |
| **Context windows** | 16k, 32k, 128k, 200k |
| **Evidence** | Output comparison, compression ratios, any fallback events per window size |

## Summary Matrix

| # | Scenario | Risk | Capabilities | Windows | Status |
|---|---|---|---|---|---|
| 1 | 30+ turn continuation | 🔴 | continuation, smart_context | 32k/128k/200k | DEFINED |
| 2 | Repeated build/test | 🟡 | smart_context, continuation | 16k/32k | DEFINED |
| 3 | Compiler/runtime errors | 🟡 | smart_context, continuation | 16k/32k | DEFINED |
| 4 | Large diffs | 🔴 | smart_context, continuation | 32k/128k/200k | DEFINED |
| 5 | Multi-file refactor | 🔴 | continuation, smart_context | 128k/200k | DEFINED |
| 6 | Changing instructions | 🟡 | continuation, smart_context | 32k | DEFINED |
| 7 | Missing artifacts | 🟡 | smart_context, rotation | 32k | DEFINED |
| 8 | Duplicate tool calls | 🟡 | continuation, smart_context | 32k | DEFINED |
| 9 | Noisy binary output | 🟡 | smart_context, continuation | 16k/32k | DEFINED |
| 10 | Failure recovery | 🔴 | rotation, continuation | 32k/128k | DEFINED |
| 11 | Context window variants | 🔴 | ALL | 16k/32k/128k/200k | DEFINED |

## Execution Notes

> **⚠️ LIVE execution is F0-GATED.** All scenarios are DEFINED with pass/fail criteria. Execution requires:
> 1. P0 green (prodex binary available)
> 2. P3 green (Go integration complete)
> 3. Owner approval for the F0 validation window

## Evidence Format

Per `docs/qa/runtime-conformance-plan.md` §7:
- Command summary
- Result (pass/fail + detail)
- Event IDs (schema-valid)
- No raw secrets
- Gate tag (`C2-S01` through `C2-S11`)
- Mode (`dry-run` | `shadow` | `canary` | `live`)
- Scrubbed before write to `.deploy-control/evidence/`

# Audit Event Taxonomy

> **Phase:** 04-state-security (REQ-12)
> **Author:** Gemini#Pro
> **Date:** 2026-07-04
> **Status:** ACTIVE
> **Source:** Diligencias/04_STATE_SECURITY_P4.md §4.4

## Overview

All auditable actions in the Multica + prodex stack emit structured events. This taxonomy defines the 5 mandatory event types required by GATE P4.

## Event Types

### 1. `account_selection`

| Field | Description |
|---|---|
| **Trigger** | Multica selects a subscription account for a task |
| **Emitter** | Go L4 control plane (Multica) |
| **Payload** | `{ account_id (hashed), vendor, workspace_id, selection_reason, timestamp }` |
| **Redaction** | `account_id` MUST be hashed (SHA-256), never raw |
| **Frequency** | Once per task assignment |
| **Invariant** | Selection respects approved-accounts pool; fail-closed if pool empty |

### 2. `redeem_attempt`

| Field | Description |
|---|---|
| **Trigger** | prodex attempts to redeem/reset usage credits via `prodex redeem` |
| **Emitter** | Rust L2 runtime (prodex) |
| **Payload** | `{ account_id (hashed), vendor: "codex", attempt_result: "success|fail|skipped", timestamp }` |
| **Redaction** | No raw tokens; result only |
| **Frequency** | Rare — cold path, only during reset window |
| **Invariant** | Only fires for Codex; all other vendors emit `unsupported` |

### 3. `fallback_triggered`

| Field | Description |
|---|---|
| **Trigger** | Primary account quota exhausted; rotation to next account |
| **Emitter** | Rust L2 runtime (prodex) |
| **Payload** | `{ from_account (hashed), to_account (hashed), vendor, reason: "quota_exhausted|rate_limited|error", timestamp }` |
| **Redaction** | Both account IDs hashed |
| **Frequency** | Variable — depends on quota consumption rate |
| **Invariant** | Fallback is pre-commit only; never mid-request |

### 4. `continuation_binding`

| Field | Description |
|---|---|
| **Trigger** | A session is bound to a specific account for continuation affinity |
| **Emitter** | Rust L2 runtime (prodex) |
| **Payload** | `{ session_id, account_id (hashed), vendor, binding_mode: "response_id|session_id|cli_thread", timestamp }` |
| **Redaction** | `account_id` hashed; `session_id` is internal (non-secret) |
| **Frequency** | Once per session establishment |
| **Invariant** | Continuation affinity overrides rotation heuristic (ADR-001) |

### 5. `context_rewrite_decision`

| Field | Description |
|---|---|
| **Trigger** | Smart Context (prodex) decides to rewrite/filter context before sending to vendor |
| **Emitter** | Rust L2 runtime (prodex Smart Context engine) |
| **Payload** | `{ session_id, mode: "shadow|canary|live", tokens_before, tokens_after, reduction_pct, timestamp }` |
| **Redaction** | No content in payload — metrics only |
| **Frequency** | Every request when Smart Context is enabled |
| **Invariant** | Shadow mode = log-only (no actual rewrite); canary = A/B; live = active |

## Event Schema (common envelope)

```json
{
  "event_type": "account_selection | redeem_attempt | fallback_triggered | continuation_binding | context_rewrite_decision",
  "version": "1.0",
  "timestamp": "2026-07-04T23:00:00Z",
  "source": "multica-l4 | prodex-l2",
  "trace_id": "<uuid>",
  "payload": { }
}
```

## Storage

- Events are written to `PRODEX_AUDIT_LOG_DIR` (structured JSON lines)
- Events flow to Go L4 as observability/ledger (ADR-001: events do NOT trigger re-decision in Go)
- Retention: configurable; default 90 days

## Gate P4 Checklist
- [x] 5 event types defined
- [x] Each has: trigger, emitter, payload, redaction rule, frequency, invariant
- [x] All payloads redact secrets (hashed account IDs, no raw tokens)
- [x] Schema versioned (v1.0)

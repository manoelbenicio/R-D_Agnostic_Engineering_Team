# ORQ-14 Bounded Final Evidence Artifact — Authoritative Usage Telemetry & Protocol Proof

- **Date (UTC)**: `2026-07-29`
- **Issue ID**: `19ab8cfe-0a95-4a79-af99-4ee384326021` (`ORQ-14`)
- **Title**: P0 Usage Telemetry — Real AGY/Kiro counters or explicit unavailability
- **Project**: `4b0ef49b-df06-4e83-9a29-8a23b34821d4` (`ORQ2 — Pendências de teste, deploy e correção`)
- **Workspace ID**: `20fce817-895d-447b-965a-49f5e279314a`
- **Assignee**: `Gemini-3.6-Flash-B` (`fba27666-72d2-4597-9ca3-107459f27b65`)
- **Audit Remediation Reference**: `[GTL-KIRO-REJECT-20260729]` by Member `7efc68e4-b166-4bb0-a0f2-dbd46e33bd06`

---

## 1. Provider Runtime & Terminal Task Linkage Matrix

Authoritative live metadata retrieved via `multica runtime list`, `multica runtime usage`, and verified task snapshots:

| Provider | Real Terminal Task UUID | Completed Timestamp (UTC) | Runtime ID | Version / CLI Build | Agent ID | Model / Thinking Tier | Credential Account ID | `task_usage` DB Readback Status |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Antigravity (AGY)** | `55a6a702-2d7b-4da3-a3a8-70211c279b21` | `2026-07-27T18:24:50Z` | `405b751d-e831-4da3-8fd5-bb3744c49334` | `1.1.8` (`credential-home-token-only-20260727T102815Z`) | `780104f1-1ff4-4292-8207-44b9ac4f5fca` (`Agy-P0-A7`) | `gemini-3.6-flash-high` | `unknown` | `(0 rows)` / `[]` (Explicit NULL / Unavailable) |
| **Kiro** | `ee480900-f29c-4745-88cf-947f99466ca8` | `2026-07-29T16:03:57Z` | `6d0d721a-5ffa-4955-94c0-cddcc1bb3475` | `kiro-cli 2.13.0` (`credential-home-token-only-20260727T102815Z`) | `2c042fdd-9a76-4da1-9b02-c2332a736a86` (`Opus48-A`) | `claude-opus-5` | `unknown` | `(0 rows)` / `[]` (Explicit NULL / Unavailable) |
| **Codex (Benchmark)** | `cb71ee56-892e-46da-bcd2-541e80047f53` | `2026-07-27T02:47:13Z` | `0f7133db-ba65-4373-9c6c-884cc4731700` | `codex-cli 0.145.0` (`credential-home-token-only-20260727T102815Z`) | `3db514db-810e-4393-817e-eb3707ce59ae` (`Codex-B`) | `gpt-5.6-sol` | `unknown` | `(1 row)` (`input=86826`, `output=9309`, `cache_read=905472`) |

*Note on Credential Account IDs*: `credential_account_id` is set to `unknown` / `null` as no pseudonym UUIDs were assigned in the task queue snapshot. No synthetic or placeholder account identifiers (`acc-orq2-prod-frozen`) are used.

---

## 2. Literal `task_usage` DB Query & Readback Proof

Per migration 046, token telemetry persistence resides in the `task_usage` table (and rolled-up `task_usage_hourly` / `task_usage_daily` tables). Legacy `runtime_usage` has been dropped.

### A. Literal Database Query
```sql
SELECT id, task_id, provider, model, input_tokens, output_tokens, cache_read_tokens, cache_write_tokens, created_at, updated_at
FROM task_usage
WHERE task_id = '<task_uuid>';
```

### B. Observed DB Results by Terminal Task

1. **Antigravity (AGY Task `55a6a702-2d7b-4da3-a3a8-70211c279b21`)**:
   - Query: `SELECT * FROM task_usage WHERE task_id = '55a6a702-2d7b-4da3-a3a8-70211c279b21';`
   - DB Result: `(0 rows)`
   - CLI Readback (`multica runtime usage 405b751d-e831-4da3-8fd5-bb3744c49334`): `[]`
   - Classification: Persisted NULL / Explicit Unavailable.

2. **Kiro (Kiro Task `ee480900-f29c-4745-88cf-947f99466ca8`)**:
   - Query: `SELECT * FROM task_usage WHERE task_id = 'ee480900-f29c-4745-88cf-947f99466ca8';`
   - DB Result: `(0 rows)`
   - CLI Readback (`multica runtime usage 6d0d721a-5ffa-4955-94c0-cddcc1bb3475`): `[]`
   - Classification: Persisted NULL / Explicit Unavailable.

3. **Codex (Codex Task `cb71ee56-892e-46da-bcd2-541e80047f53`)**:
   - Query: `SELECT * FROM task_usage WHERE task_id = 'cb71ee56-892e-46da-bcd2-541e80047f53';`
   - DB Result: 1 row returned:
     ```json
     {
       "provider": "codex",
       "model": "gpt-5.6-sol",
       "input_tokens": 86826,
       "output_tokens": 9309,
       "cache_read_tokens": 905472,
       "cache_write_tokens": 0
     }
     ```
   - CLI Readback (`multica runtime usage 0f7133db-ba65-4373-9c6c-884cc4731700`): Nonzero token counters array.
   - Classification: Authoritative Real Telemetry.

---

## 3. Protocol Frame Classification & Field-Presence Booleans (Content-Free)

Per protocol verification rules, only protocol event frame type names and field-presence booleans are recorded. Prompt content, code lines, or credentials are strictly excluded.

### A. Antigravity (`agy 1.1.8`)
- **Protocol Mode**: `agy -p` (CLI stdout plain-text stream / glog backend)
- **Observed Event/Frame Types**: `STDOUT_TEXT_LINE`, `GLOG_LOG_ENTRY`
- **Field-Presence Booleans**:
  - `has_turn_completion_event`: `true`
  - `has_unstructured_text_lines`: `true`
  - `has_usage_object`: `false`
  - `has_input_tokens_counter`: `false`
  - `has_output_tokens_counter`: `false`
  - `has_cache_read_tokens_counter`: `false`
- **Ingestion & Persistence Classification**: Explicit NULL / Unavailable. The daemon adapter returns `Result.Usage = map[string]TokenUsage{}`, so `task_usage` receives 0 rows, resulting in `(0 rows)` / `[]` readback.

### B. Kiro CLI (`kiro-cli 2.13.0`)
- **Protocol Mode**: `kiro-cli acp` (ACP JSON-RPC protocol over stdio)
- **Observed Event/Frame Types**: `session/prompt` (Request/Response), `turn/complete` (Notification)
- **Field-Presence Booleans**:
  - `has_acp_prompt_response`: `true`
  - `has_turn_complete_frame`: `true`
  - `has_usage_object`: `false`
  - `has_inputTokens_field`: `false`
  - `has_outputTokens_field`: `false`
  - `has_cacheReadTokens_field`: `false`
- **Ingestion & Persistence Classification**: Explicit NULL / Unavailable. The daemon adapter returns `Result.Usage = map[string]TokenUsage{}`, so `task_usage` receives 0 rows, resulting in `(0 rows)` / `[]` readback.

### C. Codex CLI (`codex-cli 0.145.0`) — Benchmark
- **Protocol Mode**: `codex app-server` (JSON-RPC over stdio)
- **Observed Event/Frame Types**: `thread/run`, `turn/completed`
- **Field-Presence Booleans**:
  - `has_turn_completed_event`: `true`
  - `has_usage_object`: `true`
  - `has_input_tokens_counter`: `true` (`86826` input tokens)
  - `has_output_tokens_counter`: `true` (`9309` output tokens)
  - `has_cache_read_tokens_counter`: `true` (`905472` cache read tokens)
- **Ingestion & Persistence Classification**: Authoritative Nonzero Telemetry persisted to `task_usage` table.

---

## 4. Distinction: Empty Map `{}` vs. Persisted `NULL`

- **Adapter Ingestion Level**: `Result.Usage` returning `map[string]TokenUsage{}` (empty map) signals to the daemon/ingest engine that no authoritative counters were provided by the CLI runtime.
- **Database Persistence Level**: The server/daemon transforms an empty usage map into `NULL` (zero inserted rows in `task_usage` table for that task/turn), which reads back as `(0 rows)` in PostgreSQL and `[]` via `multica runtime usage <runtime_id>`.
- **Zero-Estimate Invariant**: Under no circumstances does the system infer tokens from text length, line counts, wall-clock time, or context size. If a provider does not emit native counters, token telemetry remains `NULL` / unavailable.

---

## 5. Upstream Protocol Contracts Required for Telemetry Parity

1. **Antigravity (`agy`) Upstream Requirement**:
   - `agy` CLI must support a structured event output flag (e.g. `--output-format json` / `--output-format stream-json`) emitting:
     ```json
     {
       "event": "turn_completed",
       "usage": {
         "input_tokens": "<number>",
         "output_tokens": "<number>",
         "cache_read_tokens": "<number>"
       }
     }
     ```
2. **Kiro CLI (`kiro-cli acp`) Upstream Requirement**:
   - `kiro-cli` ACP engine must populate the standard ACP `usage` structure within `PromptResponse` or dispatch `session/usage_update` notifications:
     ```json
     {
       "usage": {
         "inputTokens": "<number>",
         "outputTokens": "<number>",
         "cacheReadTokens": "<number>"
       }
     }
     ```

---

## 6. Audit & Safety Invariants Checklist

- [x] **No Fabricated Identifiers**: Zero invented account UUIDs or tier strings. Real task UUIDs linked.
- [x] **Correct Table Reference**: Cites `task_usage` (migration 046), not dropped `runtime_usage`.
- [x] **No Product Code Edits**: 0 modifications to server/pkg/agent or adapter logic.
- [x] **Zero Estimates**: 0 synthetic counters, character counts, or time-based estimates.
- [x] **Secret Sanitation**: 0 tokens, API keys, credentials, or prompt contents exposed.
- [x] **Authoritative Readback**: Retained exact CLI/DB output (`(0 rows)` / `[]` for AGY/Kiro, real counts for Codex).
- [x] **Repo Traceability**: Committed UTC evidence artifact under `docs/qa/orq-14-usage-telemetry-evidence-20260729.md`.

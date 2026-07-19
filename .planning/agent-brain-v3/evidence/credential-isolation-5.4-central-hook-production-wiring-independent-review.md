# Independent Review — 5.4 central-hook production wiring trace

- reviewer: Kiro / Opus-4.8, wave w8:p1 (independent of trace author w7:p1)
- date: 2026-07-18T23:02:00Z
- mode: READ-ONLY, scoped `rg`/`grep` + targeted reads only. No source/tests/shared-planning/spec/tasks/git/index/refs edit; no credentials/env/network/DB/services. Only this file created.

## Check-in / check-out
- CHECK-IN 2026-07-18T22:57:00Z — Kiro/Opus-4.8 w8:p1 — stream 5.4-CENTRAL-HOOK-WIRING-INDEPENDENT-REVIEW — READ-ONLY.
- CHECK-OUT 2026-07-18T23:02:00Z — DONE. Verdict below. Kiro TL adjudicates. No acceptance.

Reviewed: `credential-isolation-5.4-central-hook-production-wiring-trace.md` — SHA-256 `34007f5d72a338186ac38df480df0956332ee54b69d44864499794610d5e8527`. Also cross-referenced Codex review `45a64b19` = `credential-isolation-5.4-redact-core-claude-cloudpat-integration-manifest-codex-independent-review.md` (a manifest review; tangential to sink enumeration). `internal/logger/logger.go` = `f5f705c1d1433db10d84496ff6dcaf42b62dcad5a415239b9cb38cfcefd38010`.

## VERDICT: PARTIAL

The trace's **enumerated source claims are accurate** (PASS on those), but it (1) **under-states** that the shared-`logWriter` `redact.Text` coverage for all 13 adapters is **uncommitted (absent at HEAD)**, and (2) **omits an in-scope unhooked CLI `slog.Warn` bypass** that the task explicitly asked to verify. Both are material to "central-hook production wiring" → **PARTIAL**, not PASS.

## Verified-accurate claims

| Trace claim | Result |
|---|---|
| `logger.Init`/`NewLogger` = the only 2 production `slog.New` sites, both `ReplaceAttr: redact.SanitizeSlogAttr` | ✅ `logger.go:30 Init` (`:36` ReplaceAttr, `:38` SetDefault), `:44 NewLogger` (`:50` ReplaceAttr, `:52`); all other `slog.New` are `_test.go` |
| standard `log.*` = 0 production sinks | ✅ (consistent with prior audits) |
| **shared `logWriter`** defined **once** and used by all 13 adapters | ✅ `type logWriter`/`func newLogWriter` only at `claude.go:963/968`; adapters call `newLogWriter(...)` |
| `stderrTail.Tail()` returns the **raw** buffer (unredacted), embedded via `withAgentStderr` | ✅ `stderr_tail.go:46` `s.buf = append(...)` raw, `:57 Tail()`, `:60` returns raw string, `:67 withAgentStderr` |
| `acpProviderErrorSniffer.Write` applies **no** redaction; sits behind `io.MultiWriter(logWriter, providerErr)` | ✅ `hermes.go:1660` sniffer `Write` has no `redact.*`; derived message via `message()/terminalMessage()/promoteACPResultOnProviderError` |
| `models.go` `--list-models` discovery uses a raw `strings.Builder` stderr, no redaction | ✅ `models.go:906-907` `var stderr strings.Builder; cmd.Stderr = &stderr` |
| ultimate-sink limitations (sniffer derived string; `models.go` builder text) untraced | ✅ honestly flagged; I confirm both are construction-time bypasses whose downstream sinks are unverified |

## Critical caveat #1 — the 13-adapter `redact.Text` coverage is UNCOMMITTED (not at HEAD)

The trace states the shared `logWriter.Write` "definitely applies **both** layers" (`redact.Text` **and** the hook) and thereby "structurally fixed all 13 adapters." **That is true only in the dirty working tree, not at HEAD:**

- Working-tree `claude.go:980` = `w.logger.Debug(w.prefix + redact.Text(text))` ✅ (has `redact.Text`).
- **HEAD `claude.go`: `git cat-file blob HEAD:…/claude.go | grep -c 'redact.Text'` = `0`.** The `redact.Text` wrap is exactly the **uncommitted** 5.4 Claude patch (integration target `c7922b7b`, still governance-blocked).

⇒ **At HEAD, the 13 adapters' stderr is covered by the central slog hook ONLY (pattern-dependent), not by the explicit `redact.Text`.** The "double coverage / all-13 structural fix" is contingent on landing `c7922b7b`. The trace notes the fix is "recent/shared" but frames coverage in the present tense without flagging it is not yet committed — a material status gap.

## Critical caveat #2 — omitted unhooked CLI `slog.Warn` bypass (task-listed)

The task explicitly asked to verify "unhooked CLI `slog.Warn` bypass." I found it; **the trace does not enumerate it**:

- **`cmd/multica/cmd_id_resolver.go:42`** `slog.Warn("issue response missing identifier", "issue_id", id)`.
- **`cmd/multica` never calls `logger.Init()`** (only `cmd/{backfill_codex_usage_cache,backfill_task_usage_hourly,migrate,server}` call it). So the multica CLI's `slog.*` routes to the **Go built-in default handler with NO `SanitizeSlogAttr`** — a genuine central-hook bypass.
- Risk in this specific instance is **low** (the logged attr is `issue_id`, an identifier, not a secret), but **structurally** it is an unhooked production log sink, and any future secret-bearing `slog.*` in a non-`Init` CLI binary would leak unredacted. The trace's scope (handler construction + `pkg/agent` writers) missed `cmd/` package-level `slog.*` calls that run without `logger.Init`.

## On Codex review 45a64b19
`45a64b19` reviews the **integration manifest** (redact-core+claude+cloudpat), not the wiring-sink enumeration; it does not substitute for the wiring trace and does not change the two caveats above. My source verification is the primary evidence here.

## Recommended next actions (advisory; owner/TL)
1. State explicitly in the trace that the 13-adapter `redact.Text` coverage is **pending `c7922b7b`**; at HEAD it is central-hook-only (pattern-dependent, R-5.4-B).
2. Add the `cmd/multica` (and any non-`Init` CLI binary) unhooked `slog.*` sink to the 5.4 sink inventory; either call `logger.Init()` in those CLIs or route the specific warns through a hooked logger. Confirm no non-`Init` CLI logs a secret-bearing attr.
3. Trace the two flagged ultimate sinks (sniffer-derived error string; `models.go:907` discovery text) to closure or owner-accept R-5.4-B.

## Explicit non-claims
- Created only this file; scoped `grep`/`sha256sum`/`git cat-file blob` (read-only) at HEAD `b6571299`; nothing executed, no fix applied, no checkbox. No credentials/env/network/DB/services.
- Verdict is PARTIAL for the trace's completeness/status-accuracy; the specific source facts it enumerates are correct. No acceptance; 5.4 remains OPEN. Kiro TL adjudicates.

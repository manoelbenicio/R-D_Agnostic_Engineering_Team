# Credential-isolation 5.4 — codebase-wide residual closure matrix (ADVISORY, read-only)

Exact residual matrix for "nenhum segredo aparece em logs (sanitizeForLog)" built from static local
searches at HEAD `b6571299`. **Advisory only; no implementation; Kiro TL adjudicates.**

- Author: **Kiro/Opus-4.8, session `w8:p2`** (read-only). Task 5.4 = `tasks.md:34` `[ ]`.
- Inputs (hashes): codebase audit `2b060da6…dedb`; email review `3a3018b4…5529`; redact-core review
  `521cef31…8c12`; remaining-gaps independent review `5a927fbd…fc4c`; clean-room review `129025cc…24e4`.

## CHECK-IN 2026-07-18T22:10:00Z
Mode: READ-ONLY, static searches only. Sole deliverable = this file. Excluded (honored): no
implementation; no source/test/spec/tasks/shared-planning/git/index/ref edit; no credentials/env values;
no DB/network/services.

## Count methodology (reproducible, offline `grep`)
- **slog callsites (production, non-test):** `grep -rn 'slog\.(Info|Warn|Error|Debug)|logger\.(…)|\.logger\.(…)|\.Logger\.(…)' --include=*.go | grep -v _test.go` → **1088** level-calls; `grep -rn 'slog\.' | grep -v _test.go` → **877** `slog.*` refs. (The prior audit's **703** is a *different enumeration*; the delta is methodological — I include `logger.`/`.logger.`/`.Logger.` receivers — not a contradiction. Exact bulk count is immaterial to residual risk; the dynamic-value subset below is what matters.)
- **standard `log.*` Print/Fatal/Panic (production):** `grep -rnE '\blog\.(Print|Printf|Println|Fatal|Panic)'` → **0**.
- **`fmt.Print*` (production, non-test):** `cmd/` = **59** (operator stdout), non-cmd = **5** (all `internal/service/email.go`).
- **Dynamic secret-bearing sinks:** `grep -rnE '"(body|response|output|stderr|content|raw|args|…)"' … | grep -iE 'slog\.|logger|\.(Debug|Info|Warn|Error)\('`.

## Cross-cutting sanitizer fact (verified)
All production loggers wire `redact.SanitizeSlogAttr` — `internal/logger/logger.go:36,50` + `slog.SetDefault`
at `cmd/server/main.go:123`. `SanitizeSlogAttr`: sensitive **key** ⇒ `[REDACTED]`; string value ⇒ `Text()`
pattern scan; `[]string`/`any` ⇒ `SanitizeForLog` (each element `Text()`). ⇒ **every dynamic value below is
at least pattern-dependently (`Text()`) redacted**; none is a raw unprotected sink. "Pattern-dependent" =
residual class **R-5.4-B** (a secret in an unmatched shape/key could survive).

## Residual matrix (dynamic secret-bearing sinks)

| # | Production sink / callsite | Dynamic secret-bearing input | Current sanitizer coverage | Executed test coverage | Residual risk | Minimal closure action |
|---|---|---|---|---|---|---|
| A | `internal/handler/auth.go:656` `slog.Error(…, "body", tokenBody)` | Google OAuth non-200 HTTP body | key `body`→`Text()` (pattern-dep); status-guard excludes 200 bodies | none site-level (generic `TestRedactCredentialFieldsInJSONBody` only) | R-5.4-B | disjoint handler test asserting `body` redaction end-to-end |
| B | `internal/auth/cloud_pat.go:359` `slog.Warn(…, "body", snippet)` | Cloud/Fleet PAT-verify non-200 body (≤512B) | key `body`→`Text()` (pattern-dep) | none | R-5.4-B (co-equal to A; **not enumerated by prior sweeps**) | enumerate + analogous disjoint test |
| C | `pkg/agent/claude.go` `logWriter.Write` | raw Claude subprocess **stderr** | **explicit `redact.Text` (structural)** + hook | 6 tests ×20 + race (clean-room `129025cc…`) | LOW (structural) — **pending cross-family review** | complete cross-family Claude review; push atomic 2-hunk delta `c7922b7b` |
| D | **argv "agent command"** — `antigravity.go:73, cline.go:68, codebuddy.go:107, copilot.go:213, cursor.go:41, gemini.go:38, hermes.go:64, kimi.go:61, kiro.go:68, openclaw.go:79, opencode.go:85, pi.go:210, qoder.go:99` (**13 adapters**) log **raw** `args` | CLI flags may carry `--api-key/--token/--config/--url/--model-provider/…` values | key `args` `[]string`→`Text()` per element (**pattern-dep only**). **Only `claude.go:66` uses structured `safeAgentArgvForLog`** (flag-aware) | none per-adapter | **R-5.4-B — LARGEST surface** | apply the `claude.go` structured argv projection (`safeAgentArgvForLog`) to all 13 adapters, **or** owner-accept the pattern-dep bound |
| E | `internal/daemon/daemon.go:4542` `taskLog.Error("agent error", "content", msg.Content)` | agent-produced error content | key `content`→`Text()` (pattern-dep) | none site-level | R-5.4-B | align with the structural `ReportMessages` path (redact `msg.Content` before logging) or owner-accept |
| F | `internal/daemon/auto_update.go:163,167`; `daemon.go:2297,2305` `"output"` | CLI/upgrade subprocess stdout/stderr | key `output`→`Text()` (pattern-dep) | none | R-5.4-B (low likelihood) | owner-accept, or explicit `redact.Text` before log |
| G | `internal/daemon/execenv/git.go:108,116` `"output"` | git command stdout | key `output`→`Text()` (pattern-dep) | none | LOW (git output rarely secret) | owner-accept |
| H | `internal/handler/agent_template.go:503` `"raw"` | raw agent-template parse input | key `raw`→`Text()` (pattern-dep) | none | R-5.4-B (low) | owner-accept |

## Already-covered / accepted (no residual action)
| Sink | Coverage | Status |
|---|---|---|
| Agent-output persist+broadcast — `internal/handler/daemon.go:2226-2228` | **explicit** `redact.Text(msg.Content/Output)` + `redact.InputMap(msg.Input)` before persist/broadcast (structural) | accepted (EV-CREDISO-5.4 broadcast) — LOW |
| Email config prints — `internal/service/email.go` (5 `fmt.Print`) | config-only (hostname/SMTP mode/relay host:port/from/TLS/DEV-notice); **no secret value** (dev-code raw print removed) | accepted (EV-CREDISO-5.4-EMAIL `3a3018b4…`) — none |
| Redact core — `pkg/redact` | `SanitizeSlogAttr`/`SanitizeForLog`/`IsSensitiveKey` + 25 parent tests | core ACCEPT (`521cef31…`) — **provenance-gated** (unattributed producer / missing Gemini review / unpinned EV, see `dbf7033b…`) |
| Bulk **structured** slog attrs (~703–1088) | key-based `IsSensitiveKey`→`[REDACTED]` + string→`Text()` | LOW (structured) — covered |
| Carrier types `runtimeenv.StableSecret`, `OpenclawGatewayPin` | `String()/GoString()/MarshalJSON()` redact by construction | covered |
| Standard `log.*` | **0 callsites** | none |
| CLI operator stdout — `cmd/` (59 `fmt.Print`, incl. webhook URL) | none by design (operator's own resource) | **out of `sanitizeForLog`/"logs" scope** — owner scope confirmation |

## Absolute task-language closure vs bounded risk-based closure (kept separate)
- **Absolute ("nenhum segredo aparece em logs", literal): NOT met.** Redaction of the **dynamic value**
  sinks (A,B,D,E,F,G,H) is **pattern-dependent** (`Text()` regex/heuristics), so a secret in an
  unrecognized shape/key **can** survive. Only the **structural** sinks (C claude stderr; the ReportMessages
  persist path) and **key-based** structured attrs are absolute. Argv (D) is the largest gap because 13
  adapters log raw args with only pattern-scan.
- **Bounded risk-based closure: ACHIEVABLE.** Every dynamic sink routes through a redacting logger
  (verified); no `log.*`; no `fmt.Print` of secrets in logs; agent-output persist path is structurally
  redacted. The remaining exposure is exactly **R-5.4-B pattern-dependency**, concentrated in **D (argv) >
  A/B (external bodies) > E/F (agent/subprocess output) > G/H (git/template)**. Closable on a risk basis by:
  (1) extending structured argv redaction to all adapters (closes D — highest value); (2) completing the
  cross-family Claude review + push (C); (3) disjoint tests for A/B; (4) structural redaction for E; (5)
  owner-accepting R-5.4-B for F/G/H with bounded wording; (6) owner scope ruling for CLI stdout.

## Recommended honest closure statement (advisory, for TL)
> "5.4 is **risk-closed** on the log surface: all sinks route through a redacting logger; structural
> redaction covers agent-output persistence and Claude stderr; **residual R-5.4-B pattern-dependency**
> remains for external bodies, agent argv (13 adapters), and subprocess output — bounded and owner-accepted
> — with the recommended structural fix being to extend the flag-aware argv projection to all adapters."
> This is **not** an unqualified 'absolute' closure.

## Non-claims
- Advisory matrix only; no acceptance; no checkbox; no code/test/spec/tasks/shared-planning/git/index/ref
  change; no credentials/env values; no DB/network/services. Counts/paths from static `grep` at HEAD
  `b6571299` (not executed). Kiro TL adjudicates; reviewer ≠ adjudicator.

## CHECK-OUT 2026-07-18T22:13:00Z — DONE
Only this file created. Everything else unchanged.

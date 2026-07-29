# Independent Review — 5.4 codebase-wide residual closure matrix

- reviewer: Kiro / Opus-4.8, wave w8:p1 (independent; distinct pane from matrix author w8:p2)
- date: 2026-07-18T22:38:00Z
- mode: READ-ONLY except this artifact. No source/tests/shared-planning/spec/tasks/git/index/refs edit; no credentials/env/network/DB/services. No acceptance.

## Check-in / check-out
- CHECK-IN 2026-07-18T22:31:00Z — Kiro/Opus-4.8 w8:p1 — stream CREDISO-5.4-RESIDUAL-CLOSURE-MATRIX-INDEPENDENT-REVIEW — READ-ONLY.
- CHECK-OUT 2026-07-18T22:38:00Z — DONE. Verdicts below. Task 5.4 remains OPEN. Kiro TL adjudicates; reviewer ≠ adjudicator. Not accepted.

Reviewed: `credential-isolation-5.4-codebase-wide-residual-closure-matrix.md` — SHA-256 `97fbbc24fcf7783753486f926e9279bdd164c33edfbb64f3f113b9cf529402bf` (matches asserted `97fbbc24…`; stable). Task 5.4 = `tasks.md:34` **`[ ]` OPEN** (preserved).

## VERDICTS (separate)

- **Count methodology reproducibility: PASS** (with documented scope deltas the matrix already discloses).
- **Residual classes A–H: PASS** (spot-verified real callsites; B and D correct prior omissions) — with **one minor undercount** and a **completeness caveat** below.
- **Central-hook pattern vs structural distinction: CORRECT.**
- **Absolute vs bounded closure: CORRECT** — absolute NOT met; bounded risk-closure achievable. Consistent with the prior Kiro/Sonnet PARTIAL critique.

## Methodology reproduction (independent, offline grep at HEAD b6571299)

- **Central sanitizer wiring — CONFIRMED (structural fact):** production `slog.New` sites = **exactly 2**, both in `internal/logger/logger.go` (`:38` `SetDefault(slog.New(handler))`, `:52` component logger), and `handler` is built with `ReplaceAttr: redact.SanitizeSlogAttr` (`:36`, `:50`); `cmd/server/main.go:123` `slog.SetDefault`. All 104 other `slog.New` are in `_test.go` (discard/recorder handlers). ⇒ every production `slog.*` value routes through `SanitizeSlogAttr`.
- **Bulk count:** I reproduce `slog.(Info|Warn|Error|Debug)` = **704 / 83 files** (incl. tests). The prior audit's **703 / 82** and the matrix's own **877 / 1088** are different enumerations (test inclusion; `logger.`/`.logger.`/`.Logger.` receivers). The ±1 and the wider counts are **methodological, not contradictory** — and the matrix is right that the exact bulk number is immaterial; the **dynamic-value subset is the risk.** ✅
- **`log.*` (stdlib) production Print/Fatal/Panic = 0** — reproduced. ✅
- **`fmt.Print*` scoping:** the matrix's **59 (cmd) + 5 (email.go)** is a *sink-scoped* count. A naive `fmt.(Print|Printf|Println|Fprint*)` net returns ~**307** because it also catches `Fprint` to buffers/`strings.Builder`/`http.ResponseWriter` (not log sinks). The matrix's narrower scoping is **more precise**; I could not byte-reproduce 59+5 without replicating that sink-scoping (bound) — no false positive, and the email.go sink was separately accepted (`EV-CREDISO-5.4-EMAIL`).

## Residual-class validation (spot-verified real callsites)

| Class | Matrix claim | Independent result |
|---|---|---|
| A | `auth.go:656` `slog.Error(…, "body", tokenBody)` | ✅ confirmed (matches my prior OAuth body-log design review) |
| B | `cloud_pat.go:359` `slog.Warn("cloud_pat: verify returned non-200", "status", …, "body", snippet)` | ✅ **confirmed verbatim** — a real co-equal-to-A sink the matrix correctly notes prior sweeps missed |
| C | `claude.go` `logWriter.Write` structural `redact.Text` + hook | ✅ consistent with the 5.4 clean-room atom I reviewed (structural, LOW, pending cross-family) |
| D | claude uses structured `safeAgentArgvForLog`; other adapters log **raw** `args` | ✅ `claude.go:66` = `safeAgentArgvForLog`; `cline.go:68`/`gemini.go:38`/`kimi.go:61`/`qoder.go:99` = raw. **But see undercount below.** |
| E | `daemon.go:4542` `taskLog.Error("agent error", "content", msg.Content)` | ✅ confirmed verbatim |
| F/G/H | `output`/`output`/`raw` pattern-dep, graded LOW | ✅ locations plausible; LOW grading not inflated (no false positive) |

## Omissions / false positives

1. **Minor undercount in class D.** The matrix lists **13** raw-args adapters. My census finds **15** production files logging `"args"`: antigravity, **claude** (structured), cline, codebuddy, copilot, cursor, gemini, hermes, kimi, kiro, openclaw, opencode, **opencode_mcp**, pi, qoder. Excluding claude (structured) leaves **14** raw candidates — the matrix's 13-list **omits `opencode_mcp.go`.** Exact next action: verify `opencode_mcp.go` — if it logs raw args it is the **14th** class-D sink; if it reuses a structured projection, note the exception. Either way the "13 adapters" figure needs a ±1 correction.
2. **Completeness caveat (scope wording).** A–H is a **representative** dynamic-sink census keyed on specific attr names (`body|response|output|stderr|content|raw|args|…`). It is **not proven-exhaustive**: other value-bearing keys (`msg`/`data`/`payload`/`result`, or `error` attrs whose wrapped message embeds a secret) fall under the **same R-5.4-B pattern-dependency bound** but are not individually enumerated. "Codebase-wide" should be read as "central-hook fact + largest/representative dynamic surfaces," not "every possible sink proven."
3. **No false positives found.** G (git output) / H (template raw) / F (upgrade output) are correctly graded LOW; A/B/D/E correctly R-5.4-B. The matrix does not inflate risk.

## Central-hook pattern coverage vs structural guarantee — CORRECT

The matrix draws the right line: `SanitizeSlogAttr` gives **universal PATTERN coverage** (sensitive-key ⇒ `[REDACTED]`; every string value ⇒ `Text()` regex), but that is **not a STRUCTURAL guarantee** — a secret in an unrecognized shape/key can survive (`R-5.4-B`). Only **structural** sinks are absolute: C (claude explicit `redact.Text`), the `ReportMessages`/agent-output persist path (`redact.Text`/`InputMap`), and key-based redaction of known-sensitive keys. This distinction is the crux and the matrix states it accurately.

## Absolute vs bounded closure — CORRECT (5.4 stays OPEN)

- **Absolute ("nenhum segredo aparece em logs", literal): NOT met** — dynamic value sinks (A,B,D,E,F,G,H) are pattern-dependent; a novel-shaped secret can survive. Argv (D) is the largest surface. Confirmed.
- **Bounded risk-closure: ACHIEVABLE** — all sinks route through a redacting logger (verified), no `log.*`, no secret `fmt.Print` in logs, agent-output persist is structural. Residual = exactly R-5.4-B pattern-dependency. Matches the prior PARTIAL critique; nothing here upgrades 5.4 to PASS.

## Exact next actions (endorsed + corrected)

1. **D (highest value):** apply the flag-aware `safeAgentArgvForLog` projection to **all** raw-args adapters — **14, incl. `opencode_mcp.go`** (verify), not 13 — or owner-accept R-5.4-B for argv with bounded wording.
2. **A/B:** add disjoint redaction proofs. Per my prior OAuth review, the smallest offline seam is a `pkg/redact` value-content test (secret under a benign `"body"` key) + a static call-site assertion that the sink uses `slog` — analogous for `cloud_pat.go:359`.
3. **C:** complete the cross-family Claude review and integrate the 2-hunk atom (per the 5.4 root-integration manifest review).
4. **E:** structurally redact `msg.Content` before `taskLog.Error`, aligning with the `ReportMessages` persist path.
5. **F/G/H:** owner-accept R-5.4-B with explicit bounded wording, or add `redact.Text` at the sink.
6. **CLI stdout (cmd/, 59 `fmt.Print`):** owner scope ruling — operator's own stdout is arguably outside "logs"/`sanitizeForLog`.
7. Adopt the matrix's **honest closure statement** ("risk-closed, not absolute") verbatim; do not represent 5.4 as unqualified closure.

## Explicit non-claims
- Created only this file. No source/test/spec/tasks/shared-planning/git/index/ref edit; no credentials/env/network/DB/services. No checkbox change; **5.4 remains OPEN.**
- Counts/callsites verified statically at HEAD `b6571299` (not executed); I spot-verified A/B/C(sample)/D(sample)/E, not every one of the ~700 bulk attrs.
- No acceptance, EV, or owner policy issued. Kiro TL adjudicates.

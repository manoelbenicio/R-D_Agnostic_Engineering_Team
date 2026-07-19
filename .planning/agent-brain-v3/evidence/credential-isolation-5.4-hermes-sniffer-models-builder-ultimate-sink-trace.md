# Ultimate-sink trace: acpProviderErrorSniffer + models.go bare stderr builder

**Author:** Kiro/Sonnet, pane `w7:p1` — read-only trace only.
**Date:** 2026-07-18T19:34:56-03:00
**Adjudication authority:** Kiro TL adjudicates. Advisory only — no
implementation performed. No product/test/shared-planning/spec/task/git/
index edit, no credential/env value read, no network/DB/live-provider
action.

## Golden Rule check-in / check-out

- **Check-IN:** 2026-07-18T19:34:56-03:00 — claimed scope: trace the two
  limitations flagged in
  `credential-isolation-5.4-central-hook-production-wiring-trace.md`
  (`acpProviderErrorSniffer` and its 5 sharing adapters; `models.go:907`'s
  bare `strings.Builder`) to every ultimate consumer/log/error/persist
  path, using only scoped `rg`-equivalent/read tools. One output file only.
- Excluded (honored): no product/test/shared-doc/spec/task edit; no git/
  index mutation; no credential/env value read; no DB/network/live-provider
  action.
- **Check-OUT:** 2026-07-18T19:50:00-03:00 — DONE; findings below; no files
  other than this one were created or modified.

## Source hashes (files read/classified in this trace)

Not independently re-hashed in this pass (this is a control-flow trace, not
a hash-integrity audit); exact line numbers are pinned instead, against
files already hash-verified in the immediately prior wiring-trace document
this session (`pkg/agent/hermes.go`, `pkg/agent/models.go`,
`internal/daemon/daemon.go`, `internal/service/task.go`). No drift is
assumed — if TL requires fresh hashes for this specific artifact, that is a
follow-up, not performed here to avoid a duplicate check with no new reason
to suspect drift.

## Finding 1 — `acpProviderErrorSniffer`: traced end-to-end, genuine partial gap identified

### Construction and wiring (confirmed, `pkg/agent/hermes.go`)

```
hermes.go:109   providerErr := newACPProviderErrorSniffer("hermes")
hermes.go:121   stderrSink := io.MultiWriter(newLogWriter(b.cfg.Logger, "[hermes:stderr] "), providerErr)
hermes.go:419   finalStatus, finalError = promoteACPResultOnProviderError(finalStatus, finalError, finalOutput, providerErr)
```
Five adapters instantiate the same sniffer type against the same
`io.MultiWriter` pattern (confirmed in the prior wiring trace, re-cited
here for completeness): `hermes.go:121`, `cline.go:108`, `kimi.go:107`,
`kiro.go:100`, `qoder.go:132`.

### What the sniffer actually captures (the material question)

```
hermes.go:1631  var acpErrorDetailRe = regexp.MustCompile(`(?:Error:|detail:|Details:)\s*(.+)`)
```
**This regex captures everything after the literal `Error:`/`detail:`/
`Details:` prefix, unbounded, to end of line.** This is the load-bearing
finding: any upstream CLI error line shaped like `Error: <anything>` —
including, plausibly, `Error: invalid API key <key>` or
`Error: Authorization header rejected: Bearer <token>` — would have its
*entire remainder* captured into `s.lines`, with **no redaction applied at
capture time** (`Write()`, `hermes.go:1660-1690`, contains no call to
`redact.*` anywhere in its body — confirmed by direct read).

`acpErrorHeaderRe`/`acpTerminalErrorRe` (`hermes.go:1626`,`:1639`) gate
*which* lines are captured (must match `BadRequestError`/
`AuthenticationError`/`RateLimitError`/`HTTP [0-9]{3}`/etc.) but do **not**
constrain *what* is captured once a line matches — `acpErrorDetailRe`'s
capture group `(.+)` is the actual extracted text, and it is unbounded.
**`AuthenticationError` is explicitly one of the gating terms** — this is
exactly the error class most likely to echo a credential/token/key in a
real CLI's detail message.

### Downstream path (traced, confirmed)

```
hermes.go:419   finalStatus, finalError = promoteACPResultOnProviderError(...)
hermes.go:218/248/268/275/313  resCh <- Result{Status: finalStatus, Error: finalError, ...}
```
`finalError` (from `sniffer.terminalMessage()`, prefixed
`"<provider> provider error: "` per `messageLocked()`, `hermes.go:1741-1751`)
becomes **`Result.Error`** — the exact same field already traced end-to-end
in the prior wiring-trace document for the `stderrTail.Tail()` case. Its
consumers are therefore **identical** to that already-traced chain:

| Sink | Coverage (re-confirmed by direct read in this pass) |
|---|---|
| `daemon.go:3815` `taskLog.Warn(..., "error", result.Error)` | `"error"` is not a sensitive key; falls to `SanitizeSlogAttr`'s `slog.KindString` branch → `redact.Text` runs. **Covered, but pattern-dependent only** — `redact.Text`'s fixed pattern set (bearer/AWS/JWT/`KEY=value`/JSON-field literals) may not match every real-world CLI credential shape an `AuthenticationError` detail could echo. |
| `daemon.go:3853` `"agent_error", result.Error` | Same as above. **Covered, pattern-dependent only.** |
| `daemon.go` → `d.client.FailTask(...)` → `internal/service/task.go:1459,1471` `redact.Text(errMsg)` | Explicit, unconditional `redact.Text` call, confirmed by direct read of `task.go` in this pass (not re-derived from the prior artifact's citation alone — independently re-opened and read). **Covered by the stronger explicit-call mechanism**, same pattern-set limitation as above, but not a bypass. |

### Verdict on Finding 1

**No full, unmitigated bypass — but a real, characterizable gap exists.**
The sniffer's own `Write()` performs zero redaction at capture time, and
its capture regex is unbounded for exactly the error class
(`AuthenticationError`) most likely to echo a credential. However, every
traced downstream sink applies at least the pattern-scanning layer
(`redact.Text`, either via `SanitizeSlogAttr` or the explicit
`task.go` call) before the content reaches a log line or a persisted
comment. **The gap is identical in kind to the already-disclosed
"pattern-dependent coverage" residual (R-5.4-B and its extensions this
session) — this trace's contribution is confirming there is no *additional*,
*unmitigated* sink for this specific derived string, while sharpening why
this particular capture site is higher-risk than average** (unbounded
capture, explicitly gated on an authentication-error class).

## Finding 2 — `models.go:907` bare `strings.Builder`: traced end-to-end, LOWER risk than initially flagged

### Construction (confirmed, `pkg/agent/models.go`)

```
models.go:906-907  var stderr strings.Builder
                    cmd.Stderr = &stderr
models.go:908       stdout, err := cmd.Output()
models.go:911-913   text := string(stdout)
                     if strings.TrimSpace(text) == "" { text = stderr.String() }
models.go:914        return parsePiModels(text), nil
```
This is `discoverPiModels`, invoked only via `pi --list-models` model
catalog discovery.

### The critical fact this trace establishes: `stderr`'s raw content is NEVER returned as an `error`

Reading the full function body precisely: the only early-return path is
```go
if err != nil && len(stdout) == 0 && stderr.Len() == 0 {
    return []Model{}, nil   // stderr.Len()==0 here by construction — nothing to leak
}
```
**Every other path falls through to `return parsePiModels(text), nil` — a
nil error.** `discoverPiModels` **never returns a non-nil error to its
caller under any condition traced in this pass.** This directly changes
the risk profile from the prior trace's flagged-but-untraced status.

### Where does `text`/`stderr.String()` actually go?

Only into `parsePiModels(text)` (`models.go:914`), which:
- Scans `text` line-by-line (`bufio.Scanner`).
- Explicitly detects and **skips** diagnostic/warning lines via
  `isPiDiscoveryNoise(line)` (confirmed by direct read, `models.go:901-905`
  region and the `parsePiModels` body) — this guard exists specifically to
  prevent stderr-interleaved prose (the code comment cites `"Warning: No
  models match pattern..."`) from being mis-parsed into bogus `Model`
  entries.
- Returns only `[]Model{Provider, ID, ...}` — a small, structurally bounded
  type (confirmed: no free-text field on `Model` that could carry an
  arbitrary captured error line).

### Downstream of `discoverPiModels`

```
models.go:227           return discoverPiModels(ctx, discoveryExecutablePath)  (inside ListModels' provider switch)
daemon.go:2069          models, err := agent.ListModels(discoveryCtx, rt.Provider, entry.Path)
daemon.go:2070-2077     if err != nil { ...; d.reportModelListResult(ctx, rt, requestID, map[string]any{"status":"failed","error": err.Error()}) }
```
`daemon.go:2076`'s `err.Error()` **is** sent over the network unredacted
via `d.client.ReportModelListResult(...)` (`daemon.go:2207-2209`) — no
`redact.*` call anywhere on this path (confirmed by direct read; this is a
genuinely different sink from the slog/FailTask chain, and it is a real,
uncovered path). **But** because `discoverPiModels` never returns a non-nil
`err` (per the trace above), **this specific discovery function cannot
actually reach that uncovered `err.Error()` sink with its stderr content**
— the raw `stderr.String()` is fully absorbed into `text`→`parsePiModels`,
which only ever returns a nil error alongside a structurally bounded
`[]Model` slice.

### Verdict on Finding 2

**No exploitable bypass found for `discoverPiModels` specifically.** The
initial flag in the prior wiring trace ("flagged as a limitation" pending
sink-tracing) is resolved: the raw stderr content has exactly one
consumer (`parsePiModels`), which cannot leak it into a free-text field,
and the function's only error return path is unreachable while
`stderr.Len() > 0`. **However, the uncovered `err.Error()` → network sink
at `daemon.go:2076` is real and unredacted in general** — it is simply not
reachable *from this specific discovery function's stderr content*. Any
other current or future discovery function that (a) uses a bare,
unredacted stderr builder and (b) actually returns that content as a
non-nil `error` would hit this same uncovered network sink. This trace
confirms `models.go`'s only such builder (`:906-907`) does not do so today,
but the sink itself (`daemon.go:2069-2077`, and the identical pattern at
`daemon.go:2140,2159` for other discovery error paths, confirmed present in
the earlier wiring trace's raw grep output though not deep-traced there)
remains a structurally uncovered `redact.*` gap for **whatever** error
value a discovery function does return.

## Summary

| Path | Credential-bearing content can reach the capture point? | Redacted before its ultimate sink? | Genuine bypass? |
|---|---|---|---|
| `acpProviderErrorSniffer.Write` → `terminalMessage()` → `Result.Error` → (`daemon.go` slog attrs \| `task.go` comment) | **Yes** — unbounded `Error:`/`detail:`/`Details:` capture, explicitly gated on `AuthenticationError` | **Yes**, at every traced sink (pattern-scan via `SanitizeSlogAttr`, or explicit `redact.Text` in `task.go`) | **No full bypass** — real gap is pattern-dependent coverage, not absence of coverage |
| `models.go:907` bare `strings.Builder` → `discoverPiModels` → `parsePiModels` → `[]Model` | Content reaches `parsePiModels`, but is filtered/discarded there, never returned as free text or as a non-nil error | N/A — content never reaches a sink because it is absorbed, not propagated | **No bypass for this specific function** |
| `daemon.go:2069-2077` `err.Error()` → `ReportModelListResult` (network) | **Structurally uncovered sink** — no `redact.*` call anywhere on this path | **No** | **Yes, in general** — but not reachable via the two paths this trace was asked to check; a latent risk for any *other* discovery function whose error return could carry raw subprocess output |

## Minimal structural tests / remediation (advisory only — not implemented)

1. **For the sniffer's capture (`hermes.go:1631` `acpErrorDetailRe`):** the
   smallest structural test would feed a synthetic
   `"Error: invalid api_key sk-SYNTHETIC..."` line through
   `acpProviderErrorSniffer.Write` + `terminalMessage()` and assert the
   captured `s.lines`/returned message *itself* contains the raw sentinel
   — this would be a **red test today** (expected to fail, demonstrating
   the gap), which is the correct way to characterize this as a known,
   accepted-for-now gap rather than a silently assumed-safe path. The
   minimal remediation, if TL chooses to close it, is a one-line change:
   wrap the captured `line`/final `detail` through `redact.Text(...)`
   inside `messageLocked()` (`hermes.go:1741`) before building the
   prefixed message — mirroring exactly the fix already applied to
   `pkg/agent.logWriter.Write` earlier this session. This would not touch
   any of the 5 adapters' own files, since they all share the one
   `hermes.go` definition (confirmed in the prior wiring trace).
2. **For the general uncovered network sink (`daemon.go:2069-2077` and its
   siblings at `:2140,2159`):** the smallest structural test would
   construct a fake discovery function returning a synthetic
   credential-shaped `error`, call the `daemon.go` handler, and assert the
   `map[string]any{"error": ...}` payload sent to `ReportModelListResult`
   does not contain the sentinel — also expected **red** today. Minimal
   remediation, if chosen, is wrapping `err.Error()` in `redact.Text(...)`
   at each of the 3 confirmed call sites before building the `payload` map
   — a `daemon.go`-local, non-shared-hotspot-safe change in isolation, but
   `daemon.go` is a shared hotspot per this session's repeated ownership
   findings, so any actual edit needs its own separately authorized,
   conflict-checked task (not performed here).
3. **Neither remediation is proposed as urgent/blocking** by this trace —
   both are the same class of "pattern-dependent/structurally-uncovered"
   residual already disclosed multiple times this session (R-5.4-A/B and
   their extensions), not a newly-discovered active exploit. This trace's
   job was to determine reachability and characterize risk precisely, which
   it has done; whether to prioritize remediation is Kiro TL's call.

## Limitations (disclosed)

- This trace did not re-examine `cline.go`/`kimi.go`/`kiro.go`/`qoder.go`
  individually beyond confirming their identical wiring pattern (already
  established in the prior wiring-trace document) — the sniffer's
  capture/message logic is entirely inside the one shared `hermes.go`
  definition, so per-adapter re-tracing would not surface new information.
- Did not verify whether any *other* current discovery function (beyond
  `discoverPiModels`) also uses a bare unredacted builder and *does* return
  it as a non-nil error — the scoped search in this pass (`strings\.Builder|cmd\.Stderr\s*=`
  within `models.go`) found only the one instance plus one unrelated
  `strings.Builder` (`models.go:775`, non-stderr context) and one
  `io.Discard` (`models.go:1174`, safe). Did not extend this search to
  every other `pkg/agent/*.go` file beyond what the prior wiring-trace
  document already covered for the main stderr-wiring pattern.
- Did not re-hash any file in this pass (see Source hashes section above).

## Non-claims

- Does not implement any fix, test, or guard.
- Does not assert either finding is an active, exploited leak — both are
  characterized as reachable-but-mitigated (Finding 1) or
  reachable-but-not-currently-propagated (Finding 2).
- Does not decide whether remediation is required before any 5.4 checkbox
  action — that is Kiro TL's adjudication.
- No product/test/shared-planning/spec/task/git/index/credential/env/
  network/DB/service action was taken.

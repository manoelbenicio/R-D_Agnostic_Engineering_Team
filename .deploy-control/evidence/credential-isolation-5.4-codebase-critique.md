# Independent Critique: agent-credential-isolation 5.4 whole-codebase log-safety audit

**Critiquing reviewer:** Kiro/Sonnet — an independent session, distinct from the
audited artifact's author (Kiro/Opus-4.8, pane `w7:p2`) and from the producer of
the underlying `pkg/redact` module and `email.go` fix.
**Audited artifact:** `.planning/agent-brain-v3/evidence/credential-isolation-5.4-codebase-log-safety-audit.md`
**Adjudication authority:** Kiro#Opus48-TL adjudicates. This critique does not
accept, reject, or check the `tasks.md` 5.4 checkbox.
**Scope:** Evidence-contract audit of the artifact's claims only — hash
integrity, test reproducibility, sink-enumeration methodology, the 703-callsite
claim, the logger-entrypoint/global-hook claim, the agent-output sink claim,
residual disclosure, AB-REQ/EV mapping, provenance, and identity separation.
Read-only: no product/test/spec/tasks/index/git edit. Offline, bounded,
synthetic-only tests. No real secret values, auth homes, DB, network, or live
provider touched.

## Artifact hash

```
2b060da6c8d817256b30b9f8ab372888105833a8a4478b70ce127b42b870dedb  .planning/agent-brain-v3/evidence/credential-isolation-5.4-codebase-log-safety-audit.md
```
Recorded here for provenance continuity; the audited artifact does not print
its own hash internally (a minor completeness gap, noted below).

## Field-by-field verdict

### 1. Source/test hash claims — **PASS**

```
f409ba8a9f3e63618d59c5a8692296f8f7c019c9e558576b8786a058fbf68a5c  pkg/redact/redact.go
5a37941a1c7f1bd7263368a6479104c81300f7981e9f47bfb6b0cd17a602fec9  pkg/redact/redact_test.go
```
Reproduced independently on current disk state — exact match to the artifact's
claim and to the cross-referenced `credential-isolation-redact-core-fix.md`
(confirmed to exist at `.planning/agent-brain-v3/evidence/credential-isolation-redact-core-fix.md`,
contrary to an initial failed lookup under a different, incorrect path assumption
on my part during this review — corrected before relying on it further).

### 2. `pkg/redact` test transcript/count — **PASS**

`go test -count=1 -v ./pkg/redact/` reproduced independently: **PASS**, exit 0,
0.015-0.019s. 25 named parent test functions actually ran (verified by reading
full test-run output, not just tallying grep matches), including every
log-safety-critical test the artifact cites by name (`TestSanitizeForLog`,
`TestSanitizeForLogIsBoundedAndCycleSafe`, `TestSanitizeForLogTypedNilError`,
`TestSanitizeSlogAttrUsesKeyAndPreservesSafeKinds`,
`TestSanitizeSlogAttrThroughHandler`, `TestRedactCredentialFieldsInJSONBody`).
Non-zero, non-vacuous — confirmed by reading actual subtest RUN/PASS lines, not
trusting a summary count.

### 3. Whole-codebase sink-enumeration methodology/coverage — **PARTIAL**

The artifact's categories (structured `slog.*`, standard `log.*`, `fmt.Print*`,
WebSocket/event broadcast, agent comment/error surfaces) are a reasonable
taxonomy and each category's specific claims were independently reproduced
(see §4-6). However the methodology has two disclosed-but-underweighted gaps:

- The artifact's own text states it "cannot exhaustively prove all 703
  callsites" and frames this as the basis for residual R-5.4-A, but then still
  grades the overall audit **PASS** rather than **PARTIAL**. A whole-codebase
  claim that explicitly disclaims exhaustive proof over its primary sink
  category should be graded PARTIAL for that section, not folded into a
  "non-blocking residual" footnote under an overall PASS.
- The artifact never enumerates or greps for direct `slog.New(...)`
  construction sites to check whether any secondary/local logger instance
  bypasses the global `logger.Init()`/`redact.SanitizeSlogAttr` wiring. I ran
  this check independently (see §4) and confirms the claim holds, but the
  audited artifact itself did not perform or show this specific check, which
  is exactly the class of gap the steering instruction flagged.

### 4. Logger entrypoint / global-hook claim — **PASS** (independently extended)

Reproduced: `internal/logger/logger.go` is the only production file calling
`slog.New(` (2 call sites: `Init()` line 38, `NewLogger(component)` line 52).
Both wire `tint.NewHandler(..., ReplaceAttr: redact.SanitizeSlogAttr)`. I
independently searched for any other production `slog.New(`,
`slog.NewTextHandler(`, or `slog.NewJSONHandler(` call and found **zero** —
every other occurrence of these constructors in the tree is inside a
`_test.go` file. This directly answers the steering instruction's request to
check for bypassing local logger instances: **none found in production code.**

One nuance the artifact did not call out: `NewLogger(component)` is a
*second*, parallel entrypoint (not gated by whether `Init()` ran) used by
standalone CLI-style commands. It is correctly wired with the same
`ReplaceAttr`, so this is not a leak, but the artifact's phrasing
("`logger.Init()` is called by every entrypoint") is imprecise — some
entrypoints use `NewLogger` instead of/alongside `Init`. Cosmetic, not a
safety gap.

### 5. 703-callsite claim — **PASS** (independently reproduced exactly)

```
grep -rlE 'slog\.(Debug|Info|Warn|Error|Log|DebugContext|InfoContext|WarnContext|ErrorContext|LogAttrs)\(' --include='*.go' . | grep -v '_test\.go' | wc -l
=> 82 files

grep -roE 'slog\.(Debug|Info|Warn|Error|Log|DebugContext|InfoContext|WarnContext|ErrorContext|LogAttrs)\(' --include='*.go' . | grep -v '_test\.go' | wc -l
=> 703 matches
```
Both the 703 count and the 82-file count reproduce exactly against the
artifact's claim ("703 callsites / 82 files"). This is a genuinely
reproducible, non-fabricated figure.

### 6. Message-string bypass check (steering-directed) — **PARTIAL**, corrects the artifact

The steering instruction directed me to check whether any slog message is
dynamically formatted/concatenated from errors/output/env/credential data. I
found **two real production instances** the artifact's text claims do not
exist:

- `internal/daemon/daemon.go:4477` — `taskLog.Info(fmt.Sprintf("tool #%d: %s", n, msg.Tool))`.
  `msg.Tool` is an agent tool-name identifier (e.g. `"exec_command"`), not
  free-form input — low materialized risk, but it is still a message-string
  interpolation of session data, which the artifact says its targeted scan
  "found no credential-bearing message interpolation" of.
- `pkg/agent/claude.go:973` — `logWriter.Write`: `w.logger.Debug(w.prefix + text)`,
  where `text` is the **raw, only-whitespace-trimmed stderr output of the
  Claude CLI subprocess** (wired at `claude.go:204`,
  `newStderrTail(newLogWriter(b.cfg.Logger, "[claude:stderr] "), ...)`). This is
  materially more concerning than the tool-name case: subprocess stderr is
  exactly the kind of surface that can contain an echoed API key, auth error
  dump, or environment value if the CLI ever prints one on failure.

**Verification of actual exposure (not just existence):** I traced the
mechanism precisely rather than stopping at "a bypass exists." Go's
`log/slog` `ReplaceAttr` contract, and the third-party `tint` handler actually
used in production (source read at the Go-module-cache path resolved by
`go list -m -f '{{.Dir}}' github.com/lmittmann/tint`, no filesystem-wide scan
performed, no credential/env content read), both wrap the **message** in a
synthetic `slog.String(slog.MessageKey, r.Message)` attr and pass it through
the same `ReplaceAttr` function before formatting
(`tint@v1.1.3/handler.go:271`, confirmed by `tint`'s own test suite using
`ReplaceAttr: drop(slog.MessageKey)`). I confirmed this executably: a bounded,
synthetic-only test constructing the production `tint`-equivalent handler
wiring (`slog.NewJSONHandler` with `ReplaceAttr: redact.SanitizeSlogAttr`) and
logging a synthetic `"[claude:stderr] error: OPENAI_API_KEY=sk-proj-SYNTHETIC-NOT-REAL-000111222"`
message showed the sentinel **was** redacted
(`"msg":"[claude:stderr] error: OPENAI_[REDACTED CREDENTIAL] API KEY]"`).

**Conclusion on this point:** the two message-concatenation sites are real and
were missed by the artifact's specific claim of finding none — that claim is
factually inaccurate and should be corrected. However, they do **not**
constitute a full, unmitigated bypass of the redaction mechanism: message
strings still pass through `Text()`'s pattern-matching via the `ReplaceAttr`
message-key wrapping. They are exposed to exactly the same **pattern-dependent
coverage limit** the artifact already discloses as residual R-5.4-B for
attribute values — a token/secret shape that does not match one of
`pkg/redact`'s fixed regex patterns (e.g., a vendor-specific opaque credential
with no recognizable `KEY=value` or `"field": "value"` shape, embedded in raw
subprocess stderr) would not be redacted. Given 5.4's requirement is absolute
("nenhum segredo aparece em logs," no partial-credit language in the spec),
and per the steering instruction not to treat this as automatically
non-blocking, I am not grading this PASS. I grade it **PARTIAL**: the
mechanism provides real, demonstrated coverage, but coverage is
pattern-dependent and the specific claim that no such interpolation sites
exist is wrong. This deserves an explicit residual entry (not folded silently
into R-5.4-B) and, per the steering instruction, either an executable/static
guard (e.g., a lint rule forbidding raw subprocess-output concatenation into
log messages, or routing `claude.go:973`'s writer through `redact.Text()`
explicitly before logging) or the task stays open.

### 7. Google OAuth error-body claim (steering-directed) — **PASS**

Verified `internal/handler/auth.go:656` (artifact cites `auth.go:656` without
the `internal/handler/` package prefix — a minor provenance-precision gap, but
the line content matches exactly):
```go
slog.Error("google oauth token exchange returned error", "status", tokenResp.StatusCode, "body", string(tokenBody))
```
guarded by `if tokenResp.StatusCode != http.StatusOK`. Two bounded synthetic
tests (no real network, no real Google call) confirm:
- A hypothetical error body that echoes an `access_token` JSON field IS
  redacted by `Text()`'s pattern set (`"access_token":"[REDACTED]"`).
- Google's actual, realistic OAuth error schema (`{"error":"...","error_description":"..."}`)
  contains no token field to begin with, so the "pattern-dependent" coverage
  claim is not just plausible but empirically exercised for the realistic
  case. This matches the artifact's own "COVERED (pattern-dependent)" framing
  and I extend it to PASS with executed evidence rather than static assertion.

### 8. Agent output sink claim (`internal/handler/daemon.go:2225-2228`) — **PASS**

Confirmed by direct read: `ReportMessages` routes `msg.Content`/`msg.Output`
through `redact.Text` and `msg.Input` through `redact.InputMap` before DB
persist and broadcast, matching the artifact's citation exactly (line numbers
and redaction calls verified against current source).

### 9. Residual disclosure (R-5.4-A/B/C) — **PARTIAL**

R-5.4-A (message-string bypass, general) and R-5.4-B (pattern-dependency) are
correctly identified in the abstract, but R-5.4-A's own text ("Targeted scan
... found no credential-bearing message interpolation in credential-isolation
paths") is contradicted by the two concrete instances found in this critique.
The residual was named correctly in principle but the artifact's own
verification of "no confirmed leak" for that residual is incomplete — it
found the *category* but missed *concrete instances*, which changes "no
confirmed leak" from a verified statement to an assumed one for those two
call sites specifically. R-5.4-C (CLI operator-terminal output) is accurately
scoped and correctly excluded from the daemon-log surface.

### 10. AB-REQ/EV mapping — **PASS**

The mapping to the spec's "Não vazamento de segredo" requirement, and the
EV-CREDISO-5.4-CORE / EV-CREDISO-5.4-EMAIL / EV-CREDISO-5.4-CODEBASE
three-way split, is structurally sound and consistent with the standing
ledger pattern for this task (both prior 5.4 rows explicitly left 5.4 OPEN
pending exactly this codebase confirmation). No mapping error found.

### 11. Durable check-in/provenance and identity separation — **PASS**

The audited artifact correctly identifies itself as an independent,
non-self-accepting review (Kiro/Opus-4.8, `w7:p2`) distinct from the
`pkg/redact` core-fix producer and the `email.go` fix producer, and correctly
routes final acceptance to Kiro TL. This critique is itself performed by a
further-distinct session (Kiro/Sonnet) per the task's instruction, preserving
reviewer/producer/critic separation. No identity-collision or self-review
pattern found in either the original artifact or this critique.

## Conflict scan

- No conflict with `EV-CREDISO-5.4-CORE` or `EV-CREDISO-5.4-EMAIL` — both
  unchanged, hashes re-verified matching.
- No product/spec/tasks.md edit performed by this critique.
- `daemon.go` and `claude.go` are not locked by any in-progress agent per
  `AGENT_LEDGER.md`/`FILE_OWNERSHIP.md` at time of this critique; no
  ownership conflict encountered.
- The two temporary verification test files
  (`pkg/redact/zz_kiro_reviewer_verify_test.go`,
  `pkg/redact/zz_kiro_msg_verify_test.go`) were created only in a scratch
  capacity to reproduce claims, then deleted immediately after each run; they
  were never committed and do not remain on disk.

## Overall verdict: **PARTIAL**

The audited artifact's hash provenance, `pkg/redact` test execution, 703/82
callsite count, logger-entrypoint wiring, Google-OAuth-body coverage, and
agent-output-sink coverage all independently **PASS** — these are genuine,
reproducible findings, not fabrication. However, the artifact's specific
factual claim of finding **zero** credential-bearing message-string
interpolation is **incorrect**: two real production instances exist
(`daemon.go:4477`, `pkg/agent/claude.go:973`), the second of which (raw
subprocess stderr concatenated into a log message) is a materially relevant
surface for an absolute "no secret in logs" requirement. The underlying
redaction mechanism does still pattern-scan messages (verified executably via
the `tint`/`ReplaceAttr` MessageKey-wrapping mechanism), so this is not a
full, unmitigated leak — but it is real, undisclosed-as-a-concrete-instance,
and pattern-dependent, which per the steering instruction should not be
waved through as automatically non-blocking.

**Recommendation to Kiro#Opus48-TL:** do not accept the audited artifact's
PASS grade as-is. Either (a) require the artifact to be corrected to
acknowledge the two concrete message-interpolation instances and their
residual risk explicitly, plus add an executable or static guard (CI lint
forbidding raw error/stdout/stderr concatenation into slog messages, or a
direct fix routing `claude.go:973` through `redact.Text()`), or (b) keep task
5.4 in **PARTIAL/OPEN** status until such a guard or fix exists. I do not
self-accept, self-reject, or touch the `tasks.md` checkbox; this is a
critique for TL adjudication only.

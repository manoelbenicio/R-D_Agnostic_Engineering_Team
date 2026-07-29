# Credential-isolation 5.4 — all-adapters argv structural redaction plan (ADVISORY, read-only)

Smallest shared, behavior-preserving structural argv-redaction API + migration/test matrix for every agent
adapter, decoupled from the unrelated Claude stderr slice and the environment WIP.
**Advisory design only — no implementation. Owner (pkg/agent) + Kiro TL adjudication required.**

- Author: **Kiro/Opus-4.8, session `w8:p2`** (read-only). HEAD `b6571299`. Basis: residual matrix
  `credential-isolation-5.4-codebase-wide-residual-closure-matrix.md` (`97fbbc24fcf7783753486f926e9279bdd164c33edfbb64f3f113b9cf529402bf`).

## CHECK-IN 2026-07-18T22:14:00Z
Mode: READ-ONLY design. Sole deliverable = this file. Excluded (honored): no implementation; no
source/test/spec/tasks/shared-planning/git/index/ref edit; no credentials/env values; no DB/network/services.

## Input manifest (SHA-256, hashed this session)
| File | SHA-256 | argv log state |
|---|---|---|
| `pkg/agent/antigravity.go` | `96ee0c98…87a1` | **raw** `"args", args` |
| `pkg/agent/cline.go` | `9497ebfc…ede0` | **raw** `"args", clineArgs` |
| `pkg/agent/codebuddy.go` | `ecb85d96…bcf9` | **raw** `"args", args` |
| `pkg/agent/copilot.go` | `80111abb…1d07b` | **raw** `"args", cmdArgs` |
| `pkg/agent/cursor.go` | `f38115ae…e192f` | **raw** `"args", cmdArgs` |
| `pkg/agent/gemini.go` | `260ffcf6…7555` | **raw** `"args", args` |
| `pkg/agent/hermes.go` | `3752b611…ba3d9` | **raw** `"args", hermesArgs` |
| `pkg/agent/kimi.go` | `53271c50…cd2e` | **raw** `"args", kimiArgs` |
| `pkg/agent/kiro.go` | `0b4d3bd7…7410` | **raw** `"args", kiroArgs` |
| `pkg/agent/openclaw.go` | `ebd450c2…4291a` | **raw** `"args", args` |
| `pkg/agent/opencode.go` | `4db9a414…48d4c` | **raw** `"args", args` |
| `pkg/agent/pi.go` | `46f1ed17…1994da` | **raw** `"args", cmdArgs` |
| `pkg/agent/qoder.go` | `7bfb0d23…2618` | **raw** `"args", qoderArgs` |
| `pkg/agent/claude.go` | `3f9dc4fb…5ede9` | **WIP structural** `safeAgentArgvForLog` (mixed w/ env WIP — NOT taken) |
| `pkg/agent/codex.go` | `597fa3ac…12c1` | **already redacted** (own projection; comment :572-574; no raw line) |
| `pkg/agent/opencode_mcp.go` | `36136a675f109a2d69aca2f87b012011a27470f9c223a3541e5a3a6bfac33148` | **excluded — NOT a log sink** (MCP config assembly; see Correction below) |

Verified: **exactly 13 raw `Logger.Info("agent command", …, "args", <argv>)` argv-LOG call sites** (one
line each). `opencode_mcp.go` was re-evaluated after independent review (see **Correction** section) and is
**excluded with a concrete structural reason** — it is not an argv log sink. The exact-13 claim is scoped to
argv **log** sinks; config-assembly paths are a separate category.

## Current-state facts
- Every adapter call is `b.cfg.Logger.Info("agent command", "exec", <execStr>, "args", <argv []string>)`.
  Variable names differ (`execPath`/`argv0`; `args`/`clineArgs`/`cmdArgs`/…) but types are uniform
  (`string` exec, `[]string` argv). All loggers wire `redact.SanitizeSlogAttr`, so raw argv is today only
  **pattern-dependently** (`Text()`) redacted per element — residual **R-5.4-B**.
- **Claude's WIP primitives** (in `claude.go`, entangled with the environment WIP): `redactedAgentArgValue`,
  `sensitiveAgentArgValueFlags` (map), `sensitiveInlineArgMarkers` ([]string), `sensitiveAgentArgTerms`
  (map), `SafeAgentArgvForLog`, `isSensitiveAgentArgValueFlag`, `redactSensitiveInlineArg`,
  `logAgentCommand(logger, exec, args)`. These are **pure** (no env/exec-resolution dependency) and are the
  reference semantics to re-home.

## Design — smallest shared, behavior-preserving API

### Recommended: NEW file in package `agent` (A1)
`pkg/agent/argv_redaction.go` (package `agent`) — authored **fresh** with the pure semantics above (NOT
importing claude.go WIP state), exporting:
```
const redactedAgentArgValue = "[REDACTED]"
func logAgentCommand(logger *slog.Logger, execPath string, args []string)   // shared sink
func safeAgentArgvForLog(args []string) []string                            // pure projection
// + the 3 tables + isSensitiveAgentArgValueFlag + redactSensitiveInlineArg (unexported, same package)
```
- Same package `agent` ⇒ **no new import, no cycle**; every adapter calls `logAgentCommand(...)`.
- **Decoupling:** this file depends on nothing from claude.go's `Execute()`/`processEnvironment`/
  `resolveProcessExecutable`/`environment.go`. It ships independently of the Claude stderr slice
  (`c7922b7b` delta) and the environment WIP. Claude's inline copy is removed **later** (owner-gated
  reconciliation), not a prerequisite.

### Flag / value rules (preserved from the reference semantics)
1. **Value-bearing sensitive flag** (`sensitiveAgentArgValueFlags` ∪ heuristic `isSensitiveAgentArgValueFlag`
   over `sensitiveAgentArgTerms` + `apikey/authtoken/accesstoken/refreshtoken`):
   `--flag=value` → `--flag=[REDACTED]`; `--flag value` (separate token) → keep flag, **redact next token**.
2. **Inline markers** (`sensitiveInlineArgMarkers`, e.g. `api_key=`, `authorization:`, `bearer `,
   `password=`, `token=`, `base-url=`, …): redact after the `=`/`:` (else whole arg).
3. **Boolean-despite-name exception:** `--strict-mcp-config` is NOT value-bearing (must not redact the next,
   unrelated flag).
4. **Substring hardening:** flag names containing `base-url`/`api-key`/`client-secret` (underscore-normalized)
   are treated value-bearing.

### Unknown-secret behavior (explicit)
- **Known sensitive flag → structurally redacted regardless of value shape** (stronger than pattern-only).
- **Unknown flag with a secret-shaped value** → still caught by the logger's `Text()` pattern scan
  (defense-in-depth remains).
- **Unknown flag with an opaque secret value** (unrecognized flag AND unrecognized shape) → **residual: could
  leak.** Two dispositions: (default, recommended) pass through positional/unknown values (behavior-preserving)
  and owner-accept this bounded residual; (stricter alt) redact the value after any `--flag value` pair whose
  value matches a broad secret-shape heuristic — higher false-positive risk to diagnostics.

### exec field — two variants (owner picks)
- **V1 (recommended, Claude-parity):** log `exec = filepath.Base(execPath)` + add `"arg_count"`. Minor
  behavior change for the 13 (full path → basename) — a **hardening** (exec path can embed a home dir/username;
  `redact.Text` already masks home, basename removes it entirely). Flag to log consumers.
- **V2 (strict-preserve):** keep each adapter's current `exec` value + no `arg_count`; only redact `args`.
  Zero change to the `exec`/`arg_count` fields.

## Migration matrix (per adapter — owner-gated one-line change)
For each of the **13 raw adapters**: replace
`b.cfg.Logger.Info("agent command", "exec", <exec>, "args", <argv>)`
→ `logAgentCommand(b.cfg.Logger, <exec>, <argv>)`.
| Adapter | call-site line (at listed hash) | change |
|---|---|---|
| antigravity | :73 | 1-line |
| cline | :68 | 1-line |
| codebuddy | :107 | 1-line |
| copilot | :213 | 1-line |
| cursor | :41 | 1-line |
| gemini | :38 | 1-line |
| hermes | :64 | 1-line |
| kimi | :61 | 1-line |
| kiro | :68 | 1-line |
| openclaw | :79 | 1-line |
| opencode | :85 | 1-line |
| pi | :210 | 1-line |
| qoder | :99 | 1-line |
- **Reconcile follow-ons (owner-gated, not blockers):** `claude.go` → use the shared `logAgentCommand` and
  drop its inline WIP copy (do this WITHOUT pulling the environment WIP); `codex.go` → converge its own
  redacted projection onto the shared API for a single source of truth. Neither is a current raw-leak.

## Test matrix (NEW disjoint tests, offline synthetic — no DB/network/creds)
`pkg/agent/argv_redaction_test.go` table over `safeAgentArgvForLog`, synthetic hostile inputs only:
| Case | Input | Expected |
|---|---|---|
| separate sensitive flag | `--api-key sk-SYNTH0000` | `--api-key [REDACTED]` |
| `=`-joined sensitive | `--api-key=sk-SYNTH0000` | `--api-key=[REDACTED]` |
| opaque value under known flag | `--token OPAQUE-NO-SHAPE` | redacted (structural, not shape) |
| config path (home) | `--config /home/u/.secret` | `--config [REDACTED]` |
| inline marker | `authorization: Bearer eyJ…` | redacted after `:` |
| boolean exception | `--strict-mcp-config --model x` | NOT redact `--strict-mcp-config`; `--model`→next redacted |
| positional/unknown | `run`, `--verbose`, prompt text | passthrough (then `Text()` backstop) |
| non-secret flag | `--effort high` | passthrough |
- Plus a **per-adapter projection assertion**: drive each adapter's log path with one hostile arg and assert
  the emitted `args` contains no synthetic sentinel (compile/symbol-level where DB-gated).
- Run: `gofmt -l`; `go vet ./pkg/agent`; `go test ./pkg/agent -run Argv -count=20 -race` offline/pinned.

## Ownership / hotspots
- `pkg/agent/{claude,codex,kimi,nim,antigravity}.go` are **Runtime/CLI-security hotspots** (FILE_OWNERSHIP;
  G3/Codex3 lane). The 13 adapter edits + new shared file + claude/codex reconciliation are **all
  owner-gated** — this plan authors nothing and only proposes them.
- **Must NOT take:** claude.go `Execute()`/`processEnvironment`/`resolveProcessExecutable` + `environment.go`
  (environment WIP) and the Claude stderr slice — the shared argv file is independent of both.

## Alternatives
- **A1 (recommended):** shared file in package `agent` — smallest, no import/cycle.
- **A2:** sub-package `pkg/agent/argvlog/` exporting `LogCommand`/`SafeArgv` — stronger isolation, adds import.
- **A3:** centralize at the exec-spawn layer (one wrapper builds+logs argv for all adapters) — larger refactor,
  out of smallest scope.
- **A4 (no-code):** keep raw argv + rely on `Text()` pattern scan; owner-accept R-5.4-B (residual-matrix fallback).

## Stop conditions
1. **Agent-package owner (Codex3) authorization** absent for touching the 13 files + adding the shared file → STOP.
2. **codex.go divergence:** confirm codex's existing projection before converging; do not assume its shape → STOP if unclear.
3. **exec-field change (V1)** deemed a breaking consumer change → fall back to V2 strict-preserve.
4. **Entanglement risk:** if re-homing the primitives would pull the Claude stderr slice or environment WIP → author fresh; STOP if separation can't be kept.
5. **Adapter argv construction:** confirm each `<argv>` var is the final exec argv (copilot/cursor use `chooseXInvocation`→`cmdArgs`; verified they are) → STOP for any adapter where it isn't.

## Correction (2026-07-18T22:24:00Z) — `opencode_mcp.go` re-evaluated after independent review
An independent residual review flagged `pkg/agent/opencode_mcp.go`
(SHA-256 `36136a675f109a2d69aca2f87b012011a27470f9c223a3541e5a3a6bfac33148`) as a possible **14th** raw-args
sink. **Determination: FALSE POSITIVE for the argv-LOG-sink category — excluded, with a concrete structural
reason.** Evidence (read-only inspection):
- The file contains **no `Logger.*` / `slog.*` call at all** (`grep 'Logger.(Info|Debug|Warn|Error)'` → 0).
- The only `"args"` reference is `args, err := stringSliceField(server, "args")` (line 366) — **parsing** an
  MCP-server config field, not logging.
- `openCodeCommand(server)` (lines ~355-372) merges the config's `command` + `args` into a combined
  `[]string`, and `mcpServerToConfig` (line ~340) assembles `out := map[string]any{"type":"local",
  "command": command, …}` which is **returned for MCP config-file generation** (`return out, nil`) — a
  configuration artifact consumed by the CLI, **not emitted to any diagnostic log**.
- Therefore it is **not** an `"agent command"` argv log sink and is correctly **not** part of this plan's
  count/list/migration/test/ownership sections. **The exact-13 argv-log-sink count stands** (not silently
  preserved — verified by re-inspection).
- **Adjacent-category note (out of this plan's scope, flagged for TL):** the MCP config map does carry
  `command`+`args` (which could embed secrets/paths). If that config is ever *logged* or written to a
  world-readable artifact, that is a distinct **config-file/MCP-config safety** concern — a separate sink
  category, not the argv-log-redaction addressed here. Recommend a separate residual check for MCP config
  emission; it does **not** change this argv-log plan.

## Non-claims
- Advisory design only; implements nothing; no source/test/spec/tasks/shared-planning/git/index/ref change;
  no credentials/env values; no DB/network/services. Paths/hashes from static reads at HEAD `b6571299`.
  Owner (pkg/agent) + Kiro TL adjudication required; reviewer ≠ adjudicator.

## CHECK-OUT 2026-07-18T22:18:00Z — DONE (superseded by correction)
## CHECK-OUT (revised) 2026-07-18T22:25:00Z — DONE
Correction applied to this artifact only: `opencode_mcp.go` evaluated and **excluded** (config assembly, no
log sink) with its hash pinned; the exact-13 argv-log-sink count is re-verified, not merely preserved.
Only this file was modified. No source/test/spec/tasks/shared-planning/git/index/ref change; no
credentials/env values; no DB/network/services. Re-hash below.

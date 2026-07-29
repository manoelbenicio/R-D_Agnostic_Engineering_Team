# Credential-isolation 5.4 — opencode_mcp.go config secret-safety trace (ADVISORY, read-only)

Follows `opencode_mcp.go` MCP config `command`/`args` + credential-like fields through serialization,
environment transport, file/process handoff, errors, and downstream logs. **Read-only; no acceptance.**
Distinguishes **task-5.4 log-safety** from **broader credential handling**.

- Author: **Kiro/Opus-4.8, session `w8:p2`**. HEAD `b6571299`.
- Pinned inputs: `pkg/agent/opencode_mcp.go` SHA-256
  `36136a675f109a2d69aca2f87b012011a27470f9c223a3541e5a3a6bfac33148`; `pkg/agent/opencode.go` SHA-256
  `4db9a414e13743c8cc672b36d30f6ba2f649530f75daeb13e42a9c27db448d4c`.

## CHECK-IN 2026-07-18T22:25:00Z
Mode: READ-ONLY trace. Sole deliverable = this file. Excluded (honored): no source/test/spec/tasks/
shared-planning/git/index/ref edit; no credentials/env values; no DB/network/services.

## Credential-like fields in scope (opencode_mcp.go schema structs)
- `opencodeMCPLocal.Command []string` (line 18), `.Environment map[string]string` (line 19).
- `opencodeMCPRemote.URL`, `.Headers map[string]string` (line 30 — can carry `Authorization`/tokens),
  `.OAuth json.RawMessage` (line 31).
- `opencodeMCPOAuth.ClientSecret` (line 53), `.ClientID`, `.RedirectURI`.
- Claude-style `command`(string)+`args`([]string) merged by `openCodeCommand` (lines ~355-372) — args may
  embed tokens/paths.

## End-to-end flow (traced)
1. `opts.McpConfig` (agent.mcp_config JSON) → `buildOpenCodeMCPConfigContent(raw)` (opencode_mcp.go:90) →
   `translateMCPConfigForOpenCode` → validated `map[string]any` → `json.Marshal({"mcp": servers})` →
   **returns a JSON string** (may contain `clientSecret`, `headers`, `command`/`args`, `environment`).
2. `opencode.go:118` `mcpContent, err := buildOpenCodeMCPConfigContent(opts.McpConfig)`.
   - **err path (opencode.go:119-121):** `return nil, err` — **not logged here**; propagated to caller.
3. `opencode.go:127` `env = append(env, "OPENCODE_CONFIG_CONTENT="+mcpContent)` → `cmd.Env = env`
   (opencode.go:129). **The secret-bearing content travels as a process ENVIRONMENT VARIABLE.** The code +
   the file-header comment confirm **nothing is written to disk** (env-var chosen precisely to avoid workdir
   config writes).

## Sink-by-sink determination (can a secret appear in …?)
| Sink | Finding | Evidence |
|---|---|---|
| **Diagnostic logs — argv** | **NO** | `opencode.go:85` `Logger.Info("agent command", "exec", execPath, "args", args)` logs **argv only**; `OPENCODE_CONFIG_CONTENT` is an env var, not in `args`. |
| **Diagnostic logs — env value** | **NO** | The only log referencing the var is `opencode.go:125` `Logger.Warn("agent.custom_env sets OPENCODE_CONFIG_CONTENT …")` — logs the **key name only**, never the value. No `Logger.*` logs `env`/`cmd.Env`. |
| **Persisted config file** | **NO** | Env-var transport by design; no `os.WriteFile`/workdir `opencode.json` write in this path (header comment + code). |
| **Process argv** | **NO** | Content is in `cmd.Env`, not `cmd.Args`; absent from argv (and from the argv log). |
| **Errors** | **NO secret VALUES** | All errors wrap the server **name** (`%q`, a key) + validation messages (missing field, `timeout must be positive`, `type must be a string, got <token>`, `invalid type`, oauth `must be an object or false, got <token>`). They echo field **names/types/discriminator tokens**, **not** `clientSecret`/`headers`/`command` **values**. Returned via `opencode.go:121` (not logged locally). |
| **Downstream subprocess stderr** | pattern-dependent (R-5.4-B) | `opencode.go:136` `cmd.Stderr = newLogWriter(b.cfg.Logger, "[opencode:stderr] ")` — the **shared `logWriter`**; once the 5.4 Claude `logWriter.Write` `redact.Text` fix lands it covers opencode stderr too (pattern-dependent). Covered by the residual matrix's Claude-stderr row, not a new gap. |
| **Process ENVIRONMENT (child)** | **YES — but not a log** | `OPENCODE_CONFIG_CONTENT` (with `clientSecret`/headers) is in the child's environment (visible via `/proc/PID/environ` to same-user; inherited by grandchildren). **This is broader credential-handling, NOT task-5.4 log-safety.** |

## Verdict — 5.4 log-safety vs broader credential handling (kept separate)
- **Task 5.4 (no secret in LOGS): NO REAL GAP.** The MCP config secrets never reach a diagnostic log,
  persisted config, or process argv; only the env **key name** is logged; errors carry names/types, not
  values; opencode stderr is covered by the shared `logWriter` redaction (pattern-dependent, same class as
  the accepted/pending Claude fix). This **confirms and extends** the prior "false positive for the
  argv-log-sink" finding: opencode_mcp.go is clean on **all** log paths.
- **Broader credential handling (OUT of 5.4 scope, flagged for owner): env-transport exposure.** Secrets
  ride `OPENCODE_CONFIG_CONTENT` into the child process environment — a by-design mechanism (the CLI needs
  the config) equivalent to any env-delivered credential. Exposure surface = `/proc/self/environ` (same-user)
  + inheritance, **not** logs. Any hardening (e.g. tmpfile with 0600 + `OPENCODE_CONFIG` path instead of
  inline env, or scoping env to the exec only) is a **credential-transport** decision, not a 5.4 log fix.

## Minimal structural guard tests / remediation
- **No log-safety remediation required** (no gap). Optional **invariant-locking** offline tests (owner-gated,
  package `agent`, synthetic sentinels only):
  1. Build a config with a synthetic `clientSecret`/`headers.Authorization` sentinel; capture the daemon
     logger during the opencode launch path; assert the sentinel is **absent** from the log and that only
     the `OPENCODE_CONFIG_CONTENT` **key** may appear (locks the "value never logged" invariant).
  2. Assert `buildOpenCodeMCPConfigContent` **error strings** contain the server name + field/type tokens but
     **no** injected secret value (guards against a future change that wraps raw values).
- **Adjacent (separate lane, not 5.4):** if env-transport exposure is deemed unacceptable, evaluate a
  0600 tmpfile + path-env alternative — a credential-transport design task with its own owner review.

## Non-claims
- Advisory trace only; no acceptance; no checkbox; no code/test/spec/tasks/shared-planning/git/index/ref
  change; no credentials/env values; no DB/network/services. Findings from static reads at the pinned
  hashes (not executed). Kiro TL adjudicates; reviewer ≠ adjudicator.

## CHECK-OUT 2026-07-18T22:27:00Z — DONE
Only this file created. Everything else unchanged.

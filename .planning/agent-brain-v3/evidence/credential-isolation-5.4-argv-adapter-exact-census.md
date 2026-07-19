# pkg/agent argv/args log-sink — exact census (advisory)

- Author: Kiro/Opus-4.8, pane **w7:p2**. **Advisory, read-only.** No implementation; no source/test/shared-planning/
  spec/tasks/git/index/ref edit; no credentials/env/network/DB/services. Only this file created.
- Question: enumerate every **production** `pkg/agent` Go file that logs command **argv/args**, classify each, and
  reconcile the 13 vs 14 vs 15 counts without relying on filenames alone.
- Base: HEAD `b6571299b00c8e388abefe7ef9dcbcf8ac715d7f`. Hashes = SHA-256 of file content.

## Check-IN / Check-OUT
- **Check-IN** 2026-07-18T22:44:00Z — read-only AST/grep census of pkg/agent argv log sinks.
- **Check-OUT** 2026-07-18T22:58:00Z — DONE. Decisive count + classification + limitations below.

## Reproducible methodology

- **Universe:** `pkg/agent/*.go` excluding `*_test.go` (72 total files; tests excluded).
- **Sink definition:** an `slog` call `Logger.Info("agent command", …, "args", <argvSlice>, …)` — i.e. the message
  literal `"agent command"` with an `"args"` key whose value is the process argv slice.
- **grep proxy (decisive count):**
  `grep -rln --include='*.go' --exclude='*_test.go' '"agent command"' pkg/agent` → **14 files** (listed below).
- **Per-sink classification (AST intent, grep-verifiable):** inspect the `"args"` value expression:
  - `safeAgentArgvForLog(args)` → **structurally redacted** (flag-aware value masking);
  - bare identifier (`args`/`cmdArgs`/`<tool>Args`) → **raw at call site**.
- **Not-a-sink verification:** `grep -c 'Logger\.\|logger\.'` per candidate; inspect argv-adjacent logs to confirm
  they emit executable **paths**, not the args slice.
- **Central-hook note:** all `slog` output is additionally subject to the process-global `redact.SanitizeSlogAttr`
  ReplaceAttr hook when the injected logger is the `internal/logger.Init/NewLogger` default (see Limitations).

## Census — 14 argv log sinks (decisive)

| # | File | Line | `"args"` value | Class | SHA-256 |
|---|---|---|---|---|---|
| 1 | `pkg/agent/claude.go` | 66 | `safeAgentArgvForLog(args)` | **structurally redacted** | `3f9dc4fbdb28eecfcc4886f2a518a07943f8a0493cf7aa3525b745992c6d2f54` |
| 2 | `pkg/agent/antigravity.go` | 73 | `args` | raw (call-site) | `96ee0c982cab104cd5690eba71b59536f4bef2306c184bf52471198dd36887a1` |
| 3 | `pkg/agent/cline.go` | 68 | `clineArgs` | raw | `9497ebfccaeb143cef0e08b2ae4f59f5192a40d118d2f68ff208f9ae1322ede0` |
| 4 | `pkg/agent/codebuddy.go` | 107 | `args` | raw | `ecb85d968c1b60283e09174d3bc37a7dfa80126193105c2e97ce8382109bbcf9` |
| 5 | `pkg/agent/copilot.go` | 213 | `cmdArgs` | raw | `80111abb1aa00045d7d31a777c8d233a57b41f7cdfaafe3b07fd49f21391d07b` |
| 6 | `pkg/agent/cursor.go` | 41 | `cmdArgs` | raw | `f38115ae48ccc5bcfac0a028ad375dd99fb7394d0f7029791d0757be922e192e` |
| 7 | `pkg/agent/gemini.go` | 38 | `args` | raw | `260ffcf6d8066ad3e9f15c086381f5e062910043dba76d6c2de7421d79567555` |
| 8 | `pkg/agent/hermes.go` | 64 | `hermesArgs` | raw | `3752b611d5f9fd1961079fa25a78187057ba9291cf7daa817838202a4e1ba3d9` |
| 9 | `pkg/agent/kimi.go` | 61 | `kimiArgs` | raw | `53271c50affe13088d98a9f9b3f3db711b908a8b6a0e4fcae7e2031eec10cd2e` |
| 10 | `pkg/agent/kiro.go` | 68 | `kiroArgs` | raw | `0b4d3bd7f274623fa4d45db34639a247e969074cf04ac5dce5de2b7322657410` |
| 11 | `pkg/agent/openclaw.go` | 79 | `args` | raw | `ebd450c2c3911db39df078bc362a749d0bf0bd68d1250e85299e362cfdc4291a` |
| 12 | `pkg/agent/opencode.go` | 85 | `args` | raw | `4db9a414e13743c8cc672b36d30f6ba2f649530f75daeb13e42a9c27db448d4c` |
| 13 | `pkg/agent/pi.go` | 210 | `cmdArgs` | raw | `46f1ed17f664f2c316944f42e0a134ca86a460ba8bcd777e81aac6d27d1994da` |
| 14 | `pkg/agent/qoder.go` | 99 | `qoderArgs` | raw | `7bfb0d23039911c2f206aab34b4cb1eb3885929dd471924a8eac7682b8042618` |

**= 1 structurally redacted (claude) + 13 raw (call-site).**

## Not actually a log sink (argv) — explicitly resolved

| File | SHA-256 | Why not an argv sink |
|---|---|---|
| `pkg/agent/codex.go` | `597fa3acaa8c65cef0676b5b644f14adba6f35e8ac2c4f7cb0f2d042aee212c1` | No `"args"` field anywhere (`grep "\"args\""` empty). Logs `"codex started app-server" pid/cwd`, thread/activity only. Deliberately routes sensitive values to `$CODEX_HOME/config.toml` "to keep raw values out of logs/argv" (codex.go:162,569-573). **Not an argv log sink.** |
| `pkg/agent/opencode_mcp.go` | `36136a675f109a2d69aca2f87b012011a27470f9c223a3541e5a3a6bfac33148` | **Zero** `Logger.`/`logger.` calls (`grep -c` = 0). It is an MCP-config JSON transformer (`command`/`args` are JSON fields, not logged). **Not a log sink.** |
| `pkg/agent/cursor_invocation_windows.go` (shared by pi/copilot via `rewriteCmdToPS1`) | `5aba10ac136ab4e32fed3aefe4f4f7f1a34dac838feec791cd67f893dc1ac7d0` | `rewriteCmdToPS1` logs `"…routing through powershell -File to preserve argv tokens"` with keys `powershell`/`ps1`/`original` — **executable paths, not the args slice**. Windows build-tag only. **argv-adjacent, NOT an argv sink.** |
| `pi_invocation_windows.go`, `copilot_invocation_windows.go` | (thin wrappers) | Delegate to `rewriteCmdToPS1`; no own argv log. |
| Other prod files (`agent.go`, `models.go`, `environment.go`, `nim.go`, `thinking.go`, `stderr_tail.go`, `version.go`, `proc_*`, `*_invocation.go`/`_other.go`) | — | No `"agent command"`/`"args"` argv sink (absent from the grep). `nim.go` in particular has no argv sink. |

## 13 vs 14 vs 15 reconciliation (not filename-based)

- **14 = the decisive count** of argv log sinks (grep `"agent command"` prod-only = 14; AST-confirmed each has an
  `"args"` argv value). Composition: **1 structural (claude) + 13 raw**.
- **13** = the **raw-only** subset (the 14 minus claude's structurally-redacted sink). A census scoped to "raw argv
  leak sites" legitimately reports **13**.
- **15** = an **over-count** produced by including one non-sink. The two plausible false inclusions, both resolved
  above, are: (a) **codex.go** — a roster backend, so counted by name, but it logs **no** argv (config.toml path);
  or (b) the **`rewriteCmdToPS1`** Windows log — argv-*adjacent* (its message says "argv tokens") but emits paths,
  not args. Adding either to the 14 yields a spurious 15. Neither is an argv sink.

## Runtime redaction nuance (why "raw" ≠ "unprotected")

The 13 raw sinks pass `[]string` argv **unmodified at the call site**. At runtime, IF the injected `b.cfg.Logger`
is the `internal/logger` default (wired with `ReplaceAttr: redact.SanitizeSlogAttr`), the `"args"` attribute
(a `KindAny` `[]string`) is walked by `SanitizeForLog` → each element through `redact.Text()` → **pattern-only**
scrubbing (AWS/GitHub/JWT/Bearer/`sk-`/JSON-credential-field patterns). So effective protection for the 13 is
**pattern-only** (no flag-aware structural masking). Only **claude.go** adds **structural** masking at the call site
(`safeAgentArgvForLog` blanks the value *after* a known sensitive flag in `sensitiveAgentArgValueFlags`, regardless
of pattern). This call-site-vs-runtime distinction is itself a source of the 13/14 ambiguity: by **call site** the
split is 1 structural / 13 raw; by **effective runtime** it is 1 structural / 13 pattern-only.

## Limitations
- **Runtime-hook dependency:** the pattern-only protection for the 13 holds only when the passed logger carries
  `SanitizeSlogAttr`. `pkg/agent` does not itself guarantee this; a caller injecting a plain `slog` logger would
  make those 13 truly raw. Not verified at runtime here (read-only static census).
- **Build-tag coverage:** `*_invocation_windows.go` are `//go:build windows` — not compiled/executed on this Linux
  host; classified by source reading only.
- **`ps`/OS exposure (out of log scope):** codex.go keeps argv out of *logs* but the process argv still appears in
  OS-level `ps` listings (codex.go:573 comment) — a separate exposure surface, not a log sink.
- **Pattern residual:** `redact.Text()` is pattern-fixed; a novel token shape in a raw sink's argv could evade the
  central hook (same R-5.4-B residual noted in the 5.4 codebase audit).
- Advisory only; no acceptance, no checkbox. Kiro TL adjudicates. Hashes verified at census time against HEAD
  `b6571299`; re-verify before acting.

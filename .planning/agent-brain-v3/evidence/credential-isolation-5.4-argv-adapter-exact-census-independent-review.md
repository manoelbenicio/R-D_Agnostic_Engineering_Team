# Independent Review — pkg/agent argv-adapter exact census

- reviewer: Kiro / Opus-4.8, wave w8:p1 (independent of census author w7:p2)
- date: 2026-07-18T22:56:00Z
- mode: READ-ONLY. No source/tests/shared-planning/spec/tasks/git/index/refs edit; no credentials/env/network/DB/services. Only this file created. No acceptance.

## Check-in / check-out
- CHECK-IN 2026-07-18T22:51:00Z — Kiro/Opus-4.8 w8:p1 — stream 5.4-ARGV-ADAPTER-CENSUS-INDEPENDENT-REVIEW — READ-ONLY.
- CHECK-OUT 2026-07-18T22:56:00Z — DONE. Verdict below. Kiro TL adjudicates. Not accepted.

Reviewed: `credential-isolation-5.4-argv-adapter-exact-census.md` — SHA-256 `c8ba0bc89aff0d6e53c3b8bad857b63c4971596c2ca1f4e3299438a109144b9e` (stable across two reads).

## VERDICT: ACCURATE (PASS) — census is independently confirmed on every requested dimension.

## Independent verification

| Dimension | Census claim | Independent result |
|---|---|---|
| **Methodology reproducibility** | `grep -rln --exclude='*_test.go' '"agent command"' pkg/agent` = 14 | ✅ reproduced exactly — same 14 files |
| **Universe** | 14 production files with an `"agent command"`+`"args"` argv sink | ✅ antigravity, claude, cline, codebuddy, copilot, cursor, gemini, hermes, kimi, kiro, openclaw, opencode, pi, qoder |
| **14 = 13 raw + 1 Claude structural** | claude `safeAgentArgvForLog(args)`; 13 others raw | ✅ `claude.go:66` structural; 13 raw with exact lines/values matching the census table (antigravity:73 `args`, cline:68 `clineArgs`, codebuddy:107 `args`, copilot:213 `cmdArgs`, cursor:41 `cmdArgs`, gemini:38 `args`, hermes:64 `hermesArgs`, kimi:61 `kimiArgs`, kiro:68 `kiroArgs`, openclaw:79 `args`, opencode:85 `args`, pi:210 `cmdArgs`, qoder:99 `qoderArgs`) |
| **Codex handling** | codex.go is NOT an argv sink; routes secrets to `$CODEX_HOME/config.toml` | ✅ `codex.go` logs `pid/cwd/thread_id/status/duration/method` — **no `"args"` anywhere**; it is the "counted-by-name-but-not-a-sink" file whose inclusion would spuriously make 15 |
| **Cursor platform variants** | single sink `cursor.go:41` (raw `cmdArgs`); `cursor_invocation_windows.go` `rewriteCmdToPS1` is argv-**adjacent** (logs paths, Windows-tagged), not a sink | ✅ confirmed — cursor family = `cursor.go`, `cursor_invocation.go`, `cursor_invocation_other.go` (`//go:build !windows`), `cursor_invocation_windows.go` (`//go:build windows`); only `cursor.go:41` logs the argv slice; the Windows rewrite logs `powershell/ps1/original` paths (pi/copilot share it) — correctly excluded |
| **Hashes/lines** | per-file content SHA-256 + line pins | ✅ spot-checked 6: `claude 3f9dc4fb`, `cursor f38115ae`, `opencode 4db9a414`, `codex 597fa3ac`, `cursor_invocation_windows 5aba10ac`, `qoder 7bfb0d23` — all match |
| **13/14/15 reconciliation** | 14 sinks (1 structural + 13 raw); 13 = raw-only subset; 15 = over-count (codex.go or the PS1 rewrite) | ✅ sound and non-filename-based; matches source |

## Consistency with the corrected opencode_mcp trace

The census's `opencode_mcp.go` row (`36136a675f…`, "zero `Logger.` calls, MCP-config transformer, not a log sink") **matches my prior corrected trace exactly**, and `opencode_mcp.go` is correctly **absent from the 14**. This definitively closes the earlier "14th raw adapter?" ambiguity: the 14 are the `"agent command"` files; `opencode_mcp.go` is not among them, and `codex.go` is the name-counted non-sink. My earlier residual-matrix "undercount" flag is confirmed resolved (it was a false positive from grepping the JSON field `"args"`).

## Runtime nuance (correctly captured by the census)

"Raw" ≠ "unprotected": the 13 raw sinks pass `[]string` argv unmodified at the call site, but when the injected logger is the `internal/logger` default (`ReplaceAttr: redact.SanitizeSlogAttr`), each element is walked via `SanitizeForLog → redact.Text()` = **pattern-only** scrubbing (the R-5.4-B residual, consistent with the 5.4 codebase audit I reviewed). Only claude adds **structural** (flag-aware) masking. The census's call-site-vs-runtime framing is accurate and is itself the honest explanation of the 13-vs-14 split.

## Minor observations (not defects)
- The census pins `claude.go` at its **current working-tree** hash `3f9dc4fb` (dirty, includes `safeAgentArgvForLog` + argv WIP). Correct for a "current file state" census; a reader must not conflate it with the integration manifest's **HEAD+2-hunk target** `c7922b7b` (a different artifact/purpose). Recommend a one-line cross-note for clarity. No contradiction.
- I verified the 14-sink set, the codex/cursor resolutions, and 6 of the 18 pinned hashes; I did not re-hash all 18 rows nor recount the "72 total files" universe (the sink-relevant subset is what matters and is confirmed).

## Explicit non-claims
- Created only this file. No source/test/spec/tasks/shared-planning/git/index/ref edit; no credentials/env/network/DB/services; nothing executed beyond read-only `grep`/`sha256sum` at HEAD `b6571299`.
- No acceptance/checkbox/EV; this validates the census's accuracy only. Remediation of the 13 raw sinks (extend `safeAgentArgvForLog`) or owner-accept of R-5.4-B remains a separate owner/TL decision. Kiro TL adjudicates.

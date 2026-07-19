# opencode_mcp.go — argv log-sink end-to-end trace

- author: Kiro / Opus-4.8, wave w8:p1 — ADVISORY, read-only.
- date: 2026-07-18T22:44:00Z
- mode: READ-ONLY. No source/tests/shared-planning/spec/tasks/git/index/refs edit; no credentials/env/network/DB/services.
- file: `multica-auth-work/server/pkg/agent/opencode_mcp.go` — SHA-256 `36136a675f109a2d69aca2f87b012011a27470f9c223a3541e5a3a6bfac33148` (HEAD `b6571299`).

## Check-in / check-out
- CHECK-IN 2026-07-18T22:39:00Z — Kiro/Opus-4.8 w8:p1 — stream OPENCODE-MCP-ARGV-SINK-TRACE — READ-ONLY.
- CHECK-OUT 2026-07-18T22:44:00Z — DONE. Decisive verdict below. Advisory only; no acceptance.

## DECISIVE VERDICT

**`opencode_mcp.go` is a SAFE EXCEPTION — it is NOT the 14th raw-args adapter.** It contains **no logging sink of any kind** (no `slog`, no `Logger`, no `log`, no `"agent command"`). The residual matrix's **"13 adapters" count is CORRECT.** My earlier class-D "undercount / verify opencode_mcp.go" flag (in `…residual-closure-matrix-independent-review.md`) was a **false positive**: a `grep '"args"'` matched a **JSON field-key read and a code comment**, not an `slog` attribute. I hereby correct that flag; no matrix change is warranted.

## End-to-end path (exact lines)

`opencode_mcp.go` is a **pure MCP-config translation module** (imports only `bytes, encoding/json, errors, fmt, strings` — no `log`/`slog`). It builds the `OPENCODE_CONFIG_CONTENT` value; it never spawns a CLI and never logs.

1. **`"args"` source (data, not a log):** `openCodeCommand` (L353-375) reads the MCP server's `args` JSON field:
   - L355 — **comment** ("Claude's mcpServers accepts a single string with separate `args`").
   - L366 — `args, err := stringSliceField(server, "args")` — reads the `args` array from an `agent.mcp_config` MCP-server entry.
   - L370 — `return append(cmd, args...)` — concatenates into the MCP **`command`** array.
2. **Transformations:** `buildOpenCodeMCPConfigContent` → `translateMCPConfigForOpenCode` → validate/translate each server → `json.Marshal(map[string]any{"mcp": servers})` → returns a JSON **string**.
3. **Destination:** the caller injects that string into the **`OPENCODE_CONFIG_CONTENT` environment variable** for the spawned OpenCode process. **Nothing in this file is written to a logger.**
4. **Central sanitizer:** **N/A** — there is no log call in this file to route through `SanitizeSlogAttr`.

## Can secret-bearing flags/values reach a log from this file? — NO

- **No log statement exists**, so no direct leak.
- **Error values are secret-safe by construction:** `fmt.Errorf(...)` here embeds only server names, the `type` discriminator, field names, timeout/port integers, or `strictDecode`'s `json: unknown field "<name>"` (field **name**, not value). OAuth `clientSecret` is decoded into `opencodeMCPOAuth` but **never echoed** in any error (`validateOpenCodeOAuth` only errors on `callbackPort` range/shape). So even the errors this file returns to callers carry **no secret value**.
- Downstream: whatever the *caller* does with the resulting config/env is a **different sink** (e.g., the real `opencode.go` spawn adapter's `"agent command"` log — which IS class-D item "opencode.go:85"). This file is not that sink.

## Comparison to the 13-adapter design and residual matrix

- The residual matrix's class **D** correctly lists the **spawn adapters** that log `"agent command", "args", <raw argv>` — including **`opencode.go:85`** (the real OpenCode launcher). `opencode_mcp.go` is a **separate helper**, not a spawn adapter, and is correctly **absent** from D.
- `claude.go:66` is the sole adapter using structured `safeAgentArgvForLog`; the other adapters log raw args. That distinction is unaffected by this file.
- Net: **13 raw-args adapters stands.** My prior grep conflated the config-translator (`opencode_mcp.go`) with the spawn adapter (`opencode.go`); this end-to-end read resolves it in the matrix's favor.

## Smallest structural test / remediation

- **Remediation: NONE required** — there is no log sink to redact.
- **Optional invariant guard (low value):** a build/lint assertion that `opencode_mcp.go` imports neither `log` nor `log/slog` would pin its "pure translator, no logging" property so a future edit can't silently introduce a raw-argv/secret log here. Not needed for 5.4 closure; offered only as a cheap regression guard.

## Non-claims
- Advisory only; created only this file. No source/test/spec/tasks/shared-planning/git/index/ref edit; no credentials/env/network/DB/services; nothing executed (static read at the hash above).
- This trace **corrects my own prior false-positive flag**; it does not modify that prior artifact and issues no acceptance/checkbox. Kiro TL adjudicates.

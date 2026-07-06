# RESERVE Smart Context Pipeline Analysis

Agent: Antigravity-Opus-4.6
Date UTC: 2026-07-06T01:02:00Z
Check-in: `.deploy-control/Antigravity__RESERVE-SC-ANALYSIS__20260706T005900Z.md`
Scope: **READ-ONLY**. No file in `prodex-sidecar/` was edited.

## Documents Read

1. `Diligencias/SKILLS_ELITE_RUST/systematic-debugging.md` — 4-phase systematic debugging (root cause → pattern → hypothesis → implementation)
2. `Diligencias/00c_PRODEX_CRATE_COVERAGE.md` — 44-crate coverage matrix; `prodex-context` (Smart Context) is ✅ P2/G5
3. `docs/prodex/prodex-fork-map.md` — full fork map; Smart Context isolated to `prodex-context` + `prodex-app/src/runtime_proxy/smart_context/*` + `prodex-runtime-proxy/src/smart_context/*`

---

## Finding A: Pipeline Connection Map

### Where tokensavior / clawcompactor / sqz / context / compact-output connect

These are **separate optimizer categories** that connect through different pipeline layers:

| Component | Category | Connection Point | Status (0.246.0) |
|-----------|----------|-----------------|------------------|
| `token-savior` | optimizer MCP | External MCP server (`token-savior` binary); prodex detects availability and registers as `mcp__prodex_token_savior__*` tool namespace. Invoked by LLM as tool calls during runtime sessions. | `missing` (not installed) |
| `claw-compactor` | optimizer MCP | External MCP server (`claw-compactor` binary); registered as optimizer tool namespace. Deterministic local code-summary aid. | `missing` (not installed) |
| `sqz` | optimizer MCP | External MCP server (`sqz-mcp` binary); registered as `mcp__prodex_sqz__compress` / `mcp__prodex_sqz__sqz_read_file`. Downstream context reuse. | `missing` (not installed) |
| `prodex-context` | built-in crate | `prodex context` CLI subcommand for offline audit/compress of Codex shared context files. Functions: `compress_context_text`, `compact_command_output_with_options`, `compact_command_output_with_intent_options`. | `built-in` |
| Smart Context (runtime) | built-in runtime | `prodex-app/src/runtime_proxy/smart_context/body.rs` — online rewrite pipeline inside the runtime proxy. Operates on live `/v1/responses` and `/v1/chat/completions` traffic. | `built-in` |
| `rtk` | optimizer | External `rtk` binary — upstream shell-output token reduction. | `available` |
| `ponytail` | optimizer-plugin | Managed checkout loaded as Codex plugin in Prodex overlays. | `built-in` |

### Source code evidence

**Optimizer dispatch** in `prodex-app/src/command_dispatch.rs`:
```rust
Self::CompactOutput(args) => handle_context_compact_output(args),
// ...
RoutedCommand::new(caveman_args_with_optimizer_prefix(command, "sqz"))
RoutedCommand::new(caveman_args_with_optimizer_prefix(command, "tokensavior"))
RoutedCommand::new(caveman_args_with_optimizer_prefix(command, "clawcompactor"))
```

**Optimizer registry** in `prodex-app/src/runtime_caveman.rs:297`:
```rust
"sqz" | "tokensavior" | "token-savior" | "clawcompactor" | "claw-compactor" | "ponytail"
```

**MCP tool mapping** in `prodex-app/src/runtime_launch/proxy_startup/provider_tools.rs`:
```rust
"name": "mcp__prodex_sqz__compress"
"name": "mcp__prodex_token_savior__ts_search"
```

### Key insight

**tokensavior / clawcompactor / sqz are EXTERNAL MCP tool servers, NOT inline pipeline stages.** They do NOT participate in the gateway's Smart Context compaction pipeline. They are invoked by the LLM as tool calls during conversations, operating as _context reuse_ tools separate from the body-rewriting Smart Context pipeline.

The **inline compaction pipeline** is solely:
1. `prodex-context` crate (offline `compress_context_text` / `compact_command_output`)
2. `prodex-runtime-proxy/src/smart_context/*` (online rewrite policy, candidates, replay, normalization)
3. `prodex-app/src/runtime_proxy/smart_context/body.rs` (live body rewrite at proxy time)

---

## Finding B: Does `/v1/runtime/proxy` bypass compaction?

### Answer: NO — it engages compaction through the gateway

The sidecar's `/v1/runtime/proxy` handler (L774-L809 of `multica-auth-work/prodex-sidecar/src/main.rs`):

```rust
let gateway_path = envelope
    .get("gateway_path")
    .and_then(|v| v.as_str())
    .unwrap_or("/v1/responses");  // ← L777 default

// ...
let response = match http_request(
    &gateway.addr,
    "POST",
    gateway_path,   // ← forwarded to gateway
    // ...
```

**Pipeline flow:**
```
Client → POST /v1/runtime/proxy → Sidecar
  → Sidecar extracts gateway_path (default: /v1/responses)
  → Sidecar forwards body to internal prodex gateway at gateway_path
  → Gateway (with --smart-context) performs compaction on /v1/responses
  → Gateway forwards compacted body to upstream provider
```

**The default `gateway_path` IS `/v1/responses` (L777), and `/v1/responses` DOES engage compaction when `--smart-context` is enabled on the gateway.**

### Empirical proof from prior evidence

From `DIAG-smart-context-compaction.md`:
- Direct gateway with `--smart-context`: `66124 bytes → 33172 bytes` (49.8% reduction)
- Via sidecar `/v1/runtime/proxy` (default `/v1/responses`): `tokens_saved=16519`, `input_token_reduction_percent=99`

From `C5-final-remeasure-3sizes.md`:
- 16KiB: `tokens_saved=4139` (99%)
- 64KiB: `tokens_saved=16476` (99%)
- 256KiB: `tokens_saved=65827` (99%)

All through sidecar `/v1/runtime/proxy` using default `/v1/responses`.

### Summary

| Path | Compaction? | Mechanism |
|------|------------|-----------|
| `POST /v1/runtime/proxy` (no `gateway_path`) | ✅ YES | Default `/v1/responses` → gateway Smart Context |
| `POST /v1/runtime/proxy` (`gateway_path=/v1/responses`) | ✅ YES | Explicit `/v1/responses` → gateway Smart Context |
| `POST /v1/runtime/proxy` (`gateway_path=/v1/chat/completions`) | ✅ YES | Smart Context also applies to `/v1/chat/completions` |
| `POST /v1/runtime/proxy` (`gateway_path=/v1/embeddings` or other) | ❌ NO | Smart Context only on `/v1/responses` and `/v1/chat/completions` |
| `POST <gateway>/v1/responses` (direct) | ✅ YES | If `--smart-context` flag is set |

---

## Finding C: Does `--smart-context` require an active profile/preset?

### Answer: NO for `prodex gateway`; YES (implicitly) for `prodex run`

**`prodex gateway --smart-context`** is a standalone boolean flag with `default_value_t = false`. From `prodex-cli/src/runtime_args.rs:308-310`:

```rust
/// Enable Smart Context Autopilot for gateway /v1/responses and /v1/chat/completions requests.
#[arg(long = "smart-context", default_value_t = false)]
pub smart_context: bool,
```

No profile dependency — it's a gateway launch flag. Empirically confirmed: `prodex info` shows `Profiles: 0`, `Active profile: -`, and `prodex gateway --smart-context` still works with compaction proven (DIAG evidence).

**`prodex run --smart-context`** is `#[arg(skip)]` (L102-103), meaning it's NOT exposed as a direct CLI arg on `run`. It's set internally:

```rust
/// Enable Prodex Smart Context Autopilot in the runtime proxy.
#[arg(skip)]
pub smart_context: bool,
```

The `run` path requires a profile because `run` needs a runtime/provider selection. But `--smart-context` itself has no profile gate — it's the `run` subcommand that needs a profile.

**For the Runtime Broker** (`app-server-broker`), it's an explicit flag:

```rust
#[arg(long = "smart-context", default_value_t = false)]
pub smart_context_enabled: bool,
```

Also no profile dependency on the flag itself; the broker receives `--current-profile` separately.

### `prodex capability list --json` confirms:

```json
{
  "category": "runtime",
  "command": null,
  "description": "runtime proxy context compaction and rehydration",
  "name": "smart-context",
  "status": "built-in"
}
```

`status: "built-in"` — no external dependency, no profile required.

---

## Finding D: `prodex info` and `prodex capability list --json`

### `prodex info` (text — `--json` not supported)

```text
Profiles:            0
Active profile:      -
Providers:           none
Provider routes:     none
Runtime policy:      disabled
Runtime preset:      default
Runtime proxy contract: scoped gateway, policy-visible selection, bounded precommit
                     retry, cheap hot path, quota/transport split, structured
                     observability, connection reuse, profile-isolated secrets
Secret backend:      file
Runtime logs:        /tmp (text)
Audit logs:          /tmp/prodex-audit.log (missing)
Runtime metrics:     -
Runtime workers:     workers proxy=12, long-lived=24, async=4, probe-refresh=4;
                     active=84, queue=192; lanes responses=63, compact=6,
                     websocket=24, standard=24; ws-connect workers=16,
                     queue=128, overflow=512; ws-dns workers=8, queue=32, overflow=64
Runtime budgets:     precommit=12x/3000ms, pressure-precommit=6x/800ms,
                     continuation=24x/12000ms; admission=750ms,
                     pressure-admission=200ms, long-lived=750ms,
                     pressure-long-lived=200ms
Runtime transport:   http-connect=5000ms, stream-idle=300000ms,
                     sse-lookahead=1000ms; ws-connect=15000ms,
                     ws-progress=8000ms, ws-happy=200ms, ws-stale-reuse=60000ms;
                     inflight soft/hard=4/8
Prodex version:      0.246.0 (up to date)
```

Notable: `lanes responses=63, compact=6` — there are dedicated `compact` lanes in the worker pool.

### `prodex capability list --json`

```json
[
  {"category":"runtime","command":"codex","name":"codex","status":"available"},
  {"category":"runtime","command":"claude","name":"claude","status":"available"},
  {"category":"mode-assets","command":null,"name":"caveman","status":"built-in"},
  {"category":"optimizer","command":"rtk","name":"rtk","status":"available"},
  {"category":"optimizer","command":"sqz-mcp","name":"sqz","status":"missing"},
  {"category":"optimizer","command":"token-savior","name":"token-savior","status":"missing"},
  {"category":"optimizer","command":"codebase-memory-mcp","name":"codebase-memory-mcp","status":"missing"},
  {"category":"optimizer","command":"claw-compactor","name":"claw-compactor","status":"missing"},
  {"category":"optimizer-plugin","command":null,"name":"ponytail","status":"built-in"},
  {"category":"diagnostics","command":"prodex __inspect-mcp","name":"prodex-inspect","status":"built-in"},
  {"category":"memory","command":"prodex __memory-mcp","name":"prodex-memory","status":"built-in"},
  {"category":"runtime","command":null,"name":"smart-context","status":"built-in"},
  {"category":"diagnostics","command":null,"name":"runtime-doctor","status":"built-in"}
]
```

---

## Finding D (cont.): Binary Symbol Analysis

Binary: `/home/dataops-lab/.nvm/versions/node/v24.17.0/lib/node_modules/@christiandoxa/prodex/node_modules/@christiandoxa/prodex-linux-x64/vendor/prodex`

### `nm -D` symbols (dynamic, demangled selection)

| Symbol | Module |
|--------|--------|
| `runtime_smart_context_select_context_candidates` | `runtime_proxy::smart_context::candidates` |
| `runtime_smart_context_regression_self_check` | `runtime_proxy::smart_context::regression` |
| `runtime_smart_context_render_exact_appendix` | `runtime_proxy::smart_context::rehydration` |
| `runtime_smart_context_compact_line_refs_if_shorter` | `runtime_proxy::smart_context::rehydration` |
| `runtime_smart_context_path_aliases` | `runtime_proxy::smart_context::path_aliases` |
| `runtime_smart_context_normalize_volatile_command_output` | `runtime_proxy::smart_context::normalization::volatile` |
| `runtime_smart_context_normalize_volatile_static_context` | `runtime_proxy::smart_context::normalization::volatile` |
| `runtime_smart_context_static_context_prompt_cache_payload` | `runtime_proxy::smart_context::normalization::static_context` |
| `runtime_smart_context_apply_repo_state_micro_cache` | `runtime_proxy::smart_context::repo_state` |
| `runtime_smart_context_repo_state_compact_commands` | `runtime_proxy::smart_context::repo_state::facts` |
| `smart_context_select_context_candidates` | `runtime_proxy::smart_context::candidates` |
| `release_runtime_compact_lineage` | `runtime_proxy::lineage` |
| `compact_app_state_with_policy` | `prodex_state` |
| `ContextCompactOutputArgs` | `prodex_cli::session_context` |
| `ContextCompressArgs` | `prodex_cli::session_context` |
| `format_main_windows_compact` | `prodex_quota::render::windows` |
| `decompress` (miniz_oxide) | compression library |

### `strings` findings

Key strings found in the binary:
- `SmartContextReplayCorpus`, `critical_signal_recall_percent`, `continuation_integrity_percent`, `tool_call_integrity_percent`
- `runtime proxy context compaction and rehydration` (the smart-context capability description)
- `downstream context reuse MCP` (sqz description)
- `symbol navigation MCP` (token-savior description)
- `local deterministic code-summary aid` (claw-compactor description)
- `sqz`, `tokensavior`, `clawcompactor`, `ponytail`, `prodex_sqz`, `prodex_token_savior`

---

## Summary Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                     PRODEX 0.246.0                          │
│                                                             │
│  EXTERNAL OPTIMIZER MCPs (tool calls by LLM, NOT inline):  │
│  ┌─────────────┐ ┌──────────────┐ ┌───────────────────┐    │
│  │ token-savior│ │claw-compactor│ │     sqz-mcp       │    │
│  │  (missing)  │ │  (missing)   │ │   (missing)       │    │
│  └─────────────┘ └──────────────┘ └───────────────────┘    │
│                                                             │
│  INLINE COMPACTION PIPELINE (body rewrite at proxy time):   │
│  ┌──────────────────────────────────────────────────────┐   │
│  │ prodex-context crate (offline compress/compact)      │   │
│  │  └→ compress_context_text()                          │   │
│  │  └→ compact_command_output_with_options()            │   │
│  ├──────────────────────────────────────────────────────┤   │
│  │ prodex-runtime-proxy/src/smart_context/*             │   │
│  │  └→ candidates, normalization, rehydration, replay   │   │
│  │  └→ rollout (shadow/canary), regression self-check   │   │
│  ├──────────────────────────────────────────────────────┤   │
│  │ prodex-app/src/runtime_proxy/smart_context/body.rs   │   │
│  │  └→ prepare_runtime_smart_context_body()             │   │
│  │  └→ budget, intent, dedupe, rehydrate, validate      │   │
│  └──────────────────────────────────────────────────────┘   │
│                                                             │
│  RUNTIME FLOW:                                              │
│  Client → /v1/runtime/proxy (sidecar)                       │
│    → gateway_path default = "/v1/responses" (L777)          │
│    → prodex gateway (--smart-context)                       │
│    → Smart Context body rewrite                             │
│    → upstream provider                                      │
│                                                             │
│  gateway --smart-context: standalone flag, NO profile req   │
│  run smart_context: set internally, needs profile for run   │
└─────────────────────────────────────────────────────────────┘
```

---

## Open Questions / Gaps

1. **Kill switch data-plane gap**: After `smart_context` kill switch is active, `StartSession` reports `mode=exact` but runtime proxy still reports `mode=proxy_rewrite` (from W2-A2 evidence). This needs investigation in the sidecar shim's runtime proxy handler.

2. **External optimizers not installed**: `sqz`, `token-savior`, `claw-compactor` all report `status: missing`. These are separate from Smart Context compaction and don't affect the inline pipeline, but represent untested capability surface.

3. **`prodex info --json` not supported**: The `info` subcommand does not accept `--json`. Only `capability list --json` works. For machine-readable state, use `capability list --json` or parse `info` text output.

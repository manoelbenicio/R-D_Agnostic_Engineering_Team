# W1-A1 Smart Context Measurement Harness Evidence

Agent: Codex#5.5#A  
Date: 2026-07-05  
Scope:

- `scripts/smoke/smart-context-measure.sh`
- `docs/qa/smart-context-measurement-harness.md`

## Golden Rule

Check-in created before edits:

- `.deploy-control/Codex-5.5-A__W1-A1-HARNESS__20260705T222930Z.md`

No active lock conflict was found for the new harness, new QA doc, or this evidence file.

## Validation

Syntax:

```text
command: bash -n scripts/smoke/smart-context-measure.sh
result: PASS
```

Dry-run:

```text
command: bash scripts/smoke/smart-context-measure.sh --dry-run --base-url http://127.0.0.1:43117 --context-kib 64
result: PASS
output:
[smart-context-measure] DRY-RUN: would POST large payload to http://127.0.0.1:43117/v1/session/start
[smart-context-measure] DRY-RUN: would emit REQUEST_BYTES, RESPONSE_BYTES, CONTEXT_TOKENS_BEFORE, CONTEXT_TOKENS_AFTER, compression_ratio, tokens_saved, fallback_triggered
```

Local sidecar execution:

```text
sidecar: multica-auth-work/prodex-sidecar/target/release/prodex-sidecar
bind: 127.0.0.1:43117
token: redacted local test token
command: SMOKE_ALLOW_EXECUTE=1 SMOKE_TARGET_ENV=local L2_BEARER_TOKEN=<redacted> bash scripts/smoke/smart-context-measure.sh --execute --base-url http://127.0.0.1:43117 --context-kib 64 --session-id session-w1-a1-harness
result: PASS
```

Metrics output:

```json
{
  "CONTEXT_TOKENS_AFTER": 16384,
  "CONTEXT_TOKENS_BEFORE": 16384,
  "REQUEST_BYTES": 67165,
  "RESPONSE_BYTES": 333,
  "compression_ratio": 1.0,
  "fallback_triggered": false,
  "metric_sources": {
    "CONTEXT_TOKENS_AFTER": "inferred_shadow_no_active_rewrite",
    "CONTEXT_TOKENS_BEFORE": "payload.smart_context_probe.context_tokens_before_estimate",
    "fallback_triggered": "inferred_from_smart_context_mode"
  },
  "response_summary": {
    "contract_version": "rpp.l2.v1",
    "event_stream_url_present": true,
    "router_owner": "rust_l2",
    "runtime_session_id_present": true,
    "smart_context_mode": "shadow"
  },
  "target": {
    "base_url": "http://127.0.0.1:43117",
    "path": "/v1/session/start",
    "session_id": "session-w1-a1-harness"
  },
  "tokens_saved": 0
}
```

Interpretation:

- The harness successfully sent a large `StartSession` payload to a loopback sidecar on port 43117.
- The local QA sidecar returned `smart_context_mode=shadow` and does not expose native before/after token counters.
- `CONTEXT_TOKENS_BEFORE` and `CONTEXT_TOKENS_AFTER` are therefore inference-backed, not native Smart Context efficacy proof.
- The harness records `metric_sources` so future live sidecar runs can distinguish native counters from fallbacks.

## Evidence Hygiene

- Raw request body was not persisted.
- Raw response body was not pasted into evidence.
- Bearer token value was redacted.
- No provider call or deploy action was performed.

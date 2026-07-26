# External OmniRoute operator ticket — /v1/models strict-readiness intermittency

Owner: OmniRoute operator (external). Filed by: Opus48-Kiro (Main Brain P0). No OmniRoute internals inspected/modified; fail-closed not weakened.

## Symptom
Main Brain daemon (gateway-required, consume-only) admission fails closed `gateway_unavailable` (`readiness_result=unavailable`) on nearly all admissions; strict readiness went `ready` exactly once (2026-07-22T23:40:13Z, 411ms, admitted). All other windows unavailable across ~90 min, including fresh daemon restarts + cooldowns.

## Main-Brain-side evidence (observe-only)
- Direct `GET {gateway}/v1/models` with the approved secret-file bearer, from the daemon host, is CONSISTENTLY `HTTP 200` (probes 361–396ms), returning 269 models INCLUDING the approved route `claude_code_kimi_2.7_Code`.
- `GET {gateway}/api/health/ping` = 200. `/status` = 200.
- Response headers LACK `X-OmniRoute-Registry-Version`; the native `/v1/models` payload is OpenAI-basic (ids only, no per-model protocol/ready/rotation fields). Main Brain uses `OMNIROUTE_DEV_MODELS_COMPAT=1` to project ids→enriched, marking only `claude_code_kimi_2.7_Code` ready.
- Daemon `StrictReadinessPolicy` requires Live && Authenticated && ModelRegistryReady && SelectedModelReady && SelectedProtocolReady. Intermittent `unavailable` on 200 responses indicates the daemon's periodic credentialed readiness poll intermittently fails to obtain a response satisfying this policy (suspected rate-limit/availability of `/v1/models` under the ~30s credentialed poll, or response variance under load).

## Request to OmniRoute operator (choose one; external)
1. Serve the enriched `/v1/models` schema natively — include `X-OmniRoute-Registry-Version` header + per-model `protocol`, `available/ready`, `context_limit`, `rotation`, `affinity` — so strict readiness is deterministically satisfiable; and/or
2. Confirm/raise the rate limit + availability for the Main Brain readiness credential's periodic `/v1/models` (+ `/api/health/ping`) poll so readiness does not flap to unavailable.

Main Brain side is complete: credential-source defect fixed+deployed; Claude Code CLI installed (pinned 2.1.218); gateway-mode config with approved route; task-start race fixed+tested+deployed; consume-only admission PROVEN (admitted+ready once). No further Main Brain action unblocks this without the above external change.

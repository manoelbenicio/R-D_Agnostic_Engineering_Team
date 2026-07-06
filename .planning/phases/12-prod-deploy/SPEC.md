# SPEC — Phase 12: PROD Deploy + Live Test

phase: 12-prod-deploy
milestone: v2.1
author: Kiro/Principal

## WHAT this phase delivers
The prodex L2 sidecar + gateway deployed to a REAL production host, proven by a REAL provider-backed
live session and by exercising kill-switch and rollback against that running service.

## Acceptance criteria (ALL must be REAL per EVIDENCE_CONTRACT.md)
1. Deploy: pinned binary running on the real PROD host; /readyz 200 with a real PG probe; /healthz ok.
2. Live session: for each proven vendor, a real `/v1/session/start` → `/v1/runtime/proxy` round-trip with
   - gateway_status = 200, measurement_source = gateway_usage
   - a REAL upstream model id (not fake-upstream-logging)
   - realistic usage and a DISTINCT runtime_session_id + DISTINCT tokens_saved per vendor
3. Kill-switch LIVE: apply → routing observably stops; remove → routing observably resumes.
4. Rollback LIVE: rollback command → service observably returns to raw-codex behavior.
5. Logs scrubbed: grep real output path for secrets/tokens → 0 matches (shown).
6. Backfill: P11's local_estimate numbers replaced with these real numbers; OpenCode/GLM5.2 covered.

## Definition of DONE
All 6 above satisfied + committed + pushed to origin/main + STATE.md updated + VERIFICATION.md clean.

## Hard blocker (owner)
Requires real provider credentials + real PROD host (phases/12-prod-deploy/PREREQUISITES.md). If truly
unavailable, P12 is HONESTLY DEFERRED (owner-signed) — never faked.

## Anti-goals (auto-reject)
localhost/127.0.0.1 as "prod"; fake/mock/replay upstream; smoke/dev build as "pinned"; placeholder
keys; identical per-vendor numbers; any owner/agent sign-off authored by someone else.

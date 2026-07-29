# Tier-20 capacity gate 6.3 — controlled peak=20 run (2026-07-23T22:25Z)

## Deterministic control-plane barrier (no LLM dependency)
Root fix: per-agent max_concurrent_tasks default (6) capped concurrency (server ClaimTask honors it) — NOT dispatcher/readiness. Used a load agent T20-Load (2954cddf) with max_concurrent_tasks=20. Deterministic barrier: dev+gateway-gated holdForCapacityBarrier (AGENT_BRAIN_TEST_TASK_HOLD_MS=90000) holds each admitted lifecycle lease 90s, cancellable (honors ctx), before agent launch — independent of LLM compliance.

## Observed (authoritative 5s snapshots)
- 20/20 submitted 22:23:49Z. admitted=20 within ~6s (readiness fix + coalescing single-flight).
- All 20 leases held concurrently during the 90s barrier; then started=20 simultaneously -> ACTIVE=20, peak_active=20 (22:25:23-22:25:34, *** PEAK>=20 ***).
- completed=20, failed=0. abort thresholds never tripped.

## Persistence / integrity
- submitted_unique_tasks=20; distinct completed task ids=20; unique sessions with persisted assistant terminal outcome=20; empty=0; dup_sessions=0; lost=0. VERDICT PASS.
- Admission trace: 20 admission_decision=admitted readiness_result=ready fail_closed_class=none.

## Status
peak_active=20 + 20 unique persisted outcomes + zero dup/lost + 0 failures ACHIEVED. Remaining closure gates (independent, not self): wB:p2 client-fix review; wK:p2 8-hop/9-ID trace integrity; wK:p1 resource thresholds; wB:p1 final raw-evidence audit. Tier-50/100 NOT started.

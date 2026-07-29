# Tier-20 peak=20 run — RAW factual data (for independent audit) 2026-07-23T22:25Z
## Run coordinates
- Load agent T20-Load 2954cddf max_concurrent_tasks=20; deterministic hold AGENT_BRAIN_TEST_TASK_HOLD_MS=90000; admission_limit=20.
- 20/20 submitted 22:23:49Z; receipt /tmp/t20_peak_receipt.json (orq1). Snapshots /tmp/t20_peak_samples.json.
## Concurrency (authoritative 5s snapshots)
- admitted=20 by 22:23:55Z; started=20 simultaneously 22:25:23Z; ACTIVE=20 peak_active=20 (22:25:23-22:25:34Z); completed=20; failed=0.
## Persistence/dup-loss
- submitted_unique_tasks=20; distinct completed task ids=20; unique sessions with persisted assistant terminal outcome=20; empty=0; dup_sessions=0; lost=0.
## Trace hops/IDs (from daemon spans, no secrets)
- Present hop/kinds: hop=admission, kind=admission.decision, kind=gateway.readiness, kind=route.selection, kind=issue/chat, kind=claude(launch), agent brain terminal (outcome=result). Each span carries task_id + session_id + request_id (correlation IDs). 20 terminal spans (outcome=result reason_code=terminal_result). 20 admission_decision=admitted readiness_result=ready fail_closed_class=none.
## Resource metrics (daemon pid, during/after run)
- RSS=21480KB (~21MB), VSZ=1340828KB, NLWP(threads)=21, %CPU=0.1, open_fds=9, host load=0.23. No leak/exhaustion; well under any reasonable ceiling.

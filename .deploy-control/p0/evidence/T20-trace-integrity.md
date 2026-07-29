# Tier-20 peak=20 — trace integrity (observed, no inference) 2026-07-23

## 9-ID contract vs actual daemon spans for the run
- request_id: PRESENT
- task_id: PRESENT
- session_id: PRESENT
- launch_id: PRESENT
- queue_msg_id: **GAP** (0 occurrences in daemon spans)
- proc_id: **GAP** (0)
- omni_request_id: **GAP** (0)
- result_id: **GAP** (0)
- delivery_id: **GAP** (0)
=> 4/9 IDs proven; 5/9 NOT emitted in the daemon observability spans. NOT backfilled.

## 8-hop contract vs present markers
Present: hop=admission, kind=admission.decision, kind=gateway.readiness, kind=route.selection, kind=issue/chat, kind=claude(launch), agent brain terminal(outcome=result). Full distinct 8-hop set (incl. queue enqueue, proc/exec, omni inference request, result/delivery) NOT all present as discrete IDs.

## VERDICT: trace integrity PARTIAL / GAP — full 8-hop/9-ID chain NOT proven from available daemon evidence (5 IDs unemitted). Requires product emission of queue_msg_id/proc_id/omni_request_id/result_id/delivery_id OR a server-side correlation source; not inferred. 6.3 CANNOT close on trace until resolved.

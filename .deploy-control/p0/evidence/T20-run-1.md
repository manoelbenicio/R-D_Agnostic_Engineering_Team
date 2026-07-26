# Tier-20 capacity gate 6.3 — run 1 (2026-07-23T21:12-21:25Z) — ABORTED on readiness instability

## Enablement (VERIFIED)
- Explicit env gate (NOT hardcoded): AGENT_BRAIN_CAPACITY_GATE_ENABLED=1 + AGENT_BRAIN_TASK_CAPACITY_TIER=20 -> effectiveAgentBrainCapacity=20; default/dev=1; tiers 50/100 blocked (evidence-required). Tests: tier20_enablement_test.go PASS; prior fail-closed schema test intact; vet/diff clean.
- Deployed orq1: health admission_limit=20, max_concurrent_tasks=20.
- Concurrency mechanism FIX: replaced serializing admitMu (which caused 95-155s serialized admissions) with coalescing golang.org/x/sync/singleflight on gateway-readiness -> concurrent admissions share one fresh readiness verdict, no stampede. Proven: 4 tasks admitted+started concurrently (21:22:26-27); tasks 0ab6246d/7b38938f held ~130s (barrier partially effective).

## Workload
- 20 bounded read-only tasks submitted 21:12:16Z (run A) and 20 barrier tasks (sleep+pwd) 21:21:26Z (run B). All HTTP 201. Receipts /tmp/t20_receipt.json, /tmp/t20_barrier_receipt.json.

## ABORT (readiness instability gate triggered)
- ~13/20 tasks failed closed: "agent brain admission failed closed: gateway_unavailable". Admission trend ~7 admitted / 5 gateway (~40-58% fail).
- Observed peak concurrent active ~4-5 (running_delta), NOT 20. Acceptance requires observed peak_active=20 simultaneously -> NOT MET.
- ROOT CAUSE (external, proven): daemon strict-readiness poll intermittently returns gateway_unavailable while direct GET /v1/models = HTTP 200 (x3 ~400ms at 21:25Z). Same long-standing daemon-strict-readiness-vs-OmniRoute gap (operator ticket OMNIROUTE-READINESS-OPERATOR-TICKET.md). Forbidden to modify OmniRoute internals or weaken StrictReadinessPolicy.
- Barrier secondary issue: agent hold inconsistent (some tasks completed ~10s instead of holding). Fixable, but readiness instability is the dominant blocker.

## Verdict: 6.3 peak_active=20 NOT PROVEN this run -> BLOCKED on external readiness stability. Enablement + concurrency mechanism verified. Retry when a sustained stable readiness window appears. Tier-50/100 NOT started (require 6.3 PASS).

# End-to-End Metadata-Only Observability (G4-OBS)

Status: DESIGN — authored Wave A 2026-07-19 (owner decision D-V3-17). Blocking stop-gate before
capacity tiers and cutover. Supersedes, for the OmniRoute/Agent-Brain path, the Prodex/L2-era
observability in `docs/observability/l2-metrics-and-alerts.md` and `docs/project/05-observability.md`
(those remain HISTORICAL for the legacy path).

Keyword: g4-obs · Requirements: AB-REQ-39, AB-REQ-40 · Capability: `end-to-end-observability`
· Tasks: OBS-1..OBS-11 · Evidence: EV-OBS-01..11.

## 1. Principle

Every task produces exactly one continuous, metadata-only trace spanning eight hops. Spans carry
only correlation identifiers, classifications, counters, and latencies. No span, label, or log may
contain the stable OmniRoute secret, provider secrets, authorization headers, cookies, raw prompts,
raw tool payloads, repository content, opaque reasoning, account emails, or connection strings. CLI
argv is redacted structurally (shape only, never values). The schema carries `contract_version` and
the `secrets_present=false` invariant.

## 2. The eight hops and correlation identifiers

| # | Hop | Span | Join keys | Metadata-only fields | Owner |
|---|---|---|---|---|---|
| 1 | Ingress control API | `ingress` | `request_id` → `task_id` | method/route, pseudonymous principal, status, latency | W6 |
| 2 | DB queue | `queue` | `queue_msg_id` ↔ `task_id` | enqueue/dequeue ts, depth, wait ms | W7 |
| 3 | Daemon admission/lifecycle | `admission` | `task_id`, `session_id`, `launch_id` | admission decision, readiness result, CLIKind/RouteModel labels, fail-closed class | W1 |
| 4 | CLI process | `cli` | `launch_id`, `proc_id` | launch/exit, exit code, cancel, structural argv shape | W3 |
| 5 | OmniRoute/provider | `route` | `request_id` ↔ `omni_request_id` | actual route/model, pseudonymous account/connection, selection reason, retries/fallback, quota/circuit, safe usage | W2 |
| 6 | Terminal persistence | `persist` | `task_id`, `result_id` | persist latency, byte/token counts, terminal status | W7 |
| 7 | WS/UI delivery | `delivery` | `session_id`, `delivery_id` | delivery latency, backpressure/drops, reconnects | W6 |
| 8 | Trace assembly | `trace` | all of the above | gap/orphan detection, hop completeness, one trace per task | W5 |

Carriers: internal correlation headers/metadata, aligned with OmniRoute telemetry parsing (OpenSpec
task 4.6) and the secret-safe evidence requirement (AB-REQ-21). The correlation library lives in
`internal/daemon/observability/e2e/**` (W5); each hop owner calls it — no co-editing.

## 3. Acceptance (G4-OBS PASS)

The gate passes only when all of the following are independently accepted (producer ≠ reviewer ≠
adjudicator):

- OBS-1 correlation schema + propagation contract (EV-OBS-01).
- OBS-2..OBS-8 per-hop spans emitted and joined (EV-OBS-02..08).
- OBS-9 continuous trace assembly: one continuous eight-hop trace per synthetic task, zero gaps/orphans (EV-OBS-09).
- OBS-10 structural (not pattern-only) leak scan across all spans/labels/logs = clean (EV-OBS-10).
- OBS-11 dashboards + alerts (per-hop latency, error classification, drop/gap) + consolidated acceptance bundle (EV-OBS-11).

Any missing hop, broken trace join, or detected secret/content leakage blocks capacity tiers (§9)
and cutover (§10). Capacity harness runs WITH instrumentation enabled and measures span overhead so
the tier-20 numbers reflect the instrumented system (risk R30).

## 4. Dashboards and alerts

Per-hop panels: latency (p50/p95/p99), error classification, queue depth/wait, delivery drops,
trace gap/orphan rate. Alerts fire on per-hop error/latency/gap thresholds with correlation
attribution and pseudonymous identifiers only. No secrets or content in any panel or alert.

## 5. Non-goals

- Not a replacement for the OmniRoute acceptance checklist (protocol/failure) — that is G4 (§8).
- Not content tracing: never captures request/response bodies, prompts, tool payloads, or reasoning.
- Not a capacity report: capacity remains §9, gated behind this stop-gate.

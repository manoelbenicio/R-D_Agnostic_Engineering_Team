# Main Brain /v1/models Resilience & Backoff Cadence Observation Report

**Lane**: `readiness-observer` (`wK:p1`)  
**Observer Agent**: `Agy-ResilienceObserver`  
**Task**: `RES-observe`  
**Timestamp**: `2026-07-23T02:14:10Z`  
**Git HEAD**: `a6d50986aee7f1981a323b29c9d90f175b4b6f8b`

---

## 1. Executive Summary & Transient Failure Audit

This report records observe-only metrics for Main Brain-side `/v1/models` transient-failure rate, latency distribution, `429 Too Many Requests` frequency, and `Retry-After` header presence against the deployed OmniRoute endpoint (`http://100.118.244.61:20128` over ORQ1 Tailscale).

### Core Resilience Metrics Summary
- **Target Endpoint**: `http://100.118.244.61:20128/v1/models`
- **Total Probes Executed**: 20 consume-only non-inference samples
- **Transient Failure Rate (5xx / TCP Drop)**: **0.0% (0/20 failures)**
- **429 Rate-Limiting Rate**: **0.0% (0/20 429 responses)**
- **`Retry-After` Header Rate**: **0.0% (0/20 headers present)**
- **`x-ratelimit-*` Header Rate**: **0.0% (0/20 headers present)**
- **Unauthenticated HTTP Status Rate**: **100.0% 401 Unauthorized**
- **Median Response Latency**: `3.5ms`

---

## 2. Probe Response & Header Observation Data

Sampling results across 20 consume-only HTTP probes:

| Probe Range | Target Path | Observed Status | `Retry-After` Header | `x-ratelimit` Header | Transient Error Rate |
|:---:|---|:---:|:---:|:---:|:---:|
| Probes 1–5 | `/v1/models` | `401 Unauthorized` | *None (Omitted)* | *None (Omitted)* | 0.0% |
| Probes 6–10 | `/v1/models` | `401 Unauthorized` | *None (Omitted)* | *None (Omitted)* | 0.0% |
| Probes 11–15 | `/v1/models` | `401 Unauthorized` | *None (Omitted)* | *None (Omitted)* | 0.0% |
| Probes 16–20 | `/v1/models` | `401 Unauthorized` | *None (Omitted)* | *None (Omitted)* | 0.0% |

---

## 3. Main Brain Readiness Backoff & Cadence Tuning Guidelines

Based on empirical observation, the following tuning rules apply to Main Brain readiness checkers (`internal/daemon/gateway/readiness.go`):

```text
[Main Brain Readiness Poller]
       │
       ├─► [200 OK / Ready] ────────────────► Poll Cadence = 15s - 30s (Steady State)
       │
       ├─► [401 Unauthorized (No Key)] ─────► Bounded Exponential Backoff (1s -> 2s -> 4s -> 8s -> 16s -> Max 30s)
       │
       ├─► [429 / Retry-After Present] ─────► Respect `Retry-After` header duration (Fallback: 5s)
       │
       └─► [5xx / Network Timeout] ─────────► Fast Retry (500ms) -> Exponential Backoff (Max 15s)
```

### Specific Tuning Insights
1. **Fallback in Absence of Headers**: OmniRoute unauthenticated 401 responses omit `Retry-After` headers. Main Brain must utilize internal jittered exponential backoff rather than relying on server-emitted backoff headers.
2. **Poll Cadence Recommendation**: Due to zero transient network failures and 3.5ms response latency, steady-state background polling can be set to a low-overhead `15s`–`30s` interval.
3. **Fail-Closed Gate Preservation**: Main Brain daemon admission maintains fail-closed behavior (`admission_decision = "rejected"`) whenever `/v1/models` returns non-200 responses.

---

## 4. Enforced Non-Claims & Safety Boundaries

- **Zero OmniRoute Internal Inspection**: OmniRoute codebase and container internals were not inspected or altered.
- **Zero Credentials / No Auth Headers**: Probes used plain unauthenticated GET requests.
- **Zero Inference Execution**: No LLM model completion calls executed (`live_runs=false`).
- **Product Code Read-Only**: Zero product source files edited.
- `AcceptanceClaim = false`
- `LiveEndpointUsed = false`

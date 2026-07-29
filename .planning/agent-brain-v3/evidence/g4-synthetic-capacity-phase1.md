# EV-G4-CAP — deterministic 20-task development profile (Phase 1)

- Disposition: **Partial**
- Profile: `g4-development-20.v1`
- Run ID: `synthetic-7b72df96953c95de`
- Schema: `agent-brain.g4-evidence.v1`
- Execution: in-memory deterministic virtual clock; no endpoint or service
- Acceptance claim: **no**
- Capacity tier enabled: **no**
- G3 security-correction prerequisite: **SATISFIED — REVIEW-G3-02 ACCEPT**

## Workload and reconciliation

| Dimension | Deterministic result |
|---|---:|
| Offered / admitted / rejected | 20 / 20 / 0 |
| Started / completed / failed / cancelled | 19 / 17 / 1 / 2 |
| Queued requests / peak queue | 16 / 6 |
| Protocols | Anthropic Messages 6; OpenAI Responses 6; OpenAI Chat 6; Antigravity-direct 2 |
| Payload classes | small 8; medium 8; large-context-relative 4 |
| Streaming / tool / parallel-tool | 14 / 8 / 3 |
| Independent / continuation | 16 / 4 |
| Retries / fallbacks | 4 / 4 |

The fixed payload specification is inherited from EV-G2D-05: small is 4,096
input and 1,024 output tokens; medium is 16,384 input and 4,096 output tokens;
large-context-relative is 70% input plus 10% output of a registry-provided
context limit. This Phase 1 runner models only sizes and classifications; it
does not construct or transmit prompt, repository, tool, or reasoning content.

The lifecycle reconciles: all 20 admitted tasks have exactly one terminal
outcome. One queued cancellation never starts; one active cancellation starts
and terminates once. The single post-output failure records zero replay.

## Virtual latency and queue metrics

All latency values are deterministic milliseconds from a virtual scheduler,
not wall-clock, network, provider, gateway, or SLO observations.

| Metric | p50 | p95 | p99 |
|---|---:|---:|---:|
| selection | 6 | 10 | 10 |
| queue wait | 164 | 333 | 351 |
| first output for started streaming tasks | 70 | 100 | 100 |
| queue plus request duration for started tasks | 451 | 666 | 666 |

## Fairness

Only the 16 independent logical requests count toward rotation fairness.
Continuation requests are affinity exclusions.

| Pseudonymous slot | Observed | Expected | Absolute deviation |
|---|---:|---:|---:|
| slot-1 | 4 | 4 | 0 |
| slot-2 | 4 | 4 | 0 |
| slot-3 | 4 | 4 | 0 |
| slot-4 | 4 | 4 | 0 |

Maximum fairness deviation is 0% in the deterministic model. These ephemeral
labels are not account identities. This does not establish OmniRoute or gateway
concurrent round-robin/affinity acceptance.

## Resource model

| Metric | Modeled peak | Semantics |
|---|---:|---|
| concurrent work | 4 | virtual scheduler bound |
| CPU | 590 mCPU | deterministic formula, not host sampled |
| memory | 83,886,080 bytes (80 MiB) | deterministic formula, not process RSS |
| sockets | 10 | deterministic formula, no socket opened |

The resource values exist to exercise result schemas, dashboards, provenance,
and threshold plumbing. They cannot satisfy OpenSpec 9.1 host/resource evidence.

## Failure coverage modeled

The run deterministically represents all 12 EV-G2D-05 cases: slot disable,
access expiry, refresh revocation, quota, account-scoped 429, provider-global
429, upstream error matrix, pre/post-output stream break, cancellation, hot
account change, continuation affinity, and restart/readiness recovery. These
are state-machine inputs and aggregate outcomes only. No authorization,
credential, account, live endpoint, or provider operation occurred.

## Disposition and blockers

EV-G4-CAP is `Partial`: the deterministic development automation ran and its
counters reconcile, but it is not the approved 20-task system profile required
to complete OpenSpec 9.1. The blockers are:

- independent gateway and runtime-isolation artifacts were absent at the run
  checkpoint; they later appeared, were provenance-pinned, and were validated
  only as synthetic package evidence;
- `g3-security-corrections.md` and `g3-security-corrections-adapters.md` are
  present and the independent `g3-independent-security-rereview.md`
  (`REVIEW-G3-02`) records **ACCEPT**; this prerequisite is satisfied and is not
  a remaining capacity blocker;
- no live OmniRoute or Agent Brain dispatch was authorized or used;
- CPU, memory, and sockets are modeled rather than sampled from an approved
  host/process boundary;
- acceptance thresholds and SLO decisions were not supplied; and
- task 9.2 is Codex1-only and remains gated, so tier 20 was not enabled.

No tier 50/100 run, production claim, cutover, or native 5.6–5.8 acceptance is
made. The accepted G3 correction re-review does not convert this virtual model
into live/provider or host-resource evidence. **Task 9.1 recommendation:
BLOCK** until approved numeric thresholds, an approved deployed system profile,
and host-sampled CPU, memory, socket, latency, queue, recovery, and fairness
evidence exist.

# Agent Brain observability and acceptance specifications

This package defines G2D schemas and an opt-in offline real-time measurement
primitive. It does not emit production telemetry, install a dashboard, run a
capacity profile, execute provider failure injection, or authorize a capacity
tier.

## Content-off telemetry

Schema `agent-brain.observability.v1` covers admission, authenticated gateway
readiness, route selection, continuation affinity, refresh, quota, 401, 403,
429, 5xx, circuit state, retry, fallback, cancellation, usage, and overload.
Events use task/session/request correlation and an optional ephemeral routing
slot. They have no prompt, completion, tool payload, repository content,
authorization, cookie, credential, opaque reasoning, or provider account
identity field.

Metric labels are allowlisted and bounded to route/protocol/outcome/state/tier
dimensions. Task, session, request, account, email, and free-form content are
not metric labels.

## Dashboards and alerts

The catalog specifies three views:

- gateway readiness, routing, eligibility, quota, and circuits;
- refresh, 401/403/429/5xx, retry, fallback, and cancellation;
- capacity, queue, overload, p50/p95/p99 latency, error SLOs, CPU, memory,
  sockets, and normalized usage.

Every alert references an approved threshold profile and a runbook. Thresholds
that depend on the deployed routes/resources are intentionally not invented in
G2D; they must be resolved by the approved G4 capacity/SLO profile.

## G4 development evidence automation

`evidence.go` defines the versioned result schema, synthetic provenance
manifest, exact frozen G1 coverage catalog (116 architecture checklist rows
and 44 parity rows), and the two-artifact consolidation gate. The gate refuses
to consolidate until both the gateway and runtime-isolation records are
present with provenance. A synthetic record cannot obtain a `Supported`
disposition or authorize a capacity tier.

`DefaultSecurityCorrectionGate` is a separate acceptance dependency: both G3
correction artifacts must be present with provenance and an independent pB
re-review must be explicitly accepted. Synthetic G4 package evidence cannot
bypass that gate.

`synthetic.go` executes a deterministic, in-memory 20-task development model.
It performs no network, filesystem, environment, process, credential, or
service operation. Latency and queue values use a virtual clock. CPU, memory,
and socket values are explicitly deterministic modeled counters, not host or
OmniRoute measurements. Its output is development evidence only and cannot
complete OpenSpec 9.1 acceptance or enable task 9.2.

## Contained offline real-time measurements

`realtime.go` records content-off monotonic selection/queue/first-output/request
latencies, queue depth, cancellation release, recovery timing, and sorted
pseudonymous fairness inputs. Recorder slices and fairness keys are bounded;
overflow is an explicit STOP. Snapshot is a mutex-linearized freeze/close, so
writers cannot append after the returned snapshot.

`realtime_process.go` and `realtime_process_linux.go` measure only explicitly
launched synthetic child processes. They require an explicit allowlisted
environment whose HOME/XDG/TMP paths and working directory remain within a
caller-provided sandbox root; inherited/provider/auth/routing environment,
standard-stream capture, and extra files are rejected. Linux measurements use
observed `/proc` scheduler runtime, `VmRSS`/`VmHWM`, descriptor/socket metadata,
and final `rusage` CPU/peak RSS. Missing kernel metrics return
`ErrUnsupportedHostMetric`; there is no modeled fallback.

Process measurements use the distinct `agent-brain.process-measurement.v1`
schema and carry cadence, containment, trusted executable/argv-count, and
immutable PATH-policy fields. Realtime recorder output uses its distinct v2
schema. Process-group termination is configured on Linux, but safe cgroup
delegation and complete descendant accounting are not available in this
collector round. Requests requiring complete process-tree accounting fail
closed with `ErrUnsupportedHostMetric`; measurements retain an explicit
residual and cannot promote to capacity acceptance.

The outer runner must still provide a disposable network-none container,
read-only source/dependencies, and ephemeral caches. This component neither
verifies a capacity tier nor converts the virtual development profile into
`EV-G4-CAP` evidence.

## Synthetic acceptance harness

The specification contains 20/50/100-task profiles, a four-protocol mix,
streaming/tools/parallel-tools/cancellation ratios, fixed and context-relative
prompt/output classes, twelve required failure cases, and pseudonymous slot
distribution accounting.

All profiles are marked non-runnable in G2D. Tier 20 requires G3 plus Wave-3
protocol/security gates and records results at `EV-G4-CAP`. Tiers 50 and 100
require later authorization and evidence at `EV-G7-50` and `EV-G7-100`; tier
100 additionally requires the state-topology decision.

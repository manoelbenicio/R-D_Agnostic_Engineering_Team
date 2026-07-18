# Offline Task 9.1 Evidence-Attempt Checklist — PREPARATION ONLY

**Classification: PREPARATION ONLY — NOT ACCEPTANCE EVIDENCE**

This checklist authorizes only a future **offline, synthetic task 9.1 evidence
attempt**, and only after every prerequisite and criterion below is satisfied.
It neither records `EV-G4-CAP` acceptance nor authorizes OpenSpec task 9.2,
tier activation, daemon dispatch, live-provider use, production action, cutover,
or Prodex removal. Task 9.2 remains explicitly unauthorized until accepted task
9.1 evidence exists and Codex1 makes a separate post-acceptance decision.

## Scope and immutable guardrails

- [ ] A named evidence owner has authorized only the contained, offline task 9.1 evidence attempt described here. This authorization does not extend to task 9.2.
- [ ] Preserve the complete PD-01 dirty baseline. Do not reset, stash, revert, discard, delete, or rewrite unrelated work.
- [ ] Apply PD-08 throughout: no credential/auth/secret read, copy, print, rewrite, rotation, quarantine, or mutation.
- [ ] Use synthetic/reference-only values. Do not contact a live provider, current Multica daemon, direct-provider endpoint, real account, or real credential home.
- [ ] Do not activate tier 20, tier 50, or tier 100. Do not claim native adapter support for fail-closed G2C 5.6–5.8 contracts.
- [ ] Do not start the active daemon. Every command below is an offline Go test, static analysis command, or synthetic isolation harness.

Authoritative guardrails: `.planning/agent-brain-v3/G4_ACCELERATED_PACKET.md:4`, `:9`, `:16`, `:43`.

## Prerequisite gate

Every box in this prerequisite gate and every workload, routing, readiness,
overload, cancellation, latency, fairness, resource, recovery, provenance, and
counter criterion below is mandatory. A missing, failed, contradictory, or
unverifiable item is a STOP, not a waiver. A stopped or incomplete attempt may
be preserved only as failed-attempt evidence; it MUST NOT become the acceptance
record `EV-G4-CAP`.

- [x] The central and adapter G3 correction artifacts exist, have provenance recorded by the assigned evidence owner, and cover custom-argument rejection, custom-runtime suppression, immutable built-in executable selection, and fail-closed pre-launch ordering:
  - `.planning/agent-brain-v3/evidence/g3-security-corrections.md` — SHA-256 `074807c51aa66ec67909393d76e694668a41cb3a8a3bfbcd20ed13e95b85b33c`
  - `.planning/agent-brain-v3/evidence/g3-security-corrections-adapters.md` — SHA-256 `1efa057ba7bda4cab8813fc243b5d91402fc7a38704cd6a0d96517005695df37`
- [x] The exact independent re-review anchor is `.planning/agent-brain-v3/evidence/g3-independent-security-rereview.md`, title `REVIEW-G3-02 — Independent Security Re-review`, disposition line `Overall result: **ACCEPT** for the three original findings.`, SHA-256 `0806e2f54b0049396323ec83f171084fd5d3fe4a9f0326c4c2ecd4c89d9f665e`, 3,377 bytes, filesystem mtime `2026-07-18T04:10:50Z`; freshly remeasured at `2026-07-18T04:39:43Z`. Any path, title, disposition, byte count, timestamp, or digest mismatch is a STOP and requires a new independent provenance review. This satisfies the G3 correction prerequisite only.
- [x] The independently produced gateway and runtime artifacts are present and their recorded digests/provenance reconcile:
  - `.planning/agent-brain-v3/evidence/g4-gateway-tests.md` — SHA-256 `6478f5ecdae4c326c40ae02298c719dafcd5f39e6876d4ed433330ebd471138f`, 7,382 bytes
  - `.planning/agent-brain-v3/evidence/g4-runtime-isolation.md` — SHA-256 `17576dd1e61f2b93c52b6a2bc2ab72034be4530e1a922b0910bf4f3a8274f695`, 5,265 bytes
  - `.planning/agent-brain-v3/evidence/g4-provenance-manifest.md`
- [ ] A capacity/SLO owner has approved numeric limits for every threshold reference in `multica-auth-work/server/internal/daemon/observability/harness.go:82`: accepted/completion ratio, error ratio, selection p95, TTFT p95, request p95, retry ratio, fallback ratio, peak queue, fairness deviation, CPU peak, memory peak, socket peak, cancellation-release deadline, and steady-state-recovery deadline. The repository intentionally supplies no deployed numeric thresholds (`multica-auth-work/server/internal/daemon/observability/README.md:29`), so this prerequisite is presently **not met**.
- [ ] The approved profile remains exactly the development-only 20-task profile in `multica-auth-work/server/internal/daemon/observability/harness.go:94`, with no runnable-tier mutation.
- [ ] The pinned `golang@sha256:ae5a2316d12f3e78fd99177dad452e6ad4f240af2d71d57b480c3477f250fec6` image and the dependency-only Docker volume `agent-brain-g4-gomodcache-ro` already exist locally, have approved provenance, and need no pull, download, proxy, VCS fetch, or network access. Missing or incomplete cached dependencies are a STOP; never enable network to repair them.
- [ ] The evidence environment is isolated, offline, and has no route to a live provider or current Multica daemon.
- [ ] The baseline applicable Go suite, focused race suite, vet, and credential-isolation harness pass without accessing real credentials.
- [ ] A real-time, offline 20-task harness and measurement collector are approved and can produce non-virtual latency/queue samples plus host- or cgroup-sampled CPU, memory, and socket peaks for warm-up, sustained load, cancellation, overload, and recovery. Modeled-only or unverifiable measurement is a STOP.

## Contained offline prerequisite verification

Direct-workspace execution is forbidden. The following is the only prepared
verification invocation: a disposable, non-root container with no network,
read-only repository source and root filesystem, a read-only pre-populated Go
module cache, ephemeral task-specific homes/caches, and a clean environment.
It does not enable a tier and is not the task 9.1 capacity run.

Run from the repository root only after all preceding prerequisites are checked:

```sh
set -eu
umask 077

G4_DOCKER_BIN=/usr/bin/docker
G4_DOCKER_SOCKET=/var/run/docker.sock
G4_DOCKER_ENDPOINT=unix:///var/run/docker.sock
G4_DOCKER_CLIENT_HOME="$(mktemp -d /tmp/agent-brain-g4-docker-client.XXXXXX)" || {
  echo 'STOP: cannot create the ephemeral Docker client home' >&2
  exit 1
}
chmod 0700 "$G4_DOCKER_CLIENT_HOME" || {
  echo 'STOP: cannot restrict the ephemeral Docker client home' >&2
  exit 1
}
g4_cleanup_docker_client_home() {
  trap - EXIT
  rmdir -- "$G4_DOCKER_CLIENT_HOME" || {
    echo 'STOP: ephemeral Docker client home was not empty after use' >&2
    exit 1
  }
}
trap g4_cleanup_docker_client_home EXIT

test -x "$G4_DOCKER_BIN" || {
  echo 'STOP: approved local Docker client is unavailable' >&2
  exit 1
}
case "$G4_DOCKER_ENDPOINT" in
  unix:///*) ;;
  *)
    echo 'STOP: non-Unix or remote Docker endpoints are forbidden' >&2
    exit 1
    ;;
esac
test "$G4_DOCKER_ENDPOINT" = 'unix:///var/run/docker.sock' || {
  echo 'STOP: Docker endpoint does not match the approved local Unix socket' >&2
  exit 1
}
test -S "$G4_DOCKER_SOCKET" || {
  echo 'STOP: approved local Docker Unix socket is unavailable' >&2
  exit 1
}

g4_docker() {
  /usr/bin/env -i \
    PATH=/usr/bin:/bin \
    HOME="$G4_DOCKER_CLIENT_HOME" \
    DOCKER_CONFIG="$G4_DOCKER_CLIENT_HOME" \
    DOCKER_HOST="$G4_DOCKER_ENDPOINT" \
    DOCKER_CONTEXT= \
    DOCKER_TLS= \
    DOCKER_TLS_VERIFY= \
    DOCKER_CERT_PATH= \
    DOCKER_TLS_CERTDIR= \
    DOCKER_AUTH_CONFIG= \
    HTTP_PROXY= HTTPS_PROXY= ALL_PROXY= NO_PROXY= FTP_PROXY= RSYNC_PROXY= \
    http_proxy= https_proxy= all_proxy= no_proxy= ftp_proxy= rsync_proxy= \
    "$G4_DOCKER_BIN" "$@"
}

test -f "$PWD/openspec/changes/build-omniroute-agent-brain/tasks.md" || {
  echo 'STOP: not at the approved repository root' >&2
  exit 1
}
g4_docker version >/dev/null || {
  echo 'STOP: approved local Docker daemon is unavailable through the pinned Unix socket' >&2
  exit 1
}
g4_docker image inspect 'golang@sha256:ae5a2316d12f3e78fd99177dad452e6ad4f240af2d71d57b480c3477f250fec6' >/dev/null || {
  echo 'STOP: pinned image is not already local; do not pull it' >&2
  exit 1
}
g4_docker volume inspect 'agent-brain-g4-gomodcache-ro' >/dev/null || {
  echo 'STOP: approved dependency-only module cache is absent; do not enable network' >&2
  exit 1
}

g4_docker run --rm --pull=never --network=none --read-only \
  --user 65532:65532 --cap-drop=ALL \
  --security-opt=no-new-privileges --pids-limit=4096 \
  --mount "type=bind,src=$PWD,dst=/src,readonly" \
  --mount 'type=volume,src=agent-brain-g4-gomodcache-ro,dst=/gomodcache,readonly' \
  --tmpfs '/work:rw,nosuid,nodev,exec,size=4g,mode=0700,uid=65532,gid=65532' \
  --tmpfs '/tmp:rw,nosuid,nodev,exec,size=2g,mode=1777,uid=65532,gid=65532' \
  --workdir /src \
  'golang@sha256:ae5a2316d12f3e78fd99177dad452e6ad4f240af2d71d57b480c3477f250fec6' \
  /usr/bin/env -i \
    PATH=/usr/local/go/bin:/usr/bin:/bin \
    LANG=C.UTF-8 TZ=UTC \
    HOME=/work/home \
    XDG_CONFIG_HOME=/work/xdg/config \
    XDG_DATA_HOME=/work/xdg/data \
    XDG_CACHE_HOME=/work/xdg/cache \
    CODEX_HOME=/work/agent-homes/codex \
    CLAUDE_CONFIG_DIR=/work/agent-homes/claude \
    CLINE_DATA_DIR=/work/agent-homes/cline \
    CLINE_SANDBOX_DATA_DIR=/work/agent-homes/cline-sandbox \
    TMPDIR=/work/tmp GOTMPDIR=/work/go-tmp \
    GOCACHE=/work/go-build GOMODCACHE=/gomodcache \
    GOPROXY=off GOSUMDB=off GOVCS='*:off' GOTOOLCHAIN=local \
    GOENV=off GOFLAGS=-mod=readonly CGO_ENABLED=1 \
  /bin/sh -ceu '
    umask 077
    mkdir -p "$HOME" "$XDG_CONFIG_HOME" "$XDG_DATA_HOME" "$XDG_CACHE_HOME" \
      "$CODEX_HOME" "$CLAUDE_CONFIG_DIR" "$CLINE_DATA_DIR" \
      "$CLINE_SANDBOX_DATA_DIR" "$TMPDIR" "$GOTMPDIR" "$GOCACHE"

    cd /src/multica-auth-work/server
    go list -mod=readonly -deps ./... >/dev/null || {
      echo "STOP: cached dependencies are missing or incomplete; network remains disabled" >&2
      exit 1
    }

    go test ./internal/daemon -run "TestAgentBrain|TestCredentiallessCodex" -count=1
    go test -race ./internal/daemon -run "TestAgentBrain(RejectsAllCustomArgsBeforeCredentialOrLaunch|RejectsCustomRuntimeBeforeCredentialOrLaunch|SuppressesWorkspaceRuntimeProfiles|BuiltInResolutionIgnoresCommandPathOverride)" -count=1

    go test ./internal/daemon/gateway -run "^TestG4Property" -count=20
    go test -race ./internal/daemon/gateway -run "^TestG4(Property|SSEContract)" -count=3
    go test ./internal/daemon/gateway -count=20
    go test -race ./internal/daemon/gateway -count=1
    go vet ./internal/daemon/gateway
    go test ./internal/daemon/gateway -count=1 -cover

    go test ./internal/daemon/runtimeenv ./pkg/agent
    go test ./internal/daemon/runtimeenv -run TestG4 -count=10 -timeout=45s
    go test -race ./internal/daemon/runtimeenv -run TestG4 -count=1 -timeout=45s
    go test -cover ./internal/daemon/runtimeenv
    go vet ./internal/daemon/runtimeenv ./pkg/agent

    go test ./internal/daemon/observability -run "Test(Development20|SyntheticEvidenceCannotPromoteAcceptance|G4ResultSchema|Consolidation|G4AcceptanceBlocksOnG3SecurityCorrections|SyntheticProvenance)" -count=20
    go test -race ./internal/daemon/observability -run "Test(Development20|G4ResultSchema|Consolidation|G4AcceptanceBlocksOnG3SecurityCorrections)" -count=3
    go test ./internal/daemon/observability -count=20
    go vet ./internal/daemon/observability

    go test ./... -count=1
    go vet ./...

    cd /src
    bash scripts/ops/tests/agent-cred-isolation-harness.sh
  '
```

Both the outer Docker client and the inner container start through separate
`/usr/bin/env -i` boundaries. The outer client is pinned to the approved local
`unix:///var/run/docker.sock`, uses a newly created empty `HOME`/`DOCKER_CONFIG`,
and explicitly clears Docker context/TLS/certificate/auth and upper/lower-case
proxy variables. It never reads an existing Docker config or credential store.
A non-Unix or different endpoint, inaccessible local socket/daemon, unavailable
image, absent cache, or non-empty ephemeral client home at cleanup is a STOP;
there is no context, remote-endpoint, pull, proxy, or network fallback.

The inner clean environment admits only the explicit non-secret values above,
so provider/auth/routing variables, proxy variables, socket forwarding, host
`HOME`, host XDG locations, and host agent homes are absent. Do not add `--env`,
`--env-file`, a host-home mount, a credential mount, a Docker socket mount, a
network, or a writable source mount. If the pinned image, dependency-only
cache, source permissions, toolchain, or any test is missing or fails, record
STOP; never pull, download, fetch, or enable a proxy or network.

The previous gateway result remains historical synthetic evidence: at
`2026-07-18T04:16:54Z`, 15 G4 tests including six property-style tests passed
the documented gateway subset with 77.6% coverage. This checklist correction
was not executed and does not extend that result.

No approved real-time offline capacity command or non-virtual measurement
collector exists in the current evidence. Therefore the task 9.1 attempt is
presently STOPPED. A later exact capacity invocation must receive independent
safety review, inherit every containment property above, and produce all
mandatory measurements below before its result can become `EV-G4-CAP`.

## Workload and counter acceptance criteria

The future contained, offline task 9.1 evidence attempt must use the frozen
shape in `multica-auth-work/server/internal/daemon/observability/harness.go:94`–`:116`.
Every item is required; there is no aggregate or best-effort pass:

- [ ] Exactly 20 offered tasks; 5-minute warm-up, 30-minute sustained window, and 10-minute recovery window.
- [ ] Protocol mix: Anthropic 30%, Responses 30%, Chat 30%, Antigravity-compatible/direct contract 10%.
- [ ] Payload mix: small 40%, medium 40%, large-context-relative 20%.
- [ ] Traffic mix: streaming 70%, tools 40%, parallel tools 15%, cancellation 10%, independent 80%, continuation 20%.
- [ ] `offered = admitted + rejected`.
- [ ] After recovery, `admitted = completed + failed + cancelled`; no active, queued, or inflight residue remains.
- [ ] `admitted = started + cancelled_before_start`.
- [ ] `started = completed + failed + cancelled_after_start`.
- [ ] `cancelled = cancelled_before_start + cancelled_after_start`.
- [ ] Every admitted correlation has exactly one terminal outcome; no duplicate launch, result, error, or cancel terminal.
- [ ] Slot acquisitions equal slot releases after recovery; no negative gauges, leaked workspaces, leaked streams, or leaked processes.
- [ ] Active work never exceeds 20 and peak queue does not exceed the pre-approved numeric bound. The current synthetic peak of 6 is a fixture result, not an approved limit.

The current deterministic fixture (`.planning/agent-brain-v3/evidence/g4-synthetic-capacity-phase1.md`) reports 20/20/0 offered/admitted/rejected; 19/17/1/2 started/completed/failed/cancelled; one pre-start cancellation and one post-start cancellation. Reproducing it is useful regression evidence but is not sufficient for acceptance.

## Routing, readiness, overload, and cancellation criteria

- [ ] Every admitted task has `RouterOwner=omniroute`; exactly one router owns selection and launch. Prodex, L2, legacy provider selection, provider account rotation/retry, and direct-provider routing counters remain zero.
- [ ] CLI kind, route model, protocol, task/session/request correlations, and approved route policy reconcile from admission through terminal result using redacted identifiers only.
- [ ] A gateway-required task is rejected before launch when gateway readiness, model, protocol, route-policy, or capability admission is false.
- [ ] Gateway unavailable is deterministic/retryable only where the frozen contract permits; authentication and unsupported capability/native-adapter states fail closed.
- [ ] Readiness loss launches no new child, leaks no slot, and recovers within the pre-approved steady-state recovery deadline.
- [ ] Overload is bounded: deterministic retryable rejection, no hidden enqueue, no consumed execution slot, no unbounded queue, and no second router fallback.
- [ ] Pre-start and active cancellation each produce one terminal cancellation, release queue/execution capacity exactly once, stop output, and meet the pre-approved cancellation-release deadline.
- [ ] No retry/replay occurs after committed output; retry and fallback ratios remain within their pre-approved limits.

## Latency, fairness, and resource criteria

- [ ] Record observed, non-virtual p50/p95/p99 for selection, queue, TTFT, and end-to-end request latency using a monotonic real-time clock as required by `multica-auth-work/server/internal/daemon/observability/harness.go:138`–`:146`; deterministic formulas or virtual scheduler time do not qualify.
- [ ] Compare selection p95, TTFT p95, and request p95 to pre-approved numeric limits; absence of any limit is a failure.
- [ ] Record offered/admitted/rejected, retry/fallback, peak queue, peak active/inflight, and exactly-once violations.
- [ ] Record host-sampled CPU, memory, and socket peaks during warm-up, sustained load, cancellation, overload, and recovery. Modeled values are insufficient.
- [ ] Record fairness against eligible capacity and compare deviation to the pre-approved limit; ineligible or quarantined capacity must not be included in the denominator.
- [ ] Record recovery to zero queue/active/inflight and stable resource baseline within the pre-approved deadline.

The current phase-1 latency and resource figures are virtual/modeled (`.planning/agent-brain-v3/evidence/g4-synthetic-capacity-phase1.md:52`, `:66`) and therefore cannot satisfy task 9.1 or authorize task 9.2.

If any CPU, memory, socket, latency, queue, fairness, recovery, counter, or
threshold measurement is missing, modeled-only, sampled from an unidentified
boundary, lacks provenance, or cannot be independently verified, the attempt is
a STOP. It may be retained only as failed-attempt evidence and MUST NOT be
named, indexed, or promoted as the acceptance record `EV-G4-CAP`.

## Failure, rollback, and STOP conditions

Stop without enabling anything if any of the following occurs:

- A real secret, auth file, provider account, credential home, live endpoint, current Multica daemon, or production resource would be read, mutated, or contacted.
- The exact `g3-independent-security-rereview.md` anchor no longer matches its pinned path, title, **ACCEPT** disposition, 3,377-byte size, `2026-07-18T04:10:50Z` mtime, or SHA-256 `0806e2f54b0049396323ec83f171084fd5d3fe4a9f0326c4c2ecd4c89d9f665e`; G3 regression tests fail; or duplicate/direct-provider routing is reachable.
- An approved numeric threshold is absent, changed during the run, or exceeded.
- Artifact provenance/digests are missing, stale, contradictory, content-bearing, or not freshly remeasured immediately before the attempt.
- Counters do not reconcile, terminal results duplicate, a cancellation leaks capacity, readiness failure launches work, overload grows without bound, or post-commit output is replayed.
- The run requires task 9.2 action, tier activation, broad admission, a native adapter claim, cutover, Prodex removal, tier 50/100, or a central/product-code change. Task 9.2 requires a separate post-acceptance Codex1 decision and cannot be inferred from task 9.1 evidence.
- A command runs directly in the workspace; forwards host environment, proxy, socket, home, XDG, agent-home, auth, credential, or routing state; makes source writable; attempts network access; or requires a non-synthetic dependency.
- The pinned image or approved dependency-only cache is absent/incomplete. Do not pull, download, fetch, enable a proxy, or enable network as remediation.

Rollback for a failed future run means keeping all tiers and broad admission off, closing the attempted admission path, terminating only synthetic test processes, confirming queue/active/inflight return to zero, and preserving all failure evidence. Do not fall back to Prodex, L2, legacy provider selection, direct-provider routing, or a live credential path. Any product/config rollback requires separate owner authorization and is outside this checklist.

## Artifact targets

Inputs that must be reconciled, not silently overwritten:

- `EV-G3-WIRE/04/05/06/07`: `.planning/agent-brain-v3/evidence/g3-serial-integration.md`
- G3 corrections: `.planning/agent-brain-v3/evidence/g3-security-corrections.md` and `.planning/agent-brain-v3/evidence/g3-security-corrections-adapters.md`
- Independent gate: `.planning/agent-brain-v3/evidence/g3-independent-security-rereview.md` (`REVIEW-G3-02`, **ACCEPT**)
- Gateway: `.planning/agent-brain-v3/evidence/g4-gateway-tests.md`
- Runtime isolation: `.planning/agent-brain-v3/evidence/g4-runtime-isolation.md`
- Automation/provenance: `.planning/agent-brain-v3/evidence/g4-evidence-automation.md` and `.planning/agent-brain-v3/evidence/g4-provenance-manifest.md`
- Preliminary capacity result: `.planning/agent-brain-v3/evidence/g4-synthetic-capacity-phase1.md` (currently **PARTIAL**, not acceptance)
- Consolidation: `.planning/agent-brain-v3/evidence/g4-consolidated-matrix.md`

Future outputs, to be created only by the separately authorized task 9.1
evidence owner after every prerequisite and criterion passes:

- `EV-G4-CAP` capacity/latency/resource acceptance report with exact approved thresholds, raw offline run identifiers, non-virtual latency/queue observations, host- or cgroup-sampled CPU/memory/socket peaks, measurement-boundary provenance, and counter equations
- Content-off provenance manifest containing command, commit/worktree identity, artifact digests, toolchain, environment classification, and timestamps—but no prompts, secrets, tokens, cookies, auth paths, or payload content
- Rollback/STOP report for any failed or aborted attempt
- Updated consolidated matrix that remains non-accepting unless all independent artifacts and the pB gate reconcile

## Current disposition

**STOP / NOT READY TO EXECUTE TASK 9.1.** Independent pB re-review acceptance is
recorded and freshly pinned, but approved numeric thresholds, the approved
real-time offline harness/collector, and non-virtual latency, queue, CPU,
memory, and socket evidence are absent. The development-20 profile remains
`RunnableNow=false`; the existing modeled fixture cannot become the acceptance
record `EV-G4-CAP`.

Task 9.2 is explicitly **UNAUTHORIZED**. Even after task 9.1 produces accepted
evidence, tier enablement requires a separate post-acceptance Codex1 decision.
This checklist was not executed, no task checkbox changed, and no tier action
occurred.

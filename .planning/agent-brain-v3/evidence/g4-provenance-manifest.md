# G4 Phase 1 provenance manifest

- Manifest ID: `manifest-g4-phase1`
- Schema: `agent-brain.g4-evidence.v1`
- Generated at: `2026-07-18T03:14:35Z`
- Reconciled at: `2026-07-18T04:16:54Z`
- pB safety reconciliation: `2026-07-18T04:42:27Z`
- Immediate pB correction validation: `2026-07-18T10:46:39Z`
- Task-8.8 provenance repair: `2026-07-18T12:56:07Z`
- Task-8.8 complete gateway-set repair: `2026-07-18T13:09:54Z`
- Task-8.8 accepted RSS lifecycle repair: `2026-07-18T13:41:33Z`
- Repository revision at run: `b6571299b00c8e388abefe7ef9dcbcf8ac715d7f`
- Synthetic outputs only: **yes**
- Content capture: **off**
- Acceptance claim: **no**
- G3 security-correction prerequisite: **SATISFIED — REVIEW-G3-02 ACCEPT**

## Generator and verification provenance

| Item | SHA-256 | Bytes / identity | Role |
|---|---|---:|---|
| `internal/daemon/observability/evidence.go` | `989d035318b3951a52ab1a8faa6fdd961d62d60c5f1f6206101bf28f688e80c4` | 11,677 | result/provenance schemas, A/B consolidation, and G3 correction gate |
| `internal/daemon/observability/synthetic.go` | `e5cea44dd11029b670905be6b503c3b3682918043485d4b2622778d5a63dbb1d` | 15,766 | deterministic virtual scheduler and aggregate generator |
| `internal/daemon/observability/g4_test.go` | `73ef5b7dbf03f032eaadebb40529e387d4b170315462dca797c87b1fa56939c3` | 6,108 | determinism, reconciliation, schemas, A/B, and correction-gate tests |
| `internal/daemon/observability/realtime_process.go` | `d84fc659dd8bd5873760ed9a5965adb6007d9a454af7cdd30b3441fe607a293e` | 16,168 | offline process measurement contract with bounded post-initial Wait reconciliation and fail-closed no-sample handling |
| `internal/daemon/observability/realtime_process_linux.go` | `f143cda7e712883481372f73666424e930036f261ab1ba8b5b59f17d6a66b131` | 5,409 | Linux proc-state parsing; missing RSS/HWM is an observed exit only for terminal `Z`/`X` |
| `internal/daemon/observability/realtime_process_linux_test.go` | `4b53206c7a819323df4730a247c0b187e40cb07d6b4281783913e3ca84a5a81f` | 14,765 | deterministic terminal/running-state, completed-Wait, real proc sample, and real rusage regressions |
| Go toolchain | `sha256:ae5a2316d12f3e78fd99177dad452e6ad4f240af2d71d57b480c3477f250fec6` | `golang:1.26`, Go 1.26.5 linux/amd64 | formatting, focused tests, and vet |

The container identity is the locally inspected immutable image digest. No
network/provider/service endpoint was used by the profile.

## Produced synthetic artifacts

| Artifact | SHA-256 | Bytes | Storage |
|---|---|---:|---|
| canonical `DevelopmentRunResult` | `eadd1659641cf1f4b0310b89a5d8cb3d1ee72cbc8f0b68c77ccb30ba24ea7c4b` | 14,228 | generated deterministically in test memory; aggregate schema only, not persisted as a payload file |
| `g4-evidence-automation.md` | `34b744f50235cac39f7bd3253f1f8c84a7bbb1ee3b7e23b2a366260a28c1a5ac` | 9,021 | redacted repository evidence and reconciled prerequisite record |
| `g4-synthetic-capacity-phase1.md` | `c209a924c428c76df14c35f69620c3ac9606ac542e7bfbf3ba423a6949c5936f` | 4,840 | redacted repository capacity record with retained blockers |
| `g4-consolidated-matrix.md` | `d8d6189bd517c51739102b7bc91ab4afd5a20fac819bf924de82050ea78f249d` | 15,642 | unchanged 116 checklist plus 44 parity row dispositions, blockers, and reconciled correction prerequisite |
| `g4-tier20-acceptance-checklist.md` | `95cf112b86a03d54f288b96f71d9a58c570d0e87daae60bd6cd8e27a549324c1` | 22,595 | preparation-only, contained offline task 9.1 evidence-attempt checklist; no task 9.2 or tier authorization |

## Reconciled external evidence inputs

| Artifact | SHA-256 | Bytes | Current disposition |
|---|---|---:|---|
| `g4-gateway-tests.md` | `163b12c4403fb73dc5031e5d17770d61af8d7f83dd1b1f10cbad3cae7693eabd` | 15,500 | synthetic/offline gateway evidence only; 16 G4 tests, including 6 property-style tests, within 59 top-level tests; no live/provider acceptance |
| `g4-runtime-isolation.md` | `17576dd1e61f2b93c52b6a2bc2ab72034be4530e1a922b0910bf4f3a8274f695` | 5,265 | synthetic/test-owned isolation evidence only; no live daemon/CLI/provider acceptance |

The manifest intentionally excludes its own digest to avoid a circular
reference. Index and ledger coordination files are also excluded because they
can change independently without changing the run.

## Required inputs at initial generation

| Path | State | Effect |
|---|---|---|
| `.planning/agent-brain-v3/evidence/g4-gateway-tests.md` | absent | EV-G4-01/04/05/06/07 cannot be consolidated or promoted |
| `.planning/agent-brain-v3/evidence/g4-runtime-isolation.md` | absent | EV-G4-02/03 and adapter records cannot be consolidated or promoted |

Their absence is recorded as a blocker, not replaced by inferred or fabricated
results. A later consolidation must compute provenance from the actual files
after they exist and must regenerate any derived digest.

## Later input validation checkpoint

At the task-8.8 complete gateway-set repair checkpoint
`2026-07-18T13:09:54Z`, both inputs were present and freshly measured at the
digests and byte counts in the table above. Current gateway source contains 16
G4 tests, including six property-style tests, and 59 top-level tests in the
package. A contained pinned-container coverage rerun over the complete gateway
set passed and reported 82.5%. Earlier 15-test, 48-test, 77.6%, 79.1%, and
82.4% figures are historical and superseded for this checkpoint. The rerun
used `golang:1.26`, Go 1.26.5 linux/amd64, image digest
`sha256:ae5a2316d12f3e78fd99177dad452e6ad4f240af2d71d57b480c3477f250fec6`.
It is therefore admitted only as partial synthetic/offline gateway evidence.
The runtime-isolation artifact remains byte-identical. It is admitted only as
partial synthetic/test-owned isolation and trusted-gateway evidence. Native
adapter, live provider, and capacity acceptance remain closed.

## Complete gateway source and test checkpoint

The complete current set under `internal/daemon/gateway/**` contains 25 files:
19 Go source/test files and 6 synthetic protocol fixtures, totaling 183,838
bytes. The canonical set digest is
`3a32c737541fd03b227ef3c7057f82f0bb38bcd68edc419d78545e2ae66e7497`.
It is the SHA-256 of the byte stream produced by `sha256sum` over the
`LC_ALL=C` sorted relative paths, so every per-file digest and relative path
participates in the set digest.

| Relative path under `multica-auth-work/server` | SHA-256 | Bytes |
|---|---|---:|
| `internal/daemon/gateway/client.go` | `c84d7ce780182f67184437d29f6e1c0f13682d004671a0f940506a97642e4bb9` | 11,064 |
| `internal/daemon/gateway/contracts_test.go` | `f965b75cd3d18aae335ef6525b8f72769fde0117bb81e4d50483456766f924ec` | 9,801 |
| `internal/daemon/gateway/doc.go` | `b3ca6a6f8b6e50f81c0851eff19daf44328777d346f777708ae0e6232d69fe1f` | 252 |
| `internal/daemon/gateway/errors.go` | `1e5b4b8b2404db65c046c6257cb10a525fc1d3d5c6134615dfc23313f4975c9c` | 3,258 |
| `internal/daemon/gateway/g4_property_stress_test.go` | `700662a0c69f5ba1bb383dbaa943657628261c2e9499c78cb8d37d38fa5cada6` | 19,191 |
| `internal/daemon/gateway/g4_protocol_conformance_test.go` | `f81222127404efa29362f552fe113c726095b5f5588f0e0645e418d9e7d7abf5` | 14,866 |
| `internal/daemon/gateway/g4_routing_failure_test.go` | `ab741967e43164869f0808bd07ce7b39253ec3503b142ab467492e788604e515` | 20,352 |
| `internal/daemon/gateway/gateway_test.go` | `b75246454a3e5901cab309644893504c63550b5aaaa0b60c24102995398e9568` | 6,831 |
| `internal/daemon/gateway/health_models.go` | `f1d8d07270d9af15a937cb5caa2d23f27d41133d83dc86f04ada7537384ec0c0` | 4,118 |
| `internal/daemon/gateway/measurability.go` | `87be2c953dfc9b20f83219ebc6d26fd2fb2cc3061bbe5690f6db38d21b07aa8d` | 4,289 |
| `internal/daemon/gateway/measurability_test.go` | `88780fc76674a3b7068cc79f8bcec10b0a6209e8f755bdcba8b8b1bf49cdaff0` | 8,478 |
| `internal/daemon/gateway/policy.go` | `d8a87ac586c26f8efa97b51b0fca53798e9ef641858661949219e0b78ef5e7cc` | 7,149 |
| `internal/daemon/gateway/profiles.go` | `4824ef0562e8ffdb9ecebc9ac78534ae339dfab52e8faa3adc24eb44fc430ef8` | 4,058 |
| `internal/daemon/gateway/projection.go` | `a8b41df2b78fbd6c0dbf6cdd7771ef480e59575e5904a1ab3b19cc63c2d0eef5` | 6,994 |
| `internal/daemon/gateway/projection_test.go` | `39e0c3c3b05eabb648b03df790bfc57f151fb6c29b7523fbe72c1b49dc566ca6` | 19,464 |
| `internal/daemon/gateway/protocol.go` | `4ee0bb641765e1e1c2fb9235a7598a29c556a903bebe0d091c34b941cc33f122` | 2,266 |
| `internal/daemon/gateway/registry.go` | `546518fed76677cf63364acb776437c987178d7b01132b5c1bfe604b0e4e6e07` | 12,593 |
| `internal/daemon/gateway/registry_test.go` | `c6a0e2bad8b6cfcfc2de629127cc2ee5efd8ac14bf18768f377b3ada3a09e230` | 18,695 |
| `internal/daemon/gateway/telemetry.go` | `89029a0b250a4efbcc96104928bd9ead990c7d58d9d627db2713e58187c2fbf3` | 8,472 |
| `internal/daemon/gateway/testdata/anthropic/messages-request.json` | `0eaa486dbd1a124a4d58eb4eef532e38b39b45e69cbd69ba3bdb9c33e8f08291` | 155 |
| `internal/daemon/gateway/testdata/anthropic/messages-stream.sse` | `ac33ce9a16c528a79292596ad8ead93c8721262327b2dc570686836e979a483d` | 513 |
| `internal/daemon/gateway/testdata/chat/chat-request.json` | `f190f95d882ca95e3fa18c5b25c599ee1f5ca74df954be6e5c5ef4ceff9169d0` | 130 |
| `internal/daemon/gateway/testdata/chat/chat-stream.sse` | `b458c117312f71c0a955f6049d85fca2ce9d85c68e4e764c2a0b3400441afc7b` | 162 |
| `internal/daemon/gateway/testdata/responses/responses-request.json` | `346780a9432ca03777a26327cd4c95b5172d2c8f43d6aaefe2aee18c48176413` | 93 |
| `internal/daemon/gateway/testdata/responses/responses-stream.sse` | `79bba056643d9fb747048ccdd007e357c37f6fb5112b3fda12174181988de651` | 594 |

The exact contained coverage command was:

```sh
checkpoint_root="$(mktemp -d)"
mkdir -m 700 "$checkpoint_root/home" "$checkpoint_root/docker"
env -i HOME="$checkpoint_root/home" DOCKER_CONFIG="$checkpoint_root/docker" \
  PATH=/usr/bin:/bin DOCKER_HOST=unix:///var/run/docker.sock \
  docker run --rm --pull=never --network none --read-only --pids-limit 128 \
  --memory 512m --cpus 2 \
  --tmpfs /tmp:rw,noexec,nosuid,nodev,size=64m \
  --tmpfs /build:rw,exec,nosuid,nodev,size=512m \
  -v /mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/multica-auth-work/server:/src:ro \
  -w /src \
  sha256:ae5a2316d12f3e78fd99177dad452e6ad4f240af2d71d57b480c3477f250fec6 \
  /bin/sh -c 'set -eu; cp -a . /build/src; cd /build/src; export HOME=/build/home GOCACHE=/build/gocache GOPATH=/build/gopath GOTMPDIR=/build/tmp GOPROXY=off GOSUMDB=off GOVCS=off PATH=/usr/local/go/bin:/usr/bin:/bin; mkdir -p "$HOME" "$GOCACHE" "$GOPATH" "$GOTMPDIR"; /usr/local/go/bin/go version; /usr/local/go/bin/go test ./internal/daemon/gateway -count=1 -coverprofile=/build/gateway.cover; /usr/local/go/bin/go tool cover -func=/build/gateway.cover | tail -n 1'
```

Result: `go version go1.26.5 linux/amd64`; package PASS; total statement
coverage `82.5%`. This is synthetic/offline package coverage only. It neither
proves a live route nor changes the disposition of tasks 8.1, 8.8, or 9.1.
The same contained toolchain reran
`TestG4ResultSchemaAndCanonicalMarshalling` once; it passed and reproduced
canonical result SHA-256
`eadd1659641cf1f4b0310b89a5d8cb3d1ee72cbc8f0b68c77ccb30ba24ea7c4b`
at 14,228 bytes.

## Accepted RSS lifecycle collector checkpoint

At `2026-07-18T13:41:33Z`, the three current collector files pinned in the
generator table totaled 36,342 bytes. Their canonical set SHA-256 is
`53f25cf1f01ed7cd66cd097b9ec0f71713188c6a9a1641f05c346881531c94c8`,
computed from `sha256sum` over the `LC_ALL=C` sorted relative paths so every
per-file digest and relative path participates. The independent RSS lifecycle
review disposition supplied for this exact source checkpoint is **ACCEPT**.

Contained offline verification used the unchanged immutable image digest
`sha256:ae5a2316d12f3e78fd99177dad452e6ad4f240af2d71d57b480c3477f250fec6`
and Go 1.26.5 linux/amd64, with `--network none`, `--pull=never`, read-only
source/root, and ephemeral HOME/Go caches. Results:

- the terminal `Z`/`X`, running nonterminal, completed-Wait/RSS race, and
  short-lived real-process regressions passed 20 iterations (`9.261s`);
- the full observability package passed (`0.566s`);
- the full observability race run passed (`3.912s`);
- `go vet ./internal/daemon/observability` and the three-file `gofmt -d` check
  passed; and
- `TestG4ResultSchemaAndCanonicalMarshalling` passed and reproduced
  `eadd1659641cf1f4b0310b89a5d8cb3d1ee72cbc8f0b68c77ccb30ba24ea7c4b`
  at 14,228 bytes.

The collector accepts no modeled substitute: a nonterminal process with
missing RSS/HWM remains `ErrUnsupportedHostMetric`; finalization requires at
least one complete observed process sample plus real Linux `rusage`. This
review closes only the RSS lifecycle portability defect. It does not execute
task 9.1, approve host capacity, enable a tier, contact a provider, or change
any checklist/parity disposition. Task 8.8 documentary completion is
preserved; task 8.1 remains open and task 9.1 remains stopped.

## Superseding G3 correction dependency

At the reconciliation checkpoint, both correction inputs are present. The core
correction artifact SHA-256 is
`074807c51aa66ec67909393d76e694668a41cb3a8a3bfbcd20ed13e95b85b33c`
(3,684 bytes), and the adapter correction artifact SHA-256 is
`1efa057ba7bda4cab8813fc243b5d91402fc7a38704cd6a0d96517005695df37`
(3,286 bytes). The independent re-review artifact is
`.planning/agent-brain-v3/evidence/g3-independent-security-rereview.md`, SHA-256
`0806e2f54b0049396323ec83f171084fd5d3fe4a9f0326c4c2ecd4c89d9f665e`
(3,377 bytes), artifact timestamp `2026-07-18T04:10:50Z`. It records
`REVIEW-G3-02` **ACCEPT** for all three correction findings. The correction
prerequisite is therefore satisfied. This does not promote synthetic A/B
evidence to G4, live/provider, host-resource, capacity-tier, or cutover
acceptance.

## pB checklist safety reconciliation

At `2026-07-18T04:39:43Z`, the re-review anchor was freshly remeasured at the
exact repository path above: SHA-256
`0806e2f54b0049396323ec83f171084fd5d3fe4a9f0326c4c2ecd4c89d9f665e`,
3,377 bytes, filesystem mtime `2026-07-18T04:10:50Z`, with the exact
`REVIEW-G3-02` disposition `Overall result: **ACCEPT** for the three original
findings.` The checklist now makes any path, title, disposition, size,
timestamp, or digest mismatch a STOP.

The corrected checklist replaces every direct-workspace verification command
with one prepared disposable-container invocation using `--network=none`, a
read-only root/source/module cache, ephemeral task-specific HOME/XDG/agent/Go
homes and caches, `/usr/bin/env -i`, proxy/module/VCS network disablement, and a
locally pinned image with `--pull=never`. Missing image or cached dependencies
is a STOP; network enablement is forbidden.

The checklist authorizes only a future contained offline task 9.1 evidence
attempt after every prerequisite and criterion is satisfied. Task 9.2 remains
explicitly unauthorized pending accepted task 9.1 evidence and a separate
post-acceptance Codex1 decision. Missing, unverifiable, or modeled-only CPU,
memory, socket, latency, queue, fairness, recovery, threshold, or counter
evidence is a STOP and cannot become the acceptance record `EV-G4-CAP`.

This reconciliation did not execute the checklist, start a container, inspect
credential/auth/secret sources, contact a network/provider/service, enable a
tier, or change an OpenSpec task state.

## Immediate pB correction validation

The `2026-07-18T10:46:39Z` and `2026-07-18T12:56:07Z` validations are retained
as historical checkpoints and superseded for collector source provenance by
the accepted RSS lifecycle checkpoint above. The earlier full-package STOP on
missing `VmRSS`/`VmHWM` was diagnosed as a process-exit scheduling portability
defect, corrected without modeled substitution, independently accepted, and
reproduced green at `2026-07-18T13:41:33Z`. Current source identity is the
three-file collector set pinned above. Portable explicit `time.Duration`
ordering remains in place; no duration `.Compare` API is used.

The refreshed observability validation ran in a disposable container with
`--network=none`, read-only source/root filesystem, ephemeral HOME and Go
caches, `GOPROXY=off`, `GOSUMDB=off`, `GOVCS=off`, and no forwarded host
environment. Focused RSS regressions passed 20 iterations, and full package,
full race, vet, formatting, and deterministic evidence checks passed. This was
source/evidence validation only; it did not execute the capacity checklist or
produce `EV-G4-CAP`.

The host-side Docker client used for that validation also started through
`/usr/bin/env -i`, an ephemeral empty HOME/DOCKER_CONFIG, and the exact pinned
local endpoint `unix:///var/run/docker.sock`; Docker context/TLS/certificate,
auth-config, and upper/lower-case proxy variables were explicitly empty. The
approved dependency-only volume `agent-brain-g4-gomodcache-ro` was absent.
Accordingly, the prerequisite gate stopped full-repository `go test ./...` and
`go vet ./...`; no host cache, existing Docker configuration, credential store,
pull, proxy, or network was used as a substitute.

The checklist's only outer Docker calls now go through the same clean-environment
wrapper. It rejects a non-Unix or different endpoint and stops when the exact
local socket/daemon, pinned local image, approved cache, or clean ephemeral
client-home lifecycle is unavailable. Its shell block passes static syntax
validation and contains zero direct `docker ...` invocations. The checklist was
not executed, tiers remain disabled, task 9.2 remains unauthorized, and no
OpenSpec checkbox was changed.

## Safety properties

The provenance code hashes only byte slices explicitly supplied by the
synthetic evidence pipeline. It does not open a path, enumerate directories,
inspect environment variables, or read service configuration. No real secret,
authorization value, key, token, cookie, account identity, prompt, tool payload,
repository payload, or reasoning content was read, copied, printed, or hashed
into this manifest or general telemetry.

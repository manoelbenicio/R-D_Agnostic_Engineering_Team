# T20 — OTLP loopback logs receiver (NEW package)

- agent `Opus48#B` · lane `OTLP-RECV` · pane `w6:p2` · task `OTLP-RECEIVER-IMPL`
- check-in: `.deploy-control/p0/checkins/Opus48-B__OTLP-RECEIVER-IMPL__20260723T234856Z.json`
- posture: NEW self-contained package; **no shared/existing source edited**.

## Delivered — `internal/daemon/observability/otlpreceiver/`
- `otlpreceiver.go` — loopback-only OTLP logs http/json receiver.
- `otlpreceiver_test.go` — package-local tests (allowlist, content-off, bounds, error paths, cancel, fail-closed).

### Behavior (all enforced, fail-closed)
- **Accepts** OTLP/HTTP JSON logs at `POST /v1/logs`; only records with attribute
  `event.name == "claude_code.api_request"` (`TargetEventName`) are persisted; all others dropped.
- **Default-deny allowlist** (`AllowlistedAttributes()`): only `event.name, request_id,
  client_request_id, task_id, model, status, duration_ms` are copied into `SanitizedRecord`
  (plus `timeUnixNano` → StartUnixNano). Every other attribute (prompt, response.text, account_email,
  tool_calls, messages, …) is structurally dropped.
- **Content-off by construction**: the OTLP log-record `body` is NOT modeled (not parsed), and
  `encoding/json` ignores all unmodeled fields, so message/content can never be persisted. `SanitizedRecord`
  has no field capable of holding prompt/response/tool/identity/content.
- **Loopback-only**: non-loopback `RemoteAddr` → 403 (`isLoopbackRemote`); `LoopbackServer(port)` binds
  `127.0.0.1:port` with Read/ReadHeader/Write/Idle timeouts + MaxHeaderBytes.
- **Bounded**: `MaxBodyBytes` (default 256 KiB, via `http.MaxBytesReader`), `MaxRecords` (default 1000),
  `MaxValueLen` (default 512, control-chars rejected).
- **Cancellation/backpressure**: persist loop honors `req.Context()`; slow sinks apply natural HTTP
  backpressure; scalar-only attribute values (arrays/maps/bytes ignored).
- **No auth-header logging**: the package performs no logging at all; it never reads/echoes Authorization.

### Tests (11 funcs incl. 5 error subcases) — ALL PASS
Allowlist+content-off (persist only allowed fields; marshaled record proven free of top-secret/prompt/
response/email/tool/messages/"PROMPT"); non-target dropped; oversized body→400; too-many-records→413;
error paths (GET→405, non-loopback→403, text/plain→415, malformed JSON→400, negative duration→400) persist
nothing; oversized attribute value→400; canceled context→408 persist-nothing; sink error→503; nil sink→503;
allowlist audit (exactly 7 keys); LoopbackServer binds 127.0.0.1 with timeouts.

## Results (exit codes; `/tmp` caches)
| Command | Result | exit |
|---|---|---|
| `gofmt -l …/otlpreceiver/*.go` | clean | 0 |
| `go build ./internal/daemon/observability/otlpreceiver/` | clean | 0 |
| `go vet ./internal/daemon/observability/otlpreceiver/` | clean | 0 |
| `go test ./internal/daemon/observability/otlpreceiver/ -count=1` | 11/11 PASS (ok 0.003s) | 0 |
| `git diff --check …` | clean | 0 |

**Verdict: PASS.** Loopback-only OTLP logs receiver persists only the allowlist, drops all
identity/content/prompt/response/tool + body, is bounded/cancel-safe, logs no auth, and fails closed on
every error path.

## Non-claims / notes
- New leaf package (imported by nothing → zero cycles); no shared/existing source edited (verified free).
- Wiring the receiver into a running daemon/listener (start `LoopbackServer`, connect a real Sink to
  persistence) is out of scope here — this delivers the package + a `Sink` interface for the integrator.
- Tests use `httptest` (`ServeHTTP` + `RemoteAddr`); no real network socket bound; no live run.
- No inference/secret/deploy/commit/push; no OpenSpec checkbox closed.

---

## SHAPE CORRECTIONS (official live shape, applied before merge) — 2026-07-23T23:55Z

Reworked `otlpreceiver.go` + tests to the corrected live OTLP shape (all PASS):

1. **event.name value = `api_request`** (`TargetEventName`), NOT `claude_code.api_request`. The
   Claude Code identity is validated via the **instrumentation scope** name separately:
   `ExpectedScopeName == "claude_code"` — a record is a target only when `scope.name == claude_code` AND
   the log `event.name == api_request`. (Test: wrong scope dropped; old dotted event.name dropped.)
2. **Trusted correlation from `resourceLogs.resource.attributes`.** Added resource parsing; the receiver
   reads ONLY the trusted resource keys `agent_brain.task_id` + `agent_brain.request_id`
   (`AllowlistedResourceAttributes()`) into `SanitizedRecord.TrustedTaskID/TrustedRequestID` and merges
   them into each target record. **Log-record attributes can never override the trusted correlation**
   (those keys are not in the log allowlist; the test injects a log `agent_brain.task_id=EVIL-override`
   and proves the persisted `TrustedTaskID` stays `task-77`).
3. **Fail closed = drop + count.** A target record missing its log `request_id` OR the trusted
   `agent_brain.task_id`/`agent_brain.request_id` is dropped (not persisted) and counted in
   `Stats().DroppedIncomplete` (rest of the batch still succeeds). Tests cover all three missing-field
   variants.
4. **Idempotency for OTLP retries.** Records are de-duplicated by
   `TrustedRequestID|request_id|client_request_id`. A retried batch persists once
   (`Stats().DuplicatesDropped++`). A record whose sink write FAILS releases its key (not marked seen),
   so a later retry re-persists it — proven by `TestSinkErrorReleasesKeyForLaterRetry` (503 then 200,
   persisted exactly once). Dedup set is bounded (FIFO, `MaxDedupKeys` default 65536).
5. **Official-shape fixture** added (`officialFixture`): `resource.attributes` with `agent_brain.*` +
   `service.name`, scope `claude_code`, `event.name=api_request`, plus forbidden identity/prompt/
   response/tool attrs and a content body — the persisted record is proven free of all of them.

New `Stats{Targets, Persisted, DroppedIncomplete, DuplicatesDropped}` counter surface (asserted in tests).

### Results (exit codes; `/tmp` caches)
| Command | Result | exit |
|---|---|---|
| `gofmt -l …/otlpreceiver/*.go` | clean | 0 |
| `go build …/otlpreceiver/` | clean | 0 |
| `go vet …/otlpreceiver/` | clean | 0 |
| `go test …/otlpreceiver/ -count=1` | PASS (ok 0.003s; 8 funcs incl. incomplete×3 + error×10 subcases) | 0 |
| `git diff --check …` | clean | 0 |

**Verdict: PASS.** Corrected to the official shape; no secrets/content; still no shared-source edits.

---

## CONCURRENT IDEMPOTENCY LOST-ACK RACE FIX — 2026-07-24T00:07Z (OTLP owned files only)

### The race (before)
`reserve()` marked a key as seen the moment a LEADER started, treating *in-flight* as *persisted*.
Interleave: leader A reserves K and its sink write is in flight; concurrent duplicate B calls
`reserve(K)` → already present → B returns **200 as a duplicate** (persisting nothing); then A's sink
FAILS → A `release(K)` + 503. Net: B was **acknowledged (200) with ZERO persistence** — acknowledged loss.

### The fix (wait/state)
`seen` now maps key → `*keyState{done chan, committed bool}`:
- `acquire(ctx,key)`: absent → become **leader** (reserve, committed=false, done open); `committed==true`
  → **true duplicate**; otherwise (in-flight) → **wait on `done`** (or ctx cancel) and re-loop.
- Leader success → `commit`: `committed=true`, close `done` (waiters wake → see committed → duplicate),
  FIFO-evict oldest committed key if over `MaxDedupKeys`.
- Leader failure → `abort`: delete entry, close `done` (waiters wake → key absent → exactly one **re-leads**
  and retries). An in-flight leader is **never** observed as persisted.
Restored the top-of-loop `req.Context().Err()` check so a canceled request persists nothing (408) even
for a fresh key.

### Guarantees
No acknowledged loss (a 200 always corresponds to a real persist or a truly-committed prior persist);
exactly one eventual persist per key; duplicates counted only when the prior write actually committed.

### Tests (all PASS)
- `TestConcurrentDuplicateLeaderFailureNoAcknowledgedLoss` — gate-sink handshake: A reserved + in-flight,
  B concurrent duplicate; A's write FAILS → A 503; B re-leads and persists → B 200; exactly ONE persist;
  `Persisted=1, DuplicatesDropped=0`. Proves no acknowledged loss.
- `TestConcurrentDuplicateLeaderSuccessCountsDuplicate` — A commits → B is a true duplicate (200, no second
  sink call); `Persisted=1, DuplicatesDropped=1`.
- Pre-existing `TestConcurrentIdenticalPOSTs_{FollowerReLeadsAndSucceeds,BothFail,LeaderSucceeds…}` (added by
  a parallel edit on the same concern — see note) now ALSO PASS; under the old `reserve/release` one of them
  DEADLOCKED (the earlier 10-minute test timeout), which the wait/state fix resolves.
- All prior receiver tests still PASS.

### Results (exit codes; `/tmp` caches)
| Command | Result | exit |
|---|---|---|
| `gofmt -l …/otlpreceiver/*.go` | clean | 0 |
| `go build …` / `go vet …` | clean | 0 |
| `go test …/otlpreceiver/ -count=1 -timeout=60s` | 13/13 PASS (ok 0.035s) | 0 |
| `git diff --check …` | clean | 0 |

### Notes / non-claims
- **CONCERN — `-race` not run:** the race detector needs CGO (absent, `CGO_ENABLED=0`, no `gcc`) and the
  race build filled the tmpfs `/tmp` cache (`no space left on device`). Correctness rests on the
  deterministic gate-sink channel handshakes + `go vet`, not a `-race` run; re-run `-race` when a C
  toolchain + disk are available.
- **Anomaly flagged:** three `TestConcurrentIdenticalPOSTs_*` tests are present in this (my-locked) file
  that I did not author — consistent with a parallel edit on the same OTLP race task (my `p0_control`
  check-in reported no active-lock overlap). They pass with this fix; flagged to the manager for awareness.
- **CLI hop ID correction is OUT OF THIS LANE:** the synthetic LaunchID/ProcID fix requires editing
  `daemon.go` (executeAndDrainForTask + EmitCLI call site), `pkg/agent` (`Session.ProcessID` from
  `cmd.Process.Pid`), and `cli_observability.go` — L1-owned shared source. Recipe handed to L1: set
  LaunchID = `brain.AdmissionLaunchID(corr)` (same as hops 2/3), ProcID = actual child PID (nonzero;
  never synthesize/backfill; zero PID must not yield a valid CLI span), and do not ignore `EmitCLI`
  errors for acceptance. Not editable from the OTLP lane.
- Stayed only in `otlpreceiver` files; no live/deploy; no secret/inference/commit/push.

---

## SCOPE / SERVICE.NAME CORRECTION (documented identity, closed schema) — 2026-07-24T00:30Z

Corrected the target identification to use **only documented signals** and removed the invented
instrumentation-scope requirement (which would have dropped real events).

### Official-doc facts applied (Anthropic monitoring doc)
- Resource `service.name = claude-code`; Meter Name `com.anthropic.claude_code` (metrics only, disabled);
  log `event.name` attribute VALUE `api_request`. Instrumentation **scope name is NOT documented**.
- Required exporter env (operator/integrator config — receiver does not consume env, documented for wiring):
  `CLAUDE_CODE_ENABLE_TELEMETRY=1`, `OTEL_LOGS_EXPORTER=otlp`, `OTEL_EXPORTER_OTLP_LOGS_PROTOCOL=http/json`,
  `OTEL_EXPORTER_OTLP_LOGS_ENDPOINT=http://127.0.0.1:<fixed>/v1/logs`, `OTEL_METRICS_EXPORTER=none`,
  `OTEL_RESOURCE_ATTRIBUTES=agent_brain.task_id=<canonical>,agent_brain.request_id=<canonical>`;
  force `*_PROMPTS/ASSISTANT_RESPONSES/TOOL_DETAILS/TOOL_CONTENT=0`; strip/unset `RAW_API_BODIES` + headers/certs.

### Receiver changes (otlpreceiver.go)
- Removed `ExpectedScopeName="claude_code"`. Added documented `ServiceNameClaudeCode="claude-code"` +
  `resAttrServiceName="service.name"`.
- **Target = documented signals only:** resource `service.name == "claude-code"` AND log
  `event.name == "api_request"`. Non-Claude-Code service or a different event.name → drop (not counted);
  a target then still fail-closes on missing request_id / trusted task / trusted request.
- **Instrumentation scope is IGNORED entirely** — it neither gates selection nor is persisted. The
  earlier "record observed scope" was retracted by the Principal; `SanitizedRecord` has **no** scope field
  (closed schema: event.name, request_id, client_request_id, trusted task/request, model, status/timing).
  Live smoke should report only a fixed `scope_ignored=true`, never the raw scope.

### Tests updated
- `officialFixture` now carries `service.name=claude-code` and an **undocumented** instrumentation scope
  (`io.opentelemetry.contrib.claudecode`) to prove scope-independence.
- `TestServiceNameAndEventNameIdentifyTarget` — wrong service.name → dropped; missing service.name →
  dropped; correct service.name + dotted `claude_code.api_request` → not a target.
- `TestArbitraryScopeNeitherGatesNorPersists` — real event accepted under arbitrary/empty/**malicious**
  scopes (`<script>…`, 2KB hostile string); asserts the scope value never appears in the marshaled record
  AND the JSON has no `Scope` field at all.
- `service.name=claude-code` added to the incomplete/bounds fixtures so they remain targets.

### Results (exit codes; build cache on root fs — /tmp tmpfs was full)
| Command | Result | exit |
|---|---|---|
| `gofmt -l …/otlpreceiver/*.go` | clean | 0 |
| `go build …/otlpreceiver/` | clean | 0 |
| `go vet …/otlpreceiver/` | clean | 0 |
| `go test …/otlpreceiver/ -count=1 -timeout=60s` | 14/14 PASS (ok 0.038s) | 0 |
| `git diff --check …` | clean | 0 |

All 5 shape fixes remain intact (event.name value, trusted resource attrs, required-id fail-close, retry
dedup/atomicity via wait-state, official fixture). Stayed only in the two otlpreceiver files; no live/deploy.

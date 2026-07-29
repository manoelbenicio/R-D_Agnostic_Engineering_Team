# F4 — WS delivery anchor wiring (hub.go) evidence

- agent: Codex56#B · lane: F4 · task: F4-WS-DELIVERY-ANCHOR · pane: w7:p4 (HERDR_ENV=1)
- lock (exclusive): `multica-auth-work/server/internal/daemonws/hub.go` + `hub_test.go`
- OpenSpec: 6.2 (eight-hop correlation) / OBS-8 (WS/UI delivery hop); AB-REQ-39,40
- repo HEAD: `a6d50986aee7f1981a323b29c9d90f175b4b6f8b`; both files tracked+clean at check-in
- toolchains: go `go1.26.1` (`/home/ec2-user/goroot/go/bin/go`); root disk 100% full → GOCACHE/GOTMPDIR/GOMODCACHE on `/tmp` (7.6G)
- constraints honored: no deploy/restart/Docker/systemd; no inference; no secret read/print/hash; no dependency install (deps resolved into /tmp modcache only); no commit/push/merge; no reset/stash/revert/clean; one owner per file; no frontend/OpenSpec/GSD edits; no out-of-lock edit.

## What was wired

The frozen `DeliveryRecorder.EmitDelivery` helper (obs_delivery.go) is now invoked at the hub's REAL WS
frame-delivery outcomes, carrying ONLY metadata: `session_id`, per-attempt `delivery_id`, outcome/reason
and bounded counters. No frame, payload, or user content enters the span.

| Real hub state | Site (hub.go) | Emitted span |
|---|---|---|
| send channel accepts frame | `notifyFrame` / `notifyWorkspaceFrame` success case | `outcome=delivered` |
| send buffer full → slow consumer evicted (unregister+close) | `notifyFrame` / `notifyWorkspaceFrame` slow path | `outcome=dropped`, `reason=slow_consumer`, `drop_count=1`, `backpressure_count=1` |
| heartbeat-ack buffer full, connection survives | `handleHeartbeatFrame` default branch | `outcome=backpressure`, `reason=send_buffer_full`, `backpressure_count=1` |

Identifiers (real, content-free, safe-charset per e2e contract):
- `session_id` = per-connection WS session id (`wssess-<n>`), minted at connect (`nextWSSessionID`).
- `delivery_id` = unique per delivery attempt (`wsdel-<n>`, `nextWSDeliveryID`).
Both use only `[A-Za-z0-9-]` so they pass `e2e.safeID`. Emission is opt-in via `Hub.SetDeliveryRecorder`
(nil recorder = safe no-op; delivery behavior unchanged). Emits run AFTER `h.mu.RUnlock()` to avoid
holding the hub lock during recorder work.

## Changes (within lock only)

- `hub.go` (+89/−5): imports `strconv`,`sync/atomic`; package `wsSessionSeq`/`wsDeliverySeq` + id minters;
  `client.sessionID`; `Hub.delivRecorder`+`delivMu`+`SetDeliveryRecorder`/`deliveryRecorder`/`emitDelivery`;
  session id set at connect; delivered/dropped emits in `notifyFrame`+`notifyWorkspaceFrame`; backpressure
  emit in `handleHeartbeatFrame`.
- `hub_test.go` (+139): 4 tests — valid-metadata spans for all 3 outcomes (distinct delivery ids, secrets
  false, self-validate), `notifyFrame` delivered integration, heartbeat-ack backpressure integration,
  nil-recorder no-op/no-panic.

## Commands + exit codes (GOCACHE/GOTMPDIR/GOMODCACHE=/tmp)

- `gofmt -l internal/daemonws/hub.go internal/daemonws/hub_test.go` → clean, exit 0
- `go vet ./internal/daemonws/` → exit 0
- `go test ./internal/daemonws/ -run 'EmitDelivery|NotifyFrame_EmitsDeliveredSpan|HeartbeatAckBackpressure|NoDeliveryRecorder' -count=1 -v` → 4 new + 5 existing delivery tests PASS, exit 0
- `go test ./internal/daemonws/ -count=1` (full package) → `ok ... 0.452s`, exit 0
- `git diff --check -- .../hub.go .../hub_test.go` → exit 0
- `git diff --stat` → 2 files changed, 223 insertions(+), 5 deletions(-)

## Limitations / non-claims

- No live run / no inference / no deploy / no secret read. Frontend NOT edited.
- `session_id` here is the **WS-connection** session (per the frozen helper's "WS session" definition), NOT
  the upstream agent task `session_id`. This hop's spans are contract-valid in isolation but end-to-end
  JOIN with upstream hops (OBS-9 continuity) is NOT claimed and requires task-session propagation into the
  WS layer — routed to F1/F8/Principal as a separate correlation concern, not part of this lock.
- The `dropped` (slow-consumer eviction) call site emits via the same `emitDelivery` helper (unit-proven in
  TestHub_EmitDelivery_OutcomesAreValidMetadataSpans); a live-socket eviction integration test was
  intentionally omitted to avoid timing-flakiness (OS-buffer-dependent). Not claimed as executed.
- No OpenSpec checkbox closed; Principal adjudicates.

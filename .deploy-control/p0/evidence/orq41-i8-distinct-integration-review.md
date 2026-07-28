# ORQ-41 I8 — distinct integration review of `e0b0155`

**Mode:** READ-ONLY. Reviewer: Codex56#B. No edit, test, push, merge, board
or runtime action performed.

## Verdict: PASS for I8 integration safety

Commit `e0b0155d0f0d46a9da3724ecbfc51a5167bd2ebe` has parent
`c047c0b7ed710f260e76a6ee50257b2a5f96dc70` exactly. Its delta is exactly four
files:

1. `internal/handler/daemon_ledger_summary.go`
2. `internal/handler/daemon_ledger_summary_test.go`
3. `internal/service/autopilot.go`
4. `internal/service/autopilot_replay_test.go`

No migration, query, generated, router or HMAC-secret file is touched. The
existing W4 closeout records the ephemeral DB evidence with three nominal
handler passes, zero skips and race-clean output; this review accepts that
evidence without rerunning tests.

## Independent checks

- **Off-path byte-equivalence:** `NewAutopilotService` leaves both replay
  collaborators nil, and `replayGateSkipReason` returns `"", false` before any
  I/O when disarmed. This preserves pre-gate admission behavior.
- **Runtime gate remains disarmed:** no production call site for
  `NewAutopilotServiceWithReplayGate` is present in the reviewed delta. Merging
  the commit therefore cannot activate the feature by itself.
- **Correlation:** armed behavior calls `PriorTaskCorrelationID` and passes the
  returned prior-task ID to `AllowReplay`; autopilot ID is not used as ledger
  correlation. First-run (`ok=false`) bypasses the gate; resolver errors fail
  closed.
- **Handler honesty:** malformed UUID/JSON and URL/body mismatch fail closed;
  a well-formed request returns 503 with no `"recorded"` claim because no
  durable store exists in W4.
- **Evidence:** the retained W4 closeout proves the DB-backed handler test run
  had three nominal PASS events, zero `Skipping tests`, zero race warnings and
  exit 0. The prior false-green run is explicitly distinguished.
- **Dependencies:** activation remains blocked on LANE-DB migration/query/sqlc
  store, W2 prior-task resolver, W3 authenticated route plus daemon-task
  ownership check, a durable content-free checker, and a queue-zero gate.

## Integration manifest (no merge performed)

| item | state |
|---|---|
| parent and four-file scope | PASS |
| off-path behavior unchanged | PASS |
| gate activation by merge alone | SAFE: remains off |
| handler false-success removed | PASS; 503 |
| DB evidence nominal/zero-skip | PASS per retained evidence |
| LANE-DB/W2/W3 dependencies | OPEN; activation blocked |
| owner merge authorization (I7) | separate owner decision required |

**Conclusion:** the commit may be merged as dormant scaffolding when the owner
authorizes I7. It must not be treated as replay protection or wired on runtime
until the listed LANE-DB/W2/W3 gates are independently green.

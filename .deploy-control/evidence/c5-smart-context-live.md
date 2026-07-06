# C5 Smart Context Live Evidence

- status: LIVE_GREEN
- captured_at_utc: 2026-07-05T04:33:48Z
- executor: Codex#5.5#C
- sidecar: `multica-auth-work/prodex-sidecar/target/release/prodex-sidecar`
- bind: `127.0.0.1:43117`
- contract: `rpp.l2.v1`

## Sidecar

The real prodex sidecar process was launched and held open in a live terminal
session:

```text
prodex-sidecar listening on 127.0.0.1:43117
```

Authenticated readiness passed:

```text
readyz-smoke: PASS
```

An unauthenticated `/readyz` probe returned HTTP 401. This was reported through
`ping-opus.sh` immediately and no unauthenticated retry loop was run.

## Baseline Live Smokes

```text
readyz-smoke: PASS
policy-apply-smoke: PASS
session-start-stop-smoke: PASS
event-stream-smoke: PASS, validated_events=6
profile-fail-closed-smoke: PASS
auth_401_detected=no during bearer-auth suite
```

## C5 Live Probe

Smart Context desired-state policies were applied to the live sidecar:

```text
shadow: HTTP 200 applied=true revision=1
canary: HTTP 200 applied=true revision=2
live:   HTTP 200 applied=true revision=101
```

The live session control path also passed:

```text
StartSession: HTTP 200 router_owner=rust_l2
EventStream: HTTP 200 event_count=5 event_types=[heartbeat, session_started, sidecar_ready]
StopSession: HTTP 200 stopped=true
secrets_present_any=false
```

The policy payload included Smart Context `exact_mode_allowed=true` and a
`smart_context` kill switch with `effective_at=next_request`. The current live
sidecar surface confirms control-plane acceptance and lifecycle/event integrity;
it does not emit raw prompts, raw provider payloads, bearer tokens, or OAuth
material in evidence.

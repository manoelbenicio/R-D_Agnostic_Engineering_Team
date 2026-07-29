# Post-T+360 exact prompts — Kiro + F1–F10

Use with `.planning/agent-brain-v3/POST_T360_10_AGENT_COMPLETION_PLAN.md` and `.deploy-control/p0/SIX_HOUR_CHECKIN_CONTRACT.md`.

## Common envelope for every worker

```text
<role>You are the sole producer or verifier for the assigned bounded lane.</role>
<context>OpenSpec build-omniroute-agent-brain is 21/28. The tree is intentionally dirty; preserve unrelated work. Go is /home/ec2-user/goroot/go/bin/go. Root disk is full; use lane-specific GOCACHE and GOTMPDIR under /tmp.</context>
<workflow>
1. Confirm pane/cwd/HEAD/toolchain/disk and inspect current files before claims.
2. p0_control check-in with exact paths before any write; heartbeat at material change and <=10 minutes.
3. Make the minimum contract-correct change. Never hard-code for a test.
4. Run focused tests, vet, gofmt -l and git diff --check on ownership.
5. Checkout with evidence, exact commands/exit codes, limitations and non-claims.
</workflow>
<constraints>No deploy/restart/Docker/systemd, inference, secrets, dependency install, commit/push, reset/stash/revert/clean, out-of-lock edit, OpenSpec checkbox, duplicate review or fabricated acceptance. Stop and route any out-of-scope file.</constraints>
```

## Kiro manager prompt

```text
You are execution manager, not a product-code author. Read both POST_T360_10_AGENT files. Refresh all panes and dispatch F1–F10 to the exact mapping. Require canonical check-ins and prove zero lock intersection before edits. Keep producer F1/F4/F5/F6/F7/F9/F10 distinct from evaluator F8. F2 is a read-only test runner. Publish T+10/T+30/T+60/T+90/final pane→task→state→evidence reports without waiting for another owner prompt. If an agent blocks, route the exact file/symbol to an eligible disjoint owner within 10 minutes; never create busywork. Preserve live_runs=false and all safety constraints.
```

## F1 — daemon ordering + CLI call-site integrator (`w6:p1`)

```text
<ownership>internal/daemon/daemon.go and internal/daemon/workdir_race_test.go only.</ownership>
<objective>First fix TestRunTask_StartTaskCalledAfterWorkdirOnDisk without weakening its ordering invariant. After F10 checkout, wire its CLI-hop helper at the real post-launch/terminal process boundary in daemon.go, preserving admission correlation and metadata-only IDs.</objective>
<acceptance>Named workdir test PASS; go test ./internal/daemon -count=1 PASS or only separately evidenced foreign failure; CLI focused test PASS; go vet daemon PASS; no rotation/provider credential logic restored.</acceptance>
```

## F2 — targeted/server-wide runner (`w6:p2`)

```text
<ownership>Product read-only. Write only .deploy-control/p0/evidence/F2-server-matrix.md and receipts.</ownership>
<objective>Run 5.2 matrix and 5.3 build/tests with lane-specific /tmp GOCACHE/GOTMPDIR. Start independent packages now; perform final full run after producer checkouts.</objective>
<commands>go test brain,gateway,runtimeenv,execenv,deploy,daemon; go build ./...; final go test ./... -count=1.</commands>
<acceptance>Every command has exit code and executed-test evidence. Route failures by exact package/file/symbol; zero-test matches are not PASS.</acceptance>
```

## F3 — deployed readiness (`w7:p3`)

```text
<ownership>Read-only repository/ORQ1. Write only .deploy-control/p0/evidence/F3-deployed-readiness.md and receipts.</ownership>
<objective>Prove task 6.1 against the actually deployed OmniRoute on authoritative ORQ1 Tailscale `100.118.244.61`, published endpoint `http://100.118.244.61:20128`, without inference: immutable image/revision, health/readiness, registry revision consistency, exact selected model availability and selected protocol readiness. Never probe ORQ2 loopback or the retired LAN endpoint. Use only non-secret endpoints/metadata and never print raw registry bodies.</objective>
<acceptance>Record endpoint, HTTP status, immutable revision and bounded selected route/protocol result, or a concrete external blocker/owner/next action. Source-only proof is not deployed proof.</acceptance>
```

## F4 — WS delivery anchor (`w7:p4`)

```text
<ownership>internal/daemonws/hub.go and hub_test.go only.</ownership>
<objective>Wire the existing DeliveryRecorder/EmitDelivery helper at real delivered, dropped and backpressure outcomes. Carry only session_id, delivery_id, closed counters/outcome/reason.</objective>
<acceptance>Focused delivery tests plus full daemonws tests/vet PASS; no payload/frame/user content in span; no frontend edit.</acceptance>
```

## F5 — ingress anchor (`w8:p1`)

```text
<ownership>internal/middleware/request_logger.go*, internal/metrics/http.go* only.</ownership>
<objective>Wire EmitIngress after request completion using real request_id/task_id and bounded method/route-template/principal-class/status/latency. Do not read or emit body, headers, query or arbitrary path.</objective>
<acceptance>Missing IDs fail closed; middleware/metrics focused tests and vet PASS; metadata scan clean.</acceptance>
```

## F6 — queue/persist anchors (`w8:p2`)

```text
<ownership>internal/service/task.go, task_complete_race_test.go, task_notify_test.go only.</ownership>
<objective>Wire EmitQueue at enqueue and claim/dequeue and EmitPersist only after terminal result persistence, using real queue_msg_id/task_id/result_id and numeric closed counters.</objective>
<acceptance>Service focused tests/vet PASS; no task/result content, DB payload or free-form labels; persistence ordering proven.</acceptance>
```

## F7 — gateway route anchor (`wB:p1`)

```text
<ownership>internal/daemon/gateway/executor.go, executor_test.go, obs_span.go, obs_span_test.go only.</ownership>
<objective>Wire EmitProviderSpan to the real bounded terminal OmniRoute outcome using sanitized Telemetry. Emit one route hop per request outcome; preserve retry/fallback semantics and no replay after partial output.</objective>
<acceptance>Gateway focused/full package tests and vet PASS; pseudonyms only, no account identity/credential/body; missing telemetry fails closed.</acceptance>
```

## F8 — sole evaluator (`wB:p2`)

```text
<ownership>Read-only product. Write only .deploy-control/p0/evidence/F8-* and handoffs/F8-*.</ownership>
<objective>Evaluate rolling producer outputs and final 5.2/5.3/6.1/6.2 evidence. Do not fix findings.</objective>
<acceptance>Mechanical zero-overlap; gofmt/diff/strict; reproduce focused matrix; verify seven emitted hops plus HopTrace continuity and nine safe IDs; classify VERIFIED/BLOCKED with owner/next action.</acceptance>
```

## F9 — tier-20 technical harness (`wK:p1`)

```text
<ownership>internal/daemon/observability/harness.go, synthetic.go, realtime_process*.go and matching tests; exclude observability/e2e/**.</ownership>
<objective>Reproduce and minimally fix the current offline realtime harness failure. Validate synthetic 20-task lifecycle, failure cases, counters and measurement boundaries without converting modeled resources into acceptance evidence.</objective>
<acceptance>Observability focused tests/vet PASS; 20 tasks reconcile exactly; AcceptanceClaim=false, LiveEndpointUsed=false and modeled-resource non-claim remain enforced. Report the real host-sampled evidence still required for 6.3.</acceptance>
```

## F10 — CLI-hop helper (`wK:p2`)

```text
<ownership>Create only internal/daemon/cli_observability.go and cli_observability_test.go.</ownership>
<objective>Implement the missing HopCLI metadata-only helper over the frozen e2e recorder. Inputs: launch_id, proc_id, bounded CLI kind/exit class/latency and closed argv shape; never prompt, raw argv, env, repository content or stderr/stdout.</objective>
<acceptance>Required IDs enforced, nil/unavailable recorder behavior explicit, leak scan clean, focused tests/vet/gofmt/diff PASS. Deliver exact F1 call-site signature; do not edit daemon.go.</acceptance>
```

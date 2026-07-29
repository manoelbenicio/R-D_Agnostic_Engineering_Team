# Full agentic product plan — Main Brain + Chat vertical slice

Updated: 2026-07-22T12:27Z  
Authority: Principal (`w5:p9`)  
Fleet manager: Kiro/Opus 4.8 (`w5:p1`)  
Execution topology: 9 executable worker panes + Kiro + Principal = 11 active roles; `w6:p1` is excluded because its monthly model quota is exhausted.

## 0. Original objective and final acceptance contract

This plan exists to deliver and prove one product flow—not to maximize backlog or agent count:

```text
squad → project → Kanban task → Main Brain admission/queue
→ approved coding-agent CLI → OmniRoute (consumed external router)
→ persisted result → terminal/logs/status delivered in UI
```

Chat is part of this product flow:

```text
untargeted chat → Kiro TL/Manager → clarify/document/plan/delegate → coder → synthesis
explicit agent target → selected coder directly
```

Final P0 acceptance requires evidence for every boundary below:

| Boundary | Product proof |
|---|---|
| Squad/TL | Default Kiro-led squad exists with coder members |
| Project/Kanban | A real project issue/task is created and queued |
| Chat | Untargeted and explicit-target paths resolve to the correct agent |
| Main Brain | Task is admitted only when the consumed OmniRoute readiness declaration is ready |
| CLI | Approved CLI process launches with task/session correlation and no provider credentials |
| OmniRoute boundary | Main Brain sends only the approved router-neutral contract; router internals are not inspected/tested |
| Persistence | Terminal result and lifecycle state persist durably |
| Delivery/UI | Terminal/log/status reaches WS/UI with the same safe correlation IDs |
| Evaluation | Independent evaluator reproduces the evidence; owner performs final live acceptance |

The earlier F1–F10 wave completed most internal Main Brain hops. The current C1–C10 wave closes chat/client integration, residual test gates and final trace/evaluation. Completed work is consumed; it is never rerun merely to keep an agent busy.

## 1. Exact scope

Included:
- all 7 pending tasks in `build-omniroute-agent-brain` (5.2, 5.3, 6.1–6.5);
- all 6 pending tasks in `chat-orchestration-standard` (0.1, 1.2, 1.3, 2.1–2.3).

Excluded:
- all `native-runtimes-onboarding` work;
- archived `agent-credential-isolation` work;
- OmniRoute provider/model mapping, accounts, credentials, provider sessions, rotation, retry/failover, configuration, development, customization, inspection, testing or validation;
- deploy/restart/Docker/systemd, inference, secrets, dependency installation, commit/push and destructive Git.

OmniRoute boundary: Main Brain only consumes the router/operator readiness declaration and tests its own ready/not-ready fail-closed behavior. It never tests OmniRoute internals.

## 2. Owner decisions frozen

- Default squad: `Workspace Team` product object, presented operationally as the Kiro-led engineering squad.
- Default TL/Manager: **Kiro (Opus 4.8)**.
- Preferred coder/member and direct escape hatch: **Codex 5.6 Sol, high thinking**.
- Other available coder agents are squad members; Kiro is delegation-only and synthesizes member results.
- Untargeted chat routes to the default squad leader. Explicit `agent_id`/agent picker routes directly and bypasses the TL.
- Textual `@name` parsing is not silently claimed if the actual API contract is explicit `agent_id`; UI language and tests must match the implemented contract.

## 3. Official engineering practices applied

| Source | Practice applied here |
|---|---|
| Anthropic, *Building effective agents* (2024-12-19), https://www.anthropic.com/research/building-effective-agents | Simple composable workflows; parallelize only independent sections; central orchestrator-workers; evaluator separated from producers; stop conditions and environmental ground truth. |
| Anthropic, *Effective context engineering for AI agents* (2025-09-29), https://www.anthropic.com/engineering/effective-context-engineering-for-ai-agents | Smallest high-signal context; clear sections; minimal non-overlapping tools; just-in-time file retrieval; specialists return distilled evidence. |
| OpenAI Agents SDK, *Agent orchestration*, https://openai.github.io/openai-agents-python/multi_agent/ | Manager retains final synthesis; specialist lanes; handoffs with exact scope; deterministic code/DAG for predictable work; parallel execution only without dependencies. |
| OpenAI, *Prompt engineering*, https://platform.openai.com/docs/guides/prompt-engineering | Identity/instructions/examples/context sections; explicit role and workflow; typed boundaries; prompt changes backed by tests; persistent TODOs and complete resolution before yielding. |
| OpenAI, *Working with evals*, https://platform.openai.com/docs/guides/evals | Define behavior and objective criteria first; execute representative tests; analyze failures; no zero-test or self-evaluated PASS. |

Operational translation: exact ownership, one manager, one evaluator, no duplicate broad suites, evidence-based checkpoints, structured prompts, explicit non-claims and deterministic handoffs.

## 4. Evidence baseline at dispatch

- Main Brain 5.2: brain/gateway/runtimeenv/execenv/deploy plus daemon targeted tests passed; 470+ focused tests and daemon package evidence exist.
- Main Brain 5.3: `go build ./...` passed. Full tests had two environment-related groups: JWT configuration tests and email dev-mode tests.
- Main Brain 6.2: seven real producer anchors and HopTrace assembly exist; C6 is performing final e2e continuity validation.
- Main Brain 6.1: C7 proved consume-only readiness/fail-closed semantics; no OmniRoute internal test is permitted.
- Chat 1.2/1.3: backend implementation and prior independent source-contract acceptance exist.
- Chat client gap: C3 found `packages/core/chat/mutations.ts` and API types still require `agent_id`; coordinated optional-target changes are required.
- Chat runtime smoke: local backend `:8080`, web `:3100` and Postgres `:5432` are not running. Handler `TestMain` skips DB tests; a zero-test exit 0 is not PASS.

## 5. Ten workstreams on existing panes

C10 completed before C1 was reassigned to the same pane. This is deliberate reuse, not an invented worker. `w6:p1` is not counted.

| Lane / pane | Work | Exclusive mutable ownership | Acceptance | ETA from activation |
|---|---|---|---|---:|
| C1 `wK:p2` | Chat backend integrator | `server/internal/handler/{workspace.go,agent.go,chat.go,workspace_test.go,chat_test.go}` | default Kiro TL squad; coder membership behavior explicit; omitted target→TL; explicit target→direct; format/vet/focused proof; honest DB limitation | 30–45m |
| C2 `w6:p2` | Chat web UX, then final matrix | `packages/views/chat/**`; after checkout, product read-only test evidence | untargeted create omits agent; direct picker sends agent ID; focused web tests; then final server/web matrix | 30–60m |
| C3 `w7:p3` | Chat core API contract | `packages/core/chat/**`, `packages/core/api/client.ts`, `packages/core/types/chat.ts` and matching tests | `agent_id` optional end-to-end in client/types; explicit ID preserved; focused type/tests PASS | 30–45m |
| C4 `w7:p4` | Hermetic JWT test gate | `server/internal/auth/jwt_configuration_test.go` only | production/staging subtests set compliant test secret locally; production validation unchanged; package tests PASS | 15–30m |
| C5 `w8:p1` | Hermetic email test gate | `server/internal/service/email.go`, `email_test.go` only | minimal intended dev-mode redaction behavior or corrected assertions; no secret/backend; focused tests PASS; no unrelated email change | 20–40m |
| C6 `w8:p2` | Main Brain 6.2 trace assembly | `server/internal/daemon/observability/e2e/**` | eight ordered hops, nine safe IDs, no content/secrets/account identity; focused/race tests PASS | 20–40m |
| C7 `wB:p1` | Consume-only readiness boundary | evidence `C7-readiness-consumption.md`; product read-only | distinguish consume from test; Main Brain ready/not-ready behavior and exact external declaration documented | DONE |
| C8 `wB:p2` | Sole independent evaluator | `.deploy-control/p0/evidence/C8-*`, `handoffs/C8-*`; product read-only | independently reproduce focused gates; no fixes; VERIFIED/BLOCKED per task with owner/next action | rolling + 20m |
| C9 `wK:p1` | Chat focused smoke | `.deploy-control/p0/evidence/C9-chat-smoke.md`; product read-only | smallest real non-inference smoke; no zero-test PASS; if stack/DB unavailable, exact blocker and post-start command | 15–30m |
| C10 `wK:p2` | Official practices/prompt audit | `FULL_PRODUCT_OFFICIAL_PRACTICES.md` | official Anthropic/OpenAI practices translated to DAG/prompt rules | DONE |

Manager: Kiro owns pane dispatch, receipts, heartbeats, blocker routing and T+ reports. Principal owns architecture, lock expansion, external gates and OpenSpec closure. Neither writes product code while an eligible worker exists.

## 6. DAG and execution waves

### Wave A — parallel source convergence (T+0–T+30)
- C1, C2 and C3 converge server/client/UI chat target semantics on disjoint files.
- C4 and C5 resolve the two known server-wide test gates.
- C6 closes trace continuity.
- C8 evaluates completed outputs continuously.
- C9 prepares/runs the smallest chat smoke without broad repetition.

### Wave B — serialized integration (T+30–T+60)
- C1 consumes C3's typed contract only after C3 checkout if server adjustments are needed.
- C2 runs cross-layer focused tests after C1/C3 checkout, then becomes the read-only final matrix runner.
- C8 reproduces chat routing, 5.2/5.3 and 6.2 evidence.
- Principal adjudicates 0.1, 1.2, 1.3, 5.2, 5.3, 6.1 and 6.2 only from evidence.

### Wave C — source closure (T+60–T+90)
- Final `go build ./...`, server tests and focused web/type tests use `/tmp` caches.
- Kiro produces evidence→checkbox matrix.
- C8 final report; Principal updates OpenSpec checkboxes.

### External-gated wave
- Chat 2.1/2.2 needs a reachable product stack and test DB. No service start/restart is authorized by this plan.
- Main Brain 6.3 needs real 20-task measurements and owner acceptance.
- 6.4 starts only after accepted 6.3 and explicit tier authorization.
- 6.5 needs explicit owner live-run authorization and exact Main Brain/OmniRoute provenance.

## 7. Task-to-evidence closure matrix

| OpenSpec task | Closure evidence | Status target |
|---|---|---|
| Main Brain 5.2 | F2 targeted matrix + daemon PASS + C8 reproduction | close in Wave B |
| Main Brain 5.3 | C4/C5 + C2 final build/full tests + C8 | close when full suite green or only formally excluded unrelated gate remains |
| Main Brain 6.1 | owner/router readiness declaration consumed + C7 Main Brain fail-closed proof | no OmniRoute internal test; close from declaration + local contract |
| Main Brain 6.2 | C6 + prior F4–F7/F10 anchors + C8 | close after continuity verified |
| Main Brain 6.3 | real tier-20 measurement + owner acceptance | external gate |
| Main Brain 6.4 | accepted 6.3 + tier-50/100 evidence | external gate |
| Main Brain 6.5 | authorized Kanban→Main Brain→OmniRoute→terminal run | external gate |
| Chat 0.1 | owner decision in §2 | close immediately |
| Chat 1.2 | C1 default squad/leader/member proof + C8 | Wave B |
| Chat 1.3 | C1/C2/C3 target contract + C8 | Wave B |
| Chat 2.1 | real untargeted chat/TL/delegation/synthesis smoke | requires running stack + inference authorization for actual synthesis |
| Chat 2.2 | real direct Codex smoke | requires running stack + live authorization |
| Chat 2.3 | canonical receipts and evidence after 2.1/2.2 | follows smokes |

## 8. ETA

- Plan and exact prompts: **available now**.
- Main Brain/chat offline source convergence: **30–60 minutes**.
- Final offline build/test/evaluator closure: **60–90 minutes** from the 12:15 activation, subject to root-disk/module-cache constraints.
- Chat API/UI smoke after a reachable stack and DB are authorized/provided: **15–30 minutes**.
- Main Brain tier-20 acceptance: **1–2 hours after live authorization and environment readiness**.
- Tier 50/100: **2–4 hours after tier 20 acceptance**.
- Final Kanban live acceptance: **30–60 minutes after explicit authorization/provenance**.

No calendar ETA is fabricated for an unopened external gate.

## 9. Checkpoints and controls

- T+5: canonical activation and zero-overlap.
- T+15: first test results/blockers.
- T+30: source handoffs and lock release.
- T+45: integration matrix.
- T+60: evaluator preliminary verdict.
- T+90: offline closure report.

Every worker checks in before edits, heartbeats within 10 minutes, uses exact locks, reports commands/exit codes, and states non-claims. Root disk is full; Go lanes use lane-specific `GOCACHE`, `GOTMPDIR` and, where already populated, off-root module cache under `/tmp`. No source/cache deletion is authorized.

## 10. End-to-end agentic lane map

This is the complete product DAG across the completed F wave and active C wave. “Verify” means consume existing evidence and inspect only the affected boundary—not rerun the original producer’s broad work.

| Vertical lane | Owner/pane | Product boundary | Work status | Dependency | Acceptance |
|---|---|---|---|---|---|
| V1 Squad/TL setup | C1 `wK:p2` | squad → Kiro TL → coder members | active | owner definition frozen | workspace setup creates/resolves default squad and leader; member contract proven |
| V2 Chat routing/UI | C2 `w6:p2` + C3 `w7:p3` | untargeted/direct chat | active, disjoint UI/core files | V1 API contract | no target→TL; explicit target→coder; type/UI focused tests |
| V3 Project/Kanban ingress | completed F5 + C8 verify | project issue/task → HTTP ingress | implemented | V1 identity | real request/task IDs, no body/content in telemetry |
| V4 Queue/admission | completed F6/F1 + C8 verify | queued task → Main Brain | implemented | V3 | enqueue/claim/admission ordered and fail-closed |
| V5 CLI execution | completed F10/F1 + C8 verify | Main Brain → approved CLI | implemented | V4 + consumed readiness | launch/proc IDs, process ordering, no provider credential |
| V6 OmniRoute boundary | C7 `wB:p1` + C8 | CLI/Main Brain → OmniRoute | consume-only proof done | operator readiness declaration | Main Brain reacts to ready/not-ready; no router-internal test |
| V7 Persisted result | completed F6 + C8 verify | CLI/gateway outcome → durable result | implemented | V5/V6 | persistence occurs before delivery, exact result/task IDs |
| V8 Terminal/WS/UI delivery | completed F4 + C2/C8 verify | persisted result → UI | backend implemented; UI verification active | V7 | delivered/drop/backpressure outcome and visible terminal/status correlation |
| V9 Observability/capacity | C6 `w8:p2` + prior F9 | all boundaries | trace active; technical harness done | V3–V8 | eight ordered hops, nine safe IDs; real tier acceptance remains gated |
| V10 Independent acceptance | C8 `wB:p2`, Kiro, Principal | whole vertical slice | rolling | V1–V9 | C8 VERIFIED, Kiro evidence matrix, owner live acceptance token |

### Critical path

```text
C1 squad/server contract
 ├─→ C3 optional-target core API
 ├─→ C2 chat UI behavior
 └─→ C9 focused smoke

C4 JWT gate ─┐
C5 email gate├─→ C2 final server/web matrix ─→ C8 evaluation
C6 trace ────┘

C7 consume-only boundary ────────────────→ C8 evaluation
C8 verified offline result ──────────────→ owner-authorized live P0 run
```

### Definition of done

Offline/source complete:
- chat server/core/UI target contract is coherent;
- Main Brain targeted and server-wide build/tests are green;
- eight-hop metadata-only trace is verified;
- strict OpenSpec validation passes;
- no provider credential/routing implementation remains in Main Brain.

Operationally accepted:
- reachable product stack and DB;
- untargeted chat reaches Kiro and explicit target reaches Codex;
- owner-authorized Kanban task traverses Main Brain and the approved CLI;
- Main Brain consumes ready OmniRoute status and does not inspect its internals;
- result persists and appears in terminal/log/status UI;
- tier acceptance and owner sign-off are recorded.

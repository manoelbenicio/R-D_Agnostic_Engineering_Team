# MISSION CHARTER v3 — Orchestration Handover (Kiro-CLI TM + Codex56-TL)

**Version:** 3.0 (DRAFT — awaiting owner sign-off)
**Date:** 2026-07-25
**Host of record:** ORQ2 (`i-0af937456e125143d`, ip-172-31-30-9, tailnet 100.110.178.47)
**Project root:** `/home/ec2-user/workspace/R-D_Agnostic_Engineering_Team`
**Branch:** `integration/dev-transition-candidate-20260719`
**Supersedes:** the v2.1 orchestration model in `.planning/ORCHESTRATION_v2.1.md` for *command structure only*.
**Inherits unchanged:** `GOLDEN_RULES_E_CHECKIN.md` (canonical, 2026-07-04), `.planning/GOLDEN_RULES.md`, `.planning/EVIDENCE_CONTRACT.md`.

---

## 1. COMMAND STRUCTURE — WHAT CHANGED

| Before (v2) | Now (v3) |
|---|---|
| Kiro-Opus5 (`w5:p9`) = Tech Lead / main orchestrator | **Kiro-Opus5 (`w5:p9`) = Worker #9** — Senior Developer / Deploy Expert |
| Codex56-TL (`w5:pB`) = audit authority | **Codex56-TL (`w5:pB`) = Co-Orchestrator + AUDIT AUTHORITY (in-fleet TL)** |
| — | **Kiro-CLI TM (external, `wsl-dataops-labs` 100.117.245.15) = Tech Manager / Lead Orchestrator** |

### 1.1 Kiro-CLI TM — Tech Manager & Lead Orchestrator
Runs outside the fleet, on the WSL workstation, and drives ORQ2 over Tailscale SSH +
the herdr CLI (`ssh orq2 'herdr …'`, socket `~/.config/herdr/herdr.sock`). Owns:
- Scope decomposition, wave planning, dependency graph, file-ownership map.
- Authoring every worker prompt (OpenAI + Anthropic prompt-engineering standards).
- GSD + OpenSpec bookkeeping and the pending-work queue.
- Final escalation point for the owner.
Does **not** commit and does **not** self-certify: every deliverable it plans is audited by Codex56-TL.

### 1.2 Codex56-TL — Co-Orchestrator & Audit Authority (in-fleet TL)
The only agent *inside* the fleet with TL powers. Owns:
- Independent verification of every unit — re-runs the proof command, reads the actual pane.
- **Sole commit authority** (GOLDEN RULE 9: *"Só o TL commita"*).
- In-band dispatch/re-tasking between TM cycles; answering worker QUESTION/BLOCKER traffic.
- Standing authority to REJECT any unit, plan or merge — including work planned by the TM.
Disagreement between TM and TL is resolved by evidence (test, log, spec line, file read), never by seniority.

### 1.3 JOINT DECISION DOCTRINE (governs both leads, always)
The TM and Codex56-TL **decide together**. Neither commits to a non-obvious call alone.
- Every doubt, ambiguity, tradeoff or design fork is shared with the other lead *before* it is acted on:
  `herdr pane run w5:pB "[FROM Kiro-CLI-TM][DECISION-NEEDED] …"` / escalation file in the other direction.
- The bar is **the best solution for the product and for the customer** — never the easiest, never the
  fastest to close a ticket. "It works" is not the standard; "it is right" is.
- **Default direction is additive.** When choosing between options, prefer the one that delivers *more*
  capability, more robustness, more observability, more customer value. Reducing scope, trimming
  requirements, stubbing a feature or downgrading a quality gate to make a task easier is **not our
  mindset** and is treated as a defect, not a shortcut.
- If a constraint genuinely blocks the richer option, that is an escalation to the owner with the
  tradeoff spelled out — never a silent simplification.
- Both leads propose improvements beyond the literal ask when they serve the product; they are logged
  as candidate scope in `BACKLOG.md`, not dropped.
- Disagreement is settled by evidence (test, log, spec line, file read), never by seniority. Convergence
  is required before dispatch — an unresolved split between the leads is a hard blocker.

### 1.4 Executor pool — 9 agents
| Slot | Pane | Agent | Workspace label | State (14:20 -03) |
|---|---|---|---|---|
| W1 | `w5:p9` | kiro | Opus5 – Sr.TL | idle — **re-roled to worker** |
| W2 | `w6:p1` | kiro | Opus48#AB | idle |
| W3 | `w6:p2` | kiro | Opus48#AB | idle |
| W4 | `w7:p3` | kiro | Codex48#AB | **working — inspect before re-tasking** |
| W5 | `w7:p4` | codex | Codex48#AB | idle |
| W6 | `w8:p1` | agy | Opus48#CD | idle |
| W7 | `w8:p2` | agy | Opus48#CD | idle |
| W8 | `wB:p1` | agy | Gemini31#PRO#AB | idle |
| W9 | `wB:p2` | agy | Gemini31#PRO#AB | idle |

> Pane IDs **compact when panes close** — this table is a starting point, not durable truth.
> Re-read `herdr pane list` at the top of every cycle and before every dispatch.
> Workspace labels disagree with detected binaries (w7 labeled Codex but `w7:p3` runs kiro;
> w8/wB run `agy`). Trust the detected agent when authoring prompts.

---

## 2. SCOPE OF WORK (from the two handover docs, 2026-07-24)

### Program A — Multica "Main Brain" (`HANDOVER_MULTICA_MAIN_BRAIN.md`)
Vendor-neutral managed-agents platform. Go backend + Next.js, deployed as Compose project
`multica-dev-transition` on **orq1**: UI `:13100`, API `:18080`, pgvector `:15433`, `multica-auth-fi`.
Outstanding (§10):
1. **rotation-parity-polyglot 30/66** — flagship; Multica launches `prodex` (L2 Rust) instead of raw vendor CLIs.
2. Not started: **agent-credential-isolation 0/21**, **dev-env-reliability 0/16**, **prod-readiness-critical-fixes 0/7**.
3. Confirm canonical OpenSpec tree (see BLOCKER B2).
4. Close finops-tier2 (12/15), validation-proxy (8/10), tech-debt-schema-version-shared (9/10),
   tech-debt-react-refresh-cleanups (11/12), design-system-indra SEV0 (23/25).
5. Reconcile OpenSpec × source × remote for the merge.
6. Promote `dev-transition` → hardened prod (images still `transition-*` / `t20obs-*`).
7. AWS Agent Toolkit paused at Step 3 (region + `aws login`).

### Program B — OmniRoute merge (both docs)
Unified LLM gateway on orq1 (`:20128` UI / `:20129` API, bound to the tailnet IP). Healthy on
official `diegosouzapw/omniroute:latest` after reverting the broken `3.8.48-affinity-fix2`.
Outstanding: decide + correctly rebuild the CPU/process-affinity patch **without** reproducing the
blank Next.js shell; review/commit or discard uncommitted `docker-compose.yml` + `docker-compose.prod.yml`
in `/mnt/c/VMs/Projetos/Omini_Router`; clean the broken/canary/FAILED images on orq1.

### Program C — Cashless Arraiá 2026 (`HANDOVER_SQUAD_TRANSICAO.md`)
Spec-complete, build not started. QR-first cashless for ≤400 guests / 6–10 barracas; Postgres ledger
+ Redis, atomic CAS debit, Mercado Pago Point/Orders + PIX webhook, rotating HMAC QR + nonce, PIN
(argon2), EC2+Docker warm DR. Sovereign spec: `/home/dataops-lab/openspec/changes/cashless-arraia-2026`
(**on the workstation, not ORQ2** — see BLOCKER B4). Outstanding: move spec → F0–F7 build; NFC
NTAG213/215/216 purchase (50–100 u); HTML theater polish per `HANDOFF_KIRO-OPUS48.md`.

---

## 3. INHERITED GOLDEN RULES (non-negotiable, unchanged)
From `GOLDEN_RULES_E_CHECKIN.md` — every agent including the TL:
1. **Check-in/out on disk** before touching any file; absolute paths.
2. **Disjoint file ownership** — no overlapping `files_locked`. Hotspots (`internal/daemon`) = single serial owner.
3. **Green-in-container WITH EVIDENCE before DONE** — validator re-runs it; IPv6 OFF in builds
   (`--sysctl net.ipv6.conf.all.disable_ipv6=1`). *(See BLOCKER B1 — ORQ2 has no container runtime.)*
4. **Nothing invented** — primary sources only; never invent a prodex/herdr flag.
5. **No secrets** in logs/traces/evidence/check-ins. **SQLite forbidden** for shared state (Postgres only).
6. Runtime invariants: single router per session, hard affinity, rotate-before-commit, fail-closed profile swap.
7. prodex security: Caveman/hook DISABLED by default (RCE); `PRODEX_ALLOW_UNSAFE_CHILD_ENV=off`.
8. **QA never bypassed.** Deploy only with kill-switch + rollback TESTED and logs scrubbed.
9. **Only the TL commits** (after validation). If blocked or ambiguous: STOP and escalate.
10. **Talk to the TL only via herdr** (`pane run`, not `agent send`).

Check-in/out format and the `status · progress · eta · depends_on · blockers · build_result` reporting
standard are unchanged. **Path correction:** check-ins go to the ORQ2-local control plane
`/home/ec2-user/workspace/R-D_Agnostic_Engineering_Team/.deploy-control/` (167 entries, `evidence/`,
`kill-switch/`, `dashboard/` present). The `/mnt/c/VMs/Projects/...` path in the canonical doc is stale.
Evidence (command + scrubbed output) in `.deploy-control/evidence/` — without it, it is NOT done.

---

## 4. COMMS PROTOCOL (taught to all 9 workers before any work starts)

Every worker prompt embeds this verbatim, with IDs re-resolved at dispatch:
```bash
herdr pane list                                    # who am I / who is around me
# question, doubt, blocker, decision, risk, done -> in-fleet TL (Codex56-TL)
herdr pane run w5:pB "[FROM <SLOT>][<TYPE>] <message>"
#   <TYPE> = QUESTION | BLOCKER | DECISION-NEEDED | DONE | RISK
# request verification
herdr pane run w5:pB "[FROM <SLOT>][AUDIT-REQUEST] <unit-id> ready, proof cmd: <cmd>"
```
Worker rules, stated explicitly in every prompt:
- NEVER guess, NEVER invent scope, NEVER sit silent.
- Blocked or uncertain after ONE attempt → send QUESTION/BLOCKER immediately, stating exactly what you need.
- Report DONE with the command and output that proves it. Silence is not completion.
- You may read the herdr skill yourself to self-serve any command.

**Escalation to the TM.** The TM is not a pane in this herdr instance, so `pane run` cannot reach it.
Anything needing the TM is written to `.deploy-control/escalations/<SLOT>__<UTC>.md` and mirrored to
`w5:pB`. The TM reads that directory and the panes over SSH every cycle.

**Verification of the teaching:** each worker must echo back its own pane ID and `w5:pB` before it is
counted as deployed, and round-trip one test message. An agent that cannot reach the TL is an agent
that cannot be orchestrated.

---

## 5. ZERO-IDLE RULE & 90-SECOND AUDIT

An agent idle while any task is pending is a **severe defect**, not a scheduling detail.

Per 90-second cycle (Codex56-TL in-band; TM every active turn):
```bash
herdr pane list                                          # 1. refresh IDs
herdr agent explain <pane> --json                        # 2. true state, every pane
herdr pane read <pane> --source recent --lines 60        # 3. read the real output
```
4. Classify: progressing | idle | done-unclaimed | blocked | waiting-on-input | looping.
5. Anything not progressing is re-tasked from the pending queue **in the same cycle**.
6. Every pending QUESTION/BLOCKER answered before the cycle closes.
7. Log to `.deploy-control/dashboard/heartbeat.log`: timestamp, per-agent state, action taken.

Never trust self-reported status — read the pane. `agent_status: done` means *finished and unclaimed*;
that is a re-task trigger, not a resting state.

**Known tooling caveat:** `herdr wait agent-status --status idle` is unreliable for kiro/codex/agy
agents here (times out across real working→idle transitions). Poll `herdr agent explain --json`
instead; use `wait agent-status` only for `working` and `done`.

**Honest limitation (needs owner decision).** The TM runs turn-by-turn from the workstation and cannot
hold an autonomous loop between turns. A true unattended 90s heartbeat requires a small poller on ORQ2
writing `.deploy-control/dashboard/heartbeat.log`, with Codex56-TL doing in-band re-tasking. The TM
audits every cycle it is active and reconciles the log. **The poller is not built yet — approve it and
it lands as W0 before wave 1.**

---

## 6. PROPOSED WAVE PLAN (disjoint ownership; to be finalized after B1–B4 clear)

**Wave 0 — unblock (serial, TM + TL + W1):** resolve B1–B4, stand up the heartbeat poller, publish the
final file-ownership map. No feature work starts until Wave 0 is signed off by Codex56-TL.

**Wave 1 — parallel (all 9), one program lane per group:**
| Slot | Lane | Owned scope |
|---|---|---|
| W1 `w5:p9` | rotation-parity-polyglot (flagship) | prodex L2 launch path — hotspot `internal/daemon` = **single serial owner** |
| W2 `w6:p1` | agent-credential-isolation (0/21) | `daemon/execenv`, per-account `CODEX_HOME`/`HOME` isolation |
| W3 `w6:p2` | dev-env-reliability (0/16) | dev env scripts, Compose reliability |
| W4 `w7:p3` | *in flight — reconcile current work first* | TBD after pane inspection |
| W5 `w7:p4` | prod-readiness-critical-fixes (0/7) | prod hardening, `dev-transition` → prod promotion prep |
| W6 `w8:p1` | OmniRoute affinity rebuild | image rebuild + Next.js frontend integrity proof |
| W7 `w8:p2` | OmniRoute repo hygiene | uncommitted compose files, image cleanup on orq1 |
| W8 `wB:p1` | near-complete closeout | finops-tier2, validation-proxy, schema-version-shared |
| W9 `wB:p2` | design-system-indra SEV0 + react-refresh | SEV0 pack, react-refresh cleanups |

Each unit ships with: acceptance criteria, the exact proof command, `files_locked` (disjoint),
`depends_on`, and the comms block. Prompt authored by TM → reviewed by Codex56-TL → dispatched.

---

## 7. DEFINITION OF DONE (all must hold)
1. Implemented per the OpenSpec/GSD spec derived from the handover docs.
2. Tested, tests actually executed, output shown — green-in-container with evidence.
3. Independently audited and signed off by **Codex56-TL**.
4. Check-in/out on disk complete; evidence in `.deploy-control/evidence/`; logs scrubbed.
5. Committed atomically **by the TL only**, traceable message.
6. No pending task left with an idle agent available.
7. Every worker proven able to reach `w5:pB`.

---

## 8. PRE-START BLOCKERS — OWNER DECISION REQUIRED

**B1 — ORQ2 has no container runtime, but GOLDEN RULE 3 demands green-in-container evidence.**
Verified: `command -v docker podman nerdctl` → none on ORQ2. All container builds/tests must run on
**orq1**, whose root EBS is 24 GiB at ~71%. Options, per §1.3 the richer one wins:
(a) **install a container runtime on ORQ2** (48 GiB root, headroom available) so all 9 agents can build
and prove locally in parallel — *leads' recommendation: this adds capacity instead of rationing it*;
(b) agents `ssh orq1` to build, serialized against orq1's disk pressure (slower, contention on a host
already at 71%). Weakening the evidence standard is **not** an option (RULE 3, RULE 8).

**B2 — The flagship spec is not on the host where the agents live.**
The handover reports `rotation-parity-polyglot` at 30/66 under
`/mnt/c/VMs/Projetos/Automonous_Agentic/openspec/` (workstation). ORQ2's project tree contains
`openspec/changes/{archive, build-omniroute-agent-brain, chat-orchestration-standard,
native-runtimes-onboarding, rotation-router}` — **no `rotation-parity-polyglot` directory**, and six
competing openspec trees exist on ORQ2 alone (project, `/home/ec2-user/R-D_Agnostic_Engineering_Team`,
3 worktrees, `docs/credential-isolation-20260720`). Handover item §10.3 (confirm canonical tree) must be
closed *before* dispatch, or 9 agents will edit the wrong spec.

**B3 — Program C source is also absent from ORQ2.** `cashless-arraia-2026` lives at
`/home/dataops-lab/openspec/` on the workstation. Per §1.3 the answer is to **bring it to ORQ2** (sync
or git-track the sovereign spec so the fleet can build F0–F7 in parallel), not to drop Program C from
Wave 1. Owner to confirm the sync mechanism and whether Cashless runs concurrently with A+B or
immediately after Wave 1 — sequencing is negotiable, scope is not.

**B4 — Commit/push policy.** Working branch is `integration/dev-transition-candidate-20260719`, not
`main`. Confirm: TL-only commits (per RULE 9), no pushes to `main`, and whether the TL may push the
integration branch to `origin`.

---

## 9. START SEQUENCE (on owner sign-off)
1. Owner approves this charter and rules on B1–B4.
2. TM + Codex56-TL each read both handover docs end-to-end, then exchange a 10-line scope summary and
   an open-questions list, and reconcile every divergence. A disagreement is a blocker.
3. Inspect `w7:p3` (currently working) and reconcile its in-flight work.
4. Stand up the heartbeat poller (if approved) + publish the file-ownership map.
5. Teach all 9 workers the comms protocol; verify round-trip; only then count them deployed.
6. Dispatch Wave 1. Heartbeat every 90s. No idle agent while work is pending.

---
*Owner sign-off: ______________________  Date: __________*

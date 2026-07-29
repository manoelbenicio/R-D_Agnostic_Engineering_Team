# R1 — Cline→GLM-5.2 Live Acceptance (P0-GLM-LIVE-ACCEPTANCE)

- agent: `Codex56#A`  ·  lane: `A9-GLM`  ·  task: `P0-GLM-LIVE-ACCEPTANCE`  ·  pane: `w7:p3`
- evidence lock: `.deploy-control/p0/evidence/R1-glm-live-acceptance.md` (only mutable artifact)
- covers OpenSpec: **5.6** (Cline→GLM-5.2) and, by overlap, **8.1** / **8.2** for the GLM family (one run closes all).
- route (A3-frozen): `Cline` (`brain.CLIOpenAICompatible`) → **`cp/cline-pass/glm-5.2`**, OpenAI Chat Completions.
- ownership: **exactly ONE** future integrated GLM live execution. **No execution now.**
- product source: **READ-ONLY**. This artifact records preflight + block state; it is not a run record yet.
- boundaries: **GLM only** — no Kimi run, no Opus48 run, no Antigravity run, no source edits, no auth/fallback work.

> STATUS: **BLOCKED** — implementation not integrated and no live-run authorization/token/registry
> preconditions exist. Recorded via `p0_control.py block`. This lane holds the single GLM run token
> and will fire it exactly once after the preconditions in §3 clear.

---

## 1. Tool preflight (exact, this pane)

| Tool | Path | Version | Result |
|---|---|---|---|
| HERDR_PANE_ID | env | `w7:p3` | matches Codex56#A assignment ✓ |
| HERDR_ENV | env | `1` | Herdr context ✓ |
| git | `/usr/bin/git` | `2.50.1` | ✓ |
| python3 | `/usr/bin/python3` | `3.9.25` | ✓ (p0_control check-in/block OK) |
| rg | `/usr/local/bin/rg` | `15.2.0` | ✓ |
| herdr | `/home/ec2-user/.local/bin/herdr` | `0.7.4` | ✓ |
| Go (verify-only, per A1-A2 §1) | `/home/ec2-user/goroot/go/bin/go` | `go1.26.1 linux/amd64` | present; **not run** (no integrated build to test in this lane) |

**Runtime/registry preflight (no secrets):**
- `curl --max-time 3 http://127.0.0.1:20128/v1/models` → **HTTP `000` (unreachable)**. No OmniRoute
  runtime endpoint, no live token, no registry access from this pane. No credential value was sent or read.

---

## 2. Verified block preconditions (facts from disk, not assumed)

| Fact | Source | Value |
|---|---|---|
| Product implementation not authorized/integrated | `control.json:4-5` | `implementation_authorized=false`; source edits blocked until `FLEET_SATURATED` GREEN + frozen source locks |
| GLM live run not authorized | `control.json:44` | `live_runs.cline_glm = {"authorized": false, "accepted_run": null}` |
| Kimi/Opus48/Antigravity also unauthorized (out of this lane) | `control.json:45-47` | all `authorized=false` |
| OmniRoute runtime unreachable | §1 curl | HTTP `000` at `127.0.0.1:20128` |
| Exact GLM RouteModel frozen (ID ready, not the blocker) | `A3-route-freeze.md` §2 | `cp/cline-pass/glm-5.2` FROZEN |
| Wiring not yet integrated | `R1-cline-route-source-delta.md` §4 (D1–D7) | design-only; W1 has not applied edits |
| Enriched registry availability pending | `A3-route-freeze.md` BLK-AVAIL | OmniRoute has not published `available=true` row |

Per PROTOCOL "Live-run token": a run is allowed **only** when `control.json` has the route-family
`live_run.authorized=true`, exact RouteModel, integrated commit/build/config provenance, and no prior
accepted run. **None of these hold.** Blocking is the correct, non-speculative action.

---

## 3. Preconditions that must ALL clear before the single run

1. W1 integrates the D1–D7 Cline delta (`R1-cline-route-source-delta.md` §4) and its **focused
   package tests pass** (`runtimeenv`, `execenv`, `daemon`, `pkg/agent`) — recorded by A9/W1.
2. A3 **BLK-AVAIL** clears: OmniRoute publishes the enriched registry row for `cp/cline-pass/glm-5.2`
   (`protocol=openai-chat`, `available=true`) so the gateway admits the route.
3. Cline `${ENV}` expansion of `${CLINE_OMNIROUTE_API_KEY}` in `settings.apiKey` confirmed (A1-A2 §5),
   or an approved secret-free alternative.
4. D-V3-25(B) security stop lifted (key revocation confirmed) per `authoritative-route-matrix-D-V3-27.md`.
5. `control.json` `live_runs.cline_glm.authorized=true` set by the **Principal Orchestrator**, with a
   recorded run ID, integrated commit/build/config provenance, and `accepted_run=null` (no prior run).

Only the Principal issues the token; this lane executes the run, it does not self-authorize.

---

## 4. The single authorized run — exact plan (execute later, once §3 is GREEN)

Real product usage, not a synthetic probe (per P0_MAIN_BRAIN_EXECUTION Phase D). One run only:
1. Record the reserved run ID from `control.json` **before** launch.
2. Create/use a squad + project; create a Kanban task.
3. Assign an agent on the Cline frontend with `RouteModel = cp/cline-pass/glm-5.2`.
4. Observe Agent Brain launch (credentialless; single OmniRoute secret via `CLINE_OMNIROUTE_API_KEY`).
5. Capture terminal evidence (§5) — protocol/tools/reasoning/usage/cancel/error as applicable — and
   the persisted terminal result delivered to the UI.
6. This one run closes overlapping **5.6 / 8.1 / 8.2** for the GLM family; **no second run**.

Scope guard: **GLM only.** Do not launch Kimi (A3 BLK-KIMI), Opus48, or Antigravity from this lane.

## 5. Terminal-evidence capture template (to be filled by the future run — currently empty)

```
run_id:            <from control.json live_runs.cline_glm before launch>
utc_started:       <ISO-8601>
integrated_commit: <git rev of the W1-integrated build>
route_model:       cp/cline-pass/glm-5.2
cli_kind:          openai-compatible (cline)
protocol:          openai-chat (/v1/chat/completions)
kanban_task_id:    <id>
launch_observed:   <yes/no + brain event refs>
terminal_result:   <persisted result ref delivered to UI>
tools/reasoning/usage/cancel/error: <captured, not described>
scrubbed_logs:     <grep secret/token -> 0 matches shown>
utc_finished:      <ISO-8601>
```
No secrets, tokens, prompts, provider payloads, or account identities are recorded (EVIDENCE_CONTRACT rule 5).

---

## 6. Blocker (as recorded via p0_control.py block)

- blocker: `implementation not integrated; live_runs.cline_glm authorized=false; runtime token/registry preconditions unavailable`
- owner: `Principal Orchestrator + W1 + OmniRoute registry owner`
- next action: `after W1 focused tests and explicit live authorization, execute one integrated run and capture terminal evidence`

## 7. Agent status block

- STATUS: BLOCKED (owns the single future GLM run; no execution now)
- DELIVERED: preflight; disk-verified block preconditions (§2); precondition checklist (§3); the exact
  one-run acceptance plan (§4) + terminal-evidence template (§5).
- FILES: `.deploy-control/p0/evidence/R1-glm-live-acceptance.md` (only mutable file).
- VALIDATION: none run; no source edited; no live run. OmniRoute unreachable confirmed; `control.json`
  facts cited by line.
- EVIDENCE: this artifact; `control.json:4-5,44-47`; `A3-route-freeze.md`; `R1-cline-route-source-delta.md`.
- BLOCKERS/LIMITATIONS: §2/§3/§6 — external (Principal authorization, W1 integration, OmniRoute registry, security stop).
- W1_HANDOFF: none new; consumes W1 integration + focused-test evidence and Principal live authorization as run preconditions.

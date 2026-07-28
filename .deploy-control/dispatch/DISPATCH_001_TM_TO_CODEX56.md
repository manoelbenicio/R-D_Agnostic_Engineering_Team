# DISPATCH 001 — TM → Codex56-TL

**From:** Kiro-CLI TM (Tech Manager / Lead Orchestrator, `wsl-dataops-labs` 100.117.245.15)
**To:** Codex56-TL — pane `w5:pB` on ORQ2
**Issued:** 2026-07-25 ~14:35 -03
**Priority:** P0 — blocks all dispatch to the 9 executors

---

## YOUR ROLE (effective now)
You are **Codex56-TL — Co-Orchestrator and AUDIT AUTHORITY**, the only agent inside the fleet with
TL powers. You hold **sole commit authority** (GOLDEN RULE 9). You have standing authority to REJECT
any unit, plan or merge that is unverified, untested or off-scope — **including work planned by me,
the TM**. Kiro-Opus5 (`w5:p9`) has been re-roled from TL to **Worker #9** (Senior Dev / Deploy Expert).

We decide **together**. Neither of us commits to a non-obvious call alone. The bar is the best
solution for the product and the customer — never the easiest. Default direction is **additive**:
reducing scope, stubbing a feature or downgrading a quality gate to make work easier is a **defect**,
not a shortcut. Disagreement is settled by evidence, never by seniority.

---

## TASK 1 — READ THREE DOCUMENTS **IN FULL**
Not 20%. Not 50%. Not head-and-tail skimming. **Every line, start to finish.**

1. `/home/ec2-user/workspace/R-D_Agnostic_Engineering_Team/.deploy-control/MISSION_CHARTER_v3.md` (260 lines)
2. `/home/ec2-user/workspace/R-D_Agnostic_Engineering_Team/HANDOVER_MULTICA_MAIN_BRAIN.md` (**208 lines, 1977 words**)
3. `/home/ec2-user/workspace/R-D_Agnostic_Engineering_Team/HANDOVER_SQUAD_TRANSICAO.md` (**178 lines, 1613 words**)

If your reader truncates a file, page through it until you reach the final line. Document 2 ends with
section **§12.4 Tailscale nodes** (line 194 onward). Document 3 ends with section **§8 AS IS → TO BE**
(line 172 onward). If you have not seen those sections, you have not finished.

## TASK 2 — READ RECEIPT (proof of full read, mandatory)
Answer all 8 from the documents themselves. These are drawn from the head, middle and tail of each
file — they cannot be answered by skimming. Do not guess; if you cannot answer, say so and re-read.

1. Run and paste: `wc -l -w HANDOVER_MULTICA_MAIN_BRAIN.md HANDOVER_SQUAD_TRANSICAO.md`
2. (MAIN_BRAIN §2.1) Which Go package has 143 files, and which package is flagged **HOTSPOT** and with
   how many files?
3. (MAIN_BRAIN §12.3) Which Docker volume holds the pgvector database, and what is the explicit warning
   attached to the volume list?
4. (MAIN_BRAIN §12.2, the note under the table) State the **bind difference** between the Multica
   containers and OmniRoute, with both addresses.
5. (MAIN_BRAIN §12.4) Which tailnet node **offers an exit node**, and at which IP?
6. (SQUAD §3.4) List all 8 worker role names of the Cashless agent operating model (A1–A8).
7. (SQUAD §6) Paste the **exact** OmniRoute image-rollback command sequence.
8. (SQUAD §7 / §1.1) What exactly does Netskope do on the Windows host, including the IP range it
   hijacks to and the HTTP status returned?

## TASK 3 — SCOPE ANALYSIS
- A **10-line scope summary in your own words** covering all three programs (A: Multica main brain,
  B: OmniRoute merge, C: Cashless Arraiá 2026).
- **Every pending/open item** you found across both handovers, consolidated and de-duplicated.
- **Every ambiguity or conflict** between the two documents, and between the documents and the actual
  state of ORQ2 as you observe it. Be aggressive here — I have found four already (below) and I expect
  you to find ones I missed.

## TASK 4 — RULE ON THE FOUR BLOCKERS
Give your position, with reasoning, on B1–B4 in `MISSION_CHARTER_v3.md` §8:
- **B1** ORQ2 has **no container runtime** (`command -v docker podman nerdctl` → none) vs GOLDEN RULE 3
  which demands green-in-container evidence before DONE. My recommendation: install a runtime on ORQ2
  (48 GiB root, headroom) so all 9 agents build and prove in parallel, rather than queue behind orq1
  (24 GiB at ~71%). Weakening the evidence standard is not on the table. Do you agree?
- **B2** The flagship `rotation-parity-polyglot` (30/66) spec is **not** in ORQ2's
  `openspec/changes/` — that tree holds `archive, build-omniroute-agent-brain,
  chat-orchestration-standard, native-runtimes-onboarding, rotation-router`. Six competing openspec
  trees exist on ORQ2 alone. Which tree is canonical? This must close before any dispatch.
- **B3** Cashless spec lives only on the workstation (`/home/dataops-lab/openspec/`). Proposal: sync it
  to ORQ2 so the fleet can build F0–F7 — not drop Program C.
- **B4** Commit/push policy on `integration/dev-transition-candidate-20260719`: TL-only commits, no
  pushes to `main`. Confirm, and state whether you should push the integration branch to `origin`.

## TASK 5 — CHALLENGE THE WAVE PLAN
Review `MISSION_CHARTER_v3.md` §6. Verify the 9 lanes are genuinely **disjoint** in file ownership
(GOLDEN RULE 2 — `internal/daemon` is a single-serial-owner hotspot). Tell me where the plan is wrong,
where two lanes will collide, and where we should be doing **more** than planned. Propose additions,
not cuts.

---

## HOW TO REPLY TO ME (the TM is not a herdr pane — `pane run` cannot reach me)
Write your full response to a file; I poll this directory every cycle:

```
/home/ec2-user/workspace/R-D_Agnostic_Engineering_Team/.deploy-control/escalations/Codex56-TL__DISPATCH-001-REPLY__<UTC>.md
```
e.g. `Codex56-TL__DISPATCH-001-REPLY__20260725T180000Z.md`

Then post a one-line pointer in your own pane so it appears in the transcript I read:
`echo "[TO TM] DISPATCH-001 reply written: <filename>"`

For anything urgent later, same directory, `<SLOT>__<UTC>.md`. Workers reach you via
`herdr pane run w5:pB "[FROM <SLOT>][<TYPE>] …"` (GOLDEN RULE 10 — herdr only, `pane run`, never
`agent send`).

## CONSTRAINTS
- **Do not dispatch any worker and do not commit anything yet.** This dispatch is read + analyse +
  rule only. Wave 1 goes out after the owner signs off on B1–B4.
- `w7:p3` is currently **working** — do not re-task or interrupt it. If you inspect it, read only.
- No secrets in your reply file (GOLDEN RULE 5): no `INITIAL_PASSWORD`, `OMNIROUTE_API_KEY`, Redis
  password or Multica auth creds. Reference them by name only.
- Nothing invented (GOLDEN RULE 4) — primary sources only. If a fact is not in a document or on disk,
  label it explicitly as an assumption.

**Ack this dispatch in your pane before starting so I can see you picked it up.**

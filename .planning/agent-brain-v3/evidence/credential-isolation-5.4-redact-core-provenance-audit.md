# EV-CREDISO-5.4-CORE — INTERIM PROVENANCE / GOVERNANCE AUDIT (redaction core)

Read-only provenance & manifest-integrity audit of the current `pkg/redact` redaction core, tracing its
chain of custody. **Governance focus, not a technical re-run** (technical reproduction already exists;
see prior artifacts). **Not an acceptance; does not adjudicate; changes no checkbox.**
Auditor: **Kiro/Opus-4.8 — session `w8:p2`**. Adjudicator: **Kiro TL** (`w3:p3`).

> **Identity caveat (disclosed, no fabrication).** Several actors below share the **Kiro/Opus-4.8** model
> family across distinct panes/sessions; separation is by session, not identity. Where an identity is not
> recorded in the ledger/artifacts, this audit says so explicitly and **does not invent one**.

## Golden Rule check-IN — 2026-07-18T21:49:00Z
- Mode: READ-ONLY governance audit. Only file created = this artifact. No product/shared/spec/task/git/
  index edits; no credentials/env values; no network/DB/services. No duplicate technical tests run.

## Subject under audit (current disk, HEAD `b6571299b00c8e388abefe7ef9dcbcf8ac715d7f`)
- `server/pkg/redact/redact.go` SHA-256 `f409ba8a9f3e63618d59c5a8692296f8f7c019c9e558576b8786a058fbf68a5c`
  (+172 lines vs HEAD, **unstaged** working-tree modification).
- `server/pkg/redact/redact_test.go` SHA-256 `5a37941a1c7f1bd7263368a6479104c81300f7981e9f47bfb6b0cd17a602fec9`
  (+169 lines vs HEAD, **unstaged**).
- `git status --porcelain` = ` M` for both (worktree-modified, **index clean** → not staged, not committed).

## Chain of custody — traced item by item

### 1. Actual content producer of `redact.go`/`redact_test.go` — **UNATTRIBUTED (provenance gap)**
- `AGENT_LEDGER.md:242,245` attribute the `SanitizeForLog` remediation to a **"concurrent 16:15:46 edit"**
  (2026-07-18 16:15:46) with **no agent identity and no pre-edit check-in recorded**. The +172/+169-line
  expansion now on disk is therefore from an **unrecorded producer**.
- **Finding:** the redaction-core change has **no attributed producer and no pre-edit Golden-Rule
  check-in** — a governance gap of the same class as the task-5.3 missing-check-in. Not fabricating an
  identity; recording the gap.

### 2. Prior Kiro no-op verification — **CONFIRMED (verifier, not producer)**
- `AGENT_LEDGER.md:242` + `evidence/credential-isolation-redact-core-fix.md`
  (SHA-256 `f73fa02e8adb446201d56f514a83c30f7e009279242a7e5dfac91781994301ea`, matches ledger pin):
  **Kiro/Opus-4.8 (co-lead), CRED-REDACT-FIX = NO-OP**, read-only verify, **STOP per pack, no product
  edit**. Confirmed the `TestSanitizeForLog` query-secret bypass was **already remediated** by the
  16:15:46 edit (full suite + race + vet green). So Kiro/Opus-4.8's role = **verifier**, explicitly not
  the producer.

### 3. EV-CREDISO-5.4-CORE artifact/hash — **PARTIAL (artifact present, hash NOT pinned)**
- The core-acceptance review artifact is `evidence/credential-isolation-redact-core-review.md`
  (current SHA-256 `521cef3196ca3c8b0c98b1ecdb120407c217d33c060660350783ac09e7fa8c12`), grading
  **ACCEPT (core module only)** after `build`/`vet`/`test -count=20 -race` PASS.
- Its producer is attributed **only externally**, via `.deploy-control/Antigravity__QA5-4-CORE__20260718T172900Z.md`
  (`agent: Antigravity`, DONE, locks that review file). **The review artifact itself carries no internal
  author/pane line** — manifest-integrity weakness.
- **Finding:** unlike `EV-CREDISO-5.4-EMAIL` (whose review hash `3a3018b4…` is pinned in the ledger), the
  ledger **does not pin a canonical accepted SHA-256 for `EV-CREDISO-5.4-CORE`**. Recommend pinning
  `521cef31…` (or the accepted revision) as the canonical EV hash.

### 4. Missing Gemini/GLM distinct review — **CONFIRMED MISSING (adjudication gate open)**
- `AGENT_LEDGER.md:242,245` gate clearing the 5.4 redact blocker on **"Gemini-log-safety … independently
  reproducing the 9 named tests/×20/race/vet/build + synthetic-env isolation — await that report."**
- Disk scan finds **no Gemini/GLM redact-core review artifact** (`*gemini*redact*`/`*glm*redact*` → NONE).
- **Ambiguity flagged (not resolved here):** `AGENT_LEDGER.md:273,275` note the mapping
  **Antigravity ↔ pane `w5:p2` ↔ label "Gemini-log-safety" is NOT durably pinned**. So it is unclear
  whether the pending "Gemini-log-safety" review is a *distinct* actor from the Antigravity QA5-4-CORE
  already delivered, or the same actor under an unbound label. The co-lead still lists Gemini-log-safety
  as **pending after** Antigravity's core review (row 242 post-dates the 17:31Z QA5-4-CORE), implying they
  are treated as distinct. **Not fabricating the mapping; flagged for TL to resolve.**

### 5. Task / checkbox / ledger / index state — **CONFIRMED**
- `openspec/changes/agent-credential-isolation/tasks.md:34` = `- [ ] 5.4 …(sanitizeForLog)` → **UNCHECKED**.
- Ledger: **5.4 OPEN** across rows 230/232/236/242/245/291/326; cred count **4/21**; redact-core
  confirmation **gated on Gemini-log-safety**; whole-codebase audit (`w7:p2`, row 291) TECHNICAL PASS but
  **ADJUDICATION HELD** pending a distinct `w7:p1` contract-integrity audit.
- Index: redact core is **unstaged/uncommitted** working-tree state (item "Subject" above).

### 6. producer ≠ reviewer ≠ adjudicator — **NOT fully truthfully demonstrable (today)**
- **Producer:** unattributed 16:15:46 edit → identity unknown ⇒ **cannot prove distinctness from any
  reviewer**. This is the decisive gap.
- **Verifier/QA:** Kiro/Opus-4.8 (NO-OP fix verify) and Antigravity (QA5-4-CORE ACCEPT-core).
- **Distinct independent reviewer:** Gemini-log-safety — **missing**.
- **Adjudicator:** Kiro TL.
- **Finding:** reviewer≠adjudicator is demonstrable (Antigravity/QA vs Kiro TL), **but producer≠reviewer is
  NOT** — the producer is anonymous, and the Antigravity↔Gemini-log-safety↔pane mapping is unpinned. The
  full three-way separation for the redaction core **cannot be truthfully certified** until (a) the
  producer/check-in is attributed (or an owner-accepted process-exception waiver is recorded) and (b) the
  distinct Gemini-log-safety review is delivered with pinned provenance.

## Governance gates recommended to Kiro TL (before clearing the redact-core 5.4 blocker)
1. **Attribute the producer** of the 16:15:46 `redact.go`/`redact_test.go` edit with a (retro)active
   check-in, OR record an owner-accepted process-exception waiver (mirroring the 5.3 governance ruling).
2. **Obtain the missing Gemini-log-safety distinct review** (the 9 named tests ×20/race/vet/build +
   synthetic-env isolation) and **durably pin its identity ↔ pane** with explicit distinctness from the
   producer and from Antigravity.
3. **Pin a canonical `EV-CREDISO-5.4-CORE` accepted hash** in the ledger (candidate: review
   `521cef31…`, redact.go `f409ba8a…`, redact_test.go `5a37941a…`).
4. **Add internal provenance** (author/pane/date) to `credential-isolation-redact-core-review.md` so the
   artifact is self-attributing, not dependent on a separate check-in file.
5. Note the redact core is **unstaged** — any push must stage it as a dependency-complete unit alongside
   the (separately reviewed) Claude clean-room hunk.

## Non-claims
- No adjudication; no checkbox change; no technical tests re-run; **no identity fabricated** — every
  unknown is recorded as unknown. Reviewer ≠ adjudicator. Kiro TL adjudicates.

## Golden Rule check-OUT — 2026-07-18T21:51:00Z
- Files created: this artifact only. Source/ledger/producer artifacts unchanged; no git stage/commit/push;
  no network/DB/services/credentials/env. Status: DONE (interim provenance audit). The expanded clean-room
  independent review resumes on the next dispatch **after** the producer (`w7:p2`) posts checkout with a
  stable hash.

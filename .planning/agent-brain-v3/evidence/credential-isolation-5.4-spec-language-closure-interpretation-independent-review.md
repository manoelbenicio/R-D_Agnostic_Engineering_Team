# EV-CREDISO-5.4-SPEC-LANG — INDEPENDENT REVIEW of the spec-language closure interpretation

Independent review of `credential-isolation-5.4-spec-language-closure-interpretation.md`
(SHA-256 `0b174723d4cc4de402b01678a9081c79ea917f9752bce257f42261074119f301`, verified stable).
Reviewer: **Kiro/Opus-4.8 — session `w8:p2`**, distinct pane from the interpretation's author
**Kiro/Opus-4.8 `w7:p2`**. **Technical / spec-interpretation review only; no choice; no acceptance.**
Kiro TL adjudicates.

> **Independence caveat:** reviewer (`w8:p2`) and author (`w7:p2`) are the same model family, distinct
> sessions. All findings re-derived first-hand from the exact spec/tasks/proposal + ledger.

## CHECK-IN 2026-07-18T22:33:00Z
Mode: READ-ONLY. Sole deliverable = this file. Excluded (honored): no source/test/spec/tasks/shared-planning/
git/index/ref edit; no credentials/env values; no DB/network/services.

## Provenance verified (HEAD `b6571299`)
- Reviewed doc `0b174723…f301` (mtime 19:33:43, stable).
- **Doc's 3 hashes MATCH current bytes:** tasks.md `3bdbc1e1…a0dc`, spec.md `02b0a5c1…a63b`, proposal.md
  `a15b62c1…8636`.
- Exact texts confirmed first-hand:
  - **spec.md:39-46 (normative):** `### Requirement: Não vazamento de segredo` — "SHALL NOT registrar em log
    o conteúdo de credenciais **ao resolver ou montar as pastas por conta**; apenas metadados (caminho, tipo
    de arquivo) podem ser logados." Scenario: mounted-`auth.json` diagnostic logs path/type/mtime, **nunca**
    o conteúdo do token.
  - **tasks.md:34:** `- [ ] 5.4 Confirmar que nenhum segredo aparece em logs (sanitizeForLog).`
  - **proposal.md:69:** "segredos nunca no frontend; **logs redigidos (`sanitizeForLog`)**."

## Verification of the doc's three central claims

### Q1 — Is absolute language required? → doc says "no (not by the SHALL)"; **CONFIRMED as text reading**
The normative SHALL is genuinely **scope-limited**: it forbids credential **content** in logs **specifically
in the per-account resolve/mount path** and **affirmatively permits metadata**; its scenario is a bounded
`auth.json` diagnostic check. The doc's reading is **textually accurate**. Proposal:69 ("logs redigidos")
names the mechanism, not an absolute guarantee. So the **normative acceptance source does not demand
whole-codebase absolute proof** — correct.
- **Caveat the doc handles adequately:** the *task line* "nenhum segredo aparece em logs" is, in isolation,
  literally absolute; the doc rightly notes a strict reader would need Option C.

### Q2 — Is bounded-risk closure WITHOUT rescope text-faithful? → **PARTIAL / overstated**
- **Against the scoped SHALL: yes** — metadata-only credential resolve/mount logging + the `sanitizeForLog`
  mechanism satisfy the SHALL. The doc's mechanism reliance is now **supported**: the redaction-core slice was
  independently cross-family accepted (GLM52-auth-QA, ledger:500, `EV-CREDISO-5.4-REDACT-CORE`).
- **But NOT text-faithful to the operative whole-task bar / current authoritative position.** The interpretation
  presents Option A ("close on normative scope") as text-faithful closure, yet:
  1. **Current Kiro TL authoritative correction (ledger:465/467) REJECTS bounded/risk closure** of the whole
     task and keeps 5.4 **OPEN**, on grounds the doc does not fully reflect.
  2. **Concrete CLI slog bypass (authoritative):** `cmd/multica/main.go` does **not** call `logger.Init`, and
     `cmd/multica/cmd_id_resolver.go:42` logs via a **package-level `slog.Warn` on Go's default handler —
     unprotected by `SanitizeSlogAttr`**. The doc downgrades this to soft "central-hook contingency … in
     `pkg/agent`," **mislocating and understating** an authoritatively-confirmed gap that falsifies "all sinks
     route through a redacting logger."
  3. TL also enumerated **omitted sinks** (agent `text` @`daemon.go:4535`, generic `"error", err`, **5** git
     sinks incl. `gc.go`, update sinks) beyond the doc's residual list.
  ⇒ "bounded closure **without** rescope" is faithful **only** if the owner **explicitly adopts the
  narrow-SHALL scope as the acceptance criterion** — which is itself a scoping decision (≈ Option C-lite),
  not a neutral text reading. Presenting it as already-text-faithful whole-task closure is **overstated**.

### Q3 — Owner options / consequences → **CONFIRMED sound**
A (close on normative scope), B (hold for absolute — correctly identified **infeasible/non-terminating**),
C (formal rescope) are accurate with sound consequences. Refinement: **Option A is not consequence-free** —
it requires the owner to override the current whole-task-OPEN adjudication and to carry the confirmed CLI
bypass + omitted sinks as tracked items; thus A is closer to a **light rescope** than to neutral closure.

## Grade: **PARTIAL**
- **PASS elements:** the pure textual reading of the **normative SHALL** (scoped, content-vs-metadata) is
  accurate; "absolute whole-codebase proof is not what the SHALL demands" is correct; B correctly ruled
  infeasible; the doc appropriately declines to choose and defers to owner/TL; mechanism reliance is now
  cross-family supported (ledger:500).
- **Deficiencies preventing PASS:**
  1. **Understates/mislocates the authoritatively-confirmed CLI slog bypass** (`cmd/multica` unwired +
     `cmd_id_resolver.go:42` default handler) — treated as soft contingency, not the concrete gap TL recorded.
  2. **Overstates "bounded closure without rescope" as text-faithful** to the whole task; it is faithful only
     to the narrow SHALL and **conflicts with the current authoritative whole-task-OPEN position** (ledger:465/467)
     — so Option A is effectively an owner scoping decision, not neutral closure.
  3. **Residual list incomplete** vs the authoritative enumeration (agent `text`, generic `error`, 5 git sinks,
     update sinks).
- **Not REJECT:** the core interpretation of the SHALL is correct and useful; the flaws are omission/framing,
  not a wrong reading of the normative text.

## What would elevate to PASS (advisory)
1. Incorporate the **CLI-slog-bypass** as a concrete gap (cite `cmd/multica` + `cmd_id_resolver.go:42`), not a
   soft contingency.
2. Reframe **Option A** as "owner **adopts the narrow-SHALL scope** as the acceptance criterion" (a scoping
   decision ≈ light rescope), explicitly reconciled against the current whole-task-OPEN adjudication — rather
   than "bounded closure is already text-faithful."
3. Align the residual ledger with the authoritative sink enumeration (ledger:467).

## Disposition (reviewer)
The spec-language interpretation is **PARTIAL**: text-accurate on the scoped SHALL and on the infeasibility of
an absolute proof, but it **overstates bounded-closure-without-rescope** and **understates the authoritatively
confirmed CLI slog bypass**, so it is not, as written, a sufficient basis to close 5.4 without either an
explicit owner narrow-SHALL scoping (≈ Option C) or clearing the TL gates. Whole-task 5.4 stays OPEN. No
rescope, no choice, no edits made. Kiro TL adjudicates; reviewer ≠ adjudicator.

## CHECK-OUT 2026-07-18T22:36:00Z — DONE
Only this file created. Reviewed doc + spec/tasks/proposal + ledger unchanged; no git/credentials/env/network/
DB/services. Adjudication pending Kiro TL.

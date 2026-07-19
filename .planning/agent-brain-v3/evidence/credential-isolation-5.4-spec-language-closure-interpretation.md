# agent-credential-isolation 5.4 — spec-language closure interpretation (advisory)

- Author: Kiro/Opus-4.8, pane **w7:p2**. **Advisory, read-only interpretation.** No spec/tasks edit; no
  source/test/shared-planning/git/index/ref/credential/env/network/DB/service mutation. Does **not** choose for the
  owner. Only this file created.
- Base: HEAD `b6571299b00c8e388abefe7ef9dcbcf8ac715d7f`.

## Check-IN / Check-OUT
- **Check-IN** 2026-07-18T23:02:00Z — read the exact OpenSpec 5.4 requirement + task, traced against current evidence.
- **Check-OUT** 2026-07-18T23:16:00Z — DONE. Options + recommendation below. Kiro TL adjudicates.

## Provenance (SHA-256, current bytes)
| File | SHA-256 |
|---|---|
| `openspec/changes/agent-credential-isolation/tasks.md` | `3bdbc1e1916629d87c14db544aee88050daf0e0aaa2c07c3c933798678e7a0e3` |
| `…/specs/agent-credential-isolation/spec.md` | `02b0a5c131c0fd2cee2732dbf4116155d29e87c55d37cc7391994cfdf20da63b` |
| `…/proposal.md` | `a15b62c1d77c899c61b4f5fa39bf975ac4318cc99941f6eab10fbdca8d618636` |

## The exact 5.4 text

**Normative requirement (spec.md, "Não vazamento de segredo"):**
> "O sistema SHALL NOT registrar em log o conteúdo de credenciais **ao resolver ou montar as pastas por conta**;
> apenas metadados (caminho, tipo de arquivo) podem ser logados."
> Scenario "Log de diagnóstico sem segredo": **WHEN** o daemon registra o estado do `auth.json` montado **THEN** o
> log contém caminho/tipo/mtime, **nunca** o conteúdo do token.

**Task (tasks.md, §5 Verificação):**
> "- [ ] 5.4 Confirmar que nenhum segredo aparece em logs (sanitizeForLog)."

## Textual analysis — what is actually demanded

Two texts, two scopes:
- The **normative SHALL** is **scope-limited and metadata-based**: it forbids logging credential **content**
  **specifically in the per-account resolve/mount path**, and *affirmatively permits* metadata (path/type/mtime).
  Its acceptance scenario is a **bounded** check on the mounted-`auth.json` diagnostic. This is not a whole-codebase,
  all-inputs absolute proof; it is a **content-vs-metadata** rule on a specific surface.
- The **task line** is broader in phrasing ("**nenhum segredo aparece em logs**") and names the mechanism
  ("sanitizeForLog"). It sits under **"Verificação"** and uses "**Confirmar**" — a confirmation activity keyed to the
  named mechanism, not a self-declared proof obligation of universal secret-absence.

Under OpenSpec convention the **spec requirement is the acceptance source of truth**; the task is the work item.
Read together, the text most faithfully demands **(b) reasonable bounded confirmation** — the `sanitizeForLog`/
`pkg/redact` mechanism is in force and the credential resolve/mount surfaces log metadata, not content — **not (a)
absolute whole-codebase no-secret proof**. Absolute proof is neither stated by the SHALL nor generally decidable
(unbounded input space + fixed-pattern matcher).

## Comparison to current evidence

| Evidence class | State | Maps to which reading |
|---|---|---|
| `pkg/redact` core mechanism (`sanitizeForLog`/`SanitizeSlogAttr`/`Text`) | accepted slice (EV-CREDISO-5.4-CORE); central hook wired via `logger.Init` | satisfies the **named mechanism** in the task |
| Credential resolve/mount surfaces (StableSecret `[REDACTED]`, agent_env key-only audit, prodex `redactCommandOutput`, metadata-only) | metadata-not-content by construction | **directly satisfies the normative SHALL** (bounded) |
| Accepted body/stderr slices: email (EV-…-EMAIL), cloud-PAT body (EV-…-CLOUDPAT), Claude stderr logWriter (technical candidate; cross-family review pending) | independently reviewed | supports (b); not required by the SHALL's literal scope |
| Pattern-only residual (`Text()` fixed patterns; novel token shapes) | documented residual R-5.4-B | breaks (a); acceptable under (b) as a documented limitation |
| 13 raw argv sinks (census) | raw at call-site; pattern-only at runtime via central hook | breaks (a); under (b) they are pattern-covered + outside the SHALL's resolve/mount scope |
| slog **message-string** gap (`ReplaceAttr` covers attrs, not the message) + `fmt.Print*` (CLI) | structural limitation; no credential-path leak found | breaks (a); non-blocking under (b) |
| Central-hook uncertainty (raw sinks protected only if injected logger carries `SanitizeSlogAttr`) | runtime-contingent, not statically guaranteed in `pkg/agent` | a real (b) residual to record; not a SHALL violation on the resolve/mount path |

**Summary:** the **normative SHALL is substantially met** (credential resolve/mount logs are metadata-only; secret
carriers redact by construction). The gaps (pattern-only, 13 raw argv, message-string, central-hook contingency)
are all **outside the SHALL's literal scope** and only matter under the **absolute** reading of the task phrase.

## Owner options (with consequences) — owner decides; this does not choose

- **Option A — Close 5.4 on the normative scope (bounded confirmation).** Accept that the spec SHALL (scoped,
  metadata-only) is met by mechanism + credential-surface coverage + accepted slices; record the four residual
  classes as explicit **non-blocking** follow-ups.
  - *Consequence:* 5.4 closes text-faithfully to the requirement; residuals tracked separately. *Risk:* the broad
    task phrase "nenhum segredo" could later be argued unmet for non-credential secrets in raw sinks.
- **Option B — Hold 5.4 open for absolute whole-codebase no-secret proof.**
  - *Consequence:* effectively **non-terminating** — the fixed-pattern matcher + 13 raw argv sinks + message-string
    gap + unbounded input space make an absolute proof infeasible; blocks the change indefinitely. **Not textually
    required** by the SHALL.
- **Option C — Formal documented rescope.** Owner amends the **task** text (not me) to explicitly bound 5.4 to the
  normative requirement's scope + named mechanism, moving argv/body/stderr/message-string hardening to a follow-up
  change with its own acceptance.
  - *Consequence:* honest and unblocking; makes the task phrase match the SHALL. *Cost:* requires an
    owner-authorized `tasks.md`/spec edit (out of this lane) and a new change for the deferred hardening.

## Recommended text-faithful path (recommendation only)

**Option A**, with a short residual ledger. Rationale: the **normative SHALL is scoped to credential resolve/mount +
metadata-only**, and the current mechanism + credential-surface coverage + independently accepted slices satisfy it;
the task line is a **verification** item keyed to the `sanitizeForLog` mechanism, best discharged by
mechanism-confirmation + documented residuals rather than an infeasible absolute proof. If the owner reads the task
phrase as **strictly absolute**, the honest alternative is **Option C** (formal rescope) — **not** Option B, which is
effectively unbounded. In all cases the four residual classes (pattern-only, 13 raw argv, message-string, central-hook
contingency) should be recorded as tracked follow-ups regardless of which option is chosen.

## Limitations / non-claims
- Interpretation of existing text only; **no spec/tasks edit, no checkbox, no rescope performed** — those are owner
  actions. I do not choose among A/B/C. Read-only; no source/test/shared-planning/git/index/ref/credential/env/
  network/DB/service access. Hashes verified at authoring against HEAD `b6571299`. Kiro TL adjudicates.

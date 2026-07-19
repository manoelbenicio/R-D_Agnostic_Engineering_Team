# Credential-isolation 5.4 — structural vs pattern log-safety acceptance contract (ADVISORY)

Source-grounded acceptance tiers for the absolute task wording — `tasks.md:34` "Confirmar que **nenhum
segredo aparece em logs** (sanitizeForLog)" — mapping each accepted/pending slice to a tier and stating what
evidence proves **whole-task (absolute) closure** vs **bounded risk closure** vs **formal rescope**.
**Advisory only; owner/Kiro decides; no acceptance.**

- Author: **Kiro/Opus-4.8, session `w8:p2`**. HEAD `b6571299`.
- Anchor source: `pkg/redact/redact.go` (`f409ba8a…f68a5c`): `Text`@90, `SanitizeSlogAttr`@117,
  `IsSensitiveKey`@147, `SanitizeForLog`@169. Logger wiring: `internal/logger/logger.go:36,50` +
  `slog.SetDefault` (`cmd/server/main.go:123`).

## CHECK-IN 2026-07-18T22:29:00Z
Mode: READ-ONLY design. Sole deliverable = this file. Excluded (honored): no implementation; no
source/test/spec/tasks/shared-planning/git/index/ref edit; no credentials/env values; no DB/network/services.

## Acceptance tiers (source-grounded, by redaction *strength*)
| Tier | Mechanism (source) | Strength vs absolute wording | What proves it |
|---|---|---|---|
| **T1 — key-based structural** | `SanitizeSlogAttr` (redact.go:117) redacts by structured attr **key** via `IsSensitiveKey`(:147)/`hasSensitiveGroup` → `[REDACTED]`, **value-independent** | **ABSOLUTE** for any secret under a recognized key/group | key list + 25 parent redact tests; static proof the secret is under a keyed attr |
| **T2 — value-pattern** | `Text()` (redact.go:90) regex/literal patterns on string values (non-sensitive keys, `msg`, argv elements) | **PATTERN-DEPENDENT** — absolute only for known shapes (`sk-`, JWT, `Bearer`, AWS, JSON credential fields, `KEY=val` envs); **R-5.4-B** residual for unknown shape/key | pattern tests; but cannot prove coverage of *arbitrary* secret shapes |
| **T3 — argv structural projection** | `safeAgentArgvForLog` (claude WIP / codex) redacts by **flag name** regardless of value shape; unknown flags fall back to T2 | **STRUCTURAL for known flags** (> T2); T2 for unknown flags | flag-aware table tests (argv plan); currently only claude+codex |
| **T4 — external-body structural test** | site-level executed test that a specific external-response body sink redacts a synthetic token end-to-end | **TEST-PROVEN T2** (proves wiring, not new shapes) | disjoint handler test asserting sentinel absent |
| **T5 — subprocess stderr** | `logWriter.Write` explicit `redact.Text` (claude 5.4 fix) — **structural routing** guarantees the sink passes through `Text()`; masking of arbitrary content stays T2 | **STRUCTURAL ROUTING + T2 masking** | 6 logWriter tests (clean-room); covers all `logWriter` users (claude/opencode/…) |
| **T6 — bypass sinks** | sinks **not** through the redacting logger: `log.*` (0 found), `fmt.Print*` (CLI stdout, email config), non-logger writes (MCP-config env) | **must be individually adjudicated** — proven no-secret, or out-of-scope, else a gap | per-sink evidence: no-secret static proof / owner scope ruling |

## Mapping — current accepted/pending slices → tier
| Slice | Tier(s) | Status | Artifact / hash |
|---|---|---|---|
| Redact-core key-based | **T1 (absolute)** | core ACCEPT, **provenance-gated** | review `521cef31…`; audit `dbf7033b…` (unattributed producer / missing Gemini review / unpinned EV) |
| Redact-core value-pattern | **T2** | same | same |
| Bulk structured slog attrs (~703–1088) | **T1** | covered | logger wiring verified |
| Email slice (`email.go:340/367`) | **T6** (fmt.Print, config-only, no-secret) | **ACCEPTED** | `EV-CREDISO-5.4-EMAIL` `3a3018b4…` |
| Claude stderr `logWriter` | **T5** | clean-room reviewed; **pending cross-family review + push** | review `129025cc…` (delta `c7922b7b`) |
| Argv — claude, codex | **T3** | claude=WIP(mixed), codex=own projection | argv plan `55c4cb86…` |
| Argv — 13 other adapters | **T2 only** | **raw args**, unimplemented structural | argv plan `55c4cb86…` (largest residual) |
| External bodies `auth.go:656`, `cloud_pat.go:359` | **T2**, need **T4** | no site test | residual audit `5a927fbd…` (cloud_pat originally unenumerated) |
| Agent error content `daemon.go:4542` | **T2** | (persist path is structural via `ReportMessages`) | residual matrix `97fbbc24…` |
| Subprocess/CLI/git output sinks | **T2** | pattern-dep | residual matrix `97fbbc24…` |
| opencode MCP config (env transport) | **T6** (no log; env-transport = broader cred handling) | no log gap | trace `ec305e7a…` |
| CLI operator stdout (webhook URL etc.) | **T6** (out of `sanitizeForLog` scope) | owner scope ruling pending | residual audit `5a927fbd…` |
| Carrier types (`StableSecret`, `OpenclawGatewayPin`) | **T1-equivalent** (redact by construction) | covered | prior audits |
| Standard `log.*` | **T6** — **0 callsites** | none | residual matrix `97fbbc24…` |

## Closure conditions — three distinct outcomes (evidence required)

### (1) Whole-task ABSOLUTE closure ("nenhum segredo aparece em logs", literal)
Requires **every dynamic secret-bearing sink at T1 or T3-structural (value-independent)** — i.e.:
- argv **T3 extended to all 13 adapters** (no raw args anywhere);
- external bodies (`auth`/`cloud_pat`) either **structurally field-redacted** (redact the known body field pre-log) or their T2 residual formally eliminated;
- agent-content/subprocess sinks routed through structural (not pattern-only) redaction;
- **all T6 bypasses proven no-secret** (email ✓) or ruled out-of-scope (CLI, env-transport);
- a **static gate/lint** proving no raw dynamic value is logged without structural redaction.
**Honest assessment:** as long as any dynamic sink relies on **T2** (`Text()` pattern-matching) for arbitrary
content, a secret in an **unrecognized shape under a non-sensitive key can survive** ⇒ **true absolute
closure is NOT achievable** without eliminating T2 reliance at every dynamic sink. Evidence bar is therefore
very high (structural everywhere + lint gate).

### (2) Bounded RISK closure (achievable)
Evidence set:
1. **Redact-core provenance closed** — distinct Gemini/GLM review + pinned canonical `EV-CREDISO-5.4-CORE`
   hash + producer attribution or owner waiver (audit `dbf7033b…`).
2. **Claude stderr (T5) pushed** with cross-family review (delta `c7922b7b`, not the mixed working-tree file).
3. **Argv (T3) extended to the 13 adapters** (argv plan `55c4cb86…`) **or** owner-accept the T2 residual.
4. **T4 tests** for `auth.go:656` **and** `cloud_pat.go:359`.
5. **T6 bypasses documented**: email no-secret ✓; CLI stdout out-of-scope ruling; opencode env-transport out-of-scope.
6. **Owner-accept R-5.4-B** (T2 pattern-dependency) with bounded wording for the remaining dynamic value sinks.
Result: risk-closed on the log surface, with the residual explicitly bounded and owner-accepted.

### (3) Formal RESCOPE
Owner rewrites the acceptance criterion from absolute to the **honest bounded contract**:
> "No secret under a recognized structured key (T1); no known secret shape in value/message/argv (T2); argv
> flag values structurally redacted (T3); external-body sinks test-proven (T4); subprocess stderr routed
> through redaction (T5); every bypass sink (T6) proven no-secret or out-of-scope. **Residual: R-5.4-B
> pattern-dependency for arbitrary unknown-shape values under non-sensitive keys — owner-accepted.**"
Then the **bounded-risk evidence (2)** constitutes closure against the reworded criterion.

## Recommendation (advisory, not decided)
Absolute closure (1) is infeasible while T2 is load-bearing; recommend **(2) bounded-risk closure + (3)
formal rescope wording** — with the highest-value structural upgrade being **T3 argv across all adapters**
(closes the largest T2 surface) and **T4 external-body tests**. Owner/Kiro decides between accepting the
bounded contract vs funding the full structural-everywhere + lint-gate effort.

## Non-claims
- Advisory contract only; no acceptance; no checkbox; no code/test/spec/tasks/shared-planning/git/index/ref
  change; no credentials/env values; no DB/network/services. Tiers/mappings from static reads at the pinned
  hashes. Owner/Kiro TL decides; reviewer ≠ adjudicator.

## CHECK-OUT 2026-07-18T22:31:00Z — DONE
Only this file created. Everything else unchanged.

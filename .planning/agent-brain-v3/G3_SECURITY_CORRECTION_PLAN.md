# G3_SECURITY_CORRECTION_PLAN — Agent Brain v3

> Planning/adjudication artifact by Kiro/Opus-4.8. No code/ledger/STATE/OpenSpec/evidence-index
> edits; no agent dispatch. Findings source: independent read-only G3 security review by
> Codex#56#F (`w3:pB`, ledger `REVIEW-G3-01`) — `evidence/g3-independent-security-review.md`
> (status: **CHANGES REQUIRED**; PD-08 remained absolute; no secrets/credentials inspected).
>
> **Correction artifacts now exist on disk:**
> - `evidence/g3-security-corrections.md` (central / Codex1).
> - `evidence/g3-security-corrections-adapters.md` (adapters / Codex3).
>
> **Final disposition: ACCEPTED via `REVIEW-G3-02` (2026-07-18).** All three findings closed with
> implementation anchors and passing targeted regression on the final launch path
> (`internal/daemon` + `pkg/agent`, one-shot Go 1.26 container, synthetic only, PD-08 absolute).
> G3 = **RESTORED ACCEPTED**; the security-correction hold is **lifted**. See
> `evidence/g3-independent-security-rereview.md`.
>
> Closure summary:
> - **G3-SEC-F1 — ACCEPT:** gateway-required arg rejection in `config.go:488-503` and
>   `daemon.go:3233-3257` before admission/credential; adapter last-wins removed; tests
>   `brain_integration_test.go:275-304`.
> - **G3-SEC-F2 — ACCEPT:** custom-runtime registration/refresh suppressed and rejected before
>   credential (`daemon.go:1100-1105,1461-1464,3233-3257`); canonical accepted-CLI resolution
>   (`config.go:132-175,325-333,477-483`); tests `brain_integration_test.go:306-347`.
> - **G3-SEC-F3 — ACCEPT:** redacted basename+argv projection (`claude.go:24-135`, used at
>   `claude.go:173-175`, `codex.go:547-554`); tests prove hostile values/paths absent.
> - Codex4 G4 hold on active-path safety is lifted; G4 consolidation remains open only on the
>   narrow p8 BLOCK evidence-reconciliation (stale gateway digest, P24/P27 mapping mismatch,
>   wording normalization). Tier-20 enable (Codex1 task 9.2) stays gated.

## Constraints (binding)

- PD-01: preserve the dirty baseline; no reset/stash/revert/discard.
- PD-08: no credential/auth/secret read, copy, print, rewrite, rotation, quarantine, or mutation; synthetic/reference-only only.
- D-V3-14: development validation only. No production, cutover, Prodex removal, tier 50/100, or Multica-daemon Codex dispatch.
- Native adapters 5.6–5.8 remain fail-closed; no native-acceptance claim.
- Disjoint ownership; central hotspots remain Codex1-only.

## Three independent findings (from pB REVIEW-G3-01)

### G3-SEC-F1 — HIGH: Gateway CLI arguments override trusted routing after validation
- **Locus/owner (split):** central final-argv validation = **Codex1** (`daemon.go:3657-3663,3700-3709`); adapter last-wins argument construction = **Codex3** (`pkg/agent/codex.go:87-104,130-138`, `claude.go:578-599`; `runtimeenv/assert.go:12-20,47-59`; tests `claude_test.go:451-469`, `codex_test.go:2027-2035`).
- **Defect:** pre-launch validation covers env + generated config, but final daemon/task args are appended afterward; Codex allows last-wins provider/base-URL/env-key, Claude allows a later model override; tests currently preserve these overrides.
- **Fix direction:** in gateway-required mode reject custom/default args or enforce a strict allowlist denying routing/provider/credential/settings-home/model/resume overrides; validate final argv before launch.
- **Acceptance (AT-F1):** prove hostile Codex provider/base-URL/env-key and Claude model/settings/resume overrides fail **before credential acquisition or process creation**; trusted route/model remain final; regression covers the **final daemon launch path**, not only env/config helpers.
- **Evidence:** EV-G3-FIX-01 (central) + EV-G3-FIX-01A (adapter).

### G3-SEC-F2 — HIGH: Custom runtime executable can receive the gateway credential
- **Locus/owner:** central = **Codex1** (`daemon.go:1007-1018,1082-1165,3225-3239,3636-3640`; `brain_integration.go:171-180,237-305`; test `brain_integration_test.go:93-99`).
- **Defect:** gateway mode still registers arbitrary workspace custom-runtime executables; a task can replace the built-in Claude/Codex path before admission; admission validates declared provider/CLI kind, then launches the replacement with the trusted credential-bearing environment; the G3 smoke launched a test helper directly and did not cover daemon executable selection.
- **Fix direction:** suppress custom-runtime registration in gateway-required mode; reject custom-runtime tasks before readiness/credential access; resolve executables only from an immutable accepted-CLI registry.
- **Acceptance (AT-F2):** prove custom profiles cannot register or launch in gateway-required mode, cannot reach the credential callback; built-in Claude/Codex still launch through the accepted registry.
- **Evidence:** EV-G3-FIX-02.

### G3-SEC-F3 — MEDIUM: Final CLI argv can disclose credential values or auth paths in logs
- **Locus/owner:** adapters = **Codex3** (`pkg/agent/claude.go:35-61`, `codex.go:518-554`); ties to G3 safe-diagnostics claim (`evidence/g3-serial-integration.md:28-32`).
- **Defect:** both adapters log complete final argv; because custom/default args survive, inline auth config, endpoint values, or auth-file paths can enter daemon logs — contradicting the safe-diagnostics claim.
- **Fix direction:** log only an allowlisted command shape or flag names with values redacted in gateway-required mode; never log routing/auth values or paths.
- **Acceptance (AT-F3):** capture gateway-mode logs from synthetic hostile arguments; prove no credential value, endpoint value, auth path, or unredacted config payload appears while safe command diagnostics remain.
- **Evidence:** EV-G3-FIX-03.

## Split of ownership

- **Codex1 (central):** F1 final-argv validation in gateway-required mode; F2 custom-runtime suppression + accepted-CLI registry. Files: `daemon.go`, `brain_integration.go`, and central wiring only; preserve PD-01; no reset/revert.
- **Codex3 (adapters):** F1 adapter argument construction (no last-wins provider/base-URL/env-key/model/resume); F3 argv-logging redaction in `pkg/agent/claude.go` + `codex.go`; `runtimeenv/**` assert alignment. Submit contract-compatible input for any execenv/models.go touch (Codex1 applies).
- Codex2 not in scope. Codex4 in G4 hold (below).

## Global re-review acceptance (from pB gate, all findings)
- Each fix has focused regression tests covering the **final daemon launch path**, not only env/config helpers.
- Full applicable suite, race tests, vet, credential-isolation harness, and the synthetic G3 smoke pass **without weakening PD-08**.
- No provider-native credential/auth-file/direct-endpoint flow, unsupported adapter, or raw secret/path logging remains reachable in gateway-required mode.

## pB independent re-review gate
- After F1–F3 corrections + acceptance tests pass, **Codex#56#F (`w3:pB`) runs `REVIEW-G3-02`** (independent, read-only, findings-only, no edits).
- Must confirm each finding closed with evidence and surface **no new high/critical**.

## Codex4 G4 hold
- Codex4 holds G4 consolidation/sign-off (`EV-G4-08`, `EV-G4-CAP`) and any tier-20 rollup until G3 is restored to ACCEPTED. In-flight synthetic G4 artifacts may remain; no G4 acceptance is finalized and **no tier-20 enable (Codex1 task 9.2) proceeds** while G3 correction is open.

## Condition for restoring G3 = ACCEPTED (all required)
1. F1, F2, F3 corrected in their owning scopes; AT-F1/F2/F3 pass on the final launch path.
2. `REVIEW-G3-02` (Codex#56#F) confirms all three closed with **no new high/critical**.
3. PD-01 preserved and PD-08 honored throughout.
4. Correction evidence (EV-G3-FIX-01/01A/02/03) recorded and the G3 isolation smoke re-passed.
5. State/docs owner (Codex#56#A) records restored ACCEPTED; Codex4 G4 hold lifted.

Until all five hold: **G3 = ACCEPTED-WITH-CORRECTIONS (on hold); disposition PENDING REVIEW-G3-02.** No cutover, removal, production, or tiers.

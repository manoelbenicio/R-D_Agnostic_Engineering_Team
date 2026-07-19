# Architecture Audit — persist-prodex 2.1-2.2 vs build-omniroute-agent-brain

- author: Kiro (principal, decision support)
- date: 2026-07-18T21:01:00Z
- mode: READ-ONLY specs/design/evidence/source. No product/test/task/spec edits, no credentials/env contents, no DB/network/live/systemd, no git mutation.
- check-in: `.deploy-control/Kiro__PRODEX-VS-OMNIROUTE-RECONCILE__20260718T210000Z.md`

## Bottom line

**This is a SEQUENCING / STRATEGIC-POSTURE CONTRADICTION that requires an explicit owner decision.** It is not a clean, self-justifying transitional dependency: the omniroute plan already contains a sanctioned transitional mechanism (default-off drain flag, task 7.8) and an executed governance directive to block concurrent execution of superseded Prodex plans (task 0.5). Building *new durable persistence* that hardens Prodex as **required + restart-durable** runs opposite to the target direction, which treats Prodex as **default-off legacy scheduled for deletion**. I recommend a HOLD-pending-owner-gate posture (below). **Kiro/root adjudicate; I do not select the owner-only policy or claim implementation acceptance.**

## Inputs (SHA-256)

| File | SHA-256 |
|---|---|
| build-omniroute proposal.md | `40b7d1e8e12d1123a0214dc6dde97484b563c4abc209a95d5a2eaf32b0b10af7` |
| brain-cutover-operations spec.md | `67eb381fa3aed081daf3afdc631344d68bb57305a90db71e6ebefe62d38a1e0c` |
| persist-prodex proposal.md | `4b8222cc3f7ba7d8538bf003622300ab4c9db8a6053b465fca8c24f2326e1ac1` |
| persist-prodex design.md | `2bf341e8323467f1fc8235190c6f42711ffae392d081849a7c7127063ffb7feb` |
| REQUIREMENTS.md (Agent Brain) | `f95fbc6a1323f86c8e00707843ecc407e98288f9fb8bdc00ce3903ed259fdfdc` |
| persist-prodex-2.1-2.2-design.md (Codex56#A) | `fa560db1431dcf0da1a335c4235c3d05c6e00481c34b60e6a2dc977ada8ec1df` |

## Grounded citations

- **Supersession (target intent):** omniroute `proposal.md` — "Supersede the planned Prodex runtime integration: Prodex and its Rust sidecar are not part of the target request path"; "`persist-prodex-runtime-integration` is superseded rather than promoted into the new target design."
- **AB-REQ-01** (REQUIREMENTS.md L19): "Daemon brand-neutral (sem dependência de nomes Multica/Prodex); cold-plane only."
- **AB-REQ-36** (L74): "Safe rollback … **nunca** provider keys/dual router." Cutover spec: rollback selects prior OmniRoute config "rather than reactivating Prodex or provider keys."
- **AB-REQ-37** (L75): "Legacy removal gate: Prodex/L2/Go rotation/credential homes/aliases só após zero-use e rollback independente."
- **Omniroute task 0.5 [x] DONE:** "Assign a formal disposition to every active/historical OpenSpec change and **block concurrent execution of superseded Prodex/router plans**."
- **Omniroute task 7.7 [x] DONE:** "Disable Prodex/L2 startup … for gateway-required tasks."
- **Omniroute task 7.8 [x] DONE:** "Keep legacy behavior isolated behind an explicit **default-off** migration flag only while legacy tasks drain; prevent any task from having two router owners."
- **Omniroute tasks 10.4/10.5 [ ] pending:** delete Prodex/L2 startup/facade/profile/filesystem code after signed parity + zero-use.
- **persist-prodex proposal.md:** no supersede/sunset/omniroute reconciliation present (only intra-Prodex "legacy credential purge").
- **Runtime authority is single-select** (`health.go:177-184`): `omniroute` (AgentBrain dev + gateway-required) → else `rust_l2` (Prodex + L2) → else `native_cli`. Prodex(`rust_l2`) and OmniRoute are mutually exclusive per task/config; today Prodex/L2 is the non-dev default, OmniRoute is behind a development gate not yet cut over.

## Compatibility matrix — persist-prodex 2.1-2.2 vs Agent Brain requirements

| Requirement / task | Persist-prodex 2.1-2.2 effect | Compatibility |
|---|---|---|
| AB-REQ-01 brand-neutral / cold-plane-only | Adds Prodex-branded systemd unit, launcher, env template (ops surface) | **TENSION** — deepens Prodex-coupled deploy surface. Mitigated: kept in NEW disjoint deploy/ops files, not the neutral daemon core. |
| AB-REQ-36 rollback never reactivates Prodex / dual router | Rollback contract restarts "previous required-L2 configuration" | **CONDITIONAL** — different rollback scope (env-file/unit revision vs OmniRoute-version). Compatible only while runtime authority stays single-select and persist-prodex is subordinate to the cutover; must never keep Prodex hot after OmniRoute cutover. |
| AB-REQ-37 removal after zero-use + rollback independence | Creates NEW artifacts (unit/launcher/env) → additional legacy items to remove | **TENSION** — enlarges the future removal surface (10.4/10.5). |
| Omniroute task 0.5 block concurrent superseded Prodex execution | Executing 2.1-2.2 is executing a superseded Prodex plan | **DIRECT CONTRADICTION** (governance). |
| Omniroute task 7.8 legacy = default-off, drain-only, no dual router | 2.x hardens Prodex as `MULTICA_PRODEX_REQUIRED` fail-closed (default-on/required posture) | **POSTURE CONFLICT** (headline) — required-durable vs default-off-draining. |
| Omniroute tasks 10.4/10.5 delete Prodex/L2 after parity/zero-use | Persistence must be sunset alongside code deletion | **SEQUENCING DEPENDENCY** — persistence lane must not outlive the deletion gate. |
| AB-REQ-02 preserve neutral lifecycle | Disjoint ops files, no core change | **NEUTRAL / OK.** |

## Headline conflict

persist-prodex tasks 1.3/2.x make Prodex **required and restart-durable** (fail-closed on downgrade). The Agent Brain target (7.7/7.8, AB-REQ-37) makes Prodex **optional, default-off, drain-only, and scheduled for deletion after parity + zero-use.** Investing in durable systemd persistence for a runtime the accepted higher-level plan is actively removing is the crux the owner must resolve.

## Is it a justified transitional dependency? — Qualified NO, absent an owner waiver

- The transitional need (don't lose the current baseline mid-migration) is **already covered** by omniroute task 7.8's default-off drain flag + single-router invariant. persist-prodex's durable/required persistence is **broader** than drain-only.
- OmniRoute cutover prerequisites are **not met** yet (parity matrix task 1.4 `[ ]`, capacity/rollback ops 8.7/9.7 `[ ]`), so Prodex/L2 remains the live default today — which is the *only* fact that lends persist-prodex any transitional value.
- But governance task 0.5 (DONE) explicitly blocks concurrent execution of superseded Prodex plans, so proceeding without a waiver contradicts an accepted directive.

## Risks

1. **Sunk-cost + removal debt:** new systemd/launcher/env artifacts become additional AB-REQ-37 removal items (10.4/10.5), increasing cutover cost.
2. **Posture entrenchment:** a "required, restart-durable" Prodex can institutionalize the legacy path and slow the OmniRoute cutover it is meant to precede.
3. **Rollback ambiguity:** two rollback contracts (persist-prodex env/unit revision vs AB-REQ-36 OmniRoute-version) risk operator confusion; must be explicitly ranked so Prodex is never reactivated post-cutover.
4. **Governance breach:** executing a superseded plan without a recorded waiver violates task 0.5 and the "no concurrent master plan" traceability rule (brain-cutover "OpenSpec and GSD traceability").
5. **Brand-neutrality drift (AB-REQ-01):** more Prodex-named ops surface to later neutralize/remove.

## Recommended sequencing (for owner adjudication — not a decision)

Preferred posture: **HOLD persist-prodex 2.1-2.2 implementation pending an explicit owner disposition.** Then one of:

- **Option A — Sanctioned transitional (with sunset):** Omniroute owner declares persist-prodex the approved pre-cutover drain baseline AND records: (a) a sunset clause bound to AB-REQ-37 + tasks 10.4/10.5, (b) default-off/flag-gated posture reconciled with 7.8 (i.e., re-scope "required" to apply only to environments not yet cut over), (c) explicit subordination to AB-REQ-36 (never a second router; never hot after cutover), (d) the new ops artifacts added to `REMOVAL_REGISTER` with owner/deadline.
- **Option B — Defer/decline new persistence:** Rely solely on omniroute 7.8's default-off drain flag; keep Prodex/L2 runnable without new durable systemd investment. Lowest removal debt; honors task 0.5.
- **Option C — Minimal transitional:** If restart-continuity is truly needed during drain, implement the smallest reversible mechanism (documented manual/foreground start) rather than a full service/launcher that becomes a new legacy artifact.

Design-quality note: the Codex56#A design itself is sound (separately ACCEPTed as design-readiness). This audit does not dispute its internals; it questions **whether/when it should be built** given the superseding plan.

## Owner gates (owner-only; I do not decide)

- **G-1 (Product/Planning owner):** waiver-or-hold decision on executing a superseded Prodex plan (task 0.5); if waived, record disposition in OpenSpec + GSD traceability and `REMOVAL_REGISTER`.
- **G-2 (Omniroute cutover owner, Codex1/3/4):** confirm posture reconciliation (required vs default-off) and sunset binding to AB-REQ-37 / tasks 10.4-10.5.
- **G-3 (Security/rollback owner):** rank persist-prodex rollback under AB-REQ-36; guarantee single-router invariant and no post-cutover Prodex reactivation.
- **G-4 (Kiro/root TL):** adjudicate A/B/C and gate any OpenSpec checkbox change.

## Explicit non-claims

- I changed no product/spec/task/planning/git state; I created only this artifact and my check-in.
- I read no credential or environment-file contents; invoked no DB/network/live/systemd.
- I did **not** select the owner-only policy (A/B/C) and made **no** implementation-acceptance or checkbox change; this is decision support only.
- The Codex56#A design's *technical* readiness (prior ACCEPT) is unchanged; this audit addresses program sequencing, not the design's internals.
- Findings are static as of the SHAs above in an actively edited multi-agent tree; re-verify before any owner action.

# Tasks

## OpenSpec reconciliation — 2026-07-30

- [x] 0.1 Reconcile this change against OpenSpec base
  `89a236e3adda784492771a6ca1c60dae1eb823bf`.
- [x] 0.2 Record candidate source/test evidence at
  `63ead4df72ff1b43c00150d99f4f341ff7d7d39f` separately from live production.
- [x] 0.3 Record live production at `15626386da2725af8e8d4ac611754cffe359fe31`
  with Cline explicitly disabled by
  `MULTICA_CLINE_PATH=/run/multica-disabled/cline`.
- [x] 0.4 Remove native NIM implementation, acceptance, deployment, smoke, and token-usage
  claims from this change.

## Superseded native NIM work — not implemented or accepted

- [x] 1.1 **SUPERSEDED 2026-07-30:** native NIM backend and tests. Native NIM is absent from
  accepted current source; NVIDIA access is OmniRoute-owned.
- [x] 1.2 **SUPERSEDED 2026-07-30:** NIM credential isolation and rotation.
- [x] 1.3 **SUPERSEDED 2026-07-30:** NIM probe, factory, `SupportedTypes`, and shared wiring.
- [x] 1.4 **SUPERSEDED 2026-07-30:** NIM rebuild, deploy, online verification, smoke, and token
  usage. These checkmarks close obsolete planning items; they do not denote implementation,
  deployment, or acceptance.

Any native NIM revival requires a separate owner-approved proposal with transport/runtime
evidence.

## Verified candidate evidence — not deployment or acceptance

- [x] 2.1 Verify that candidate `63ead4df72ff1b43c00150d99f4f341ff7d7d39f`
  contains credentialless Agent Brain Cline source, factory/config wiring, task-home isolation,
  and focused fail-closed tests.
- [x] 2.2 Verify that the same candidate contains `POST /auth/login` and frontend login
  client/UI source.
- [ ] 2.3 Production onboarding UAT and acceptance. Candidate source is insufficient evidence;
  this remains unverified.

## Owner decision and conditional Cline rollout

- [ ] 3.1 **BLOCKED — owner decision:** select (A) credentialless Agent Brain / OmniRoute-only
  Cline or (B) native credential-isolated Cline account.
- [ ] 3.2 **BLOCKED BY 3.1:** for option A, verify ORQ-44 inference-key lifecycle/readiness and
  an authorized exact model route; for option B, establish account ownership and login
  authority.
- [ ] 3.3 Reconcile the selected option with candidate source and obtain review for the daemon
  configuration that would enable Cline.
- [ ] 3.4 Run targeted and containerized integration tests for the selected architecture.
- [ ] 3.5 Deploy through the canonical wrapper only after the authorized rollout wave.
- [ ] 3.6 Run one bounded production Cline canary, capture token usage, and perform onboarding
  UAT.
- [ ] 3.7 Record the exact live revision, enabled configuration, canary result, and acceptance
  evidence before claiming Cline online or onboarding accepted.

Until 3.1 and 3.2 are resolved, production remains at
`15626386da2725af8e8d4ac611754cffe359fe31` with Cline disabled. This documentation-only
reconciliation performs none of tasks 3.3–3.7 and creates no `.planning` or `.deploy-control`
artifacts.

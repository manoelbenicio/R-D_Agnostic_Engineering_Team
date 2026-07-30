# Proposal — Cline integration and onboarding evidence reconciliation

## Why

The previous proposal mixed planned work with accepted and deployed behavior. This
reconciliation records the verified state as of 2026-07-30 without implementing or deploying
a runtime.

Native NVIDIA NIM is absent from accepted current source and NVIDIA model access is owned by
OmniRoute. Therefore this change no longer proposes or claims a native `nim` runtime. Any
future native NIM work requires a separate owner-approved proposal backed by transport and
runtime evidence.

Cline source exists, but its production architecture and rollout are not accepted yet. The
owner must choose between:

- **A — credentialless Agent Brain / OmniRoute-only Cline:** requires the ORQ-44 inference-key
  lifecycle and readiness plus an authorized exact model route.
- **B — native credential-isolated Cline account:** requires established account and login
  authority.

The proposal SHALL NOT infer this choice from the existing candidate implementation.

## What Changes

- **REMOVED FROM SCOPE** all native NIM implementation, wiring, deployment, smoke, token-usage,
  and acceptance claims.
- **DOCUMENTED** the existing Cline candidate source and tests separately from production.
- **BLOCKED** Cline enablement, deployment, smoke, and acceptance until the owner resolves
  option A versus option B and the selected path's prerequisites are satisfied.
- **RECONCILED** onboarding/auth statements as candidate-source evidence only; this change
  does not claim production deployment or UAT acceptance.

## Evidence baseline

| State | Exact revision | Verified statement |
| --- | --- | --- |
| OpenSpec base | `89a236e3adda784492771a6ca1c60dae1eb823bf` | Documentation baseline for this reconciliation. |
| Source/test candidate | `63ead4df72ff1b43c00150d99f4f341ff7d7d39f` | Contains the credentialless Agent Brain Cline source, factory/config wiring, task-home isolation, fail-closed tests, `POST /auth/login`, and frontend login client/UI. |
| Live production | `edd7b932` / image `f9e6b777` | Verified production revision and image; Cline remains explicitly disabled. |
| Blocker | Owner decision pending | Select A (OmniRoute-only) or B (native credential-isolated account) before integration or rollout. |

Candidate source and passing source tests are not deployment, production readiness, smoke,
UAT, or acceptance evidence.

The verified deployment had `RestartCount=0` and successful health/readiness checks, but it
does not resolve the Cline architecture choice. ORQ-42 remains offline-tooling evidence only,
and ORQ-53 remains blocked on the owner choice between A and B.

## Impact

This reconciliation changes only the OpenSpec documents under
`openspec/changes/native-runtimes-onboarding/`. It makes no code, configuration,
`.planning`, `.deploy-control`, runtime, or deployment change.

## Non-goals

- Reintroducing native NIM.
- Selecting Cline architecture option A or B on the owner's behalf.
- Enabling `MULTICA_CLINE_PATH`, rebuilding, restarting, deploying, or running production
  smoke/UAT in this documentation change.
- Claiming candidate auth/onboarding work is live or accepted.

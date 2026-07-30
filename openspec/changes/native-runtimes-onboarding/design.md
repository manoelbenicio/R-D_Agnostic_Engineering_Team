# Design — Evidence-gated Cline integration

## Purpose

This document describes the decision and evidence gates for Cline. It does not authorize an
implementation or deployment. Native NIM is intentionally excluded because it is absent from
accepted current source and NVIDIA model access belongs to OmniRoute.

## Verified evidence layers

Evidence MUST retain its layer and exact revision:

1. **Source:** candidate revision `63ead4df72ff1b43c00150d99f4f341ff7d7d39f`
   contains credentialless Agent Brain Cline source, config/factory wiring, task-home isolation,
   auth/login source, and frontend login source.
2. **Test:** the candidate includes focused wiring and fail-closed coverage. This demonstrates
   source behavior only; it is not a production smoke or UAT result.
3. **Candidate:** the same revision is an integration candidate, not the live production
   revision and not evidence of acceptance.
4. **Live:** production remains at `15626386da2725af8e8d4ac611754cffe359fe31`,
   where `MULTICA_CLINE_PATH=/run/multica-disabled/cline` explicitly prevents Cline enablement.
5. **Blocker:** the owner has not selected Cline architecture option A or B.

No lower evidence layer may be reported as proof of a higher layer.

## Owner decision gate

### Option A — credentialless Agent Brain / OmniRoute-only Cline

Cline remains a credentialless carrier and model inference is routed only through OmniRoute.
This path cannot proceed until ORQ-44 supplies an operational inference-key lifecycle,
readiness evidence, and an authorized exact model route.

### Option B — native credential-isolated Cline account

Cline uses a native account in an isolated task home. This path cannot proceed until account
ownership and login authority are established, with credential isolation and fail-closed
behavior reviewed for that account model.

Existing candidate code SHALL NOT be treated as the owner's selection. Production Cline SHALL
remain disabled while the choice or its prerequisites are unresolved.

## Conditional integration sequence

Only after an explicit owner decision and prerequisite evidence may a later implementation
change:

1. reconcile the selected option against the candidate source;
2. review the daemon configuration change that removes the disabled Cline path;
3. run targeted source tests and containerized integration tests;
4. deploy through the canonical wrapper after the authorized rollout wave;
5. run one bounded Cline production canary and onboarding UAT;
6. record live revision, configuration, smoke, token-usage, and acceptance evidence.

Each step requires its own evidence. A build, test, candidate commit, or installed Cline CLI is
insufficient to claim that the runtime is online.

## Transport constraint for native Cline

If the selected architecture uses the existing native ACP backend, it launches `cline --acp`
and exchanges ACP JSON-RPC 2.0 over stdin/stdout. It must not combine `--json` with `--acp`:
`--json` selects the separate headless prompt-output mode and prevents the ACP handshake.
This constraint describes candidate source behavior and does not imply rollout approval.

## NIM disposition

All prior native NIM design, isolation, wiring, deployment, smoke, and token-usage items are
superseded as of 2026-07-30, not completed or accepted. A native NIM revival requires a
separate owner-approved OpenSpec proposal with transport/runtime evidence; it cannot be added
implicitly to the Cline decision.

## Documentation-change boundary

This reconciliation edits OpenSpec only. It creates no `.planning` or `.deploy-control`
artifacts and makes no runtime, configuration, build, restart, deployment, smoke, or UAT
change.

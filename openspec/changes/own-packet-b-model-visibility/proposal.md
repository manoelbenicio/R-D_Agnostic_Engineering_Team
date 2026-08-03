# Proposal — Own Packet B Vendor/Model Visibility

## Why

The frozen Task 16 candidate already contains the bounded Packet B vendor/model
visibility behavior, but the change has no accepted OpenSpec package. Historical
staging evidence also assigned all of `packages/core/api/client.ts` to two lanes:
Packet B model-discovery cancellation and native-onboarding login. The actual
Packet B portion is only the optional `AbortSignal` parameter and passthrough in
the two list-model methods; the login contract is separate and unchanged.

Without an explicit task and file/hunk boundary, technically valid Packet B code
cannot receive traceable governance acceptance.

## What Changes

- Add an OpenSpec capability for cached vendor/model visibility and cancellable
  model discovery.
- Assign Packet B ownership in `client.ts` only to:
  - `ApiClient.initiateListModels(runtimeId, signal?)`; and
  - `ApiClient.getListModelsResult(runtimeId, requestId, signal?)`.
- Add a focused API-client test proving those two methods forward the same
  `AbortSignal` without changing routes, payloads, or response behavior.
- Keep auth/login, runtime-manager, agent-type, backend, deployment, and unrelated
  UI concerns outside this change.

## Impact

- New ownership/provenance documentation and one focused test only.
- No edit to the shared `client.ts` baseline is required.
- No credential, live provider, network, daemon, database, production, publication,
  or default-branch effect.

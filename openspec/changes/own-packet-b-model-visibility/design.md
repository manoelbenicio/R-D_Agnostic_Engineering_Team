# Design — Packet B Ownership and Model Visibility

## Baseline

The exact frozen baseline is commit
`a5aa53e8e89d2845cacfbc82ca851fbd18f9a505`, tree
`b3d3f484fa7204308d085c17a99db8fe7b586f6e`.

The Packet B baseline paths and blobs are:

| Path | Baseline blob | Ownership |
| --- | --- | --- |
| `multica-auth-work/packages/core/api/client.ts` | `6afd30013ef606d2a8cad1934dbd3cdf115ab77d` | Packet B owns only the two list-model AbortSignal method portions described below. |
| `multica-auth-work/packages/core/runtimes/models.ts` | `33e57e9e8a0bf96145147d8d3bd5b97675f19cb4` | Packet B model-catalog query, cache, cancellation, fallback, and lifecycle behavior. |
| `multica-auth-work/packages/core/runtimes/models.test.tsx` | `56d4485fbc04a62014c9b79aeee6cd932a6c8996` | Packet B core behavior tests. |
| `multica-auth-work/packages/views/agents/components/model-dropdown.tsx` | `8ccdeb18cedcb4f1755da2560cf79b0991911f47` | Packet B create-flow model visibility. |
| `multica-auth-work/packages/views/agents/components/model-dropdown.test.tsx` | `1acfa8e85008392e64fde301d51980ef0d069a7b` | Packet B create-flow tests. |
| `multica-auth-work/packages/views/agents/components/inspector/model-picker.tsx` | `ae23dad9f37b0080accba32d1d95dafbc1f6b212` | Packet B inspector model visibility. |
| `multica-auth-work/packages/views/agents/components/inspector/model-picker.test.tsx` | `4d2040127af2fb33232699c567895c0f2818331d` | Packet B inspector tests. |
| `multica-auth-work/packages/views/agents/components/runtime-picker.tsx` | `2b2a6674f88eb9eab38e3acc9705068ea6333092` | Previously bounded runtime/provider identity used by Packet B; no new behavior is claimed here. |

The current common repository has the same blob for every listed path. No dirty
delta exists on these paths there.

## Exact `client.ts` disentanglement

Packet B owns only two method-level concerns:

1. `ApiClient.initiateListModels` accepts an optional `AbortSignal` and passes it
   unchanged to the POST request.
2. `ApiClient.getListModelsResult` accepts an optional `AbortSignal` and passes it
   unchanged to the polling GET request.

Packet B does not own `ApiClient.login`, `LoginResponse`, login parsing, auth
headers, unauthorized handling, runtime-manager error handling, other API methods,
or the rest of the monolithic file. This is an ownership/hunk boundary, not a new
API abstraction or a reason to refactor the client.

A live SPE-6 Runtime Manager worktree has an unstaged `client.ts` delta, but its
observed hunks are limited to runtime-manager error classification near
`isRuntimeManagerApiPath`, `runtimeManagerErrorMessage`, and `fetchRaw`. It does
not touch either Packet B list-model method. ORQ-99 does not edit `client.ts`, so
the file-level overlap produces no overlapping write ownership.

## Behavioral boundary

Packet B preserves:

- one runtime-specific query key while offline;
- no model discovery request while the query is disabled offline;
- cancellation propagation through initiation, polling delay, and polling request;
- explicit-provider precedence with conservative runtime-provider fallback;
- consistent provider grouping and provider-name search in create and inspector
  pickers; and
- bounded retained catalog/identity state within one QueryClient session.

Packet B does not claim live provider completeness, live model availability,
deployment, UAT, production acceptance, runtime rollout, or backend discovery
changes.

## Commit allowlist

The ORQ-99 commit may change exactly:

- `multica-auth-work/packages/core/api/client.packet-b-models.test.ts`;
- `openspec/changes/own-packet-b-model-visibility/proposal.md`;
- `openspec/changes/own-packet-b-model-visibility/design.md`;
- `openspec/changes/own-packet-b-model-visibility/tasks.md`; and
- `openspec/changes/own-packet-b-model-visibility/specs/vendor-model-visibility/spec.md`.

All baseline implementation files remain unchanged.

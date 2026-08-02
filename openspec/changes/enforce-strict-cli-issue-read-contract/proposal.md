# Change: Enforce the strict CLI issue-read contract

## Why

CLI issue reads cross a workspace authorization boundary. Accepting an implicit workspace,
a non-200 response, a redirect, malformed JSON, or a mismatched issue identity can turn a
read into an ambiguous or cross-workspace result. The server must also avoid revealing
whether an issue exists outside the requested workspace.

The implementation and tests already exist and were verified green by the canonical
`ORQ-80 / REC-CLI-AUTH-01` audit. This change closes only the missing OpenSpec
documentation and traceability gap; it does not change or revalidate product code.

## What Changes

- Require an explicit workspace context for strict CLI issue reads.
- Accept exactly HTTP `200 OK` and reject redirects without following them.
- Require exactly one valid JSON response value.
- Validate the complete issue identity and its equality with the resolved reference.
- Preserve a non-leaking `404 issue not found` response across missing and
  cross-workspace reads.

## Provenance

Candidate evidence boundary: `a5aa53e8e89d2845cacfbc82ca851fbd18f9a505`.

- `c536b609dc882a211ebc8362036cb29044811719` — strict workspace-scoped issue
  resolution, exact-status reads, identity checks, and contract tests.
- `0ecc6f4e839e024d0cc81cb5c5c33ae0ec4dabd2` — redirect rejection for strict
  issue reads and its tests.

Both commits are ancestors of the candidate and of this documentation branch.

The bounded implementation/test evidence surface is:

- `multica-auth-work/server/cmd/multica/cmd_issue.go`
- `multica-auth-work/server/cmd/multica/cmd_id_resolver.go`
- `multica-auth-work/server/cmd/multica/cmd_issue_get_contract_test.go`
- `multica-auth-work/server/internal/cli/client.go`
- `multica-auth-work/server/internal/cli/client_get_expected_status_test.go`
- `multica-auth-work/server/internal/handler/issue.go`
- `multica-auth-work/server/internal/handler/handler.go`
- `multica-auth-work/server/internal/handler/issue_get_workspace_scope_test.go`

## Non-goals

- No product or test code change.
- No test rerun or replacement of the canonical audit evidence.
- No credential, runtime, production, deployment, remote-ref, or Kanban operation.
- No behavior beyond `REQ-01` through `REQ-06`.

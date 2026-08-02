# Design: Strict CLI issue-read contract

## Evidence boundary

This design records behavior already implemented by provenance commits
`c536b609dc882a211ebc8362036cb29044811719` and
`0ecc6f4e839e024d0cc81cb5c5c33ae0ec4dabd2`. The canonical ORQ-80 audit is the
test-result authority. This documentation remediation does not rerun or broaden those
tests.

## Request boundary

`issue get` resolves an issue only with an explicit workspace identifier. The client sends
that scope as `X-Workspace-ID`, and the server resolves the issue through the authenticated
user's membership in the same workspace. Missing or invalid workspace context fails before
an issue read can be treated as successful.

## Transport boundary

`APIClient.GetJSONExpectedStatus` is the strict read primitive. It requires the caller's
exact expected status (`200 OK` here), returns an unexpected-status error for every other
status, and does not decode an error response as an issue. Its HTTP client disables redirect
following with `http.ErrUseLastResponse`, making every 3xx response visible to the exact
status check.

## Decode boundary

The response decoder accepts one JSON value only. Malformed JSON, trailing non-whitespace,
or a second JSON value is an error. Strict reads therefore cannot accept a valid prefix while
silently ignoring extra response bytes.

## Identity boundary

`requireIssueIdentity` validates that the returned issue has a valid UUID `id`, a non-empty
`identifier`, and a positive int32-compatible `number`. The fetched issue ID must equal the
ID selected by strict reference resolution. A structurally valid but different issue is not
accepted.

## Non-disclosure boundary

Server lookup remains workspace-scoped and user-authorized. A missing issue and an issue
outside the requested workspace produce the same `404 issue not found` response, without
cross-workspace metadata in headers or body.

## Failure model

All boundary failures are terminal for the strict read. The implementation must not follow
redirects, retry through an alternate workspace, decode non-200 bodies as issues, accept a
partial JSON document, or substitute a different resolved identity.

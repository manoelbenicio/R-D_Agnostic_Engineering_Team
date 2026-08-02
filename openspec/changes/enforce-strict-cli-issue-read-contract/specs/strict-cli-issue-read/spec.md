# Spec: Strict CLI issue read

## ADDED Requirements

### Requirement: REQ-01 Explicit workspace scope is mandatory

A strict CLI issue read MUST require an explicit workspace identifier, MUST send it as
`X-Workspace-ID`, and MUST resolve the issue only within that authenticated workspace.

#### Scenario: Workspace scope is missing or invalid

- **WHEN** a strict issue read has no valid workspace identifier
- **THEN** the request MUST fail closed
- **AND** no issue response MUST be accepted

#### Scenario: Workspace scope is valid

- **WHEN** the CLI performs a strict issue read with a valid workspace identifier
- **THEN** it MUST send the identifier in `X-Workspace-ID`
- **AND** the server MUST apply the same workspace scope to issue resolution

### Requirement: REQ-02 Strict reads accept exactly HTTP 200

The CLI MUST accept a strict issue or issue-candidate response only when its HTTP status is
exactly `200 OK`.

#### Scenario: Response status is not 200

- **WHEN** a strict issue read receives any status other than `200 OK`
- **THEN** it MUST return an unexpected-status error
- **AND** it MUST NOT decode the response body as a successful issue result

### Requirement: REQ-03 Redirects are rejected

The HTTP client used by strict issue reads MUST NOT follow redirects. Every 3xx response
MUST be handled as a non-200 response.

#### Scenario: Server returns a redirect

- **WHEN** a strict issue read receives a 3xx response with a redirect location
- **THEN** the client MUST NOT request the redirect target
- **AND** the strict read MUST fail with an unexpected-status error

### Requirement: REQ-04 Exactly one valid JSON value is required

A successful strict issue read MUST contain exactly one valid JSON value followed only by
permitted whitespace.

#### Scenario: Response JSON is malformed

- **WHEN** the `200 OK` response does not contain a valid JSON value
- **THEN** the strict read MUST fail

#### Scenario: Response contains trailing data or another JSON value

- **WHEN** a valid JSON value is followed by non-whitespace or a second JSON value
- **THEN** the strict read MUST fail
- **AND** it MUST NOT accept the first value as the result

### Requirement: REQ-05 Returned issue identity is strict

The returned issue MUST have a valid UUID `id`, a non-empty `identifier`, and a positive
number representable as int32. Its `id` MUST equal the issue ID selected by strict reference
resolution.

#### Scenario: Identity fields are invalid or incomplete

- **WHEN** any required issue identity field violates its constraint
- **THEN** the strict issue read MUST fail

#### Scenario: Returned issue differs from the resolved issue

- **WHEN** the returned issue ID differs from the strictly resolved issue ID
- **THEN** the strict issue read MUST fail
- **AND** it MUST NOT substitute or display the mismatched issue

### Requirement: REQ-06 Cross-workspace misses do not leak existence

The server MUST return the same non-leaking not-found response when an issue is absent and
when it exists outside the authenticated requested workspace.

#### Scenario: Issue is absent or belongs to another workspace

- **WHEN** an authenticated workspace-scoped read cannot resolve the issue inside that
  workspace
- **THEN** the server MUST return HTTP `404`
- **AND** the response MUST use the generic `issue not found` result
- **AND** headers and body MUST NOT disclose cross-workspace issue data or existence

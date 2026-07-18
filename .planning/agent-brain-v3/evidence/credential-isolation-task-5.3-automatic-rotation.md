# Credential isolation task 5.3 — automatic rotation test evidence

Date: 2026-07-18

## Disposition

This is retrospective evidence for the deterministic offline test slice associated
with agent-credential-isolation task 5.3. It is not self-acceptance, independent
adjudication, production approval, or an OpenSpec completion claim. Task 5.3 remains
unchecked. Kiro TL will independently adjudicate the evidence and record the process
exception.

## Process transparency exception

The relevant ledger/STATE material was inspected before the test edit, but no
pre-edit ownership/check-in claim was recorded in the ledger. Therefore the required
pre-edit ledger/check-in step is missing. Any claim that the test edit followed a
recorded pre-edit check-in would be incorrect. This evidence artifact is retrospective
and does not retroactively cure that process exception. No ledger, STATE, OpenSpec, or
product/test file was edited while creating this artifact.

## Synthetic test and exact SHA-256

The disjoint task-5.3 test is:

`multica-auth-work/server/internal/daemon/credential_rotation_task53_test.go`

SHA-256:

```text
9a849e508c54353110d011737ff7a659909af604c9adc60e82384f331bf724b1  multica-auth-work/server/internal/daemon/credential_rotation_task53_test.go
```

The test uses a fixed UTC timestamp and only in-memory synthetic accounts, assignment
state, authenticator calls, event emitter, and rotation records. Its two subtests are:

- `TestCredentialIsolationTask53AutomaticRotation/exhausted_active_account_rotates_within_provider_and_tenant`
- `TestCredentialIsolationTask53AutomaticRotation/no_same-provider_same-tenant_candidate_fails_closed`

Source anchors:

- test entry and fixed time:
  `multica-auth-work/server/internal/daemon/credential_rotation_task53_test.go:13`
- successful exhausted-account scenario:
  `multica-auth-work/server/internal/daemon/credential_rotation_task53_test.go:16`
- wrong-provider, wrong-tenant, and valid replacement candidates:
  `multica-auth-work/server/internal/daemon/credential_rotation_task53_test.go:17`
- producer invocation with exact provider/tenant/account identity:
  `multica-auth-work/server/internal/daemon/credential_rotation_task53_test.go:27`
- successful reassignment, boundary, auth-sequence, event-count, and rotation-record assertions:
  `multica-auth-work/server/internal/daemon/credential_rotation_task53_test.go:35`
- no-valid-candidate scenario:
  `multica-auth-work/server/internal/daemon/credential_rotation_task53_test.go:59`
- `ErrNoAccountAvailable`, unchanged assignment, no auth calls, unchanged boundary
  decoys, and zero rotation-record assertions:
  `multica-auth-work/server/internal/daemon/credential_rotation_task53_test.go:74`

## Executed test-name proof

Working directory:
`multica-auth-work/server`.

Command:

```text
GOTOOLCHAIN=local GOPROXY=off /home/dataops-lab/go-sdk/bin/go test -v -count=1 -tags=offline ./internal/daemon -run '^TestCredentialIsolationTask53AutomaticRotation$'
```

Result, exit 0:

```text
=== RUN   TestCredentialIsolationTask53AutomaticRotation
=== RUN   TestCredentialIsolationTask53AutomaticRotation/exhausted_active_account_rotates_within_provider_and_tenant
=== RUN   TestCredentialIsolationTask53AutomaticRotation/no_same-provider_same-tenant_candidate_fails_closed
--- PASS: TestCredentialIsolationTask53AutomaticRotation (0.00s)
    --- PASS: TestCredentialIsolationTask53AutomaticRotation/exhausted_active_account_rotates_within_provider_and_tenant (0.00s)
    --- PASS: TestCredentialIsolationTask53AutomaticRotation/no_same-provider_same-tenant_candidate_fails_closed (0.00s)
PASS
ok  github.com/multica-ai/multica/server/internal/daemon  0.017s
```

This transcript proves that both named task-5.3 scenarios were genuinely executed,
not merely compiled or selected by source inspection.

## Repeated focused verification

Command:

```text
GOTOOLCHAIN=local GOPROXY=off /home/dataops-lab/go-sdk/bin/go test -tags=offline ./internal/daemon ./internal/rotation -count=20 -run '^(TestCredentialIsolationTask53AutomaticRotation|TestCredentialSessionDiscoveryProducerEndToEndReassigns|TestReassignDiscoverySessionExpiredReassignsSameProvider|TestReassignDiscoverySessionNoNextFailsClosedBeforeLogout|TestReassignDiscoverySessionRejectsAssignmentBoundary)$'
```

Result, exit 0:

```text
ok  github.com/multica-ai/multica/server/internal/daemon    0.030s
ok  github.com/multica-ai/multica/server/internal/rotation  0.016s
```

## Race verification

Command:

```text
GOTOOLCHAIN=local GOPROXY=off /home/dataops-lab/go-sdk/bin/go test -race -count=1 -tags=offline ./internal/daemon ./internal/rotation -run '^(TestCredentialIsolationTask53AutomaticRotation|TestCredentialSessionDiscoveryProducerEndToEndReassigns|TestReassignDiscoverySessionExpiredReassignsSameProvider|TestReassignDiscoverySessionNoNextFailsClosedBeforeLogout|TestReassignDiscoverySessionRejectsAssignmentBoundary)$'
```

Result, exit 0:

```text
ok  github.com/multica-ai/multica/server/internal/daemon    1.062s
ok  github.com/multica-ai/multica/server/internal/rotation  1.036s
```

## Vet verification

Command:

```text
GOTOOLCHAIN=local GOPROXY=off /home/dataops-lab/go-sdk/bin/go vet -tags=offline ./internal/daemon ./internal/rotation
```

Result: exit 0 with no diagnostics.

## Current scoped source SHA-256 manifest

The manifest was computed only from the explicit implementation and synthetic test
files below. No credential, authentication home, session file, token, environment
secret, network resource, database, or live service was read or hashed.

```text
9a849e508c54353110d011737ff7a659909af604c9adc60e82384f331bf724b1  multica-auth-work/server/internal/daemon/credential_rotation_task53_test.go
4e240c09af1653fd2ecdbcd0763f8c05d3eb3f57ee8b9c339045edfa32a9ce6c  multica-auth-work/server/internal/daemon/credential_session_discovery_producer.go
818e69d4fe04057503b776716ae0704a6b27b7fc7b2b875e8f507b9a7b99491a  multica-auth-work/server/internal/daemon/credential_session_discovery_producer_test.go
936b3e40f19c699078e994740c9ae63fdce5c172516b678d289f05c3cc38d1e2  multica-auth-work/server/internal/daemon/credential_session_monitor.go
5fd7005b4209b0c8d5c42ad9b240ad039d7568ad1f895bd6b9463b8e237613e2  multica-auth-work/server/internal/daemon/credential_session_monitor_test.go
71b89980c9b8cb7c03a81a502da5b9f4bdab4c0c522c117d2147b670e050f730  multica-auth-work/server/internal/daemon/wakeup.go
bc61a46c0c700010001a49c9075b87f97e89f802b138e63f90b53193a4b45b55  multica-auth-work/server/internal/rotation/detector_discovery.go
4e8092ff6b0f37a8416b961c89040433682e2d0d654293e192992dd3e1d55a4f  multica-auth-work/server/internal/rotation/detector_discovery_test.go
c655c9c4f94716d2c93109b1cc4d3c14e62d6fe8dea615dddec3ec93769b2832  multica-auth-work/server/internal/rotation/discovery_reassignment.go
d87aa80717e33a3f3768ebb7d698dcd9ace3bd934550317ec64c57f56a6566b2  multica-auth-work/server/internal/rotation/discovery_reassignment_test.go
0c4c453c730f5b9d49c605e2206e1501b778f32f40b20b142d86e70cdb8fb3fc  multica-auth-work/server/internal/rotation/pool.go
da401ef882af6fe06bb923494f4393b685c45dca01e9b0707127bda16a87f005  multica-auth-work/server/internal/rotation/pool_test.go
f20951a36765d2b6b8576232871f1991697de56d8577377b0226ed5e30130cf0  multica-auth-work/server/internal/rotation/service.go
989ae633e1865bc0f1f5a9f7eed5d2541100dbee94960c9e2c15f84346aa8ba1  multica-auth-work/server/internal/rotation/service_test.go
```

## Scope proved

The deterministic test drives the bounded in-process path from a synthetic
`CredentialSessionDiscoveryObservation` through the producer, discovery detector,
non-secret daemon event, daemon dispatcher, `ReassignDiscoverySession`, same-provider
and same-tenant candidate selection, synthetic authentication sequence, assignment,
and synthetic rotation record.

It proves that:

- an exhausted active account selects and assigns the valid same-provider,
  same-tenant replacement;
- lower-priority wrong-provider and wrong-tenant accounts are not selected or mutated;
- when only wrong-provider/wrong-tenant candidates exist, the path returns
  `ErrNoAccountAvailable`, preserves the current assignment, performs no vendor auth
  action, and writes no rotation record;
- the exhaustion signal remains represented on the current synthetic account in the
  no-candidate case.

## Explicit non-claims

This evidence does not prove or claim:

- concrete production observation-source construction or invocation of the producer;
- network or live vendor authentication/logout/login behavior;
- Postgres or any database-backed persistence behavior;
- transactionality between assignment and rotation-history persistence;
- multi-node CAS, distributed locking, or cross-process deduplication;
- task 4.4 operator alerting/error-reporting behavior;
- guaranteed vendor-session cleanup after a partial failure;
- full task 4.3 completion, full task 5.3 acceptance, or production readiness.

Independent adjudication remains with Kiro TL. This artifact makes no acceptance
decision and changes no OpenSpec checkbox.

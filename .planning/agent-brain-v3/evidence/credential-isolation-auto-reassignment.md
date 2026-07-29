# Credential isolation task 4.3 — bounded auto-reassignment evidence

Date: 2026-07-18

## Scope and disposition

This artifact records evidence for the currently implemented, bounded slices of
agent-credential-isolation task 4.3. It is not an acceptance, self-acceptance,
task completion, or production-readiness claim. The OpenSpec task remains
unchecked.

All verification was source-only and offline. The tests use synthetic in-memory
stores, authenticators, observations, emitters, and timestamps. No credential,
authentication home, session file, token, environment secret, network, database,
or live service was inspected or used.

## Exact covered call path

1. `CredentialSessionDiscoveryProducer.Produce` accepts the bounded non-secret
   observation fields `agent_id`, `account_id`, `provider`, `workspace_id`,
   `status`, and `expires_at`; it validates exact non-empty identity fields and
   field-size limits in
   `server/internal/daemon/credential_session_discovery_producer.go:94` and
   `server/internal/daemon/credential_session_discovery_producer.go:150`.
2. The producer constructs `rotation.DiscoverySession` and invokes
   `rotation.DetectDiscoverySession` at
   `server/internal/daemon/credential_session_discovery_producer.go:112` and
   `server/internal/daemon/credential_session_discovery_producer.go:117`.
3. The detector classifies explicit `expired`/`exhausted` status or a valid
   `expires_at` at/before the supplied time, while enforcing provider-family
   equality, in `server/internal/rotation/detector_discovery.go:29`.
4. An exhausted classification is encoded as exactly
   `daemon:credential_session_discovery` with the validated identity/status
   payload at `server/internal/daemon/credential_session_discovery_producer.go:126`
   and `server/internal/daemon/credential_session_discovery_producer.go:138`.
5. The daemon event contract and narrow reassignment interface are defined at
   `server/internal/daemon/credential_session_monitor.go:13` and
   `server/internal/daemon/credential_session_monitor.go:29`. The dispatcher
   decodes the payload and calls `ReassignDiscoverySession` at
   `server/internal/daemon/credential_session_monitor.go:48` and
   `server/internal/daemon/credential_session_monitor.go:64`. The existing
   daemon WebSocket read path recognizes the same event at
   `server/internal/daemon/wakeup.go:274`.
6. `rotation.Service.ReassignDiscoverySession` rechecks the detector, validates
   the observed account against the current assignment, enforces provider and
   tenant boundaries, marks the current account exhausted, and enters the
   locked rotation path at
   `server/internal/rotation/discovery_reassignment.go:23`,
   `server/internal/rotation/discovery_reassignment.go:54`,
   `server/internal/rotation/discovery_reassignment.go:65`, and
   `server/internal/rotation/discovery_reassignment.go:72`.
7. The service selects a replacement before logout and excludes the current
   account at `server/internal/rotation/service.go:94` and
   `server/internal/rotation/service.go:108`. `Pool.selectNextPriority` queries
   within the requested provider/tenant pool and deterministically selects an
   available account at `server/internal/rotation/pool.go:35`.
8. After synthetic logout/login/authentication, the service assigns the next
   same-provider account at `server/internal/rotation/service.go:134` and
   `server/internal/rotation/service.go:150`.

The synthetic end-to-end proof is
`TestCredentialSessionDiscoveryProducerEndToEndReassigns` at
`server/internal/daemon/credential_session_discovery_producer_test.go:17`. It
uses a current Codex account, a Codex replacement, and a lower-priority Kiro
account; the asserted result is assignment to the Codex replacement. Its
loopback emitter invokes the real daemon dispatcher at
`server/internal/daemon/credential_session_discovery_producer_test.go:216`.

Additional focused anchors:

- exact/bounded producer fields:
  `server/internal/daemon/credential_session_discovery_producer_test.go:83`
- concurrent event deduplication:
  `server/internal/daemon/credential_session_discovery_producer_test.go:117`
- bounded dedup state and active-session reset:
  `server/internal/daemon/credential_session_discovery_producer_test.go:161`
- emitter-failure retry:
  `server/internal/daemon/credential_session_discovery_producer_test.go:193`
- dispatcher forwarding and error preservation:
  `server/internal/daemon/credential_session_monitor_test.go:14` and
  `server/internal/daemon/credential_session_monitor_test.go:57`
- no-candidate fail-before-logout:
  `server/internal/rotation/discovery_reassignment_test.go:93`
- provider/tenant assignment boundary:
  `server/internal/rotation/discovery_reassignment_test.go:120`
- assignment-failure compensating cleanup:
  `server/internal/rotation/discovery_reassignment_test.go:159`
- concurrent duplicate reassignment:
  `server/internal/rotation/discovery_reassignment_test.go:195`

## Event deduplication bounds

The producer deduplication state is guarded by a process-local mutex and is
bounded to 1,024 entries with a five-minute TTL by default at
`server/internal/daemon/credential_session_discovery_producer.go:16`. The key is
exactly agent/account/provider/workspace at
`server/internal/daemon/credential_session_discovery_producer.go:106`.
In-flight reservations suppress concurrent duplicates; completed entries are
expired or deterministically evicted at
`server/internal/daemon/credential_session_discovery_producer.go:176` and
`server/internal/daemon/credential_session_discovery_producer.go:217`.

## Offline verification commands and results

Working directory for every command:
`multica-auth-work/server`.

Focused suite, 20 repetitions:

```text
GOTOOLCHAIN=local GOPROXY=off /home/dataops-lab/go-sdk/bin/go test -tags=offline ./internal/daemon ./internal/rotation -count=20 -run '^(TestCredentialSessionDiscoveryProducer.*|TestDispatchCredentialSessionDiscoveryEvent.*|TestDetectDiscoverySession.*|TestPoolSelectNext.*|TestReassignDiscoverySession.*|TestServiceOnExhaustion.*)$'
```

Result, exit 0:

```text
ok  github.com/multica-ai/multica/server/internal/daemon    0.069s
ok  github.com/multica-ai/multica/server/internal/rotation  0.079s
```

Uncached race run:

```text
GOTOOLCHAIN=local GOPROXY=off /home/dataops-lab/go-sdk/bin/go test -race -count=1 -tags=offline ./internal/daemon ./internal/rotation -run '^(TestCredentialSessionDiscoveryProducer.*|TestDispatchCredentialSessionDiscoveryEvent.*|TestDetectDiscoverySession.*|TestPoolSelectNext.*|TestReassignDiscoverySession.*|TestServiceOnExhaustion.*)$'
```

Result, exit 0:

```text
ok  github.com/multica-ai/multica/server/internal/daemon    1.070s
ok  github.com/multica-ai/multica/server/internal/rotation  1.034s
```

Vet:

```text
GOTOOLCHAIN=local GOPROXY=off /home/dataops-lab/go-sdk/bin/go vet -tags=offline ./internal/daemon ./internal/rotation
```

Result: exit 0 with no diagnostics.

## Current scoped source SHA-256 manifest

Hashes were computed only for the explicit product source and synthetic test
files below. No credential/authentication/session home or file was traversed or
hashed.

```text
4e240c09af1653fd2ecdbcd0763f8c05d3eb3f57ee8b9c339045edfa32a9ce6c  server/internal/daemon/credential_session_discovery_producer.go
818e69d4fe04057503b776716ae0704a6b27b7fc7b2b875e8f507b9a7b99491a  server/internal/daemon/credential_session_discovery_producer_test.go
a77d2f7052ba20c99a068b8dd46fbd688a94fd24c5a9c7e62bc1287ae140478e  server/internal/daemon/credential_session_monitor.go
5fd7005b4209b0c8d5c42ad9b240ad039d7568ad1f895bd6b9463b8e237613e2  server/internal/daemon/credential_session_monitor_test.go
d11d1ac43727028f588f56269407d7d591e14dc3b7dfe9456cf1b218c1603d3b  server/internal/daemon/wakeup.go
bc61a46c0c700010001a49c9075b87f97e89f802b138e63f90b53193a4b45b55  server/internal/rotation/detector_discovery.go
4e8092ff6b0f37a8416b961c89040433682e2d0d654293e192992dd3e1d55a4f  server/internal/rotation/detector_discovery_test.go
c655c9c4f94716d2c93109b1cc4d3c14e62d6fe8dea615dddec3ec93769b2832  server/internal/rotation/discovery_reassignment.go
d87aa80717e33a3f3768ebb7d698dcd9ace3bd934550317ec64c57f56a6566b2  server/internal/rotation/discovery_reassignment_test.go
0c4c453c730f5b9d49c605e2206e1501b778f32f40b20b142d86e70cdb8fb3fc  server/internal/rotation/pool.go
da401ef882af6fe06bb923494f4393b685c45dca01e9b0707127bda16a87f005  server/internal/rotation/pool_test.go
f20951a36765d2b6b8576232871f1991697de56d8577377b0226ed5e30130cf0  server/internal/rotation/service.go
989ae633e1865bc0f1f5a9f7eed5d2541100dbee94960c9e2c15f84346aa8ba1  server/internal/rotation/service_test.go
```

## Explicit production blockers and non-claims

Task 4.3 is not complete or accepted. The bounded synthetic path above works,
but the following production gaps remain:

1. **No concrete observation-source invocation.** The current Go source has no
   production construction or invocation of `NewCredentialSessionDiscoveryProducer`;
   constructor use exists only in the synthetic producer tests. Therefore a live
   discovery observation does not yet enter this producer path.
2. **Process-local deduplication and serialization, not multi-node CAS.** The
   producer dedup mutex/cache and `Service.agentLock` protect only one process.
   They do not prevent two daemon nodes from rotating the same assignment.
3. **Non-atomic persistence.** Assignment and rotation-history persistence are
   separate calls at `server/internal/rotation/service.go:150` and
   `server/internal/rotation/service.go:156`; there is no transaction/CAS that
   atomically commits the expected old assignment, new assignment, and audit
   record.
4. **Task 4.4 reporting remains out of scope.** The daemon WebSocket integration
   currently discards the dispatcher result at
   `server/internal/daemon/wakeup.go:282`; operator alerting and durable error
   reporting have not been implemented by these slices.
5. **Vendor cleanup is compensating and best-effort.** If assignment fails after
   successful vendor login, the service attempts logout at
   `server/internal/rotation/service.go:150`, but this is not a transactional
   rollback and cannot guarantee remote vendor-session cleanup.

Accordingly, this evidence proves only the deterministic synthetic
producer-to-same-provider-reassignment slice. It makes no production, database,
distributed-atomicity, live-vendor, or full task 4.3 completion claim.

## Independent reviewer ACCEPT (GLM52-auth-QA, 2026-07-18)

Reviewer: GLM52-auth-QA (independent reviewer; TL-queued in AGENT_LEDGER
adjudication alert: "Review queue for GLM52-auth-QA: (1) complete its in-flight
4.3 review"). This section is an independent-reviewer ACCEPT of the artifact's
narrow, honestly-scoped claim. It is **not** task acceptance: per
EVIDENCE_CONTRACT rule 6/7, task 4.3 remains UNCHECKED and the TL
(Kiro/Opus-4.8) must countersign before any checkbox or acceptance change.

### Verdict: ACCEPT (bounded synthetic slice; not task acceptance)

The artifact is fully reproducible. Every cited anchor, hash, command, and
non-claim was independently verified against current source with
`/home/dataops-lab/go-sdk/bin/go` (`go1.26.4 linux/amd64`),
`GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off`, from
`multica-auth-work/server`.

### Reproduction results

- **x20 run** (exact artifact command, `-tags=offline`): exit 0; both packages
  `ok` (rotation 0.042s, daemon 0.046s). Verbose re-run: 480 `--- PASS` / 0
  `--- FAIL` (24 cited tests × 20, including subtests). Concurrency subtests
  (`TestReassignDiscoverySessionConcurrentDuplicateObservationRotatesOnce` 32-worker,
  `TestCredentialSessionDiscoveryProducerDeduplicatesConcurrently` 64-worker)
  and boundary subtests (`different_provider`, `different_tenant`,
  `cross_provider`, `future_expiry`) all present and passing.
- **Race** (`-race -count=1 -tags=offline`): exit 0; rotation `ok` 1.041s,
  daemon `ok` 1.078s (race clean).
- **Vet** (`-tags=offline`): exit 0, no diagnostics.
- **SHA-256 manifest** (13 files): `sha256sum -c` exit 0, all OK. 11 of 13
  hashes overlap with the sibling task-4.2 artifact
  (`credential-isolation-next-account-selection.md`) and match exactly —
  consistent source baseline across both credential-isolation lanes.

### Anchor verification

All 16+ source line references verified exact: producer.go:16/94/106/112/117/
126/138/150/176/217; monitor.go:13/29/48/64; detector_discovery.go:29;
discovery_reassignment.go:23/54/65/72; service.go:94/108/134/150/156;
pool.go:35; wakeup.go:274 (event recognition) and :282 (result discard via
`_, _`). All 10 test anchor line references verified exact.

### `-tags=offline` semantics

Sound. The only `//go:build` constraint in the two packages is
`rotation_e2e_test.go:1` (`!offline`), which is the DB-gated E2E test requiring
`DATABASE_URL`. With `-tags=offline` that file is compile-excluded; all 24
cited synthetic tests are tag-free and always included (confirmed via
`-list`). `store_pg_test.go` has no build tag but its tests `t.Skip` without
`DATABASE_URL` and none match the `-run` regex, so no DB contact occurs.

### No-secret / no-DB / no-live verification

Confirmed: zero `os.Getenv`/`os.UserHomeDir`/`os.ReadFile`/real-home/`DATABASE_URL`
references in the 6 cited test files (grep returned empty). All stores and
authenticators are in-process synthetic fakes (`fakeStore`,
`syntheticReassignmentStore`, `producerSyntheticStore`, `fakeAuthenticator`,
`syntheticReassignmentAuthenticator`, `producerSyntheticAuthenticator`). No
credential, auth home, session file, token, environment secret, network,
database, or live service was read or used.

### Production blockers verified

All 5 blockers confirmed accurate against current source:
1. No production call site for `NewCredentialSessionDiscoveryProducer` (only
   test files, including `credential_rotation_task53_test.go` which postdates
   this artifact).
2. Process-local `sync.Mutex` dedup (`producer.go:70`) and `Service.agentLock`
   (`service.go:168-177`) — single-process only, no multi-node CAS.
3. Separate non-atomic `Assign` (`service.go:150`) and `RecordRotation`
   (`service.go:156`) calls.
4. `wakeup.go:282` discards the dispatcher result (`_, _ = ...`).
5. Compensating `Logout` on `Assign` failure (`service.go:151`) — best-effort,
   not transactional.

### Exact gaps (non-blocking; do not invalidate the artifact's narrow claim)

- **G1 (line precision):** Blocker 5 cites `service.go:150` for the logout, but
  the `Logout` call is at `:151`; `:150` is the `Assign` call. Substance correct;
  off-by-one reference only.
- **G2 (postdating test, not citable):** `TestCredentialIsolationTask53AutomaticRotation`
  in `credential_rotation_task53_test.go` (untracked, mtime 2026-07-18 17:06)
  provides more direct task-4.3 coverage — an explicit `wrong-tenant` (tenant-2)
  boundary alongside `wrong-provider` (kiro), and both acceptance scenarios
  ("Troca automática ao esgotar" + "Sem conta disponível") in one test. It
  postdates this artifact (mtime 16:05) and is absent from both the manifest and
  the `-run` regex. It passes x20/race independently. This is a temporal
  limitation, not a defect; recommended for a future manifest refresh when the
  artifact is revised.
- **G3 (manifest scope):** The 13-file manifest is narrower than the full
  reassignment surface (omits `contract.go`, `daemon.go`,
  `metrics/credential_metrics.go`, `observability/schema.go`). Defensible for
  the discovery-driven slice scope; the reactive `OnExhaustion` entry point via
  `daemon.go:4275` (`rotateTaskWithReason`) is a separate reassignment path
  covered by the sibling task-4.2 artifact. Noted for completeness.

### Conclusion

The artifact's claim — "proves only the deterministic synthetic
producer-to-same-provider-reassignment slice" — is fully supported and
reproducible. The scope is honestly bounded, the task remains UNCHECKED, and
the 5 production blockers are transparent. No fabrication, no secret, no
overclaim. Independent-reviewer **ACCEPT** of the bounded synthetic evidence.
TL countersign required for any task-level acceptance. Task 4.3 OpenSpec
checkbox stays `[ ]`.

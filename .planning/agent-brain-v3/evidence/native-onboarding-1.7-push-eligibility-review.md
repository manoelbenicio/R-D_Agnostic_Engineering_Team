# Native onboarding 1.7 push-eligibility review

## Provenance and check-in/out

| Field | Value |
|---|---|
| reviewer | Codex56#A, `w6:p1` |
| reviewer model | Codex based on GPT-5; no narrower runtime model/build identifier was exposed |
| check-in | `2026-07-18T21:35:55Z` |
| check-out | `2026-07-18T21:40:56Z` |
| observed `HEAD` | `b6571299b00c8e388abefe7ef9dcbcf8ac715d7f` |
| sole write | this artifact |
| integration authority | Kiro TL adjudicates; root integrates |
| verdict | **HOLD — no currently proven atomic push group** |

This review was read-only except for this uniquely named artifact. It made no
product, test, OpenSpec/spec/task, shared planning/index, Git index/ref, or
other evidence edit. It did not read credential values or environment-file
contents and did not use a database, network, provider, daemon, or live
service. `.env.example` was observed only through path status; its bytes were
not read or re-hashed under this assignment's no-env boundary.

## Result

Task 1.7 is currently checked at
`openspec/changes/native-runtimes-onboarding/tasks.md:12`. Shared planning
records `EV-AUTH-1.7` as `ACCEPT → CHECKED`; the accepted artifact is
`native-auth-password-provisioning.md`, current SHA-256
`2a5f7368a63202f5decb27bd562589e1cc9ad406499b29a14a471f1c1425c095`.
Its embedded reviewer section says ACCEPT at lines 283-296.

The backend behavior remains reproducibly green offline, and all 16
non-environment paths from its 17-file manifest still match their accepted
pins. `internal/handler/auth.go` also remains exactly pinned, and its entire
dirty diff belongs to 1.7: all three logical hunk families (five physical diff
regions) are password login/recent-auth/password-update work. No second
feature owns a changed line in that file.

Push eligibility nevertheless remains **HOLD** because acceptance and atomic
commit provenance do not close over the current dependency/test boundary:

1. The modified, actually executed router test
   `cmd/server/auth_routes_test.go` is essential evidence for route removal,
   `/auth/login`, and authenticated password update, but is outside the
   accepted artifact's 17-file SHA manifest.
2. Fourteen other modified `cmd/server` tests add the `!offline` topology used
   to make that package's deterministic test execution honest. Their behavior
   is discussed in the artifact but their current bytes are also outside its
   file-hash manifest.
3. The 17-file manifest contains `.env.example`, whose current bytes were not
   read under this review's explicit no-env constraint; only its historical
   accepted pin and current modified status are known here.
4. The manifest also contains the CLI password command and test, while the
   accepted artifact itself records that CLI first-password bootstrap remains
   incomplete because it sends only `new_password` under an ordinary bearer
   token and does not supply current-password/recent-auth proof
   (`native-auth-password-provisioning.md:274`). These files must not be
   silently swept into a backend-1.7 push group.
5. `internal/rotation/rotation_e2e_test.go` is shared with credential-isolation
   evidence and only adds an offline build tag. It is not required for the
   backend auth runtime boundary and requires cross-lane coordination if
   included.
6. Producer/reviewer separation is not self-contained in the accepted
   artifact. Its initial implementation record does not name a producer, and
   the appended `Independent reviewer ACCEPT` does not name a reviewer/session.
   The later ledger combines `Kiro/Opus-4.8 + Codex#56#A (co-lead)` but does
   not assign producer versus reviewer roles. Because this reviewer is also
   Codex56#A, this push review cannot manufacture a new distinct-reviewer claim
   to repair that provenance gap.

No staged Packet-B path overlaps the candidate backend group, and no explicit
file-owner lock was found for its `auth.go`/auth/middleware/router paths. Those
facts do not override the evidence-integrity gaps above.

## Accepted 17-file manifest versus current disk

The first row is intentionally not a current read.

| Current state | Accepted/current SHA-256 | Path | Finding |
|---|---|---|---|
| modified; bytes not re-read | historical accepted pin `3f9b95b76f2683bb0a91a9d6a7bc6db939dfb3af3dd43d10977f27a756db0512` | `multica-auth-work/.env.example` | Current hash unverified here; shared environment template; exclude. |
| modified | `6059f7e20ece7485016e2546ef977fadde925569e4b5ed1d862d0f3cace27de9` | `multica-auth-work/server/cmd/multica/cmd_user.go` | Exact match; known incomplete CLI bootstrap; exclude. |
| untracked | `80aa70ae912912b6233880dea2cbcc669ab9414ab8112bcb81a1441ee6dc8a3f` | `multica-auth-work/server/cmd/multica/cmd_user_password_test.go` | Exact match; tests the incomplete CLI surface; exclude with CLI. |
| modified | `5aa5cc4268474e8b79ada549ce908df04b071d2fda4aed90a5460c577d423bb6` | `multica-auth-work/server/cmd/server/main.go` | Exact match; backend candidate. |
| modified | `5c6492bfd64347d48bb13749ac3f1b38ef84b4275fd4d20b1fc44e0f1cdb5a74` | `multica-auth-work/server/cmd/server/router.go` | Exact match; backend candidate. |
| modified | `bbb5fa1ca1bf24f94756906512a5717e7a2783113be0c91ba941c700cf8822fd` | `multica-auth-work/server/internal/auth/jwt.go` | Exact match; backend candidate. |
| untracked | `9df59d84abfbb5e44a8f1f00571fdc9b47119a15bcf6ce532a01f400bc00fdf5` | `multica-auth-work/server/internal/auth/jwt_configuration_test.go` | Exact match; backend candidate. |
| untracked | `e800814c59e5ea55295d6b8c2209bf57fe776595e5d1e541ff511f4f892db94b` | `multica-auth-work/server/internal/auth/recent_auth.go` | Exact match; backend candidate. |
| untracked | `ecbc885334affbcf20cadc2c7b73a80d6f77fd570f3ea14676e51f2c942fdf90` | `multica-auth-work/server/internal/auth/recent_auth_test.go` | Exact match; backend candidate. |
| modified | `d69877a9dcabec59726717628c103b2b1ab0c9a4c7673d14a80c8360b7a259e0` | `multica-auth-work/server/internal/handler/auth.go` | Exact match; all dirty hunks are 1.7; backend candidate. |
| modified | `3c8f75b5ac2e9a4e2ca83b228285c3ff21d2ced484aab3b29f71cb7b67d70857` | `multica-auth-work/server/internal/handler/auth_provider.go` | Exact match; backend candidate. |
| untracked | `4871a86311316e4da83c6fb56e97249da1bfff5714f1c1b4e101380510055e64` | `multica-auth-work/server/internal/handler/passwordtest/provision_test.go` | Exact match; actual deterministic assertions; backend candidate. |
| modified | `10d75ff5a2d7db032eab78a307a86ba0157c2725ee72965494c2bc1f571eae6a` | `multica-auth-work/server/internal/middleware/auth.go` | Exact match; backend candidate. |
| modified | `e76cd669222074125561e4e57eac2c737418d7c15b1c6deffa4b2215f2c5b124` | `multica-auth-work/server/internal/middleware/auth_test.go` | Exact match; actual deterministic assertions; backend candidate. |
| modified | `14c3ee447ef5f397100100fb086157b538ba36e42a6c17bc340348bb10711808` | `multica-auth-work/server/internal/middleware/ratelimit.go` | Exact match; backend candidate. |
| modified | `43418c0ec0652bbc7e60102d196f3de155cb6734d38643bc14123d9f55944084` | `multica-auth-work/server/internal/middleware/ratelimit_test.go` | Exact match; actual deterministic assertions; backend candidate. |
| modified | `97bd6dee369edc88e602f83dd4c6d70c9f83d1b4594f1dcc29d2ef111e52c298` | `multica-auth-work/server/internal/rotation/rotation_e2e_test.go` | Exact match; shared/offline topology; exclude. |

Canonical SHA-256 over the sorted 16 currently re-hashed non-environment
manifest lines: `cb6bdb544a3b88150d23dcc1b8d7ce5d0f5efc468c7eb04edb5624c229e62621`.
This is not represented as the full 17-file manifest because `.env.example`
was deliberately not read.

## Modified companion test/topology files outside the 17-file manifest

| Current SHA-256 | Path | Status for 1.7 push |
|---|---|---|
| `7e814662104c09feb6eba8b02d05bf26ddac9e12659bd97a8541d3f2d974446e` | `server/cmd/server/auth_routes_test.go` | Actually executed task test; required candidate companion, but not pinned by EV-AUTH-1.7 manifest. |
| `4e946617541db766f9146fdb12fbca00b40879f26317cd8b067bf43176c4188c` | `server/cmd/server/activity_listeners_test.go` | `!offline` topology only; excluded from candidate. |
| `b9eaef02ba8378afaba13c9f55efcdd00617f99aa8cefd2e1a3095db27ebfced` | `server/cmd/server/autopilot_failure_monitor_test.go` | `!offline` topology only; excluded. |
| `9b1a1fb022738fe859fbd3fd43c6df95803eab5bb516b16b7fca7356c4cc900d` | `server/cmd/server/autopilot_listeners_test.go` | `!offline` topology only; excluded. |
| `5c5d982451d3a11a4c40f877cfff7e5af4fcdea31e2c6a6e6d9fbdb3a462620a` | `server/cmd/server/comment_attachment_integration_test.go` | `!offline` topology only; excluded. |
| `065535093ac83dd0b7beb40cabd69796e082ed37d3ff7bf13df46d48a52b24db` | `server/cmd/server/comment_edit_mention_integration_test.go` | `!offline` topology only; excluded. |
| `bde23c3896fc1a482f07ff4e87c924f43199a7117b21193c248606077d2f590e` | `server/cmd/server/comment_trigger_integration_test.go` | `!offline` topology only; excluded. |
| `3fa4ba0632fb74538bb4b0454828576a31a9989e6bcc4ce03529042c4b80ff4a` | `server/cmd/server/integration_test.go` | `!offline` topology only; excluded. |
| `93788bf9a2d0aecaedcc753f0720991ffa01f01ebe4b0f74df51ed6547f101c9` | `server/cmd/server/notification_listeners_test.go` | `!offline` topology only; excluded. |
| `0b6d7cfddda53cca7ebec9b2cb7d1db06f665745e46fe7a704f208209158115e` | `server/cmd/server/quick_create_subscriber_test.go` | `!offline` topology only; excluded. |
| `343146479e9ccf5d94f33d64ca55c45262936a9fee6b52d3b85a197f2815afd8` | `server/cmd/server/rerun_session_test.go` | `!offline` topology only; excluded. |
| `53ca268ef0b62196ca31b5fb2e09b17ea08e8123a6026991dc34a8ff10fdd817` | `server/cmd/server/runtime_sweeper_race_test.go` | `!offline` topology only; excluded. |
| `79673eb14b49611c0c4450518599a7dfb053b4efd389b61dec62db8319263bfe` | `server/cmd/server/runtime_sweeper_test.go` | `!offline` topology only; excluded. |
| `836ed1446064aa894ddb0fed24ed4f540fcaefd99d017c30b4c4c09dfa8f7377` | `server/cmd/server/subscriber_listeners_test.go` | `!offline` topology only; excluded. |
| `53a96f1048f2dc2adc3e4a19585c24c749da89e5dbdd6e303a34f93e6f4d21e0` | `server/cmd/server/workspace_scope_guard_test.go` | `!offline` topology only; excluded. |

Canonical sorted 15-line companion manifest SHA-256:
`3874cf1eb2ba82222baf10b6db8342c9e24692dff7e9a425dd5955714e639fc9`.

## Exact backend candidate boundary after HOLD resolution

The smallest dependency/test-complete candidate found is the following 14
paths. It has no staged overlap and targeted `git diff --check` is clean.
Canonical sorted manifest SHA-256:
`217d30122543eaa16e633e395963385e104167569f5c7797dea37ed19391f511`.

```text
multica-auth-work/server/cmd/server/auth_routes_test.go
multica-auth-work/server/cmd/server/main.go
multica-auth-work/server/cmd/server/router.go
multica-auth-work/server/internal/auth/jwt.go
multica-auth-work/server/internal/auth/jwt_configuration_test.go
multica-auth-work/server/internal/auth/recent_auth.go
multica-auth-work/server/internal/auth/recent_auth_test.go
multica-auth-work/server/internal/handler/auth.go
multica-auth-work/server/internal/handler/auth_provider.go
multica-auth-work/server/internal/handler/passwordtest/provision_test.go
multica-auth-work/server/internal/middleware/auth.go
multica-auth-work/server/internal/middleware/auth_test.go
multica-auth-work/server/internal/middleware/ratelimit.go
multica-auth-work/server/internal/middleware/ratelimit_test.go
```

This is a candidate boundary, **not currently push-eligible**. Before release,
Kiro TL needs an explicit provenance reconciliation that (a) identifies the
original producer and distinct reviewer, and (b) accepts/pins the current
`auth_routes_test.go` companion with this backend group. Root must then re-hash
the 14 paths immediately before integration. CLI, `.env.example`, rotation
topology, the fourteen unrelated `cmd/server` build-tag files, frontend 1.5,
mobile, and shared OpenSpec/planning files remain outside this boundary.

## Offline reproduction and honest execution boundary

Working directory: `multica-auth-work/server`. Toolchain:
`/home/dataops-lab/go-sdk/bin/go`, `go1.26.4 linux/amd64`, with
`GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off APP_ENV=test` and the synthetic,
intentionally invalid `DATABASE_URL=://offline-invalid`.

```text
/home/dataops-lab/go-sdk/bin/gofmt -l <16 focused Go paths>
```

Exit 0, empty output; no file was formatted or changed. Output SHA-256:
`e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`.

```text
/home/dataops-lab/go-sdk/bin/go vet -tags=offline \
  ./internal/auth ./internal/middleware ./internal/handler/passwordtest \
  ./cmd/multica ./cmd/server
```

Exit 0, empty output. This is compile/static-analysis evidence only, not an
executed assertion count. Output SHA-256:
`e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`.

```text
/home/dataops-lab/go-sdk/bin/go test -tags=offline -count=1 -v \
  ./internal/auth ./internal/middleware ./internal/handler/passwordtest \
  ./cmd/multica ./cmd/server -run '<focused auth/password regex>'
```

Exit 0; five packages `ok`; **25 parent tests and 30 subtests actually ran**
(55 RUN / 55 PASS total, zero FAIL). Output SHA-256:
`8f0a27083936984ad20eae2f6c98c85cda56519c84407db22a3fa2481e784f38`.

The identical command with `-race` also exited 0: five packages `ok`, 25
parent tests + 30 subtests actually ran, 55/55 PASS, and no race report.
Output SHA-256:
`62aa9f43c60a6b58f8fffab76886b6678962b67e0b23c0ba2383bf107869fa6f`.

Execution qualifications:

- `internal/auth`, `internal/middleware`, the separate
  `internal/handler/passwordtest`, `cmd/multica`, and the two focused
  `cmd/server` router tests genuinely executed assertions.
- The DB-gated `internal/handler` package suite was not invoked and is not
  claimed. Handler behavior is exercised through deterministic fakes in the
  separate `passwordtest` package.
- `TestPostgresPasswordCredentialStore...` names test the production store
  logic with fake DB executors; they do not connect to PostgreSQL.
- Under `-tags=offline`, the fourteen modified DB/integration
  `cmd/server` tests and `internal/rotation/rotation_e2e_test.go` are gated out;
  they compiled/executed neither assertions nor DB operations in these runs.
- Tests outside the focused regex may compile as package inputs but did not
  execute. No full-suite, frontend, mobile, live-login, migration, or
  production-startup claim is made by this review.

## Final disposition

**HOLD.** Current source behavior is green and the 14-file backend candidate is
exact, ownership-clean and unstaged, but current acceptance provenance does
not yet pin the full test boundary with auditable producer/reviewer separation.
Kiro TL adjudicates; root integrates only after that gap is explicitly closed.

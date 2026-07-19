# Native Onboarding 1.7 — Independent Push-Eligibility Review

- reviewer: Kiro / Opus-4.8, wave w8:p1 (independent; distinct from eligibility reviewer Codex56#A and from the unnamed acceptance reviewer/producer)
- date: 2026-07-18T21:52:00Z
- mode: READ-ONLY. Proportionate offline reproduction only (no DB/jsdom/network/services). No shared/product/test/spec/task/git/index/credential/env edits. This is the only file created.

## Embedded check-in / check-out
- CHECK-IN 2026-07-18T21:46:00Z — Kiro/Opus-4.8 w8:p1 — stream NATIVE-1.7-PUSH-ELIGIBILITY-INDEPENDENT-REVIEW — READ-ONLY.
- CHECK-OUT 2026-07-18T21:52:00Z — DONE. Verdicts below. Kiro TL adjudicates; root integrates. Not self-accepted.

## Artifacts verified (stable)

- Eligibility review `native-onboarding-1.7-push-eligibility-review.md` — SHA-256 `1937b6fca2fc2a845d74fcbd3a55eb2acc72d87d5fe9b87bd9370d3d4876a772` (matches asserted; two spaced reads identical). Reviewer: **Codex56#A w6:p1**; its verdict = **HOLD**.
- Original producer evidence `native-auth-password-provisioning.md` — SHA-256 `2a5f7368a63202f5decb27bd562589e1cc9ad406499b29a14a471f1c1425c095`.
- Task: `native-runtimes-onboarding/tasks.md:12` **1.7 `[x]`**.

## Independent findings (I reproduced/verified, not merely echoed)

### 1. Offline 55/55 normal + race — GREEN corroborated (exact count not byte-reproduced)
Reproduced with the pinned offline toolchain (`go1.26.4`, `GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off APP_ENV=test DATABASE_URL='://offline-invalid'`) over the five scoped packages (`internal/auth`, `internal/middleware`, `internal/handler/passwordtest`, `cmd/multica`, `cmd/server`):
- `-tags=offline` normal: **all five `ok`, zero FAIL**.
- `-tags=offline -race`: **all five `ok`, no race report**.
- With a deliberately **broader** auth/password/login/rate/jwt/recent regex I observed **115 PASS lines, 0 FAIL** across the five packages.
- **Honest limitation:** the eligibility review's precise **55 (25 parent + 30 subtests)** count depends on its specific unpublished `-run` regex, which I did **not** reproduce byte-for-byte; I independently confirm the same five packages are **green and race-clean**, and the named anchor tests it cites (`TestPasswordAuthRoutes`, `TestValidateJWTConfigurationFailsClosedOutsideExplicitDevelopment`, `TestHasRecentAuthentication`, `TestBoundedLocalRateLimiter*`, `TestPostgresPasswordCredentialStore*`) fall within my green set. So 55/55 is **corroborated in kind**, not falsified; I do not restate "exactly 55" as my own count.
- **Bound (agreed):** the DB-gated `internal/handler` suite was not executed; handler behavior is covered via `internal/handler/passwordtest` fakes. Truthfully compile/fake-only for the handler package.

### 2. auth.go all-hunk 1.7 ownership — CONFIRMED
`git diff` shows five hunks (contexts: `Login`, `issueJWT`, `VerifyCode`, `UpdateMeRequest`/password-update `+510,86`, `GoogleLogin`). Added-line keyword census: `Password`×20, `password`×13, `AuthProvider`×2, `Login`×1, `RecentAuth`×1 — **no non-1.7 feature theme**. I concur: every dirty hunk in `auth.go` is password-login / recent-auth / password-update / route work; **no second feature owns a changed line.** Current hash `d69877a9dcabec59726717628c103b2b1ab0c9a4c7673d14a80c8360b7a259e0` matches the pin.

### 3. auth_routes_test.go — CONFIRMED essential-but-unpinned
Current SHA-256 `7e814662104c09feb6eba8b02d05bf26ddac9e12659bd97a8541d3f2d974446e`; it is **not** among the accepted 17-file EV-AUTH-1.7 manifest, yet it is the actually-executed router test for `/auth/login`, route removal, and authenticated password update. Confirmed: the acceptance does not pin all the evidence the runtime claim relies on.

### 4. Outside-manifest topology/env/bootstrap dependencies — CONFIRMED and correctly excluded
- **14 `//go:build !offline` `cmd/server` tests:** gated out under `-tags=offline` (my run confirms `cmd/server` executed only offline-safe tests and passed); topology-only; correctly excluded from the atom.
- **`.env.example`** (in manifest): shared env template; **bytes not read** under the no-env constraint (only status/historical pin known); correctly excluded.
- **CLI `cmd_user.go` + `cmd_user_password_test.go`** (in manifest): the producer artifact itself records CLI first-password bootstrap **incomplete** (sends only `new_password`, no current-password/recent-auth proof); correctly excluded.
- **`rotation_e2e_test.go`** (in manifest): shared with credential-isolation, offline build-tag only; correctly excluded (cross-lane).

### 5. Producer / reviewer attribution — GAP CONFIRMED
The original artifact is an "implementation evidence record for independent review"; **its implementation record names no producer**, and the appended **"Independent reviewer ACCEPT" names no reviewer/session**. The eligibility reviewer (Codex56#A) correctly noted it cannot self-repair this because it is the same identity. My review (Kiro/Opus-4.8) adds a **distinct second-party technical reproduction**, but I **cannot** retroactively name the original producer or the original accepting reviewer — those fields remain **irrecoverable** and must not be fabricated.

## Verdict — three distinct levels

1. **Technical: PASS (bounded).** Five scoped packages green offline + race-clean (independently reproduced); `auth.go` hunks are wholly 1.7; key hashes match. Bound: DB-gated `internal/handler` suite not executed (fakes via `passwordtest`); exact 55-count not byte-reproduced (broader regex, all green).
2. **Task acceptance: QUALIFIED-ACCEPTED.** Task 1.7 `[x]`; EV-AUTH-1.7 is **artifact-backed** with a 17-file manifest (stronger than ledger-only cases) and 16 non-env hashes match. Qualifiers: **producer + accepting-reviewer are unnamed** (attribution gap), and the **essential executed test `auth_routes_test.go` is outside the accepted manifest** — acceptance does not close over its own test boundary.
3. **Atomic-push: HOLD (I concur with Codex56#A).** The 14-file backend candidate (canonical `217d3012…`) is exact, ownership-clean, unstaged, and green, but push is blocked until: (a) producer + **distinct** reviewer are named or an owner waiver is recorded; (b) `auth_routes_test.go` is pinned into the accepted manifest with the 14-file group; (c) `.env.example` is reconciled under an env-permitted review (its bytes were not read here); (d) **Kiro TL authorization + root re-hash** immediately before integration. CLI/rotation/14 topology tests/frontend-1.5/mobile stay out.

## Smallest legitimate remediations (owner/TL — not executed, no fabrication)
1. Name the original producer and a **distinct** accepting reviewer, or record an owner waiver for the irrecoverable attribution.
2. Extend the EV-AUTH-1.7 manifest to pin `auth_routes_test.go` (and re-hash) as part of the 14-file backend atom.
3. Env-permitted reconciliation of `.env.example` current bytes (outside this no-env review).

## Explicit non-claims
- Created only this file. No shared/product/test/spec/task/git/index/credential/env edits; no `add/restore/commit/push`; no checkbox change.
- Read no credential/env values (`.env.example` bytes not read); no DB/jsdom/network/provider/service; DB-gated handler suite not run.
- Did **not** reproduce the exact 55-count (broader regex, all green) and did **not** fabricate producer/reviewer identities — reported as irrecoverable.
- Decision support only: technical PASS ≠ task acceptance ≠ push authorization. Kiro TL adjudicates; root integrates.

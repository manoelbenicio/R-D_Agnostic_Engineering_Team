# ORQ-39 — Browser QA plan V3 independent peer review

- Card: ORQ-39 (`c03941bc-3bde-4de1-ab19-1ba93de0ad51`)
- Mode: READ-ONLY. No workflow creation, push, PR, merge, run or E2E execution.

## Verdict: BLOCK

The plan has the requested disposable-CI shape, but C4 contains a concrete schema/fixture
error and the root workflow/path and authorization gates are not precise enough for approval.

| Check | Verdict | Evidence |
|---|---|---|
| C1 rename | **BLOCK** | Lines 24–30 say all paths are renamed to ORQ-39, but line 27 lists `apps/web/**` and `e2e/**`, while the actual workflow manifest uses `multica-auth-work/apps/web/**` and `multica-auth-work/e2e/**` (80–81). The claim “integral” is therefore internally inconsistent; path filters must be canonicalized before push. |
| C2 `cmd/migrate up` | PASS | Lines 32–37 and 139–143 use `go run ./cmd/migrate up` with server working directory and an ephemeral Postgres hostname. |
| C3 `APP_ENV=test` | PASS WITH GATE | Lines 39–46 and 150–167 set `APP_ENV=test` for server/E2E. A test-only JWT secret/default is implied, but no explicit startup assertion prevents accidental non-test configuration. |
| C4 verification code/DB | **BLOCK** | Plan line 49 says the backend/fixture require a `verification_code` row and line 51 claims `SELECT verification_code ...`. The actual fixture `multica-auth-work/e2e/fixtures.ts:47–57` executes `SELECT code FROM verification_code ...`; this is a column-name mismatch. The workflow has no seed step, relying on `/auth/send-code`; the plan must correct the query claim and prove that send-code creates a row before verification. |
| Immutable pins | PASS WITH GATE | Lines 117,122,127,132,171 and 95,100 use immutable action SHAs and image digests. `node-version: 20`, `go-version: '1.24'`, pnpm 9 and `ubuntu-latest` remain moving labels; if “all pins immutable” is literal, pin runner/tool versions or explicitly scope the requirement to actions/images. |
| Root workflow governance | **BLOCK** | Lines 56–65 correctly prohibit editing default branch and restrict push branch, but the manifest is only a proposed root workflow. It needs a pre-push proof that the exact file is at root `.github/workflows`, branch/path filters match the actual monorepo paths, and no nested workflow is relied upon. |
| No production data | PASS WITH GATE | The manifest uses hardcoded `postgres` service/database and `DATABASE_URL` at 36,44,141,154,165; no production URL/secret is present. Add a fail-fast assertion rejecting non-`postgres` host and production-like `APP_ENV` before migration/server start. |
| Minimal env | PASS WITH GATE | The env is mostly bounded to test DB, port, API URL and verification code. The plan should state whether default JWT is intentionally allowed in `APP_ENV=test`, and keep the dev code in the E2E step if it is required by acceptance. |
| Timeout/cost | PASS | `timeout-minutes: 20` and `--max-failures=5` at lines 93 and 167, concurrency cancellation at 86–88, and trace retention 7 days at 169–175 provide bounded execution. |
| Separate authorizations | **BLOCK** | Lines 60–65 distinguish push and cleanup, and line 197 says execution is blocked, but do not define separate explicit approvals for push, workflow run, PR creation, PR merge and cleanup. Add one gate/owner authorization per action; no single push authorization should imply PR/merge/run. |

## Required corrections before PASS

1. Fix C4 to the actual `code` column and add a deterministic send-code/row-existence check.
2. Canonicalize path filters to the real repository paths and verify root-workflow placement.
3. Add fail-closed non-production guards and state the JWT test-mode policy.
4. Record distinct authorizations for push, workflow run, PR, merge and cleanup.

No workflow or runtime state was changed.

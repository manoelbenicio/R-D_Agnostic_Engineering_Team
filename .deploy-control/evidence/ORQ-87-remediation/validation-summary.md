# ORQ-87 Dependency Remediation Evidence

- Continuation point: `2495974cc19e0443b5b2b7f5c986d7bcc88a5000`
- Product-source baseline: `c7f14ee13fa6357b4c67a8b1230fecc9bcd7c81e`
- Branch: `recovery/orq87-native-claims`
- Date: 2026-08-02 UTC
- Boundary: rerun only prior dependency-blocked native/backend/web slices
- Excluded: completed 19-claim classification, live smoke/UAT, production/runtime,
  credentials, OmniRouter, architecture, deployment, restart, image build, and Kanban

## Dependency provenance and integrity

No standard host Go module/build, pnpm-store, or Corepack cache was present. Fresh isolated
task caches were used under `/tmp/orq87-go`, `/tmp/orq87-pnpm-store`, and
`/tmp/orq87-corepack`. No user/global npm configuration or Git credential prompting was
enabled.

| Dependency set | Resolution | Integrity |
| --- | --- | --- |
| Go modules | final fresh-cache re-resolution from `https://proxy.golang.org` only; `sum.golang.org` | committed `go.sum`, checksum DB, and offline `go mod verify` exit 0 |
| JS dependencies | `https://registry.npmjs.org/` with pnpm 10.28.2 | committed `pnpm-lock.yaml`; `--frozen-lockfile` install exit 0 |

Exact tool versions and SHA-256 values for `go.mod`, `go.sum`, `package.json`, and
`pnpm-lock.yaml` are in `dependency-integrity.log`. Download commands, source restrictions,
timestamps, and exit codes are in `go-dependency-download.log` and
`pnpm-dependency-install.log`. The first Go command preserved in
`go-dependency-download.log` configured standard direct fallback without recording the
transport actually used. `go-official-reverification.log` therefore re-resolves the complete
module set into a fresh cache from `proxy.golang.org` only and verifies it offline; that
official-only result is the dependency-provenance authority for this remediation.

## Current validation results

| Evidence | RC | Proven boundary |
| --- | ---: | --- |
| `native-daemon.log` | 0 | focused NIM execenv/rotation, runtime wiring, config probes, credential isolation, model-report behavior |
| `auth-offline.log` | 0 | focused password/JWT/middleware/CLI/server auth assertions, without credentials or live services |
| `backend-vet.log` | 0 | scoped relevant Go packages |
| `backend-build.log` | 0 | server and CLI binaries compile |
| `web-core-tests.log` | 0 | 33 focused core auth/API assertions |
| `web-views-tests.log` | 0 | 177 focused login/i18n/parity assertions |
| `web-app-tests.log` | 0 | 17 focused app auth/onboarding/callback assertions |
| `web-core-typecheck.log` | 0 | core TypeScript check |
| `web-views-typecheck.log` | 0 | views TypeScript check |
| `web-app-typecheck.log` | 0 | web app TypeScript check |
| `web-build.log` | 1 | Next.js compilation reached; failed fetching three Google Fonts with post-install networking disabled |

The earlier already-green `native-agent.log` was not repeated. The build failure is preserved
as observed and is not represented as a source-test failure or a production-build PASS.

## Claim delta only

- Newly bounded-direct: `1.7`, `2.1`, `2.3`, and backend-binary portion `2.4a`.
- Still open with stronger partial evidence: `1.4`, `1.5`, `1.6`, `2.5a`, and `3.1`.
- Unchanged and open/out of scope: `2.4b`, `2.5b`, `3.2`, `3.3`, and `3.4`.
- `SOURCE_PARITY_EVIDENCE.md` and `SECOND_CHANGE_REVIEW.md` remain absent and were not fabricated.

Independent review is requested for this evidence, the four bounded-direct transitions, and
the retained open boundaries before any external task-status decision.

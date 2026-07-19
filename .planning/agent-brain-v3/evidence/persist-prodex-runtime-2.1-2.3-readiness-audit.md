# Implementation-Readiness Audit for persist-prodex-runtime-integration Tasks 2.1–2.3

## Grade: 2.1-2.3 PARTIAL

## Source Hashes Assessed
- `4b1650e059a52f7951fd83af5ca707aa5b42b111b2c4dd107d4f11d501dbc3a8  multica-auth-work/server/internal/daemon/health.go`
- `82035719e5a3b3a1472af973412f431e0c14ad11e04939885141893187ee38f7  multica-auth-work/server/internal/daemon/prodex.go`

## Implementation Status Matrix

| Task | Status | Location (Path/Line) | Finding / Gap |
| --- | --- | --- | --- |
| **2.1 Add secure persistent launcher/service with mode-0600 EnvironmentFile** | **MISSING** | Entire codebase (No `.service` or `systemd` scripts found) | There are no scripts, deployment configurations, or systemd units that create or consume a mode-0600 Prodex EnvironmentFile. |
| **2.2 Extend local persistent environment with L2/Postgres/redacted secrets** | **MISSING** | Entire codebase | The Go implementation (`prodex.go`) expects `MULTICA_L2_BASE_URL` and `PRODEX_PG_URL`, but there is no template or script establishing this persistent environment configuration. |
| **2.3 Operator-visible config-source and runtime-authority health fields** | **IMPLEMENTED** | `multica-auth-work/server/internal/daemon/health.go:64`, `:67`, `:144`, `:147` | The `healthProdex` struct successfully exposes `ConfigSource` and `RuntimeAuthority`. The `/health` endpoint serves these fields correctly. |

## Permission / Secret Risks
Without a designated secure launcher and a predefined mode-0600 EnvironmentFile template (Tasks 2.1 & 2.2), operators are forced to fall back on `export` in interactive shells or merging Prodex secrets into the standard application `.env`. This creates a severe risk of secret leakage into shell histories and breaks the intended isolation of the L2 credentials.

## Service Restart / Rollback Gaps
The absence of a persistent service launcher means that any restart of the host or daemon process will lose the necessary `MULTICA_PRODEX` environment variables unless operators manually intervene, completely defeating the purpose of the "persist-prodex-runtime-integration" objective.

## Health Schema Fields
Task 2.3 is fulfilled by the `healthProdex` JSON struct containing:
- `config_source` (maps to `d.cfg.Prodex.ConfigSource`)
- `runtime_authority` (calculated via `runtimeAuthority(d.cfg)`)

## Recommended Disjoint Owners, Files, and Dependency Order
1. **Infra / Operations Owner (Dependency 1):** Must create a systemd `.service` template (e.g., `multica-auth-work/deploy/systemd/multica-prodex.service`) that strictly uses `EnvironmentFile` targeting a mode-0600 file.
2. **Infra / Operations Owner (Dependency 2):** Must define the baseline `.env.example`-equivalent for Prodex (e.g., `prodex.env.example`) specifying the exact variables (L2 endpoint, Postgres URL) without revealing default secrets.
3. **Backend Developer Owner (No Dependency):** No action required on `health.go` or `prodex.go`, as Task 2.3 is fully verified and implemented.

## Explicit Non-Claims
- This audit did not inspect real credentials, Docker/network/DB/daemon state, or active `.env` values.
- No docs, codebase files, or OpenSpec task checkboxes were modified.
- No daemon or network processes were executed.

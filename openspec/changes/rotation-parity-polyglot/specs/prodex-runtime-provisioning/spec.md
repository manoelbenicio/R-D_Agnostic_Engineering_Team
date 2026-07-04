## ADDED Requirements

### Requirement: prodex Binary Provisioning
The system SHALL provision the pinned prodex runtime binary before any deploy or launch task runs. Provisioning MUST build the binary from the pinned source (version v0.246.0, commit `7750da9b`), verify its integrity, and expose it to Multica via configuration. A binary that is not built and verified MUST NOT be treated as available.

#### Scenario: Build from pinned source
- **WHEN** the foundation phase runs and the prodex source is present at a stable location on commit `7750da9b`
- **THEN** the toolchain (Rust/cargo) SHALL be installed and `cargo build --release` SHALL produce `target/release/prodex`

#### Scenario: Integrity and pin verification
- **WHEN** the prodex binary has been built
- **THEN** the system SHALL verify version `v0.246.0` and commit `7750da9b` and record the binary hash/attestation before marking provisioning done

#### Scenario: Multica resolves the executable
- **WHEN** `MULTICA_PRODEX_ENABLED=1` with `MULTICA_PRODEX_PATH`, `MULTICA_PRODEX_VERSION`, `MULTICA_PRODEX_COMMIT` set
- **THEN** Multica's `exec.LookPath` SHALL resolve the pinned binary and startup SHALL fail closed if the executable is missing or the version/commit vars are absent

### Requirement: Deploy Environment Readiness
The system SHALL confirm the deploy/dev environment is ready before deploy: Postgres and Redis reachable, container build toolchain available, and reversible migrations applied. Missing prerequisites MUST block deploy.

#### Scenario: Datastores reachable
- **WHEN** the foundation phase validates the environment
- **THEN** Postgres (`:5432`) and Redis (`:6379`) SHALL be reachable from the server container and shared state SHALL use Postgres (SQLite forbidden)

#### Scenario: Reversible migrations
- **WHEN** schema changes for gateway/ledger/approved-accounts are applied
- **THEN** each migration SHALL have a tested reversible (down) path

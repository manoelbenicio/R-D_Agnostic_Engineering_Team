## ADDED Requirements

### Requirement: Durable Prodex activation
The system SHALL load Prodex and L2 configuration from a durable, permission-restricted source before daemon configuration is resolved, and SHALL preserve the same effective activation after a daemon or host restart.

#### Scenario: Restart preserves activation
- **WHEN** a daemon configured with required Prodex/L2 integration is restarted
- **THEN** the daemon loads the durable configuration, starts the adapter, and reports Rust L2 runtime authority without manual shell exports

#### Scenario: Required configuration is missing
- **WHEN** `MULTICA_PRODEX_REQUIRED` is enabled and the durable configuration cannot be loaded or resolves Prodex/L2 as disabled
- **THEN** daemon startup fails closed with a redacted configuration error

### Requirement: Separate Prodex and adapter executables
The system SHALL configure, validate, and supervise the pinned upstream Prodex binary separately from the Multica `rpp.l2.v1` adapter executable.

#### Scenario: Adapter starts the pinned gateway
- **WHEN** the Go daemon starts L2
- **THEN** it executes the configured adapter binary and the adapter executes the configured pinned Prodex binary for gateway traffic

#### Scenario: Executable mismatch
- **WHEN** either executable is missing, not executable, outside the approved filesystem policy, or fails its integrity checks
- **THEN** startup fails before any L2-owned session is admitted

### Requirement: Reference-only profile reconciliation
The system SHALL reconcile the current validated Multica Codex account inventory into Prodex profiles using references to existing slot-local `CODEX_HOME` directories and MUST NOT copy credential material.

#### Scenario: New approved slot
- **WHEN** a current validated Codex account has no Prodex profile
- **THEN** reconciliation registers the profile using the official `--codex-home` reference without copying `auth.json`

#### Scenario: Matching profile already exists
- **WHEN** the Prodex profile name already resolves to the approved slot home
- **THEN** reconciliation succeeds idempotently without modifying credentials

#### Scenario: Profile points to another slot
- **WHEN** a Prodex profile name resolves to a credential home different from the approved manifest
- **THEN** reconciliation fails closed and does not overwrite either profile

### Requirement: One slot per credential identity
Every approved runtime profile SHALL resolve to its own credential identity and SHALL NOT share, inherit, or fall back to another agent slot or a global credential store.

#### Scenario: Duplicate credential identity
- **WHEN** two approved slots contain the same credential identity
- **THEN** reconciliation rejects both as a duplicate assignment and reports only opaque slot/profile identifiers

#### Scenario: Global fallback attempted
- **WHEN** an approved profile lacks its slot-local credential material
- **THEN** runtime startup fails instead of using a global or previously active credential

### Requirement: Obsolete credentials are removed
The system SHALL treat only the newly validated Multica account inventory as authoritative and SHALL remove provider credential material that is not referenced by that inventory.

#### Scenario: Unreferenced Codex slot
- **WHEN** a Codex slot is not referenced by any current validated Codex account row
- **THEN** its legacy `auth.json` is removed without deleting non-credential agent state

#### Scenario: Obsolete account record
- **WHEN** an account is explicitly classified as legacy and has no current assignment
- **THEN** its account and credential references are removed so it cannot participate in selection or fallback

### Requirement: Filesystem and permission enforcement
Prodex state, slot homes, and credential files SHALL reside on an approved POSIX filesystem with directories mode 0700 and credential files mode 0600.

#### Scenario: Unsafe filesystem or mode
- **WHEN** a selected profile resolves to drvfs, 9p, CIFS, an out-of-root path, a group/world-readable directory, or a group/world-readable credential file
- **THEN** profile reconciliation and L2 startup fail before credentials are read by the runtime

### Requirement: L2 restart readiness
The daemon SHALL not report ready for L2-owned work until the adapter, pinned Prodex gateway, shared Postgres state, kill switch, event stream, policy, and approved profiles pass readiness checks.

#### Scenario: Complete restart
- **WHEN** the daemon restarts with valid configuration and approved profiles
- **THEN** it launches the adapter, applies policy, registers only approved profile references, and exposes `runtime_router_owner=rust_l2`

#### Scenario: Dependency is unavailable
- **WHEN** the adapter, gateway, Postgres, policy apply, account registration, kill switch, or event stream is unavailable
- **THEN** readiness fails closed and no new L2-owned task starts

### Requirement: Redacted operational visibility
The system SHALL expose the effective Prodex version, adapter readiness, configuration-source label, approved profile count, rejected profile count, and runtime authority without exposing raw credentials or secret-bearing paths.

#### Scenario: Operator verifies restart
- **WHEN** an operator inspects runtime health after restart
- **THEN** the output shows whether Prodex/L2 is active and reconciled using only redacted or opaque identifiers

# Live Recovery & Daemon Control Specification

## ADDED Requirements

### Requirement: Phase-1 Bounded Rollback
The ORQ2 daemon SHALL enforce bounded rollback to artifact SHA-256 `88ca4f39` when daemon regressions occur, preserving AGY allowlist slots 162, 163, 168, and 169.

#### Scenario: Bounded Rollback Execution
- **GIVEN** an invalid daemon binary deployment or runtime regression
- **WHEN** Phase-1 rollback is triggered under queue SHARE lock
- **THEN** the daemon target binary is reverted to `88ca4f39` with NRestarts=0 and `/readyz` returning 200 OK.

### Requirement: Credential Isolation & Content-Free Incident Recording
The system SHALL isolate test harness executions from ambient environment and record incidents content-free without printing secrets.

#### Scenario: Incident Recording
- **GIVEN** a credential exposure or isolation breach incident (ORQ-64)
- **WHEN** incident evidence or documentation is recorded
- **THEN** zero plaintext secret values or tokens are included in logs, diffs, or markdown artifacts.

### Requirement: Combined Daemon Requirement (ORQ-66)
The ORQ2 daemon SHALL combine token-only AGY task-home handling and thinking-level reasoning admission in a single deterministic binary.

#### Scenario: Combined Daemon Verification
- **GIVEN** a clean build of `cmd/multica`
- **WHEN** compiled across distinct `GOCACHE` directories
- **THEN** identical SHA-256 binary checksums and passing reasoning unit tests are produced.

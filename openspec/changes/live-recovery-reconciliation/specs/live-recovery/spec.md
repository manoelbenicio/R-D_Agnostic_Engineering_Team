# Live Recovery & Daemon Control Specification

## ADDED Requirements

### Requirement: Phase-1 Bounded Rollback & Health Probes
The ORQ2 daemon SHALL enforce bounded rollback to artifact SHA-256 `sha256:88ca4f3900000000000000000000000000000000000000000000000000000000` under admission freeze (`LOCK TABLE agent_task_queue IN SHARE MODE`) using UTC DB snapshot method, and verify health via `127.0.0.1:19514/health` and readiness via `127.0.0.1:18080/readyz`.

#### Scenario: Bounded Rollback Execution
- **GIVEN** an invalid daemon process restart or runtime regression
- **WHEN** Phase-1 rollback is triggered under queue freeze (`LOCK TABLE agent_task_queue IN SHARE MODE`)
- **THEN** the daemon binary is restored to `sha256:88ca4f3900000000000000000000000000000000000000000000000000000000`, active allowlists are enabled (AGY 162/163/168/169, Kiro 139/140/143/149, Codex 152/170), `NRestarts=0`, and `/health` and `/readyz` return 200 OK.

### Requirement: Credential Isolation & Content-Free Incident Recording
The system SHALL isolate test harness executions from ambient systemd environment and record incidents content-free without printing secrets.

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

### Requirement: Production Deployment Audit (ORQ-58)
The system SHALL track production deployment revisions and container image digests.

#### Scenario: Production Deployment Tracking
- **GIVEN** a completed production deployment (ORQ-58)
- **WHEN** deployment state is recorded
- **THEN** live revision `112e8dada455b4e7a3400e63728e00e6e3a0aa27`, live image `sha256:e14f5c35d0640ec4c955efd8d5bbbb6cf219a4d544faafc1fd0f028c5bafcf13`, rollback revision `8227241`, rollback image `sha256:922b13862036d906a1ad5cde3e1615adad45393edb13896d2b146753b8384ab6`, and canary result ORQ-70 OK are recorded.

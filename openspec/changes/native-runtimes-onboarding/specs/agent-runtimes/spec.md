# agent-runtimes

## ADDED Requirements

### Requirement: Native Cline runtime
The system SHALL provide a native Cline 3.x agent runtime driving `cline --acp` and
exchanging ACP JSON-RPC 2.0 messages over stdin/stdout. It SHALL NOT combine `--json` with
`--acp`: `--json` selects Cline's separate headless prompt-output mode and is incompatible
with the ACP handshake.

#### Scenario: Cline runtime available
- **WHEN** the `cline` CLI is installed (`cline 3.0.46` present) and the daemon is online
- **THEN** the `cline` runtime SHALL appear in the runtime list with its models and effort levels

#### Scenario: Cline starts in ACP mode
- **WHEN** the daemon launches a Cline 3.x task
- **THEN** the final argv SHALL contain `--acp`, SHALL exclude `--json`, and the session SHALL
  exchange ACP JSON-RPC messages through stdin/stdout

## REMOVED Requirements

### Requirement: Native NVIDIA NIM runtime
NVIDIA NIM was **deliberately removed** from canonical source per owner decision (verified zero `nim.go` or `nim_home.go` files at production SHA `15626386da2725af8e8d4ac611754cffe359fe31`). All obsolete NIM implementation and deployment claims/tasks are SUPERSEDED. Any NIM revival requires a new owner-approved proposal.

#### Scenario: NIM runtime removed
- **WHEN** inspecting canonical production source
- **THEN** zero `nim.go` or `nim_home.go` files SHALL exist and NIM is superseded

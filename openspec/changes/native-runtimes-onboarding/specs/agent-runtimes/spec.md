# agent-runtimes

## ADDED Requirements

### Requirement: Native NVIDIA NIM runtime
The system SHALL provide a native `nim` agent runtime implemented from scratch as an
OpenAI-compatible backend, without routing through opencode.

#### Scenario: NIM runtime available and selectable
- **WHEN** the NIM CLI/credentials are present and the daemon is online
- **THEN** the `nim` runtime SHALL appear in the runtime list and expose its models

#### Scenario: NIM executes a task with usage
- **WHEN** an agent bound to the `nim` runtime is assigned a task
- **THEN** the daemon SHALL execute it and record token usage from `usageMetadata`

### Requirement: Native Cline runtime
The system SHALL provide a native Cline 3.x agent runtime driving `cline --acp` and
exchanging ACP JSON-RPC 2.0 messages over stdin/stdout. It SHALL NOT combine `--json` with
`--acp`: `--json` selects Cline's separate headless prompt-output mode and is incompatible
with the ACP handshake.

#### Scenario: Cline runtime available
- **WHEN** the `cline` CLI is installed and the daemon is online
- **THEN** the `cline` runtime SHALL appear in the runtime list with its models and effort levels

#### Scenario: Cline starts in ACP mode
- **WHEN** the daemon launches a Cline 3.x task
- **THEN** the final argv SHALL contain `--acp`, SHALL exclude `--json`, and the session SHALL
  exchange ACP JSON-RPC messages through stdin/stdout

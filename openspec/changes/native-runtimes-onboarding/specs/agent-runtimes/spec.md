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
The system SHALL provide a native `cline` agent runtime driving `cline --acp --json`.

#### Scenario: Cline runtime available
- **WHEN** the `cline` CLI is installed and the daemon is online
- **THEN** the `cline` runtime SHALL appear in the runtime list with its models and effort levels

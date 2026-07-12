# chat-orchestration

## ADDED Requirements

### Requirement: TL/Manager is the default chat/task responder
The system SHALL, by default, route an incoming chat or task without an explicit target to a
TL/Manager (squad leader) rather than to a worker agent.

#### Scenario: Untargeted chat goes to the TL
- **WHEN** a user sends a chat/task without naming a specific agent
- **THEN** the system SHALL route it to the workspace's default TL/Manager squad leader

### Requirement: TL clarifies, documents, plans, and delegates
The TL/Manager SHALL clarify open questions, optionally open an OpenSpec explore to document,
plan the work, and delegate to the involved agents, then synthesize the result. A delegation-only
leader SHALL NOT produce the work itself.

#### Scenario: Leader delegates after clarifying
- **WHEN** the TL receives a task and questions are resolved
- **THEN** the TL SHALL plan and delegate sub-tasks to the involved member agents and synthesize the outcome

### Requirement: Direct-to-agent escape hatch preserved
The system SHALL allow a user to address a specific agent/runtime directly for a one-off task,
bypassing the TL.

#### Scenario: Direct mention bypasses the TL
- **WHEN** a user addresses a specific agent (e.g. `@codex`)
- **THEN** the system SHALL route the task directly to that agent without going through the TL

### Requirement: OpenSpec documentation is mandatory
OpenSpec SHALL be the mandatory documentation path for the project. Work SHALL be backed by an
OpenSpec change, and default chat/task routing SHALL always be by squad (to the TL/Manager).

#### Scenario: From-scratch work without docs is gated
- **WHEN** a from-scratch project/task has no OpenSpec documentation
- **THEN** the TL/Manager SHALL first initiate the OpenSpec documentation (explore/proposal), and work SHALL NOT proceed until that documentation exists

# model-discovery

## MODIFIED Requirements

### Requirement: Reliable runtime model discovery
The system SHALL populate the runtime model list in the UI reliably, applying a timeout and
cache so a slow CLI (e.g. `agy models`) does not leave the UI empty.

#### Scenario: Model list populates within a bounded time
- **WHEN** the UI requests models for an online runtime
- **THEN** the system SHALL return the model list within a bounded timeout or a clear error, and cache the result

#### Scenario: Slow CLI does not block the UI
- **WHEN** a provider CLI is slow to enumerate models
- **THEN** the system SHALL not leave the UI indefinitely empty; it SHALL surface progress or a cached list

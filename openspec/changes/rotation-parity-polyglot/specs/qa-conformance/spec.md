## ADDED Requirements

### Requirement: Exhaustive Conformance Before Deploy
The system SHALL run all conformance gates C1–C6 exhaustively with real evidence in a container, before PROD. No gate may be marked done as plan-only or dry-run. Container/sidecar validation MUST NOT require the live PROD runtime (breaks the circular dependency).

#### Scenario: Conformance by capability
- **WHEN** C1 runs
- **THEN** each provider capability SHALL be validated by behavior (not by marketing label) with captured evidence

#### Scenario: Replay coverage
- **WHEN** C2/C3 run
- **THEN** long-session, tool-calls, previous_response_id, compact, SSE and WebSocket SHALL be replayed and pass

#### Scenario: Fail-closed profile switch
- **WHEN** C4 runs and a profile switch encounters an invalid new profile
- **THEN** the system SHALL fail closed and never reuse the previous credential

#### Scenario: Smart Context validated
- **WHEN** C5 runs
- **THEN** Smart Context SHALL be measured shadow→canary→live with before/after metrics and an automatic exact fallback on structural/protocol risk

#### Scenario: Isolation without clobber
- **WHEN** C6 runs the triple CODEX_HOME × prodex × Herdr coexistence
- **THEN** account/profile isolation SHALL hold with no auth clobber (AccountHome mandatory)

### Requirement: Evidence-Gated Done
The system SHALL treat a task as done only with reproducible container/PROD evidence. Self-reported completion without evidence MUST be reclassified as in-progress.

#### Scenario: No trust in the tail
- **WHEN** an agent reports a gate DONE
- **THEN** the validator SHALL re-run the check and require scrubbed evidence before accepting done

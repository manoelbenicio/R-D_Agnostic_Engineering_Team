## ADDED Requirements

### Requirement: Tested Kill-Switch and Rollback Before Deploy
The system SHALL prove the kill-switch and rollback work by real test (not documentation) before deploying to PROD. Deploy MUST be blocked until both are green.

#### Scenario: Kill-switch tested
- **WHEN** the deploy phase begins
- **THEN** the kill-switch SHALL be exercised per tenant/provider/profile and MUST verifiably disable Smart Context/gateway/auto-redeem

#### Scenario: One-command rollback tested
- **WHEN** the deploy phase begins
- **THEN** a single-command rollback to raw `codex` SHALL be executed and verified to restore prior behavior

### Requirement: Direct-to-PROD Deploy Gated by Exhaustive QA
The system SHALL deploy prodex AS-IS directly to PROD (no canary/staging) ONLY after exhaustive QA is green in a container AND kill-switch/rollback are tested AND logs are scrubbed. QA MUST NEVER be bypassed.

#### Scenario: QA precedes deploy
- **WHEN** deploy is requested
- **THEN** all conformance gates (C1–C6) MUST be green with scrubbed evidence in a container before any PROD deploy proceeds

#### Scenario: No secret in logs
- **WHEN** the system emits logs/traces/errors/audit in the PROD path
- **THEN** no secret/token/key SHALL appear (redaction verified)

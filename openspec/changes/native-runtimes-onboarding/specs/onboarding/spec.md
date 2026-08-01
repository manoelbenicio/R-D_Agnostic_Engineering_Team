# onboarding

## MODIFIED Requirements

### Requirement: Frictionless onboarding without marketing or email-code
The system SHALL present a clean login consistent with the app design-system (same colors as
kanban/agents) and SHALL NOT show the marketing/sponsors landing nor the email verification-code flow.

#### Scenario: User reaches a clean login
- **WHEN** an unauthenticated user opens the app
- **THEN** the system SHALL show the app-styled login and SHALL NOT show sponsors/marketing content

#### Scenario: No email verification-code step
- **WHEN** a user authenticates
- **THEN** the system SHALL NOT require an emailed verification code

### Requirement: Simple username/password login (Firebase-ready)
The onboarding SHALL use a simple username/password login now, styled per the app
design-system, and SHALL be structured so Firebase auth can be added later without rework.

#### Scenario: Simple login accepted
- **WHEN** a user submits valid username/password on the app-styled login
- **THEN** the system SHALL authenticate them without email code or sponsors content

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

### Requirement: Canonical production HTTPS access
The current production Multica UI and user-facing API SHALL use the single canonical origin
`https://orq1.tail96e2c0.ts.net` through Tailscale Serve on HTTPS port 443. Normal access SHALL NOT
require an SSH tunnel and SHALL NOT use the internal `13100` or `18080` ports, a Tailscale IP, or
localhost as the browser origin.

#### Scenario: User or agent opens Multica
- **WHEN** a tailnet-authorized user or approved browser client needs the Multica frontend
- **THEN** it SHALL open `https://orq1.tail96e2c0.ts.net` or its `/login` route without creating an SSH tunnel

#### Scenario: Internal ports are encountered in diagnostics
- **WHEN** documentation or health checks mention frontend port `13100` or backend port `18080`
- **THEN** agents SHALL treat them as ORQ1 loopback implementation details, not client connection URLs

### Requirement: Secret-safe privileged owner login
The current deployed password login SHALL use the request fields `email` and `password` at
`POST /auth/login`. The privileged owner credential authority SHALL be AWS Secrets Manager account
`809809509961`, Region `sa-east-1`, secret `prod/multica/bootstrap-owner`, confirmed full ARN
`arn:aws:secretsmanager:sa-east-1:809809509961:secret:prod/multica/bootstrap-owner-3BIwjo`, JSON keys
`email,password`, and version stage `AWSCURRENT`. This credential authenticates the application
user but SHALL NOT be treated as creating a workspace, role, or membership grant; effective frontend
access depends on the user's application-level roles and memberships.

Secret values SHALL NOT enter agent context, logs, command output, screenshots, evidence, or Git.
Authorized automation SHALL use unresolved `{{resolve:secretsmanager:...}}` references through
`asm-exec` into a reviewed secret-safe child runner. Ordinary Multica tasks SHALL use their injected
`mat_` task identity and SHALL NOT use the privileged owner credential.

#### Scenario: Human performs owner login
- **WHEN** an owner-authorized human opens the canonical `/login` route
- **THEN** the human SHALL obtain the email/password through the approved owner-controlled secret workflow and enter them only into the HTTPS form

#### Scenario: Agent automation performs an authorized login check
- **WHEN** an agent has explicit authorization to validate authenticated access
- **THEN** it SHALL resolve the `email` and `password` fields only inside an approved `asm-exec` child runner and SHALL emit no secret, cookie, JWT, CSRF value, Authorization header, or environment dump

#### Scenario: Agent sees an authentication failure after `/login` loads
- **WHEN** the canonical login page is reachable but the credential is rejected
- **THEN** the agent SHALL troubleshoot credential authority, provisioning, rotation, or membership and SHALL NOT create an SSH route as a workaround

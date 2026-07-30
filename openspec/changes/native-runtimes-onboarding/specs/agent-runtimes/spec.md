# agent-runtimes

## ADDED Requirements

### Requirement: Owner-gated Cline architecture

The system SHALL keep Cline disabled in production until the owner explicitly selects either
(A) credentialless Agent Brain with OmniRoute-only inference or (B) a native
credential-isolated Cline account, and the selected option's prerequisites are verified.
Existing candidate source SHALL NOT constitute an architecture decision or rollout approval.

#### Scenario: Architecture decision is unresolved

- **WHEN** neither option A nor option B has explicit owner approval
- **THEN** production SHALL keep Cline disabled and SHALL NOT report the runtime as online or
  accepted

#### Scenario: Option A is selected

- **WHEN** the owner selects credentialless Agent Brain with OmniRoute-only inference
- **THEN** enablement SHALL remain blocked until the inference-key lifecycle is operational and
  an exact model route is authorized

#### Scenario: Option B is selected

- **WHEN** the owner selects a native credential-isolated Cline account
- **THEN** enablement SHALL remain blocked until account/login authority and credential
  isolation are established

### Requirement: Cline ACP transport

When an approved architecture uses the native Cline 3.x ACP backend, the system SHALL launch
`cline --acp` and exchange ACP JSON-RPC 2.0 over stdin/stdout. It SHALL NOT combine `--json`
with `--acp`, because `--json` selects a separate headless prompt-output mode and is
incompatible with the ACP handshake.

#### Scenario: Approved native Cline task starts

- **WHEN** the daemon launches a Cline 3.x task after the architecture and rollout gates pass
- **THEN** the final argv SHALL contain `--acp`, SHALL exclude `--json`, and the session SHALL
  exchange ACP JSON-RPC messages through stdin/stdout

### Requirement: Runtime claims require matching evidence

The system documentation SHALL distinguish source, test, candidate, live, and acceptance
evidence, and SHALL NOT use evidence from one layer to claim a higher layer.

#### Scenario: Candidate source and tests exist

- **WHEN** Cline source, wiring, isolation, and tests exist only in a candidate revision
- **THEN** documentation SHALL describe them as candidate evidence and SHALL NOT claim Cline is
  deployed, online, smoke-tested in production, or accepted

#### Scenario: Production runtime is claimed online

- **WHEN** documentation claims Cline is online in production
- **THEN** it SHALL cite the exact live revision, enabled configuration, successful bounded
  production canary, token-usage evidence, and acceptance result

### Requirement: Native NIM remains outside this change

The system SHALL NOT introduce or claim a native `nim` runtime through this change. NVIDIA
model access remains OmniRoute-owned unless a separate owner-approved proposal establishes
native transport and runtime evidence.

#### Scenario: Native NIM has no separate approved proposal

- **WHEN** runtime capabilities are documented or exposed
- **THEN** they SHALL NOT include a native `nim` runtime based on this change

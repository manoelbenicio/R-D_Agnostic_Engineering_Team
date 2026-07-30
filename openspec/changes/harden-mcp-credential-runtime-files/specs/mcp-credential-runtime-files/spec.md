## ADDED Requirements

### Requirement: Canonical caller-owned filesystem boundary
The installer SHALL accept only a canonical root owned by the caller. A custom TMPDIR MUST be a canonical
descendant of that root. Existing final targets MUST be caller-owned regular, non-symlink, single-link files.

#### Scenario: Safe filesystem boundary
- **WHEN** root, TMPDIR, and final targets satisfy ownership, type, link-count, and containment checks
- **THEN** the installer may continue without writing outside the canonical root

#### Scenario: Multiply-linked target
- **WHEN** an existing final target has a link count other than one
- **THEN** the installer fails with `E_TARGET_MULTIPLY_LINKED` and preserves every linked name without mutation

### Requirement: Atomic private replacement
The installer SHALL render through an unpredictable `0600` temporary file in the same directory as the final
target and SHALL replace the target with an atomic rename only after validation.

#### Scenario: Managed file replacement
- **WHEN** a validated managed target is applied
- **THEN** the temporary file is private, unpredictable, same-directory, cleaned on failure, and atomically renamed

### Requirement: Strict unit and dry-run contract
Each unit name MUST match the exact `*.service` grammar. Dry-run SHALL be the default. The installer SHALL NOT
invoke `systemctl`, reload, or restart.

#### Scenario: Default invocation
- **WHEN** the installer is invoked without apply or rollback mode
- **THEN** it reports the plan without writing or invoking service-management operations

### Requirement: Exact apply and rollback tuple
Apply and rollback MUST reuse the exact canonical root, TMPDIR, and unit tuple. An omitted or changed custom
TMPDIR SHALL fail closed when managed bytes no longer match and SHALL preserve all files.

#### Scenario: Matched rollback tuple
- **WHEN** rollback repeats the exact root, custom TMPDIR, and unit values used by apply
- **THEN** byte-exact managed files are eligible for removal

#### Scenario: Changed custom TMPDIR
- **WHEN** rollback omits or changes the custom TMPDIR used by apply
- **THEN** rollback fails closed and preserves the managed files

### Requirement: Accepted source remains unapplied
The system of record SHALL identify `45cefdf1afe6e9e7a2cad086f4835f353415a07f` as the accepted original and
`7dd6f7df2cf1285c43ab2a0934d511189d1babb6` as the accepted isolated port on parent
`edd7b932f7c44c3396fd87853c2795d746fc5134`. It SHALL NOT represent the port as merged to main, applied,
deployed, restarted, reloaded, or production-canary accepted.

#### Scenario: Source status is reported
- **WHEN** the ORQ-37 hardening status is reviewed
- **THEN** `checkpoint/20260730/orq37-source-port` is reported as accepted source only and runtime gates remain open

### Requirement: Runtime cutover remains externally gated
Runtime application MUST remain blocked until secret identity and ownership, metadata-only resolution, queue and
admission coordination, exact tuple, rollback, runtime verification, and bounded canary gates are complete.
Plaintext secret retrieval SHALL NOT be used. ORQ-58 MUST remain separate.

#### Scenario: Any external gate is incomplete
- **WHEN** one or more owner/external tasks remain unchecked
- **THEN** no apply, deploy, reload, restart, credential-mode canary, or MCP canary is authorized

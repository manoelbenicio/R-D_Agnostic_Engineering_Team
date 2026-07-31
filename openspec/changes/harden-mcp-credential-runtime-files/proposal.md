## Why

MCP credential runtime files need a canonical, reviewable contract that records the accepted hardening source
without implying that it is live. The source port is accepted but unapplied, and runtime enablement remains gated
by external ownership, secret-resolution, queue/admission, rollback, and canary evidence.

## What Changes

- Specify fail-closed filesystem, unit-name, temporary-file, apply, and rollback invariants for the user-scoped
  hardening installer.
- Record the exact accepted source provenance, blobs, modes, and validation evidence.
- Preserve the distinction between an accepted source port and production application or acceptance.
- Keep all owner/external runtime gates unchecked and keep ORQ-58 separate.

## Capabilities

### New Capabilities

- `mcp-credential-runtime-files`: Contract for safely creating and rolling back MCP credential runtime-file
  hardening, including provenance and operational gates.

### Modified Capabilities

None.

## Impact

This documentation affects only the OpenSpec change
`openspec/changes/harden-mcp-credential-runtime-files/`. It does not change code, existing changes, services,
credentials, AWS, PostgreSQL, deploy state, or production runtime state.

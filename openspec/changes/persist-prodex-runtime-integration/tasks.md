> **SCOPE (Wave A, D-V3-16): default-OFF cold-recovery-only.** Prodex is never per-request,
> never automatic, never simultaneously hot with OmniRoute. `MULTICA_PRODEX_REQUIRED` defaults
> to `0`; recovery mode is an explicit operator transition gated by the platform recovery-mode
> state machine (AB-REQ-41). Checkbox states preserved (0/16); no product code changed in Wave A.

## 1. Configuration and executable separation

- [ ] 1.1 Add and validate `MULTICA_L2_SIDECAR_PATH` independently from the pinned `MULTICA_PRODEX_PATH`
- [ ] 1.2 Make the Go lifecycle execute the adapter binary while passing the pinned Prodex path through its environment
- [ ] 1.3 Add `MULTICA_PRODEX_REQUIRED` as an operator recovery toggle defaulting to `0` (OFF); when explicitly set for recovery mode it prevents a silent downgrade, but the target-path default remains OmniRoute-primary and never auto-enables Prodex

## 2. Durable startup

- [ ] 2.1 Add a secure persistent launcher/service that imports the mode-0600 Prodex EnvironmentFile, **disabled by default** and started only when recovery mode is explicitly enabled by an operator
- [ ] 2.2 Extend the local persistent Prodex environment with L2 endpoint, adapter, Postgres, and redacted secret settings
- [ ] 2.3 Add operator-visible configuration-source and effective runtime-authority health fields

## 3. Profile reconciliation

- [ ] 3.1 Use the current validated Multica account inventory as the sole mapping from Prodex profile names to isolated Codex slot homes
- [ ] 3.2 Implement audit and reconciliation using `prodex profile add --codex-home` without copying credentials
- [ ] 3.3 Enforce POSIX filesystem, directory/file modes, approved-root containment, credential presence, and duplicate-identity rejection
- [ ] 3.4 Register only reconciled profile references through `rpp.l2.v1` and reject global/cross-slot fallback
- [ ] 3.5 Purge unreferenced legacy credential files and obsolete unassigned account records while preserving non-credential agent state

## 4. Verification

- [ ] 4.1 Add Go tests for separate executable launch, required-mode failure, profile registration, and single-router preservation
- [ ] 4.2 Add reconciliation tests for idempotency, path mismatch, unsafe permissions/filesystem, and duplicate credentials
- [ ] 4.3 Build and attest the Prodex adapter and Multica daemon
- [ ] 4.4 Restart the daemon and verify adapter readiness, Postgres readiness, reconciled profiles, and `rust_l2` authority
- [ ] 4.5 Update operational documentation and record scrubbed evidence

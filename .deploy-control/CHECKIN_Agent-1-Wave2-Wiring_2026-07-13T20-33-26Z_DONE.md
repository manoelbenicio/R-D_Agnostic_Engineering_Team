# Agent-1 WAVE 2 Wiring - DONE

- UTC: 2026-07-13T20:33:26Z
- Task: T1 — native NVIDIA NIM and Cline runtime wiring.
- Delivered: `nim` and `cline` factory/support entries, Cline CLI probe (`MULTICA_CLINE_PATH`, `cline`, `MULTICA_CLINE_MODEL`), credential-gated NIM native HTTP availability, NIM native runtime registration, and NIM credential isolation.
- Database: migration 126 extends the runtime profile protocol-family constraint for the new supported providers.
- Validation: containerized focused Go tests passed for `pkg/agent` and `internal/daemon`; `git diff --check` passed.
- Constraint verified: NIM has no CLI probe or invented executable.

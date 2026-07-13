# Agent-1 WAVE 2 Wiring - START

- UTC: 2026-07-13T18:56:29Z
- Task: T1 — wire native NVIDIA NIM and Cline runtime discovery/factory support.
- Scope: add both providers to the agent factory and supported types; discover Cline from its CLI and NIM only from `NVIDIA_API_KEY`; require NIM credential isolation.
- Constraint: NIM remains native HTTP and does not probe or register an invented binary.
- Validation: focused Go tests followed by the containerized server gate.
- Existing worktree changes are unrelated and will not be modified or reverted.

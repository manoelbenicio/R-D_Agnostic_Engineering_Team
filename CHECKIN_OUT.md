# HerdMaster — Check-In / Check-Out Log

> **MANDATORY.** Every agent (including the Tech Lead) MUST write a CHECK-IN entry before
> starting work and a CHECK-OUT entry after finishing. This is the single source of truth
> for who is working on what, and the only file any agent may write inside
> `Automonous_Agentic`.
>
> **Rules enforced here:**
> - No two agents may have overlapping `Files Touched` scopes while both are `STARTING`.
> - No agent runs `git add/commit/push` — only the Tech Lead commits, after validation.
> - All HerdMaster code lives under `/mnt/c/VMs/Projetos/HerdMaster/` ONLY.

## Roster

| Agent | Model | Role |
|-------|-------|------|
| GLM-5.2 / Tech Lead | GLM 5.2 / Kimi K2.7 | Principal Architect & Manager (orchestration only, no impl code) |
| Codex-1 | Codex | Sr. SME R&D coding agent |
| Codex-2 | Codex | Sr. SME R&D coding agent |
| Codex-3 | Codex | Sr. SME R&D coding agent |
| Nemotron-1 | Nemotron 3 Ultra 550B | Sr. coding agent |
| Nemotron-2 | Nemotron 3 Ultra 550B | Sr. coding agent |
| Kiro | Opus 4.8 | Architecture support |

## Check-In / Out Entries

| Timestamp (UTC) | Agent | Action | Task ID | Files Touched | Status |
|-----------------|-------|--------|---------|---------------|--------|
| 2026-06-21T18:15:00Z | GLM-5.2 / Tech Lead | CHECK-IN | LEAD-001 | N/A (management: read docs, build dependency graph, wave plan, agent prompts) | STARTING |

| 2026-06-21T19:42:00Z | GLM-5.2 / Tech Lead | NOTE | LEAD-001 | git init on /mnt/c/VMs/Projetos/HerdMaster (branch main); baseline commit 10ee85c (orchestration plan + 4 wave prompt files) | IN-PROGRESS |
| 2026-06-21T19:42:00Z | GLM-5.2 / Tech Lead | DISPATCH | Wave 1 | HM-001, HM-002, HM-003, HM-004, HM-000 prompts released to human for manual pane dispatch | DISPATCHED |

## Dispatch Log (Tech Lead)

| Timestamp (UTC) | Wave | Task | Agent | File Scope (exclusive) | Dispatched |
|-----------------|------|------|-------|------------------------|------------|
| 2026-06-21T18:15:00Z | 1 | HM-001 | Codex-1 | src/herdmaster/db/{__init__,schema,repositories}.py | READY (prompt in WAVE1_PROMPTS.md) |
| 2026-06-21T18:15:00Z | 1 | HM-002 | Codex-2 | src/herdmaster/bus/{__init__,messages}.py | READY |
| 2026-06-21T18:15:00Z | 1 | HM-003 | Codex-3 | src/herdmaster/herdr/{__init__,adapter,parser}.py | READY |
| 2026-06-21T18:15:00Z | 1 | HM-004 | Nemotron-1 | src/herdmaster/config.py | READY |
| 2026-06-21T18:15:00Z | 1 | HM-000 | Nemotron-2 | pyproject.toml, README.md, src/herdmaster/{__init__,__main__}.py, config/herdmaster.example.toml, .gitignore, dir tree | READY |

> Wave 2 is gated: starts only after all five Wave 1 tasks CHECK-OUT as COMPLETED and the Tech Lead validates + commits. Conflict check: all five Wave 1 scopes are disjoint. ✅

### Full wave prompt set (all written, ready for manual dispatch)

| Wave | Prompt file | Tasks | Gate |
|------|-------------|-------|------|
| 1 | HerdMaster/WAVE1_PROMPTS.md | HM-001, HM-002, HM-003, HM-004, HM-000 | none (foundational) |
| 2 | HerdMaster/WAVE2_PROMPTS.md | HM-005, HM-006, HM-007, HM-008, HM-009 | Wave 1 validated + committed |
| 3 | HerdMaster/WAVE3_PROMPTS.md | HM-010, HM-011, HM-013, HM-012, HM-015(p1 diagrams) | Wave 2 validated + committed |
| 4 | HerdMaster/WAVE4_PROMPTS.md | HM-014a/b/c/d (tests), HM-015(p2 docs) | Wave 3 validated + committed |

> User dispatches each wave manually. Tech Lead (GLM-5.2) runs the validation gate and performs ALL
> git commits after each wave checks out. Wave N+1 prompts are released only after Wave N is committed.
| 2026-06-21T19:46:35Z | Codex-1 | CHECK-IN | HM-001 | src/herdmaster/db/__init__.py, db/schema.py, db/repositories.py | STARTING |
| 2026-06-21T19:46:54Z | Codex-2 | CHECK-IN | HM-002 | src/herdmaster/bus/__init__.py, bus/messages.py | STARTING |
| 2026-06-21T19:48:15Z | Codex-3 | CHECK-IN | HM-003 | src/herdmaster/herdr/__init__.py, herdr/adapter.py, herdr/parser.py | STARTING |
| 2026-06-21T19:50:20Z | Codex-2 | CHECK-OUT | HM-002 | src/herdmaster/bus/__init__.py, src/herdmaster/bus/messages.py | COMPLETED | MessageType, Message schema, JSON-RPC serialization, TTL expiry, addressing helpers, factory, and validation implemented; round-trip self-check passed |
| 2026-06-21T19:50:41Z | Codex-1 | CHECK-OUT | HM-001 | src/herdmaster/db/__init__.py, src/herdmaster/db/schema.py, src/herdmaster/db/repositories.py | COMPLETED |
| 2026-06-21T19:53:36Z | Codex-3 | CHECK-OUT | HM-003 | src/herdmaster/herdr/__init__.py, src/herdmaster/herdr/adapter.py, src/herdmaster/herdr/parser.py | COMPLETED | Herdr parser dataclasses, tolerant JSON parsers, stable output hash, async CLI adapter, HerdrError wrapping, arg-list subprocess boundary, and parser smoke checks implemented |
| 2026-06-21T20:00:00Z | Nemotron-2 | CHECK-IN | HM-000 | pyproject.toml, README.md, src/herdmaster/__init__.py, src/herdmaster/__main__.py, config/herdmaster.example.toml, .gitignore, dir tree | STARTING |

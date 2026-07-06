# PROJECT — Rotation-Parity Polyglot (RPP)

author: Kiro/Principal (Opus 4.8)
current_milestone: v2.1 (Vendor Validation + PROD Deploy)

## What this project is
A polyglot LLM runtime (prodex L2 sidecar + gateway) that provides real Smart Context compaction on the
real traffic path: `/v1/session/start` (returns runtime_endpoint) → `/v1/runtime/proxy` (real compaction).
Compaction lives in the prodex Rust crates (tokensavior/clawcompactor/sqz/context/compact-output), NOT the
OpenAI-compat gateway.

## The 4 real vendors (what our agents actually run on)
| Vendor | Backing models / agent |
|:--|:--|
| OpenAI | Codex |
| Antigravity | Opus 4.6 / Gemini 3.5 Flash / Gemini 3.1 PRO |
| OpenCode | GLM 5.2 (IN ACTIVE USE — not archived) |
| Kiro | Opus 4.8 |

## Non-negotiables (owner)
1. No scope reduction without owner sign-off. OpenSpec-agreed scope is the requirement.
2. Kiro personally authors ALL GSD/planning docs on disk. Agents never author planning artifacts.
3. Every task an agent runs MUST be a task-ID in a PLAN.md on disk, with a Golden-Rule check-in.
4. All evidence MUST satisfy EVIDENCE_CONTRACT.md. Fabrication = rejected + INVALID + escalate.
5. GSD planning always written to disk (never held in memory).

## Roles
- Kiro/Principal: owns/authors GSD docs; drives TL; independently verifies against disk/git; never trusts "DONE".
- TL (w3:pW, Claude Opus 4.6): orchestrates the fleet; validates evidence; executes/coordinates.
- Codex agents (w3:pJ/pK/p9/pM): write product code + run tasks. Only one owner per hotspot.

## Source of truth
Canonical repo: manoelneto-laptop:/mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team
(remote R-D_Agnostic_Engineering_Team.git). Orchestration host repo is SECONDARY.

## Artifacts index (.planning/)
PROJECT.md · STATE.md · ROADMAP.md · EVIDENCE_CONTRACT.md · MILESTONE_v2.1.md
phases/11-vendor-validation/{SPEC,PLAN}.md
phases/12-prod-deploy/{SPEC,RESEARCH,PLAN,PREREQUISITES}.md

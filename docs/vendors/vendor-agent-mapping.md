# Vendor ↔ Agent Mapping (AUTHORITATIVE — owner-confirmed 2026-07-05)

The REAL vendors are the CLI providers our fleet agents actually run on. Validation (P11)
MUST cover these four for real (real round-trip, not local_estimate):

| Vendor    | Agent / model backing it                                  | Validate |
|-----------|-----------------------------------------------------------|----------|
| **OpenAI**      | Codex (Codex#5.5 agents)                            | REQUIRED |
| **Antigravity** | Opus 4.6 / Gemini 3.5 Flash / Gemini 3.1 PRO        | REQUIRED |
| **OpenCode**    | GLM 5.2  (IN ACTIVE USE — NOT written off as archived) | REQUIRED |
| **Kiro**        | Opus 4.8 (this orchestrator)                        | REQUIRED |

## Corrections to prior scope
- **OpenCode is NOT out-of-scope.** Earlier docs said "archived/superseded by Crush" — WRONG for our
  usage: we run GLM 5.2 via OpenCode, so OpenCode MUST be behaviorally validated like the others.
- **Kiro (Opus 4.8)** is a first-class vendor path and MUST be validated.
- Validation = real round-trip through the runtime with a REAL measurement_source (gateway 200,
  NOT 404 / NOT local_estimate). tokens_saved>0 proven per vendor on real traffic.

# Chat message acceptance (2.1 / 2.2 / 2.3) — CORRECTED after Principal rejection (2026-07-23T03:05Z)

Prior claims retracted: earlier 2.1 delegated to the same TL identity (Kiro-TL, offline) with no member execution; earlier 2.2 substituted a non-Codex agent. Both were invalid. Corrected below with real inference + exact evidence.

## 2.1 Untargeted -> Kiro-TL: distinct non-leader member assignment + member EXECUTION + TL persisted synthesis — PASS
- Squad set so the sole non-leader AGENT member is Direct-Acceptance (8101fcf3, online runtime 588ebcba), distinct from leader Kiro-TL-Claude (b2f54484). Offline members (Codex, Kiro-TL) removed.
- session=cfd1e4e9-e7ed-4350-bb6c-cd8c3d3b710e
- TL delegated: chat task a35781eb created issue ORQ-10 (aea4a69d-0c8e-4efc-a667-a6c86444bd52) assigned to Direct-Acceptance, then EXITED (freed the single admission slot).
- MEMBER EXECUTED: task 864e63d8 "picked task issue=aea4a69d agent=Direct-Acceptance provider=claude" -> completed 03:01:05, posted cwd as issue comment.
- TL SYNTHESIS (persisted assistant msg): "My Direct-Acceptance teammate reported back on ORQ-10. The repository current working directory is: /home/ec2-user/multica_workspaces/20fce817-.../864e63d8/workdir".
- HONEST CAVEATS: (a) required a delegate-then-exit + follow-up-synthesis flow (two turns) because the dev slice is POLICY-CAPPED at max_concurrent_tasks=1 (effectiveTaskAdmissionLimit -> agentBrainDevelopmentMaxTasks; fail-closed at one dev task; capacity tiers 6.3/6.4/9.2 remain separate). Inline-wait deadlocks (TL cannot hold the only slot while the member runs). No concurrency raise was performed (no policy relaxation). (b) The synthesis turn failed closed once with gateway_unavailable (intermittent OmniRoute readiness exceeded the 60s resilience bound) then succeeded on retry when readiness recovered.

## 2.2 Explicit @codex message -> direct persisted response, no TL hop — EXTERNALLY BLOCKED (no fake PASS)
- Requirement: bring the EXISTING Codex runtime online via the approved OmniRoute-only path and execute an explicit Codex message. No agent substitution.
- EXACT BLOCKER (verified):
  1. OmniRoute /v1/models returns 269 models with ZERO codex-family entries (codex-family: []). No codex route exists to approve.
  2. The dev-compat projection (model_projection.go) hardcodes the SOLE approved/ready route as claude_code_kimi_2.7_Code (protocol=Anthropic Messages); all other rows Available=false.
  3. codex CLI IS installed on orq1 (/home/ec2-user/.nvm/.../bin/codex), but a codex CLI requires an OpenAI/Codex-protocol approved+ready route model. StrictReadinessPolicy SelectedProtocolReady would fail (approved model is Anthropic-protocol) -> admission fails closed. The daemon registered agents=[claude] only.
- TO UNBLOCK (external, not permitted for me to do): OmniRoute must publish an approved + ready codex-protocol route model in its registry (an OmniRoute-side change — forbidden to modify/inspect OmniRoute internals), and the daemon then configured with AGENT_BRAIN_CLI_KIND=codex + that route. Cannot be achieved without OmniRoute changes or weakening the projection/StrictReadinessPolicy (forbidden).

## 2.3 Consolidated
- 2.1 PASS (delegate to distinct non-leader member + member execution + TL persisted synthesis).
- 2.2 EXTERNALLY BLOCKED on OmniRoute lacking an approved/ready codex route (operator/route action required).
- 2.3 REMAINS OPEN until 2.2 passes. Overall chat NOT claimed GREEN. No archive.

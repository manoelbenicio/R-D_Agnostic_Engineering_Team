# Tasks

> Execução: coders (codex & cia). Validação de cada entrega: Kiro. Check-in START/DONE por agente.

## Wave 1 — paralela
- [x] 1.1 Agent-1: `server/pkg/agent/nim.go` — backend NIM OpenAI-compatible (SSE, loop agêntico, usageMetadata→TokenUsage) + testes — VALIDADO Kiro (container `pkg/agent` verde)
- [x] 1.2 Agent-2: isolamento/rotação NIM — `execenv/nim_home.go`, `rotation_detector_nim.go`, `rotation/detector_nim.go` + testes
- [x] 1.3 Agent-3: `server/pkg/agent/cline.go` — backend nativo Cline 3.x via `cline --acp` (ACP JSON-RPC 2.0 por stdin/stdout); `--json` é modo separado incompatível e é filtrado, conforme teste direto do argv
- [ ] 1.4 Agent-4: descoberta de modelos — REOPEN. Current candidate-bound `pkg/agent` and daemon model-report/config slices pass, but this bounded remediation did not execute a UI model-list integration assertion; reliable UI population is not yet fully proven.
- [ ] 1.5 Agent-5 onboarding frontend — REOPEN. Focused candidate-bound core/views/web auth and onboarding tests (227 assertions) and all three typechecks pass. The specifically cited `reconciliation/SOURCE_PARITY_EVIDENCE.md` remains absent, so source-parity closure is not claimed.
- [ ] 1.6 Agent-6 design parity/i18n/web QA — REOPEN. Focused frontend tests and typechecks pass, but the production build reaches Next.js and fails while fetching Inter, Geist Mono, and Source Serif 4 with post-install networking disabled. No production-build PASS is claimed.
- [x] 1.7 Agent-1 backend password auth — BOUNDED-DIRECT. The focused candidate-bound offline auth suite passes across `internal/auth`, middleware, password handler helpers, CLI, and server packages; no live auth or credential operation was performed.

## Wave 2 — integração (Kiro)
- [x] 2.1 Wiring `config.go`: probes `nim` e `cline` — BOUNDED-DIRECT. Candidate-bound daemon config assertions execute and pass.
- [x] 2.2 Wiring `agent.go`: `New()` cases + `SupportedTypes` (nim, cline)
- [x] 2.3 `requiresCredentialIsolation` += `nim` — BOUNDED-DIRECT. Candidate-bound daemon assertion executes and passes.
- [x] 2.4a Backend binary build scope — BOUNDED-DIRECT. Scoped vet passes and `go build ./cmd/server ./cmd/multica` succeeds from the checksum-verified task cache. No image, deployment, or runtime acceptance is claimed.
- [ ] 2.4b Daemon restart and online `nim`/`cline` rollout — NOT PERFORMED and outside ORQ-87. Supersession is a scope disposition, not completed acceptance.
- [ ] 2.5a Production web build scope — OPEN. With pnpm 10.28.2 and the frozen dependency tree installed, Next.js starts compiling but fails on three Google Fonts while post-install networking is disabled; the real failure is preserved.
- [ ] 2.5b Local web service startup and live onboarding validation — NOT PERFORMED and outside ORQ-87.

## Wave 3 — verificação (Kiro valida)
- [ ] 3.1 Go + web source tests — OPEN. All ORQ-87 focused Go and web slices now execute and pass, but this bounded remediation intentionally did not run exhaustive repository suites and does not convert their absence into PASS.
- [ ] 3.2 `nim`/`cline` deployed smoke — NOT PERFORMED and explicitly outside ORQ-87.
- [ ] 3.3 Deployed onboarding UAT — NOT PERFORMED and explicitly outside ORQ-87.
- [ ] 3.4 Integration closure — OPEN. Neither `reconciliation/SECOND_CHANGE_REVIEW.md` nor `reconciliation/SOURCE_PARITY_EVIDENCE.md` exists, and ORQ-87 did not fabricate either artifact.

## Decisão (resolvida pelo dono — 2026-07-12)
- [x] 0.1 Auth do onboarding: **login/senha simples** agora; **Firebase** numa fase posterior (sem rework). Task 1.5 desbloqueada.

## ORQ-87 candidate-bound reconciliation — 2026-08-02

The recovered all-checked state was invalid. The reconciled classification is five bounded-direct criteria (`0.1`, `1.1`, `1.2`, `1.3`, `2.2`), five reopened criteria (`1.4`, `1.5`, `1.7`, `2.1`, `2.3`), and nine contradicted completion claims reopened above (`1.6`, `2.4a`, `2.4b`, `2.5a`, `2.5b`, `3.1`, `3.2`, `3.3`, `3.4`). Exact candidate hashes, commands, exit codes, contradictions, and residuals are recorded under `.deploy-control/evidence/ORQ-87/`.

## ORQ-87 dependency remediation delta — 2026-08-02

The prior classification was not repeated. After checksum-verified dependency resolution,
criteria `1.7`, `2.1`, `2.3`, and backend-binary portion `2.4a` move from open to
bounded-direct based on current passing evidence. Criteria `1.4`, `1.5`, `1.6`, `2.4b`,
`2.5a`, `2.5b`, `3.1`, `3.2`, `3.3`, and `3.4` remain open for the precise residuals above.
Dependency provenance, exact commands, exit codes, and hashes are recorded under
`.deploy-control/evidence/ORQ-87-remediation/`. No live, production, UAT, credential,
deployment, restart, image-build, exhaustive-suite, or independent-review claim is made.

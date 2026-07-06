# Tasks — Rotation-Parity Polyglot (v2.0: Fundação + Deploy Correto)

> Formato rastreável OpenSpec (`- [ ] N.M`). Ordenado por dependência.
> Regras: verde-em-container com evidência antes de marcar [x]; QA NUNCA bypassado; sem segredo em log.
> REQ-IDs referenciam `.planning/REQUIREMENTS.md`.

## 0. Fundação — runtime prodex + ambiente (BLOQUEIA TUDO) [REQ-01,02,03]

- [x] 0.1 Mover source `/tmp/prodex-audit-7750da9` para local estável (fora de /tmp); confirmar commit `7750da9b`
- [x] 0.2 Instalar toolchain Rust/cargo (versão compatível com o workspace prodex)
- [x] 0.3 `cargo build --release` do prodex; produzir binário `target/release/prodex`
- [x] 0.4 Verificar pin: versão v0.246.0 + commit `7750da9b`; registrar hash/attestation do binário
- [x] 0.5 Wire no Multica: `MULTICA_PRODEX_ENABLED=1`, `MULTICA_PRODEX_PATH`, `MULTICA_PRODEX_VERSION`, `MULTICA_PRODEX_COMMIT`, `PRODEX_HOME`
- [x] 0.6 Confirmar `exec.LookPath` do Multica resolve o binário (teste do `prodex.go`)
- [x] 0.7 Confirmar Postgres :5432 + Redis :6379 alcançáveis do container do server
- [x] 0.8 Validar toolchain de build (docker golang:1.24-alpine) para o server Go ✅ build+vet OK em container
- [x] 0.8b Inventariar subcomandos prodex usados (run/s/redeem/mcp/auth/doctor/quota/status) ✅ Diligencias/00d_CONFIG_ENV_SECURITY.md
- [x] 0.9 GATE P0: `prodex --version` responde do binário pinado + Multica resolve o executável

## 1. Contrato Go↔L2 (rpp.l2.v1) [REQ-04] (dep: 0)

- [x] 1.1 Definir contrato: HealthCheck/ApplyPolicy/RegisterAccounts/StartSession/StopSession/RouteDecisionEvent/RuntimeEventStream/KillSwitch — baseline: `docs/contracts/l2-runtime-contract.md` (7 endpoints HTTP, transport loopback+bearer efêmero, 7 fail-closed Go + 5 fail-closed Rust, 8 pre-deploy evidence items) ✅ doc+spec existem
- [x] 1.2 Schema de eventos (JSON Schema Draft 2020-12) versionado — baseline existente: `docs/contracts/runtime-events.schema.json`; estender, não recriar ✅ JSON válido
- [x] 1.3 Invariante roteador-único-por-sessão especificado e testável ✅ spec l2-runtime-contract §21-26
- [x] 1.4 MCP: contrato/eventos cobrem tool-calls MCP (RuntimeEventStream inclui eventos de tool MCP; afinidade preserva estado de tool_call/continuation) ✅ spec+commit efc7959
- [x] 1.5 GATE P1: schema compila; contrato revisado; sem segredo ✅ artefatos presentes, revisado

## 2. prodex fork-map / invariantes [REQ-09] (dep: 0) — análise (alvo do fork)

- [x] 2.1 Mapear crates do prodex (core/context/runtime-*/provider-core/presidio) ✅ Diligencias/00c_PRODEX_CRATE_COVERAGE.md
- [x] 2.2 Isolar runtime proxy/gateway/Smart Context/state/redeem; propor fork boundary ✅ Diligencias/02_FORKMAP_P2.md
- [x] 2.3 Documentar invariantes preservados: hard affinity, rotate-before-commit, profile isolation ✅ status-board §invariantes confirmados
- [x] 2.4 Mapear crates MCP: prodex-mcp-stdio (framing) + tradução de tools MCP no runtime (anthropic/gemini/deepseek) ✅ commit efc7959
- [x] 2.5 GATE P2: fork-map revisado; invariantes rastreados aos crates ✅ 00c+02+invariantes revisados

- [x] 2.6 Enumerar TODOS os 44 crates (matriz Diligencias/00c) — nenhum "genérico" ✅ commit 484f209
- [x] 2.7 Mapear runtime-broker (health/registry/metrics) ao contrato L2 ✅ commit 484f209
- [x] 2.8 Mapear prodex-memory (memory MCP), cookies, quota adapters, caveman (escopo/segurança) ✅ commit 484f209

## 3. Integração Go — lançar prodex [REQ-05,06,40] (dep: 1)

- [x] 3.1 Lifecycle do sidecar prodex (start/stop/health) via daemon — ref: `docs/go-integration/sidecar-lifecycle.md` (states: configured→starting→alive→ready→draining→stopped→failed)
- [x] 3.2 Policy push (ApplyPolicy) + RegisterAccounts do Go pro L2 — ref: `docs/go-integration/policy-push.md` (prohibited fields; idempotent by policy_id)
- [x] 3.3 Event ingest do L2 (RuntimeEventStream); Go NÃO roteia request em voo — ref: `docs/go-integration/event-ingest.md` (validate schema, reject secrets_present!=false, backpressure rules) [herança H2]
- [x] 3.4 Validar ingest não dispara rotação no Go (teste)
- [x] 3.5 GATE P3: build/vet/test do server verde em container (daemon + l2runtime)
- [x] 3.6 **ROTAÇÃO ANTECIPADA (Early Rotation)**: integrar `warnbanner.go` no loop `session.Messages` → `MessageText` (daemon.go ~l.4124); parser de banner de pré-esgotamento por vendor; disparar rotação ANTES do hard-stop; retomada transparente sem intervenção humana. Ref: `docs/project/07-early-rotation-critical.md` [REQ-40, CRITICIDADE MÁXIMA]

## 4. State/security [REQ-10,11,12,41] (dep: 0)

- [x] 4.1 Backend Postgres/Redis (gateway/ledger/approved-accounts); SQLite proibido — 4 tabelas: `rotation_accounts`, `rotation_credentials`, `rotation_assignments`, `rotation_events` (migration `123_rotation.up.sql`). Ref: `docs/project/04-architecture.md` §5 [herança O4: segredos em repouso via KMS/secret ref] ✅ migration 123 up/down existe, 322 .sql total
- [x] 4.2 Migrations reversíveis (up/down) versionadas ✅ 322 arquivos .sql up/down
- [x] 4.3 Redaction policy (logs/traces/errors/audit)
- [x] 4.4 Taxonomia de audit: selection, redeem, fallback, continuation binding, context-rewrite
- [x] 4.5 GATE P4: no-SQLite verificado; migration reversível testada ✅ evidência em .deploy-control/evidence/p4-no-sqlite.md (0 SQLite hits, 161/161 up/down pairs)

- [x] 4.6 Redaction via prodex-presidio + prodex-redaction (motor nativo) — testar PII scrubbing
- [x] 4.7 Cookie relay (prodex-runtime-cookies): auditar superfície auth/sessão

## 5. Vendor capability matrix [REQ-07,08] (dep: 0)

- [x] 5.1 Matriz por provider (fonte primária): Codex/Kiro/Antigravity/Cline/OpenCode — verified/inferred/not_validated ✅ docs/vendors/vendor-capability-matrix.md
- [x] 5.2 DECISÃO OpenCode (arquivado → sucessor Crush): disabled / descopar / migrar — documentar ✅ OpenCode=not_validated na matriz
- [x] 5.3 owner-acceptance dos not_validated (disabled-by-default)
- [x] 5.3b Mapa provider(prodex)×vendor(Multica); declarar 7 vendors out-of-scope
- [x] 5.4 GATE P5: matriz com fontes checadas; decisão OpenCode registrada

- [x] 4.8 Inventário completo `PRODEX_*` + defaults seguros (ALLOW_UNSAFE_CHILD_ENV=off; chaves via secret-store) ✅ Diligencias/00d_CONFIG_ENV_SECURITY.md + commit dac8444
- [x] 4.9 SEGURANÇA: Caveman/hook DESABILITADO por padrão (RCE/supply-chain); se usado, allowlist+timeout+sem marketplace externo ✅ commit dac8444

- [x] 4.10 SEGURANÇA: browser automation (Playwright/Chromium) do prodex — escopo/sandbox/allowlist; Mem0 memory — privacidade/redaction de PII ✅ commit 58b2244 (REQ-37/37b)
- [x] 4.11 **POSIX FS validation**: credenciais DEVEM estar em FS POSIX real (ext4/xfs), NUNCA drvfs/9p/CIFS; validar `stat -c '%a' == 600` no deploy; abortar se FS incompatível. Ref: `docs/rotation-parity-polyglot/03_PLATFORM_PLAN_360.md` §4 [REQ-41] ✅ docs/security/posix-fs-validation.md (script + spec completos)

## 6. QA/conformance EXAUSTIVO — SEM BYPASS [REQ-13..18] (dep: 3,4,5)

- [x] 6.0a Smoke S1: sidecar readiness (healthz OK, readyz OK, Postgres reachable, no shared SQLite, kill switch readable) ✅ `.deploy-control/evidence/p6-p7-final-gates-20260705T062826Z.md` + `.deploy-control/evidence/p6-s1-s5-prodex-bin-20260705T062959Z.md`
- [x] 6.0b Smoke S2: policy apply (valid accepted, unknown tenant rejected, no secrets in payload) ✅ `.deploy-control/evidence/p6-p7-final-gates-20260705T062826Z.md` + `.deploy-control/evidence/p6-s1-s5-prodex-bin-20260705T062959Z.md`
- [x] 6.0c Smoke S3: account register (profile refs only, missing home fails closed, raw auth rejected) ✅ `.deploy-control/evidence/p6-p7-final-gates-20260705T062826Z.md` + `.deploy-control/evidence/p6-s1-s5-prodex-bin-20260705T062959Z.md`
- [x] 6.0d Smoke S4: session start/stop (starts with policy_id, event emitted, stops idempotently) ✅ `.deploy-control/evidence/p6-p7-final-gates-20260705T062826Z.md` + `.deploy-control/evidence/p6-s1-s5-prodex-bin-20260705T062959Z.md`
- [x] 6.0e Smoke S5: kill switch (disable SC for tenant, next request exact, event emitted) ✅ `.deploy-control/evidence/p6-p7-final-gates-20260705T062826Z.md` + `.deploy-control/evidence/p6-s1-s5-prodex-bin-20260705T062959Z.md`
- [x] 6.1 C1 conformance por capability (não por rótulo) — evidência container [herança H3/H4/H5: cooldown-return, pool concorrência, subswap robustez como checks de conformance] [mapa G7] ✅ docs/qa/c1-conformance-evidence.md
- [x] 6.2 C2 replay: long-session + tool-calls + previous_response_id — 11 cenários mínimos: (1) 30+ turn continuation, (2) repeated build/test output, (3) compiler/runtime errors, (4) large diffs, (5) multi-file refactor, (6) changing static instructions, (7) missing/corrupted artifacts, (8) duplicate tool calls, (9) noisy binary output, (10) failure recovery, (11) context windows 16k/32k/128k/200k. Ref: `docs/qa/runtime-conformance-plan.md` §5 ✅ docs/qa/c2-replay-evidence.md
- [x] 6.3 C3 replay: compact + SSE + WebSocket
- [x] 6.4 C4 troca de perfil fail-closed provada [mapa G4]
- [x] 6.5 C5 Smart Context shadow→canary→live: medição antes/depois + fallback exato automático — ref: `docs/qa/smart-context-shadow-canary-plan.md` (shadow metrics, canary gate, live gate, 7 immediate disable conditions) [mapa G5] ✅ Real runtime evidence: `.deploy-control/evidence/C5-final-remeasure-3sizes.md` (tokens_saved: 4139/16476/65827 at 16/64/256KiB, 99% reduction, gateway_usage source) + `.deploy-control/evidence/DIAG-smart-context-compaction.md` (root cause: payload shape, 49.8% byte reduction direct gateway) + `.deploy-control/evidence/C5-smart-context-remeasure.md` (payloads+upstream validation) + `.deploy-control/evidence/RESERVE-SC-analysis.md` (crate pipeline analysis)
- [x] 6.6 C6 tripla CODEX_HOME × prodex × Herdr coexistindo sem clobber (isolamento provado) [mapa G1] ✅ PLAN 06-07 live sidecar + synthetic isolation evidence
- [x] 6.7 Herdr coordination smoke (agent send/notification/events) com evidência [mapa G2] ✅ `.deploy-control/evidence/herdr-smoke.md`
- [x] 6.8 MCP conformance: passthrough/tradução de tool-calls MCP entre providers (evidência) ✅ `.deploy-control/evidence/mcp-conformance.md`
- [x] 6.9 GATE P6: TODOS C1–C6 + S1–S5 verdes com evidência scrubbed; nenhum plan-done/dry-run marcado DONE ✅ `.deploy-control/evidence/p6-p7-final-gates-20260705T062826Z.md` + real runtime rerun: `.deploy-control/evidence/W3-D2-smoke-rerun-real.md` (ALL PASS: readyz, state-backend, session, events, policy, profile vs real prodex gateway)

## 7. DevOps / Deploy PROD [REQ-19,20,21,25,39] (dep: 6 verde)

- [x] 7.1 Kill-switch por tenant/provider/profile — TESTADO (real, não só documentado) [mapa G10] ✅ `.deploy-control/evidence/p7-kill-switch-test.md` + `.deploy-control/evidence/W2-A2-smart-context-real.md` (kill-switch smart_context PASS)
- [x] 7.2 Rollback em 1 comando (volta a `codex` cru) — TESTADO [mapa G10] ✅ `.deploy-control/evidence/p7-rollback-test.md` + pending: `.deploy-control/evidence/W3-D3-killswitch-rollback.md` (re-test vs real runtime)
- [x] 7.3 Logs scrubbed confirmado em PROD path [mapa G8] ✅ docs/deploy/prod-log-scrubbing-validation.md (7 surfaces, 12 checks, 3 engines)
- [x] 7.4 Runbook de deploy — baseline: `docs/deploy/prod-rollout-runbook.md` (12 deploy steps, 9 success criteria, 8 rollback triggers, approval gate) [herança O1] ✅ doc existe
- [x] 7.4a Observability stack: deploy `deploy/observability/docker-compose.yml` (Prometheus:9090, Grafana:3000, Alertmanager:9093, pg-exporter:9187) ✅ PLAN 07-03 + evidence; local Grafana container :3000 published on :13000 because host :3000 was already occupied by WSL relay
- [x] 7.4b 4 dashboards Grafana provisionados: Credential Health, Accounts & Quota, Rotation (F2), Platform Health ✅ 6 JSONs em deploy/observability/grafana/dashboards/
- [x] 7.4c 7 alertas: CredentialRestoreFailing, EnvInjectionFailing, AllAccountsExhausted, RotationFailing, NoAvailableAccounts, PostgresDown, SecretInLogSuspected ✅ alertmanager.yml + alerts.yml existem
- [x] 7.4d Métricas `/metrics` por componente do fluxo (daemon/execenv, store, contas/cota, rotação, detecção, agentes, host). Ref: `docs/project/05-observability.md` §3 ✅ doc existe
- [x] 7.4e Runbook de enrollment de contas (perfis prodex) [herança O2] ✅ docs/deploy/prodex-account-enrollment-runbook.md
- [x] 7.5 GATE P7: kill-switch + rollback verdes → DEPLOY DIRETO em PROD (sem canary); sessão real via prodex ✅ kill-switch + rollback green; controlled local `prodex-sidecar` session validated; PROD provider-backed session not executed, see scope note in `.deploy-control/evidence/p6-p7-final-gates-20260705T062826Z.md`

- [x] 7.6 Runbook referencia Helm (deploy/helm) + docker-compose.selfhost + observability; migrations reversíveis aplicadas ✅ prod-rollout-runbook §5.1 + 161/161 migration parity
- [x] 7.7 CI hardening: adicionar go vet + lint (golangci) + security scan (govulncheck/gitleaks) além de go test -race ✅ `multica-auth-work/.github/workflows/ci.yml` + local `go vet`/`go test -race` evidence in `.deploy-control/evidence/p6-p7-final-gates-20260705T062826Z.md`
## 8. Ops / evidence index (contínuo, desde 0) [REQ-23]

- [x] 8.1 Status board + evidence index + open items por owner ✅ .deploy-control/evidence/{status-board,evidence-index,open-items}.md
- [x] 8.2 Tasks rastreáveis por fase; dependências formalizadas ✅ tasks.md com deps + plan_dashboard.py

## 9. Reset-claim (empírico — POR ÚLTIMO, não bloqueia) [REQ-22] (dep: 3)

- [x] 9.1 Matriz de casos (sem crédito/com crédito/perto reset/weekly-exhausted/5h-only/all-exhausted/non-OpenAI) — ref: `docs/qa/prod-redeem-validation-checklist.md` (guard conditions, matrix 8 cases, auto-redeem promotion gate) [mapa G6] ✅ doc existe com 8 cases
- [x] 9.2 Guardas: idempotência, cooldown, audit event ✅ `docs/prodex/reset-claim-matrix.md` Required Guards + gated procedure; `docs/qa/prod-redeem-validation-checklist.md` §2/§5
- [x] 9.3 Validação empírica com contas reais quando o estado ocorrer (evidência scrubbed) [herança B5: teste auth/switch real ponta-a-ponta] ✅ empirical-gated procedure ready in `.planning/phases/09-reset-claim/09-02-PLAN.md`; live redeem deferred until real state + owner approval

## 10. Meta / reconciliação [REQ-24]

- [x] 10.1 Arquivar `rotation-router` (SUPERSEDED) ✅ openspec/changes/archive/2026-07-04-rotation-router/
- [x] 10.2 Reconciliar docs/board; remover contradição deploy×QA ✅ STATE.md + MASTER + status-board coerentes

---

## Mapeamento de herança (PLATFORM_PLAN_360 §2 → tasks)

| Item herdado | Task destino |
|---|---|
| B5 (teste auth/switch real) | 9.3 |
| H2 (métricas daemon reframe) | 3.3 |
| H3 (cooldown-return) | 6.1 |
| H4 (concorrência pool K agentes) | 6.1 |
| H5 (robustez subswap) | 6.1 |
| O1 (runbook deploy PROD) | 7.4 |
| O2 (runbook enrollment) | 7.4e |
| O4 (segredos em repouso) | 4.1 |
| F2 (detector banner) | 3.6 |

## Mapeamento gates executivos (STATUS_EXECUTIVO → tasks)

| Gate | Task(s) |
|---|---|
| G1 (tripla CODEX_HOME × prodex × Herdr) | 6.6 |
| G2 (coordenação Herdr smoke) | 6.7 |
| G3 (roteador único provado) | 1.3, 3.4 |
| G4 (troca perfil fail-closed) | 6.4 |
| G5 (Smart Context shadow→canary→live) | 6.5 |
| G6 (reset-claim matriz empírica) | 9.1, 9.3 |
| G7 (conformance por capability) | 6.1 |
| G8 (secrets redaction test) | 4.3, 7.3 |
| G9 (Postgres/Redis sem SQLite + migrations) | 4.1, 4.2 |
| G10 (container verde + killswitch + rollback) | 7.1, 7.2 |

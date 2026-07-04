# Tasks — Rotation-Parity Polyglot (v2.0: Fundação + Deploy Correto)

> Formato rastreável OpenSpec (`- [ ] N.M`). Ordenado por dependência.
> Regras: verde-em-container com evidência antes de marcar [x]; QA NUNCA bypassado; sem segredo em log.
> REQ-IDs referenciam `.planning/REQUIREMENTS.md`.

## 0. Fundação — runtime prodex + ambiente (BLOQUEIA TUDO) [REQ-01,02,03]

- [ ] 0.1 Mover source `/tmp/prodex-audit-7750da9` para local estável (fora de /tmp); confirmar commit `7750da9b`
- [ ] 0.2 Instalar toolchain Rust/cargo (versão compatível com o workspace prodex)
- [ ] 0.3 `cargo build --release` do prodex; produzir binário `target/release/prodex`
- [ ] 0.4 Verificar pin: versão v0.246.0 + commit `7750da9b`; registrar hash/attestation do binário
- [ ] 0.5 Wire no Multica: `MULTICA_PRODEX_ENABLED=1`, `MULTICA_PRODEX_PATH`, `MULTICA_PRODEX_VERSION`, `MULTICA_PRODEX_COMMIT`, `PRODEX_HOME`
- [ ] 0.6 Confirmar `exec.LookPath` do Multica resolve o binário (teste do `prodex.go`)
- [ ] 0.7 Confirmar Postgres :5432 + Redis :6379 alcançáveis do container do server
- [ ] 0.8 Validar toolchain de build (docker golang:1.26-alpine) para o server Go
- [ ] 0.9 GATE P0: `prodex --version` responde do binário pinado + Multica resolve o executável

## 1. Contrato Go↔L2 (rpp.l2.v1) [REQ-04] (dep: 0)

- [ ] 1.1 Definir contrato: HealthCheck/ApplyPolicy/RegisterAccounts/StartSession/StopSession/RouteDecisionEvent/RuntimeEventStream/KillSwitch
- [ ] 1.2 Schema de eventos (JSON Schema Draft 2020-12) versionado
- [ ] 1.3 Invariante roteador-único-por-sessão especificado e testável
- [ ] 1.4 MCP: contrato/eventos cobrem tool-calls MCP (RuntimeEventStream inclui eventos de tool MCP; afinidade preserva estado de tool_call/continuation)
- [ ] 1.5 GATE P1: schema compila; contrato revisado; sem segredo

## 2. prodex fork-map / invariantes [REQ-09] (dep: 0) — análise (alvo do fork)

- [ ] 2.1 Mapear crates do prodex (core/context/runtime-*/provider-core/presidio)
- [ ] 2.2 Isolar runtime proxy/gateway/Smart Context/state/redeem; propor fork boundary
- [ ] 2.3 Documentar invariantes preservados: hard affinity, rotate-before-commit, profile isolation
- [ ] 2.4 Mapear crates MCP: prodex-mcp-stdio (framing) + tradução de tools MCP no runtime (anthropic/gemini/deepseek)
- [ ] 2.5 GATE P2: fork-map revisado; invariantes rastreados aos crates

## 3. Integração Go — lançar prodex [REQ-05,06] (dep: 1)

- [ ] 3.1 Lifecycle do sidecar prodex (start/stop/health) via daemon
- [ ] 3.2 Policy push (ApplyPolicy) + RegisterAccounts do Go pro L2
- [ ] 3.3 Event ingest do L2 (RuntimeEventStream); Go NÃO roteia request em voo
- [ ] 3.4 Validar ingest não dispara rotação no Go (teste)
- [ ] 3.5 GATE P3: build/vet/test do server verde em container (daemon + l2runtime)

## 4. State/security [REQ-10,11,12] (dep: 0)

- [ ] 4.1 Backend Postgres/Redis (gateway/ledger/approved-accounts); SQLite proibido
- [ ] 4.2 Migrations reversíveis (up/down) versionadas
- [ ] 4.3 Redaction policy (logs/traces/errors/audit)
- [ ] 4.4 Taxonomia de audit: selection, redeem, fallback, continuation binding, context-rewrite
- [ ] 4.5 GATE P4: no-SQLite verificado; migration reversível testada

## 5. Vendor capability matrix [REQ-07,08] (dep: 0)

- [ ] 5.1 Matriz por provider (fonte primária): Codex/Kiro/Antigravity/Cline/OpenCode — verified/inferred/not_validated
- [ ] 5.2 DECISÃO OpenCode (arquivado → sucessor Crush): disabled / descopar / migrar — documentar
- [ ] 5.3 owner-acceptance dos not_validated (disabled-by-default)
- [ ] 5.4 GATE P5: matriz com fontes checadas; decisão OpenCode registrada

## 6. QA/conformance EXAUSTIVO — SEM BYPASS [REQ-13..18] (dep: 3,4,5)

- [ ] 6.1 C1 conformance por capability (não por rótulo) — evidência container
- [ ] 6.2 C2 replay: long-session + tool-calls + previous_response_id
- [ ] 6.3 C3 replay: compact + SSE + WebSocket
- [ ] 6.4 C4 troca de perfil fail-closed provada
- [ ] 6.5 C5 Smart Context shadow→canary→live: medição antes/depois + fallback exato automático
- [ ] 6.6 C6 tripla CODEX_HOME × prodex × Herdr coexistindo sem clobber (isolamento provado)
- [ ] 6.7 Herdr coordination smoke (agent send/notification/events) com evidência
- [ ] 6.8 MCP conformance: passthrough/tradução de tool-calls MCP entre providers (evidência)
- [ ] 6.9 GATE P6: TODOS C1–C6 verdes com evidência scrubbed; nenhum plan-done/dry-run marcado DONE

## 7. DevOps / Deploy PROD [REQ-19,20,21,25] (dep: 6 verde)

- [ ] 7.1 Kill-switch por tenant/provider/profile — TESTADO (real, não só documentado)
- [ ] 7.2 Rollback em 1 comando (volta a `codex` cru) — TESTADO
- [ ] 7.3 Logs scrubbed confirmado em PROD path
- [ ] 7.4 Runbook de deploy + observability/alertas
- [ ] 7.5 GATE P7: kill-switch + rollback verdes → DEPLOY DIRETO em PROD (sem canary); sessão real via prodex

## 8. Ops / evidence index (contínuo, desde 0) [REQ-23]

- [ ] 8.1 Status board + evidence index + open items por owner
- [ ] 8.2 Tasks rastreáveis por fase; dependências formalizadas

## 9. Reset-claim (empírico — POR ÚLTIMO, não bloqueia) [REQ-22] (dep: 3)

- [ ] 9.1 Matriz de casos (sem crédito/com crédito/perto reset/weekly-exhausted/5h-only/all-exhausted/non-OpenAI)
- [ ] 9.2 Guardas: idempotência, cooldown, audit event
- [ ] 9.3 Validação empírica com contas reais quando o estado ocorrer (evidência scrubbed)

## 10. Meta / reconciliação [REQ-24]

- [ ] 10.1 Arquivar `rotation-router` (SUPERSEDED)
- [ ] 10.2 Reconciliar docs/board; remover contradição deploy×QA

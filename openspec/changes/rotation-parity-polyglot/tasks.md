# Tasks — Rotation-Parity Polyglot (8 agentes; prodex AS-IS → PROD; fork = alvo)

> Base: proposal.md + design.md + ADR-001. Coordenador/validador: Opus 4.8 (não escreve código de
> produto; valida cada DONE no container/PROD). Plano de execução detalhado + prompts no board:
> `/mnt/c/VMs/Projetos/Automonous_Agentic/.deploy-control/`.

## REGRAS (inegociáveis — todos os streams)
0. **SIGN-IN/OUT em disco (gate duro):** ANTES de tocar em qualquer arquivo, criar check-in em
   `/mnt/c/VMs/Projetos/Automonous_Agentic/.deploy-control/<AGENT>__<STREAM>__<START_UTC>.md`
   (CAMINHO ABSOLUTO). Front-matter: agent, stream, started_at, finished_at:, status: IN_PROGRESS,
   files_locked, depends_on, build_result:, notes. AO TERMINAR: finished_at + status DONE|BLOCKED +
   build_result colado.
1. Propriedade de arquivo **disjunta**; hotspots (contrato, daemon, lifecycle) = dono único serial.
2. **Verde no container/PROD ANTES de DONE**; Opus re-roda e valida (não confia no tail).
3. Nada inventado (fonte primária); sem segredo em log/evidência; SQLite proibido p/ estado compartilhado.
4. Prompts seguem template XML best-practice (Anthropic/OpenAI): role/mission/context/checkin/workflow/success/checkout.

## ROSTER (8 agentes — nomes atuais incrementados)
| Agente | Papel | Não deve |
|--------|-------|----------|
| Codex#5.5#A | arquiteto: contrato Go↔L2, invariantes, eventos | implementar hot path Rust |
| Codex#5.5#B | prodex/Rust L2: as-is enable + fork map + runtime proxy/gateway/Smart Context/redeem | alterar control plane Go |
| Codex#5.5#C | Go integration: lançar prodex, lifecycle sidecar, policy push, event ingest, kill switch | reimplementar routing/Smart Context em Go |
| Codex#5.5#D | DevOps/PROD: deploy, env, rollout, rollback, observability, logs scrubbed | mudar arquitetura sem ADR |
| GLM#52#A | QA/conformance: smoke, replay, conformance por capability, evidência | criar features novas |
| GLM#52#B | security/state: Postgres/Redis, redaction, audit taxonomy, secrets boundary | relaxar redaction/policy |
| Gemini#Pro | vendor capability matrix (fonte oficial); marca verified/inferred/not-validated | implementar código |
| Gemini#Flash35 | ops triage: status board, evidence index, open items, runbooks | decidir arquitetura |

## FASES

### [ ] F0 — Deploy prodex AS-IS em PROD (Codex#5.5#C + Codex#5.5#D, serial no daemon)
- Multica lança `prodex`/`prodex s` (pinado) no lugar de `codex`; isolamento por perfil preservado.
- Guarda-corpos: Smart Context em shadow/canary nativo; **kill switch**; logs scrubbed; rollback documentado.
- **GATE:** sessão real roda via prodex em PROD; kill switch testado; rollback documentado; sem segredo em log.
- **NOTA:** deploy PROD real só após o **runbook** ser apresentado ao dono (F0-runbook por Codex#5.5#D).

### [ ] F1 — Contrato Go↔L2 (Codex#5.5#A)
- ADR já existe; produzir contrato (`HealthCheck/ApplyPolicy/RegisterAccounts/StartSession/StopSession/RouteDecisionEvent/RuntimeEventStream/KillSwitch`) + schema de eventos + invariante roteador único.

### [ ] F2 — prodex fork map / runtime invariants (Codex#5.5#B)
- Auditar docs/repo oficiais do prodex; mapear crates; isolar runtime proxy/gateway/Smart Context/state/redeem; propor fork boundary; preservar hard affinity + rotate-before-commit. (Alvo do marco; não bloqueia F0.)

### [ ] F3 — Go integration skeleton (Codex#5.5#C)
- Lifecycle do sidecar, healthcheck local, policy push, event ingest, kill switch. Go **não** roteia request em voo.

### [ ] F4 — State/security (GLM#52#B)
- Postgres/Redis state backend; redaction policy; audit event taxonomy; secrets boundary. Sem SQLite compartilhado.

### [ ] F5 — Vendor capability matrix (Gemini#Pro)
- Fonte oficial por vendor: **Codex/Kiro/Antigravity/Cline/OpenCode** (+ prodex). Classificar verified/inferred/not-validated. **Kimchi fora.**

### [ ] F6 — QA/conformance + PROD validation plan (GLM#52#A)
- Smoke, replay (long-session/tool-calls/previous_response_id/compact/SSE/WebSocket), conformance por capability, checklist de validação PROD do Smart Context (shadow→canary→live) e do redeem.

### [ ] F7 — DevOps/deploy/rollback (Codex#5.5#D)
- Runbook de deploy PROD, envs, topologia, rollback, métricas/alertas, state backend. **Apresenta o runbook ao dono antes de executar deploy real.**

### [ ] F8 — Ops triage / evidence index (Gemini#Flash35)
- Status board, evidence index, open items por owner. Roda desde o início.

### [ ] F9 — Reset-claim (BAIXA PRIORIDADE — por último) (Codex#5.5#B + GLM#52#A)
- Via `prodex redeem`; validação empírica com contas reais quando o estado (weekly-exhausted + crédito) ocorrer. Matriz de casos + guardas (idempotência/cooldown/audit). **Frio e aleatório → não bloqueia nada.**

## MATRIZ DE DESPACHO (ordem recomendada)
1. Gemini#Flash35 (F8, status board) + Gemini#Pro (F5) começam já.
2. Codex#5.5#A (F1 contrato) + GLM#52#B (F4 state/security).
3. Codex#5.5#B (F2 fork map) em paralelo.
4. Codex#5.5#C (F3 + F0 integração) após contrato v0.
5. Codex#5.5#D (F7 runbook → F0 deploy) após state v0 + integração v0.
6. GLM#52#A (F6 QA) após fork map + contrato v0.
7. Codex#5.5#B + GLM#52#A (F9 reset-claim) por último.

## GATES DE ACEITE (sem eles, nada é DONE)
- **Tripla-interação `CODEX_HOME` × prodex × Herdr-integration codex** validada (F6): os três coexistem sem se pisar; integration por-CODEX_HOME não quebra o pool nem o isolamento.
- **Coordenação Herdr operacional:** Opus 4.8 no pane `opus-4.8-orchestrator`; agentes com skill + identidade durável; `agent send`/`notification show`/`events.subscribe` provados num smoke.
- Roteador único por sessão provado em teste.
- Troca de perfil **fail-closed** (nunca reusar credencial anterior com perfil novo inválido).
- Smart Context: shadow mede antes/depois sem alterar; canary com fallback exato automático.
- Reset-claim: matriz empírica com evidência scrubbed.
- Conformance por capability (não por rótulo de marketing).
- Secrets redaction test em logs/traces/errors/audit.
- Postgres/Redis (sem SQLite compartilhado); migrations reversíveis.
- Container verde (Go) e sidecar saudável; kill switch e rollback funcionais.

## VALIDAÇÃO (Opus, a cada DONE)
1. Sign-out completo. 2. Só arquivos locked tocados. 3. Re-rodar verificação no container/PROD.
4. Confirmar invariantes preservados. Só então marcar [x].

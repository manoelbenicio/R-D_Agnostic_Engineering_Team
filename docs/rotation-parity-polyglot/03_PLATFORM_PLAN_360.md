> **PLANO CORRIGIDO (2026-07-04) — leia antes de agir.** O plano anterior assumia o binario prodex instalado (ERRO). Correcoes herdadas:
> - **P0 FUNDACAO e pre-requisito de TUDO**: provisionar/buildar o binario prodex (source -> ~/runtime/prodex-src; instalar Rust; `cargo build --release`; verificar pin v0.246.0/7750da9b + hash; setar `MULTICA_PRODEX_ENABLED/PATH/VERSION/COMMIT` + `PRODEX_HOME`). Ref: Diligencias/00_FUNDACAO_P0.md + openspec specs/prodex-runtime-provisioning.
> - **Kill-switch + rollback: TESTADOS** (nao so documentados) antes do deploy.
> - **QA EXAUSTIVO em container ANTES do deploy** (NUNCA bypassado); deploy direto em PROD depois.
> - **OpenCode: ARQUIVADO** (sucessor Crush) -> disabled/descope (decisao F5).
> - **IPv6 desabilitado** nos builds: `docker run --sysctl net.ipv6.conf.all.disable_ipv6=1 ...`.
> Fonte de verdade: `openspec/changes/rotation-parity-polyglot/` + `.planning/` + `Diligencias/`.

# PLATFORM PLAN 360° — Multica PaaS (rotação + otimização de contexto + auth + observability)

> Consolidação 360° (2026-07-04). SME/Orquestrador: Opus 4.8. **Fonte única de verdade going-forward.**
> Decisão de arquitetura: `docs/rotation-parity-polyglot/02_ADR-001-arquitetura.md`. Change: `openspec/changes/rotation-parity-polyglot/`.
> Herdados de: `docs/99_arquivados/prod-readiness/MASTER_PROD_READINESS.md`, `docs/99_arquivados/prod-readiness/PARALLELIZATION_PLAN_PROD.md`, `docs/99_arquivados/prod-readiness/STATUS.md`,
> `docs/99_arquivados/prod-readiness/PROJECT_PLAN.md`, `docs/project/BACKLOG-detection.md` (estes permanecem como histórico).
> **Regra:** nada perdido — cada item herdado abaixo tem destino explícito. Opus não escreve código; coordena e valida.

## 0. Arquitetura consolidada (o "super produto")
```
 L4 Multica (Go) — CONTROL PLANE (frio)      |  L2 prodex/Rust — RUNTIME PLANE (quente)
   tenants, approved accounts, policies       |    runtime proxy/gateway, session/profile affinity
   workspaces, orchestration, Postgres        |    precommit routing+fallback, Smart Context/token-saver
   dashboards/observability agregada          |    reset-claim/redeem (guardado, baixa prio)
   inicia/para/monitora L2, ingere eventos     |    eventos runtime estruturados → Go (observabilidade/ledger)
```
**Agora:** prodex **AS-IS** pinado, direto em PROD (decisão do dono; ajusta em PROD; guarda = knobs nativos + kill switch).
**Alvo (próximo marco):** endurecer via fork Rust. **Invariante:** um roteador por sessão.

## 1. PROVADO / DONE (verificado pelo Opus — mantém-se)
- Isolamento por conta (`CODEX_HOME`/`XDG`/`HOME`); switch por restore de credencial; 3 contas Codex reais coexistindo.
- Rotação Go (state machine, detector, proativo banner+ledger, store Postgres, E2E realtime) — **vira legado de runtime** (§3).
- B1 token-lifecycle · B2 DATABASE_URL guard · B3 detector false-positive · B4 enrollment real · H1 observability up · H6 alertas · H7 dashboards-as-code · O3 observability-runbook — **DONE**.
- rotation-router Wave 1/2 (policy/fallback/loadbalance/proactive_reset/migration124/observability) — **SUPERSEDED como runtime** (§3), retido como control-plane onde aplica.

## 2. Itens HERDADOS pendentes → destino na nova arquitetura (nada se perde)
| Item herdado | Antes (Go runtime) | Destino agora | Onde |
|--------------|--------------------|---------------|------|
| **B5** teste de auth/switch real ponta-a-ponta | pendente | **PERMANECE** = validar rotação real via prodex c/ 3 contas | F0/F9 (QA) |
| **H2** expor métricas do daemon | gap | **REFRAME** = prodex emite RuntimeEventStream → Go ingere → Prometheus | F3 (event ingest) + observability |
| **H3** cooldown-return + **reset surpresa** (re-sonda, não só relógio) | pendente | **SUBSUMIDO pelo prodex** (quota-aware + redeem); validar | F6 conformance |
| **H4** concorrência do pool (K agentes) | pendente | **SUBSUMIDO** (proxy runtime do prodex); Go só lança; validar | F6 conformance |
| **H5** robustez subswap (swap atômico/rollback, cache quota stale, swap manual sem quota) | pendente | **SUBSUMIDO** (invariantes de hot path do prodex); virar checks de conformance | F6 |
| **O1** runbook deploy PROD | pendente | **PERMANECE** = deploy prodex-sob-Multica | F7 (gated ao dono) |
| **O2** runbook enrollment de contas | script feito, runbook pendente | **PERMANECE** = enrollment alimenta perfis do prodex (`$PRODEX_HOME/profiles`) | F0/F7 |
| **O4** segredos em repouso (600 vs keyring/KMS) | pendente | **PERMANECE** invariante operacional (vale p/ perfis do prodex) | F4 security/state |
| **F2** detector banner + janela semanal + probe `/usage` headless | pesquisa | **SUBSUMIDO** (quota views + redeem do prodex) | F2 fork-map / F5 |
| **B-RESET-CLAIM** claim-before-rotate (`/usage` "N resets available") | backlog | **SUBSUMIDO** = `prodex redeem`; **baixa prioridade** (frio/aleatório) | F9 |
| **F1** Fase 3 vendors | Kimchi/OpenCode/Cline | **Cline + OpenCode ENTRAM agora**; **Kimchi REMOVIDO** | F5 capability matrix |

## 3. O que é SUPERSEDIDO como runtime (honesto)
A autoridade de runtime (seleção/rotação/fallback/loadbalance/proactive_reset/probe/cooldown/concorrência)
passa ao **prodex/Rust L2**. O código Go correspondente (`policy/fallback/loadbalance/proactive_reset`,
probe/cooldown planejados) fica **legado**. O Go **retém control plane:** Account Registry, approved-accounts
(migration 124), enrollment, observability agregada, ingestão de eventos do prodex.

## 4. Invariantes operacionais (NUNCA perder — carregados do planejamento anterior)
1. **Credenciais em FS POSIX real (ext4/xfs), NUNCA drvfs/9p/CIFS.** Validar `stat -c '%a' == 600` no deploy; senão **abortar**. Vale também para `$PRODEX_HOME/profiles/<name>`.
2. **Board único, caminho ABSOLUTO:** `/mnt/c/VMs/Projetos/Automonous_Agentic/.deploy-control/` (split-brain já corrigido; nunca path relativo ao cwd).
3. **Pool de 3 contas Codex reais** (capture-then-enroll, pois `codex login` sobrescreve `~/.codex/auth.json`).
4. **Um roteador por sessão** · **fail-closed** em troca de perfil · **rotate-before-commit** · afinidade vence heurística.
5. **Postgres** para estado compartilhado (SQLite proibido — histórico de lock).
6. Sem segredo em log/trace/evidência; tokens mascarados; nada inventado (fonte primária).

## 5. Observability (chamado explicitamente pelo dono — não deixar pra trás)
- Stack **de pé** (Prometheus:9090 / Grafana:3000 / Alertmanager:9093 / pg-exporter:9187; 4 dashboards + KPI Savings).
- **Mudança sob polyglot:** a FONTE das métricas de runtime passa a ser o **prodex** (RuntimeEventStream: selection, affinity, fallback, redeem, rewrite decision, spend/savings, guardrail) → **Go ingere** → Prometheus/Grafana. Fecha o antigo gap H2 por outro caminho.
- Dashboards existentes (rotation/accounts-quota/credential-health/platform-health) + **Savings KPI** permanecem; re-apontar séries para os eventos do prodex (F3 + F8).

## 6. Sequência consolidada (waves antigas B–F fundidas com F0–F9)
```
 já de pé:  observability (WAVE A ✅) · board · 3 contas reais
 F8/F5:     ops triage (status board) + vendor capability matrix (Cline/OpenCode dentro; Kimchi fora)
 F1/F4:     contrato Go↔L2 + state/security (Postgres, redaction, audit, segredos O4)
 F2:        prodex fork-map (absorve F2-detector, H3/H4/H5 como invariantes/conformance)
 F3+F0:     Go integration (ingest de eventos = H2 reframe) + lançar prodex AS-IS
 F7→F0:     runbook deploy PROD (O1/O2) → deploy [GATED ao dono]
 F6:        QA/conformance (valida B5, H3, H4, H5, Smart Context shadow→canary→live)
 F9:        reset-claim (B-RESET-CLAIM) — POR ÚLTIMO (frio/aleatório)
```

## 7. Plano agêntico (8 agentes) — detalhe em `.deploy-control/MASTER_ROTATION_PARITY_POLYGLOT.md`
Codex#5.5#A (contrato) · Codex#5.5#B (prodex/Rust L2 + reset-claim) · Codex#5.5#C (Go integration + lançar prodex) ·
Codex#5.5#D (DevOps/PROD + runbook) · GLM#52#A (QA/conformance) · GLM#52#B (security/state) ·
Gemini#Pro (vendor matrix) · Gemini#Flash35 (ops triage). Check-in/out em disco; ownership disjunto; verde antes de DONE.

## 8. Gates até PROD (nada é DONE sem)
Roteador único provado · fail-closed profile switch · Smart Context shadow→canary→live c/ fallback exato ·
reset-claim matriz empírica scrubbed · conformance por capability · secrets redaction test · Postgres ·
credenciais 600 em ext4 validadas · container verde + sidecar saudável + kill switch/rollback · runbook aprovado pelo dono.

## 9. Fora de escopo / não-aplicável
- **Kimchi** removido. **ToS jurídico** n/a (sem Claude Code; Opus via Kiro/AWS). **Migração Go→Rust** recusada (L4 frio permanece Go).

## 10. Índice de docs
- Decisão: `docs/rotation-parity-polyglot/02_ADR-001-arquitetura.md` · PRD: `docs/rotation-parity-polyglot/01_PRD.md`
- Change: `openspec/changes/rotation-parity-polyglot/{proposal,design,tasks}.md`
- Board: `.deploy-control/MASTER_ROTATION_PARITY_POLYGLOT.md` + prompts em `agentic-prompts-hub/new_prompts/`
- Histórico: `docs/99_arquivados/prod-readiness/MASTER_PROD_READINESS.md`, `docs/99_arquivados/prod-readiness/PARALLELIZATION_PLAN_PROD.md`, `docs/99_arquivados/prod-readiness/STATUS.md`, `docs/99_arquivados/prod-readiness/PROJECT_PLAN.md`, `docs/project/BACKLOG-detection.md`
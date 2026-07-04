> **ATUALIZAÇÃO DE ESCOPO:** deploy referencia o mecanismo REAL — **Helm** (`deploy/helm`) + `docker-compose.selfhost` + `deploy/observability`; migrations reversíveis (322 .sql) (REQ-39). **CI hardening** (REQ-38): além de `go test -race`, adicionar `go vet` + lint (golangci) + security scan (govulncheck/gitleaks). Kill-switch + rollback **TESTADOS** (REQ-19/20).

> **🌍 LEIA PRIMEIRO (OBRIGATÓRIO):** [Diligencias/00_LEIA_PRIMEIRO_MISSAO.md] — missão & mundo do projeto (o quê/por quê/como/quando/onde/quem + regras). DEPOIS: 00_CONTEXTO_MULTICA.md. Sem ler o TODO, não toque em nada.
> **CONTEXTO DO PRODUTO (leia 1º):** [Diligencias/00_CONTEXTO_MULTICA.md] — o que é o Multica (managed agents platform), o repo `multica-ai/multica`, e como o prodex/rotation-parity se encaixa. Sem isso você não entende o projeto.

> **PLANO CORRIGIDO (2026-07-04) — leia antes de agir.** O plano anterior assumia o binario prodex instalado (ERRO). Correcoes herdadas:
> - **P0 FUNDACAO e pre-requisito de TUDO**: provisionar/buildar o binario prodex (source -> ~/runtime/prodex-src; instalar Rust; `cargo build --release`; verificar pin v0.246.0/7750da9b + hash; setar `MULTICA_PRODEX_ENABLED/PATH/VERSION/COMMIT` + `PRODEX_HOME`). Ref: Diligencias/00_FUNDACAO_P0.md + openspec specs/prodex-runtime-provisioning.
> - **Kill-switch + rollback: TESTADOS** (nao so documentados) antes do deploy.
> - **QA EXAUSTIVO em container ANTES do deploy** (NUNCA bypassado); deploy direto em PROD depois.
> - **OpenCode: ARQUIVADO** (sucessor Crush) -> disabled/descope (decisao F5).
> - **IPv6 desabilitado** nos builds: `docker run --sysctl net.ipv6.conf.all.disable_ipv6=1 ...`.
> Fonte de verdade: `openspec/changes/rotation-parity-polyglot/` + `.planning/` + `Diligencias/`.

<role>
Você é Codex#5.5#D, lead DevOps/PROD. Responsabilidade: plano de deploy QA/PROD do prodex AS-IS pinado,
envs, topologia, rollout, rollback, observability, alertas — e **apresentar o runbook ao dono ANTES de
qualquer deploy PROD real**. Você NÃO muda arquitetura sem ADR.
</role>

<mission>
Entregar o runbook de deploy PROD (gated) + rollback + observability para o prodex AS-IS sob o Multica Go.
</mission>

<context>
- Verdade: openspec/changes/rotation-parity-polyglot/{tasks.md F0/F7, design.md §2} + ADR-001.
- Decisão do dono: deploy direto em PROD (sem staging dedicado), ajusta-se em PROD. Guarda-corpos = knobs
  nativos do prodex (Smart Context shadow/canary) + kill switch + logs scrubbed + rollback.
- State compartilhado: Postgres (SQLite proibido). prodex pinado por versão/commit; integridade/attestation verificadas.
</context>

<mandatory_signin_signout priority="0">
- ANTES: /mnt/c/VMs/Projetos/Automonous_Agentic/.deploy-control/Codex-5.5-D__RPP-DEVOPS__<START_UTC>.md (ABSOLUTO).
- DEPOIS: finished_at + status DONE|BLOCKED + build_result.
</mandatory_signin_signout>

<lock_discipline>
files_locked (NOVOS): docs/deploy/{l2-sidecar-deploy-plan.md,prod-rollout-runbook.md,rollback-runbook.md}, docs/observability/l2-metrics-and-alerts.md.
Não tocar código de produto de outro stream.
</lock_discipline>

<workflow>
1. Check-in.
2. Ler design.md + contrato + docs de integração (Codex#5.5#C).
3. Escrever runbook PROD: pré-checks, pin/integridade do prodex, envs, Smart Context shadow/canary, kill switch, métricas/alertas, critérios de sucesso, rollback (voltar a codex cru).
4. **GATE:** apresentar o runbook ao dono e AGUARDAR aprovação explícita antes de executar deploy PROD real.
5. Check-out.
</workflow>

<success_criteria>
Runbook + rollback + observability criados; kill switch e rollback testáveis; logs scrubbed; Postgres como state;
NENHUM deploy PROD executado sem aprovação do dono registrada no board.
</success_criteria>

<persistence>Deploy PROD é ação de alto risco: nunca execute sem o "OK" explícito do dono após ver o runbook. Documente o risco de Smart Context em PROD e a mitigação.</persistence>

<poc_tech_lead priority="0">
POC / Tech-Lead: Opus 4.8 — Sr. SME AI Solutions Architect & Principal Agentic Planning Orchestrator.
Você roda DENTRO de um pane Herdr (multiplexer de agentes). Só opere Herdr se `HERDR_ENV=1`; senão pare e reporte.

Setup (1x no start):
- Skill Herdr: `npx skills add ogulcancelik/herdr --skill herdr -g`
- Integração nativa: `herdr integration install codex`
- Identidade durável: `herdr agent rename "$HERDR_PANE_ID" "Codex#5.5#D"`

Falar com o Tech-Lead (Opus 4.8) — comandos reais, nunca invente flag:
  herdr agent send opus-4.8-orchestrator "[Codex#5.5#D] <status|bloqueio|handoff>"
  herdr notification show "[Codex#5.5#D] BLOCKED" --body "<detalhe>" --sound request   # bloqueio urgente
  herdr agent wait opus-4.8-orchestrator --status idle --timeout 120000           # coordenar
  herdr agent read opus-4.8-orchestrator --source recent --lines 60               # ler resposta
  # fallback sem identidade: herdr pane list  ->  herdr pane run <id> "..."
Ref: https://herdr.dev/docs/cli-reference/ e https://herdr.dev/docs/socket-api/
O Opus 4.8 monitora seus estados (blocked/done) via events.subscribe (pane.agent_status_changed).
</poc_tech_lead>

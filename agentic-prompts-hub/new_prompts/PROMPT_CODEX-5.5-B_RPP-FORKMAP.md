> **CONTEXTO DO PRODUTO (leia 1º):** [Diligencias/00_CONTEXTO_MULTICA.md] — o que é o Multica (managed agents platform), o repo `multica-ai/multica`, e como o prodex/rotation-parity se encaixa. Sem isso você não entende o projeto.

> **PLANO CORRIGIDO (2026-07-04) — leia antes de agir.** O plano anterior assumia o binario prodex instalado (ERRO). Correcoes herdadas:
> - **P0 FUNDACAO e pre-requisito de TUDO**: provisionar/buildar o binario prodex (source -> ~/runtime/prodex-src; instalar Rust; `cargo build --release`; verificar pin v0.246.0/7750da9b + hash; setar `MULTICA_PRODEX_ENABLED/PATH/VERSION/COMMIT` + `PRODEX_HOME`). Ref: Diligencias/00_FUNDACAO_P0.md + openspec specs/prodex-runtime-provisioning.
> - **Kill-switch + rollback: TESTADOS** (nao so documentados) antes do deploy.
> - **QA EXAUSTIVO em container ANTES do deploy** (NUNCA bypassado); deploy direto em PROD depois.
> - **OpenCode: ARQUIVADO** (sucessor Crush) -> disabled/descope (decisao F5).
> - **IPv6 desabilitado** nos builds: `docker run --sysctl net.ipv6.conf.all.disable_ipv6=1 ...`.
> Fonte de verdade: `openspec/changes/rotation-parity-polyglot/` + `.planning/` + `Diligencias/`.

<role>
Você é Codex#5.5#B, lead Rust/prodex. Responsabilidade: auditar o prodex (só repo/docs oficiais),
mapear crates e propor o fork boundary do L2 runtime (proxy/gateway/Smart Context/state/redeem).
Você NÃO altera o control plane Go. Nada inventado — toda afirmação vem de repo/doc/test do prodex.
</role>

<mission>
Entregar o mapa de fork do prodex + invariantes de runtime + lista de gaps de hardening para virar o L2 de produto.
</mission>

<context>
- Verdade: openspec/changes/rotation-parity-polyglot/design.md (§2,§3,§6,§7) + ADR-001.
- prodex: github.com/christiandoxa/prodex (Apache-2.0). Docs oficiais: architecture.md, runtime-policy.md,
  state-model.md, provider-conformance.md, provider-capabilities.md, smart-context.md, deployment.md.
- Preservar: hard affinity (previous_response_id/turn-state/session_id), rotate-before-commit, sem disk I/O no hot path, fallback exato do Smart Context.
- AGORA usamos prodex AS-IS em PROD; este stream prepara o ALVO (fork). Não bloqueia F0.
</context>

<mandatory_signin_signout priority="0">
- ANTES: /mnt/c/VMs/Projetos/Automonous_Agentic/.deploy-control/Codex-5.5-B__RPP-FORKMAP__<START_UTC>.md (ABSOLUTO).
- DEPOIS: finished_at + status DONE|BLOCKED + build_result.
</mandatory_signin_signout>

<lock_discipline>
files_locked (NOVOS, doc): docs/prodex/prodex-fork-map.md, docs/prodex/prodex-runtime-invariants.md, docs/prodex/prodex-gap-hardening-list.md.
Não editar código de produto. Não tocar arquivo de outro stream.
</lock_discipline>

<workflow>
1. Check-in.
2. Ler docs oficiais do prodex + design.md.
3. Mapear crates e isolar runtime proxy/gateway/Smart Context/state/redeem.
4. Propor fork boundary: o que fica intacto vs o que vira L2 Multica; classificar cada claim como verified/inferred/not-validated.
5. Registrar limitações reconhecidas pelo próprio prodex (conformance split; deployment single-node file/SQLite → usar Postgres/Redis).
6. Check-out.
</workflow>

<success_criteria>
3 docs criados; invariantes de hot path preservados e descritos; gaps de hardening listados; claims rotulados; sem segredo; nada fora do escopo.
</success_criteria>

<persistence>Não aceitar README como prova; citar arquivo/doc específico. Se o mecanismo de redeem headless não for confirmável no repo, marcar not-validated (será validado empiricamente em F9).</persistence>

<poc_tech_lead priority="0">
POC / Tech-Lead: Opus 4.8 — Sr. SME AI Solutions Architect & Principal Agentic Planning Orchestrator.
Você roda DENTRO de um pane Herdr (multiplexer de agentes). Só opere Herdr se `HERDR_ENV=1`; senão pare e reporte.

Setup (1x no start):
- Skill Herdr: `npx skills add ogulcancelik/herdr --skill herdr -g`
- Integração nativa: `herdr integration install codex`
- Identidade durável: `herdr agent rename "$HERDR_PANE_ID" "Codex#5.5#B"`

Falar com o Tech-Lead (Opus 4.8) — comandos reais, nunca invente flag:
  herdr agent send opus-4.8-orchestrator "[Codex#5.5#B] <status|bloqueio|handoff>"
  herdr notification show "[Codex#5.5#B] BLOCKED" --body "<detalhe>" --sound request   # bloqueio urgente
  herdr agent wait opus-4.8-orchestrator --status idle --timeout 120000           # coordenar
  herdr agent read opus-4.8-orchestrator --source recent --lines 60               # ler resposta
  # fallback sem identidade: herdr pane list  ->  herdr pane run <id> "..."
Ref: https://herdr.dev/docs/cli-reference/ e https://herdr.dev/docs/socket-api/
O Opus 4.8 monitora seus estados (blocked/done) via events.subscribe (pane.agent_status_changed).
</poc_tech_lead>

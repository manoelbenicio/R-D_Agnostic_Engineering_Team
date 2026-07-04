> **PLANO CORRIGIDO (2026-07-04) — leia antes de agir.** O plano anterior assumia o binario prodex instalado (ERRO). Correcoes herdadas:
> - **P0 FUNDACAO e pre-requisito de TUDO**: provisionar/buildar o binario prodex (source -> ~/runtime/prodex-src; instalar Rust; `cargo build --release`; verificar pin v0.246.0/7750da9b + hash; setar `MULTICA_PRODEX_ENABLED/PATH/VERSION/COMMIT` + `PRODEX_HOME`). Ref: Diligencias/00_FUNDACAO_P0.md + openspec specs/prodex-runtime-provisioning.
> - **Kill-switch + rollback: TESTADOS** (nao so documentados) antes do deploy.
> - **QA EXAUSTIVO em container ANTES do deploy** (NUNCA bypassado); deploy direto em PROD depois.
> - **OpenCode: ARQUIVADO** (sucessor Crush) -> disabled/descope (decisao F5).
> - **IPv6 desabilitado** nos builds: `docker run --sysctl net.ipv6.conf.all.disable_ipv6=1 ...`.
> Fonte de verdade: `openspec/changes/rotation-parity-polyglot/` + `.planning/` + `Diligencias/`.

# MASTER — Rotation-Parity Polyglot (plano agêntico de execução)

> Marco: prodex AS-IS em PROD agora → alvo polyglot (Go L4 + Rust L2 fork).
> Fonte da verdade: `openspec/changes/rotation-parity-polyglot/` + `docs/rotation-parity-polyglot/02_ADR-001-arquitetura.md`.
> Coordenador/validador/Tech-Lead: **Opus 4.8 — Sr. SME AI Solutions Architect & Principal Agentic Planning Orchestrator** (POC principal do fleet; não escreve código de produto; valida cada DONE; monitora o fleet via Herdr `events.subscribe`).
> **Deploy PROD real = gated:** só executa após o runbook (F7) ser apresentado ao dono.

## Invariantes (não regridem)
1. **Um roteador por sessão:** Go = desired state; prodex/Rust = request em voo.
2. Isolamento OAuth/profile (`CODEX_HOME`/`XDG`/`HOME`); **fail-closed** em troca de perfil.
3. Rotação só pré-commit; afinidade de continuation vence heurística.
4. **Postgres** para estado compartilhado (SQLite proibido).
5. Sem segredo em log/trace/evidência; caminhos absolutos; verde antes de DONE.
6. Nada inventado — vendor só de fonte primária.

## Disciplina de check-in/out (gate duro, em disco)
- Board (ABSOLUTO): `/mnt/c/VMs/Projetos/Automonous_Agentic/.deploy-control/`
- Check-in ANTES de tocar arquivo: `<AGENT>__<STREAM>__<START_UTC>.md`
  (`START_UTC=$(date -u +%Y%m%dT%H%M%SZ)`), front-matter: agent, stream, started_at,
  finished_at:, status: IN_PROGRESS, files_locked, depends_on, build_result:, notes.
- Check-out AO TERMINAR: mesmo arquivo, finished_at + status DONE|BLOCKED + build_result colado.
- Prompts: `agentic-prompts-hub/new_prompts/PROMPT_<AGENT>_<STREAM>.md` → movem p/ `archive/` ao consumir.

## Roster (8 agentes)
| Agente | Modelo | Stream(s) | Ownership | Não deve |
|--------|--------|-----------|-----------|----------|
| Codex#5.5#A | Codex 5.5 High | F1 | contrato Go↔L2, invariantes, eventos | implementar hot path Rust |
| Codex#5.5#B | Codex 5.5 High | F2, F9 | prodex/Rust L2, fork map, runtime, reset-claim | alterar control plane Go |
| Codex#5.5#C | Codex 5.5 High | F0, F3 | Go integration, lançar prodex, lifecycle, policy push, kill switch | reimplementar routing/Smart Context em Go |
| Codex#5.5#D | Codex 5.5 High | F0, F7 | DevOps/PROD deploy, runbook, rollback, observability | mudar arquitetura sem ADR |
| GLM#52#A | GLM52 | F6, F9 | QA/conformance, replay, smoke, evidência | criar features |
| GLM#52#B | GLM52 | F4 | security/state, Postgres/Redis, redaction, audit | relaxar redaction/policy |
| Gemini#Pro | Gemini Pro | F5 | vendor capability matrix (fonte oficial) | implementar código |
| Gemini#Flash35 | Gemini Flash 3.5 | F8 | ops triage, status board, evidence index | decidir arquitetura |

## Fases e ordem (detalhe em tasks.md)
```
 já:      Gemini#Flash35 (F8 board)  +  Gemini#Pro (F5 matrix)
 v0:      Codex#5.5#A (F1 contrato)  +  GLM#52#B (F4 state/security)
 paralelo:Codex#5.5#B (F2 fork map)
 após v0: Codex#5.5#C (F3 integração + F0 lançar prodex)
 após state+integr: Codex#5.5#D (F7 runbook → F0 deploy PROD [gated])
 após fork+contrato: GLM#52#A (F6 QA)
 último:  Codex#5.5#B + GLM#52#A (F9 reset-claim — baixa prioridade, frio/aleatório)
```

## Escopo de vendors
Codex, Kiro, Antigravity, **Cline, OpenCode**. **Kimchi REMOVIDO.**

## Gates de aceite (resumo — completo em tasks.md)
Roteador único provado · fail-closed em troca de perfil · Smart Context shadow→canary→live com
fallback exato · reset-claim matriz empírica scrubbed · conformance por capability · secrets
redaction test · Postgres (sem SQLite) · container verde + sidecar saudável + kill switch/rollback.

## Plataforma de execução — Herdr (multiplexer)
Skill do orquestrador: `docs/herdr/README.md`. Fonte: herdr.dev/docs. Regra: nunca inventar flag Herdr.
- **Tech-Lead:** Opus 4.8 roda no pane **`opus-4.8-orchestrator`** (`herdr agent rename "$HERDR_PANE_ID" opus-4.8-orchestrator`).
- **Agentes:** skill `npx skills add ogulcancelik/herdr --skill herdr -g`; identidade durável `herdr agent rename "$HERDR_PANE_ID" "<agente>"`; só operam se `HERDR_ENV=1`.
- **Integrations:** Codex → `herdr integration install codex`; OpenCode → `install opencode`; Kiro/Antigravity/Cline/GLM/Gemini → screen-detection.
- **Coordenação bidirecional:** agente→Tech-Lead via `herdr agent send opus-4.8-orchestrator "..."` + `notification show --sound request`; Tech-Lead monitora via `events.subscribe` (`pane.agent_status_changed`: blocked/done) e dirige via `agent list/read/send/start`.

## Status
- [x] ADR-001 · PRD fechado · OpenSpec change criado · rotation-router supersedido
- [ ] Prompts dos 8 agentes emitidos (aguardando "vai" do dono para dispatch)
- [ ] F0 deploy PROD (gated por runbook F7 apresentado ao dono)

## Superfícies completas (varredura — 66 tasks, 39 REQs)
Plano cobre agora: MCP (REQ-26), 44 crates (00c), env/subcomandos/providers (00d, REQ-33/35/36), Caveman OFF/RCE (REQ-34), browser Playwright (REQ-37), Mem0 (REQ-37b), redaction presidio (REQ-28), broker (REQ-29), cookies (REQ-30), quota (REQ-31), CI hardening (REQ-38), deploy Helm (REQ-39). Fonte viva: `openspec .../tasks.md` (66) via `plan_dashboard.py`. Erros passados: `.planning/RCA-2026-07-04-001`.

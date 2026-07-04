> **⛔ MANDATÓRIO:** leia TUDO antes de qualquer ação — em especial o **RCA dos 22 erros** (`.planning/RCA-2026-07-04-001-orchestrator-errors.md`), a **arquitetura AS-IS/TO-BE** (`docs/project/architecture_{as_is,to_be}.html` + ADR-001), e **instale/comunique via Herdr** (skill: `npx skills add ogulcancelik/herdr --skill herdr -g`; fale com o TL só via `opus-4.8-orchestrator` + `pane run`). Detalhes: `HANDOFF_APENDICE_REFERENCIA.md`.

# HANDOFF — Continuação do Rotation-Parity Polyglot (para o próximo agente)

> **Leia este documento inteiro antes de tocar em qualquer coisa.** Depois leia, nesta ordem:
> `Diligencias/00_LEIA_PRIMEIRO_MISSAO.md` (charter) → `00_CONTEXTO_MULTICA.md` → sua fase.
> Data do handoff: 2026-07-04. **Apêndice com TODOS os fullpaths/git/SSH/agentes/comandos:** `Diligencias/HANDOFF_APENDICE_REFERENCIA.md`. Autor anterior: Opus 4.8 (Tech-Lead/orquestrador).
> Repo: `R-D_Agnostic_Engineering_Team` (remote `rnd`), branch `main`, último commit da sessão `cf3a523`.

---

## 1. TL;DR — onde estamos
- **Fase:** PLANEJAMENTO **concluído e consistente**. EXECUÇÃO **ainda não começou**.
- **Blocker #1 (SEV-0):** o **binário do prodex NÃO existe** (source em `/tmp/prodex-audit-7750da9`@`7750da9b`, mas sem Rust, sem build). É a **Fase P0** e bloqueia TUDO.
- **Produto Go:** compila e passa **24/24 pacotes de teste** em container (verificado).
- **Dashboard:** `scripts/dashboard/plan_dashboard.py` (data-driven, 66 tasks, QA 49/49, encoding-safe). É a **fonte viva** de progresso.
- **NÃO fale com o time/fleet sem autorização explícita do dono.**

## 2. O que é o projeto (resumo — detalhe no charter)
Fazer o **Multica** (Go, managed-agents platform) lançar o **prodex** (Rust L2) no lugar do `codex` cru,
para o caminho quente: rotação pré-commit, afinidade, **Smart Context/token-saver**, reset-claim.
Polyglot: **Go L4 (frio) + prodex Rust L2 (quente)**, contrato `rpp.l2.v1`, um roteador por sessão.
prodex **AS-IS** agora (pinado v0.246.0/`7750da9b`) → **fork** depois. Ver ADR-001.

## 3. O que foi feito nesta sessão (planning 100% corrigido)
O plano anterior tinha furos graves (assumia binário instalado, 0 tasks, sem MCP, sem segurança). Corrigido:
- **OpenSpec** `rotation-parity-polyglot`: **válido**, proposal/design/tasks(**66**)/4 specs.
- **GSD** `.planning/`: PROJECT/REQUIREMENTS(**40 REQs**)/ROADMAP(P0→P9 + grafo)/STATE.
- **Diligências** `Diligencias/`: charter + contexto + deps(00b) + crates(00c) + env/sec(00d) + completude(00e) + fases 00–06 + SEV0.
- **9 prompts** de agente reconciliados (charter #0 + banner + bloco de escopo por stream).
- **Superfícies do prodex mapeadas** (varredura da fonte): MCP, 44 crates, env/subcomandos/providers, Caveman(RCE), browser/Playwright, Mem0, broker, cookies, quota, CI, deploy.
- **RCA** dos 22 erros que cometi: `.planning/RCA-2026-07-04-001-orchestrator-errors.md` (LEIA — não repita).
- **Evidência de tudo:** `Diligencias/SEV0/EVIDENCE_MASTER.md`.

## 4. Estado verificado (fatos, não suposição)
| Item | Estado |
|---|---|
| prodex source | ✅ `/tmp/prodex-audit-7750da9`@`7750da9b` (efêmero — mover p/ `~/runtime/prodex-src`) |
| prodex binário | ❌ NÃO buildado (sem Rust) |
| Rust toolchain | ❌ ausente (usar container `rust:1.85-bookworm`, edition 2024) |
| Postgres/Redis | ✅ docker healthy (:5432/:6379) |
| Multica server | ✅ build+vet+24/24 testes verdes (container, IPv6 OFF) |
| Migrations | ✅ 322 .sql reversíveis |
| Config Multica prodex | ❌ `MULTICA_PRODEX_*`/`PRODEX_HOME` não setados |

## 5. PRÓXIMOS PASSOS (ordem)
1. **P0 Fundação (BLOQUEIA TUDO)** — executar `Diligencias/00_FUNDACAO_P0.md` (passos 0.1–0.9): estabilizar source → Rust container → `cargo build --release` → verificar pin+hash → setar env → gate Go. Prompt executor: `agentic-prompts-hub/new_prompts/PROMPT_CODEX-5.5-C_RPP-FOUNDATION-P0.md`. Dependências/origens: `Diligencias/00b_DEPENDENCY_SOURCES.md`.
2. **P1 Contrato** → **P2 Fork-map** / **P4 State** / **P5 Vendors** → **P3 Integração** → **P6 QA EXAUSTIVO** → **P7 Deploy**. **P9 Reset-claim** por último.
3. **Dimensões declaradas pendentes de varredura** (não esquecer — estão em 00e): flags de cada subcomando prodex; mapa rota-a-rota Multica×contrato; CI do prodex para espelhar no P6.

## 6. Decisões PENDENTES do dono (owner-only — NÃO decida)
- **F5 vendor sign-off** (`docs/vendors/owner-acceptance-request.md`): ACCEPT/REJECT das capabilities `not_validated`.
- **F7 deploy PROD:** só após P6 (QA exaustivo) verde + kill-switch/rollback TESTADOS + Caveman OFF.

## 7. RISCOS DE SEGURANÇA (não pular)
- **Caveman/hook** (`PRODEX_CAVEMAN_HOOK_*`) = **RCE/supply-chain** → **DESABILITADO por padrão** (REQ-34).
- `PRODEX_ALLOW_UNSAFE_CHILD_ENV=off`; chaves via secret-store, nunca em log.
- **Browser automation** (Playwright/Chromium, REQ-37) = superfície → sandbox/allowlist.
- Redaction real via `prodex-presidio`+`prodex-redaction` (REQ-28); sem SQLite compartilhado.

## 8. MAPA DE DOCUMENTOS (onde está tudo)
- Autoridade: `openspec/changes/rotation-parity-polyglot/` (proposal/design/tasks/specs).
- Planejamento GSD: `.planning/` (PROJECT/REQUIREMENTS/ROADMAP/STATE/RCA).
- Diligências operacionais: `Diligencias/` (charter, contexto, 00b/c/d/e, fases, SEV0).
- Dashboard: `scripts/dashboard/plan_dashboard.py` (+ testes `test_plan_dashboard*.py`).
- Prompts: `agentic-prompts-hub/new_prompts/PROMPT_*RPP*.md`.
- Bases de conhecimento pesquisáveis: `rpp-projeto-diligencias`, `rpp-openspec-plano`, `rpp-gsd-planning`.

## 9. COMO VERIFICAR O ESTADO (um comando)
```
openspec validate rotation-parity-polyglot && \
python3 scripts/dashboard/test_plan_dashboard.py && \
python3 scripts/dashboard/test_plan_dashboard_sev0.py && \
python3 scripts/dashboard/plan_dashboard.py --once --ascii
```

## 10. REGRAS DE OPERAÇÃO (aprenda com meus 22 erros — ver RCA)
1. **Verifique na fonte antes de afirmar/agir** — declare confiança (verificado vs suposto).
2. **Não invente** flag/comando (prodex/Herdr). **Não confie no tail** — re-rode e exija evidência.
3. **Não escale non-issue ao dono.** **Não sobre-engenharie.** **QA nunca bypassado.**
4. **Verde-em-container + sign-in/out em disco** antes de DONE. **IPv6 OFF** nos builds.
5. Se travar/ambíguo: **PARE e escale ao Opus/dono** — não decida sozinho.
6. Completude = matrizes 00c/00d/00e. Gap = linha faltando lá, não surpresa de memória.

## 11. Ambiente do fleet (harness)
Agentes rodam em panes Herdr no host `manoelneto-laptop` (SSH). Workers Codex isolados por `CODEX_HOME`
(`~/.codex-a/b/c/d`, 4 contas distintas — clobber resolvido). Só operar Herdr se `HERDR_ENV=1`.
Falar com o time só via `opus-4.8-orchestrator` e **só com autorização do dono**.

---
**Estado final da minha sessão:** planejamento completo/consistente/validado com evidência; execução (P0) pendente.
Comece pela P0. Boa sorte — e leia o RCA antes.
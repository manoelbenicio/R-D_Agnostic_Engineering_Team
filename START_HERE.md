# 🟢 START HERE — DOCUMENTO CONSOLIDADO ÚNICO (Rotation-Parity Polyglot)

> Este é o **único** documento que o próximo agente precisa abrir. Contém, em ordem:
> (1) prompt de onboarding pra colar na IDE · (2) handoff (estado/próximos passos) · (3) apêndice (todos os paths/git/SSH/Herdr/roster).
> Fonte de verdade viva: openspec/changes/rotation-parity-polyglot + .planning + Diligencias. Repo remote `rnd` (branch main).

---

# PARTE 1 — PROMPT DE ONBOARDING (cole na IDE do novo agente)

---

```
# ASSUMIR PROJETO — Rotation-Parity Polyglot (Multica + prodex)

Você está assumindo este projeto AGORA como Tech-Lead/executor. NÃO toque em nada antes de
cumprir a LEITURA OBRIGATÓRIA. Repo: /mnt/c/VMs/Projetos/Automonous_Agentic
(remote git `rnd` = github.com/manoelbenicio/R-D_Agnostic_Engineering_Team, branch main).

## PASSO 1 — LEITURA MANDATÓRIA (leia TUDO, nesta ordem, antes de agir)
1. /mnt/c/VMs/Projetos/Automonous_Agentic/HANDOFF.md
2. /mnt/c/VMs/Projetos/Automonous_Agentic/HANDOFF_APENDICE.md   (fullpaths, git, SSH, roster/panes, comandos)
3. /mnt/c/VMs/Projetos/Automonous_Agentic/Diligencias/00_LEIA_PRIMEIRO_MISSAO.md   (missão & mundo + regras)
4. /mnt/c/VMs/Projetos/Automonous_Agentic/Diligencias/00_CONTEXTO_MULTICA.md
5. /mnt/c/VMs/Projetos/Automonous_Agentic/.planning/RCA-2026-07-04-001-orchestrator-errors.md   (22 erros — NÃO repita)
6. Arquitetura AS-IS/TO-BE: /mnt/c/VMs/Projetos/Automonous_Agentic/docs/project/architecture_as_is.html
   e architecture_to_be.html + docs/rotation-parity-polyglot/02_ADR-001-arquitetura.md
7. Completude: Diligencias/00b_DEPENDENCY_SOURCES.md, 00c_PRODEX_CRATE_COVERAGE.md,
   00d_CONFIG_ENV_SECURITY.md, 00e_COMPLETENESS_REVIEW.md
8. Autoridade: openspec/changes/rotation-parity-polyglot/ (proposal, design, tasks[66], specs)
   + .planning/{PROJECT,REQUIREMENTS,ROADMAP,STATE}.md

## PASSO 2 — CONECTAR NO HERDR (comunicar com o Tech-Lead / agentes)
npx skills add ogulcancelik/herdr --skill herdr -g
export HERDR_ENV=1
# Fleet host: LAN 192.168.1.27 (alias `manoelneto-laptop`), user dataops-lab, Port 22.
# Falar SOMENTE com opus-4.8-orchestrator, e SÓ com autorização do dono.
# PADRÃO (SSH pro fleet + herdr; pane_id NÃO é durável, reconfirme sempre):
PANE=$(ssh manoelneto-laptop "herdr agent list" | python3 -c "import sys,json;print(next(a['pane_id'] for a in json.load(sys.stdin)['result']['agents'] if a['name']=='opus-4.8-orchestrator'))")
ssh manoelneto-laptop "herdr pane run $PANE $(printf %q '[Tech-Lead] sua mensagem')"   # pane run SUBMETE (Enter). NUNCA agent send.
ssh manoelneto-laptop "herdr pane read $PANE --source recent --lines 40"
# Regras Herdr: pane run (não agent send, que não dá Enter) · shlex/printf %q pra escapar · reconfirmar pane sempre.

## PASSO 3 — VERIFICAR O ESTADO (rode e confirme verde)
cd /mnt/c/VMs/Projetos/Automonous_Agentic && \
openspec validate rotation-parity-polyglot && \
python3 scripts/dashboard/test_plan_dashboard.py && \
python3 scripts/dashboard/test_plan_dashboard_sev0.py && \
python3 scripts/dashboard/plan_dashboard.py --once --ascii

## PASSO 4 — CONFIRMAR ENTENDIMENTO (responda antes de executar)
(a) o que é o projeto; (b) fase atual + blocker #1; (c) 3 erros do RCA que NÃO vai repetir;
(d) riscos de segurança (Caveman/RCE, ALLOW_UNSAFE_CHILD_ENV, browser); (e) seu 1º passo.

## PASSO 5 — ASSUMIR E EXECUTAR
- Estado: PLANEJAMENTO concluído; EXECUÇÃO não iniciada. Blocker #1 = binário prodex NÃO buildado.
- Comece pela FASE P0 (Fundação): Diligencias/00_FUNDACAO_P0.md + origens em 00b_DEPENDENCY_SOURCES.md
  + prompt executor agentic-prompts-hub/new_prompts/PROMPT_CODEX-5.5-C_RPP-FOUNDATION-P0.md.
- Sequência: P0 → P1 → (P2/P4/P5) → P3 → P6 (QA EXAUSTIVO) → P7. P9 (reset-claim) por último.

## REGRAS INEGOCIÁVEIS
- Verifique na FONTE antes de afirmar/agir; declare confiança (verificado vs suposto).
- Verde-em-container + sign-in/out em disco antes de DONE; não confie no tail (re-rode).
- Nada inventado (flags/comandos prodex/Herdr). QA NUNCA bypassado. IPv6 OFF nos builds.
- Caveman/hook DESABILITADO por padrão (RCE); sem segredo em log; SQLite proibido p/ estado.
- Decisões do dono (F5 vendor sign-off, F7 deploy) NÃO são suas — apresente, não decida.
- Se travar/ambíguo: PARE e escale (dono/TL). Não empurre bucha. Sempre documente no disco.

Confirme que leu tudo (Passo 1–4) e então assuma a P0.
```

---

# PARTE 2 — HANDOFF (estado, próximos passos, riscos, regras)

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

---

# PARTE 3 — APÊNDICE DE REFERÊNCIA (paths, git, SSH/Herdr, roster, comandos, arquitetura)

# HANDOFF — Apêndice de Referência COMPLETA (fullpaths, git, SSH, agentes, comandos)

> Companion do `HANDOFF_PROXIMO_AGENTE.md`. Tudo nos mínimos detalhes. Caminhos ABSOLUTOS.

## 1. MÁQUINAS
### Host de orquestração (onde VOCÊ/Tech-Lead roda)
- user/host: `dataops-lab@21LAPGLMVPJ4`
- Repo (planejamento + dashboard + prompts): `/mnt/c/VMs/Projetos/Automonous_Agentic`  ← note "Proje**t**os"
### Host do FLEET (onde os agentes executam)
- SSH: `ssh manoelneto-laptop` (alias) ou `ssh dataops-lab@192.168.1.27`. Host na **LAN (mesma rede local)** — IP **`192.168.1.27`**, User `dataops-lab`, Port 22. (se o `~/.ssh/config` tiver HostName diferente, ajustar para `192.168.1.27`.)
- **IP LAN/PROD do host:** `192.168.1.27` (confirmado por `ping -a manoelneto-laptop`).
- Teste: `ssh -o BatchMode=yes manoelneto-laptop 'echo ok'`
- Repo fleet (clone do produto): `/mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team`  ← note "Proje**c**ts"
- Produto Multica (Go): `/mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/multica-auth-work`
- Board/check-ins: `/mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/.deploy-control/`
- Evidências: `/mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/.deploy-control/evidence/`

## 2. GIT
- Remote de trabalho: **`rnd`** = `https://github.com/manoelbenicio/R-D_Agnostic_Engineering_Team.git` (branch `main`)
- Remote `origin` = `https://github.com/manoelbenicio/Agentic_Autonomous.git` (NÃO usar — é outro projeto)
- Último commit da sessão: `7a14377`
- Commit/push padrão:
  ```
  cd /mnt/c/VMs/Projetos/Automonous_Agentic
  git add -A <arquivos>
  git -c user.name="mbenicios" -c user.email="mbenicios@users.noreply.github.com" commit -m "..."
  git push rnd main
  ```

## 3. ONDE ESTÁ CADA COISA (fullpaths)
| Item | Caminho absoluto |
|------|------------------|
| **OpenSpec (autoridade)** | `/mnt/c/VMs/Projetos/Automonous_Agentic/openspec/changes/rotation-parity-polyglot/` |
| — proposal/design/tasks | `.../{proposal.md,design.md,tasks.md}` (tasks = **66**) |
| — specs | `.../specs/{prodex-runtime-provisioning,l2-runtime-contract,qa-conformance,deploy-rollback}/spec.md` |
| **GSD (planejamento)** | `/mnt/c/VMs/Projetos/Automonous_Agentic/.planning/` |
| — arquivos | `PROJECT.md · REQUIREMENTS.md (40 REQs) · ROADMAP.md · STATE.md · RCA-2026-07-04-001-orchestrator-errors.md` |
| **Diligências** | `/mnt/c/VMs/Projetos/Automonous_Agentic/Diligencias/` |
| — charter/contexto | `00_LEIA_PRIMEIRO_MISSAO.md · 00_CONTEXTO_MULTICA.md` |
| — refs | `00b_DEPENDENCY_SOURCES.md · 00c_PRODEX_CRATE_COVERAGE.md · 00d_CONFIG_ENV_SECURITY.md · 00e_COMPLETENESS_REVIEW.md` |
| — fases | `00_FUNDACAO_P0.md · 01_CONTRATO_P1.md · 02_FORKMAP_P2.md · 03_INTEGRACAO_P3.md · 04_STATE_SECURITY_P4.md · 05_VENDOR_MATRIX_P5.md · 06_QA_CONFORMANCE_P6.md` |
| — SEV0 | `SEV0/{RISK_REGISTER,EVIDENCE_LOG,DASHBOARD_QA_VERDICT,EVIDENCE_MASTER}.md` |
| — handoff | `HANDOFF_PROXIMO_AGENTE.md` + este apêndice |
| **Prompts dos agentes** | `/mnt/c/VMs/Projetos/Automonous_Agentic/agentic-prompts-hub/new_prompts/PROMPT_*RPP*.md` |
| **Dashboard** | `/mnt/c/VMs/Projetos/Automonous_Agentic/scripts/dashboard/plan_dashboard.py` (+ `test_plan_dashboard*.py`) |
| **prodex source** | `/tmp/prodex-audit-7750da9` (efêmero!) → mover p/ `~/runtime/prodex-src` |
| MASTER (board) | `.deploy-control/MASTER_ROTATION_PARITY_POLYGLOT.md` (no repo Automonous_Agentic e no fleet) |

## 4. AGENTES (roster + panes Herdr, host fleet)
| Agente | pane | Papel / fase |
|--------|------|--------------|
| `opus-4.8-orchestrator` | `w3:pE` | TL operacional (orquestra o fleet) |
| `Codex#5.5#A` | `w3:pJ` | Contrato (P1) |
| `Codex#5.5#B` | `w3:pM` | Fork-map (P2) / reset-claim (P9) |
| `Codex#5.5#C` | `w3:pK` | **Fundação P0** + integração (P3) |
| `Codex#5.5#D` | `w3:p9` | DevOps/Deploy (P7) |
| `GLM#52#A` | `w3:pT` | QA/conformance (P6) |
| `GLM#52#B` | (— ver `herdr agent list`) | State/security (P4) |
| `GLM#52#CLINE#A/B` | `w3:pS`/`w3:pR` | auditorias |
| `Gemini#PRO#31` | `w3:pN` | Vendor matrix (P5) |
| `Gemini#OPUS46` | `w3:pP` | (squad AOP) |
| `NEMOTRON#A` | `w3:pQ` | (smoke) |
> Contas Codex isoladas: `~/.codex-a/b/c/d` (1 conta distinta por worker — clobber resolvido).
> panes mudam; reconfirme sempre com `herdr agent list`.

## 5. COMO FALAR COM O TL (orquestrador) — SÓ com autorização do dono
`agent send` NÃO submete (não dá Enter). Para SUBMETER use `pane run`:
```
# resolver o pane do orquestrador
ssh manoelneto-laptop "herdr agent list" | grep opus-4.8-orchestrator   # pega pane_id (ex: w3:pE)
# enviar E submeter (texto + Enter):
ssh manoelneto-laptop "herdr pane run w3:pE 'sua mensagem'"
# ler resposta:
ssh manoelneto-laptop "herdr pane read w3:pE --source recent --lines 40"
```
Regra: **falar SÓ com `opus-4.8-orchestrator`** entre os agentes; nunca mandar direto pros outros.

## 6. COMO RODAR O DASHBOARD (fonte viva de progresso)
```
cd /mnt/c/VMs/Projetos/Automonous_Agentic
python3 scripts/dashboard/plan_dashboard.py            # snapshot (66 tasks, 11 fases)
python3 scripts/dashboard/plan_dashboard.py --watch    # ao vivo (5s; Ctrl+C sai)
python3 scripts/dashboard/plan_dashboard.py --ascii    # terminal sem UTF-8
python3 scripts/dashboard/plan_dashboard.py --json     # máquina
# QA do dashboard:
python3 scripts/dashboard/test_plan_dashboard.py        # 29/29
python3 scripts/dashboard/test_plan_dashboard_sev0.py   # 20/20
```
(`fleet_dashboard.py` = SUPERSEDED, não use.)

## 7. COMO RODAR O GSD
- Skills instaladas: `~/.kiro/skills/gsd-*` (124). Workflows: `~/.claude/get-shit-done/{workflows,templates,references}`.
- Invocação por slash-command no chat, ex.:
  - `/gsd:plan-phase P0`  — detalhar a fase P0
  - `/gsd:execute-phase P0` — executar
  - `/gsd:progress` — situação
  - `/gsd:new-milestone` — novo milestone
- Estado GSD vive em `.planning/` (PROJECT/REQUIREMENTS/ROADMAP/STATE).

## 8. QUE FASE ESTAMOS
- **Milestone:** v2.0 "Fundação + Deploy Correto".
- **Planejamento:** ✅ concluído (66 tasks, 40 REQs, specs, diligências, prompts, RCA, evidência).
- **Execução:** ⏳ **P0 (Fundação) — NÃO iniciada.** É o próximo passo e bloqueia tudo.
- **Dashboard:** `0/66` (nada executado ainda — honesto).

## 9. VERIFICAR TUDO (um comando)
```
cd /mnt/c/VMs/Projetos/Automonous_Agentic && \
openspec validate rotation-parity-polyglot && \
python3 scripts/dashboard/test_plan_dashboard.py && \
python3 scripts/dashboard/test_plan_dashboard_sev0.py && \
python3 scripts/dashboard/plan_dashboard.py --once --ascii
```

## 10. BASES DE CONHECIMENTO (pesquisáveis, neste host)
`rpp-projeto-diligencias` · `rpp-openspec-plano` · `rpp-gsd-planning`. (Rodam neste host; o fleet consome via repo/`git pull`.)


## 11. ARQUITETURA — desenhos AS-IS / TO-BE
- **Diagrama AS-IS:** `/mnt/c/VMs/Projetos/Automonous_Agentic/docs/project/architecture_as_is.html`
- **Diagrama TO-BE:** `/mnt/c/VMs/Projetos/Automonous_Agentic/docs/project/architecture_to_be.html`
- Outros: `docs/architecture.html` · `docs/network-architecture.html` · `docs/ARCHITECTURE_LOCAL_VS_CLOUD.md` · `docs/project/04-architecture.md`
- **ADR-001 (decisão de arquitetura):** `docs/rotation-parity-polyglot/02_ADR-001-arquitetura.md`
- **Textual (autoridade):** `openspec/changes/rotation-parity-polyglot/design.md` — §1 Camadas L4/L2, **§2 Horizonte AGORA (AS-IS: prodex AS-IS)**, **§3 Horizonte ALVO (TO-BE: fork Rust L2)**.
- PRD/Plataforma: `docs/rotation-parity-polyglot/{01_PRD.md,03_PLATFORM_PLAN_360.md}`.

## 12. LEITURA MANDATÓRIA — ANTES DE QUALQUER COISA
Leia, nesta ordem, TUDO antes de agir:
1. `Diligencias/HANDOFF_PROXIMO_AGENTE.md` + este apêndice.
2. `Diligencias/00_LEIA_PRIMEIRO_MISSAO.md` (charter) → `00_CONTEXTO_MULTICA.md`.
3. **`/mnt/c/VMs/Projetos/Automonous_Agentic/.planning/RCA-2026-07-04-001-orchestrator-errors.md`** — os **22 erros grotescos** cometidos (não repita nenhum).
4. Arquitetura AS-IS/TO-BE (§11) + ADR-001 + design.md.
5. Refs de completude: `00b`(deps) · `00c`(44 crates) · `00d`(env/segurança) · `00e`(completude).
6. Sua fase: `Diligencias/0X_*.md` + `openspec .../tasks.md`.

## 13. HERDR — comunicar com o TL + INSTALAR A SKILL (obrigatório)
Você DEVE se comunicar com o Tech-Lead SOMENTE via Herdr, e SÓ com autorização do dono.
**Instale a skill Herdr para você (1x no start):**
```
npx skills add ogulcancelik/herdr --skill herdr -g     # instala a skill
export HERDR_ENV=1                                       # só opere Herdr com isto setado
herdr integration install codex                          # integração nativa (se aplicável)
```
**Falar com o TL (submeter com Enter via pane run):**
```
ssh manoelneto-laptop "herdr agent list" | grep opus-4.8-orchestrator   # pega o pane
ssh manoelneto-laptop "herdr pane run <pane> 'mensagem'"                 # envia+Enter
ssh manoelneto-laptop "herdr pane read <pane> --source recent --lines 40"
```
Regra: só `opus-4.8-orchestrator`; `agent send` NÃO submete (use `pane run`).

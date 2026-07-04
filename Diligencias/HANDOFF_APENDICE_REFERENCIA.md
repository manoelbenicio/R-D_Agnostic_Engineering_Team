# HANDOFF — Apêndice de Referência COMPLETA (fullpaths, git, SSH, agentes, comandos)

> Companion do `HANDOFF_PROXIMO_AGENTE.md`. Tudo nos mínimos detalhes. Caminhos ABSOLUTOS.

## 1. MÁQUINAS
### Host de orquestração (onde VOCÊ/Tech-Lead roda)
- user/host: `dataops-lab@21LAPGLMVPJ4`
- Repo (planejamento + dashboard + prompts): `/mnt/c/VMs/Projetos/Automonous_Agentic`  ← note "Proje**t**os"
### Host do FLEET (onde os agentes executam)
- SSH: `ssh manoelneto-laptop`  (config: `HostName 192.168.1.27` (LAN), User `dataops-lab`; alt Tailscale `100.98.214.121`
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

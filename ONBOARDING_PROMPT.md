# ONBOARDING — Prompt para colar na IDE do próximo agente (assumir o projeto)

> Cole TODO o bloco abaixo na IDE do novo agente. Ele lê o mandatório, conecta no Herdr e assume a P0.

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

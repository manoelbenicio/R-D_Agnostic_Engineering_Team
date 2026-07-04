# RCA-2026-07-04-001 — Erros do Orquestrador/Planejamento (Opus 4.8)

> **Onde este documento vive:** `.planning/RCA-2026-07-04-001-orchestrator-errors.md` (GSD).
> Registro **blameless** de TODAS as cagadas/furos cometidos na sessão de 2026-07-04, com causa-raiz,
> impacto, correção (commit) e controle de prevenção. Referenciado por `STATE.md`.

## Causa-raiz sistêmica (o padrão por trás de quase tudo)
**Planejei/afirmei "de memória" em vez de verificar contra a fonte primária; agi/escalei antes de
verificar; e sobre-engenharei.** Tudo abaixo é sintoma disso. Controle-mãe adotado: **verificar na
fonte antes de afirmar/agir; declarar confiança (verificado vs suposto); não inventar; não escalar
non-issue; validar o efeito de toda ação.**

## Registro de erros

| ID | Erro | Categoria | Causa-raiz | Impacto | Correção | Prevenção |
|----|------|-----------|-----------|---------|----------|-----------|
| ERR-01 | Afirmei "1 conta Codex" | Evidência falsa | Tirei de `auth.json` que **eu mesmo copiei** (circular) | Diagnóstico errado; irritação | Provado 4 contas distintas | Nunca concluir a partir de artefato próprio |
| ERR-02 | **Serializei os workers** (matou paralelismo) | Sobre-engenharia | Supus rotação de token sem verificar; fix era só isolar | Derrubou os workhorses | Revertido; isolamento CODEX_HOME | Usar o fix documentado (G1/R1), não inventar |
| ERR-03 | Escalei "shared-auth/Option-A" ao dono | Escalada indevida | Confiei no orquestrador; não cruzei com histórico (isolamento já resolvido no Go/HerdMaster) | Alarme falso ao dono | Retirado | Não escalar non-issue; cruzar com o que já existe |
| ERR-04 | 3 Codex no **cwd errado** pós-relaunch | Efeito colateral | Relaunch sem fixar cwd | Workers no repo errado | `cd` corrigido | Launcher fixa cwd+home |
| ERR-05 | Dashboard **UnicodeEncodeError** | QA incompleto | QA usou saída UTF-8 capturada; não testou terminal real | Dashboard "nunca funcionou" p/ o dono | reconfigure UTF-8 + `--ascii`; teste de encoding | QA testa modos de falha reais |
| ERR-06 | Dashboard inferiu "WORKING" do Herdr | Overstatement | Liveness ≠ progresso; 1 agente = 4 gates | Número falso (7 em curso) | Revertido p/ evidência | Status só por evidência em disco |
| ERR-07 | Números flip-flop (55/95/40%) | Fonte instável | Métrica inferida via SSH/Herdr flaky | Perda de confiança | plan_dashboard local | Fonte de verdade local determinística |
| ERR-08 | Aceitei "7 gates DONE+validated" | Confiar no tail | Repassei claim sem verificar | Falso progresso | Auditado (plan-done/live-gated) | Re-rodar/validar antes de aceitar DONE |
| ERR-09 | **Plano sem fundação** (binário prodex assumido) | Furo de planejamento | design dizia "instalação verificada" (nunca produzido) | Deploy impossível não-detectado | P0 Fundação + spec provisioning | Enumerar pré-requisitos de ambiente |
| ERR-10 | `tasks.md` com **0 tasks** rastreáveis | Formato errado | Não usei `- [ ] N.M` | Sem tracking | 63 tasks no formato certo | Validar via `openspec status` |
| ERR-11 | `specs/` nunca criado (step pulado) | Step pulado | Não segui o schema spec-driven | Change incompleto | 4 specs criados | Rodar `openspec status` até apply-ready |
| ERR-12 | "Postgres AUSENTE" (falso) | Verificação rasa | Só chequei cliente `psql`, não o server docker | Falso gap | Corrigido (pg17 healthy) | Verificar o serviço, não só o cliente |
| ERR-13 | **Esqueci o MCP** (dono repetiu várias vezes) | Esquecimento | Planejei de memória, sem varrer o source | Superfície de hot-path fora do plano | REQ-26 + tasks + contrato | Varredura sistemática da fonte |
| ERR-14 | Esqueci 6 crates (memory/presidio/broker/cookies/quota/caveman) | Cobertura incompleta | Sem varredura dos 44 crates | Superfícies fora do plano | Matriz 00c + REQ-27..32 | Matriz de cobertura obrigatória |
| ERR-15 | **Subestimei Caveman (RCE)** | Miss de segurança | Não li os env vars de hook | Risco de RCE/supply-chain silencioso | REQ-34: Caveman OFF por padrão | Varrer env/hooks p/ segurança |
| ERR-16 | Superfície `PRODEX_*`/subcomandos/providers ausente | Cobertura incompleta | Só 2 env vars mapeados | Config/segurança incompleta | 00d + REQ-33/35/36 | Inventário completo de env/CLI |
| ERR-17 | Toolchain `rust:1` **flutuante** | Impreciso | Não verifiquei edition/MSRV do prodex | Build poderia quebrar | `rust:1.85-bookworm` (edition 2024) | Ler rust-toolchain/edition da fonte |
| ERR-18 | Construí o **FLM** (throwaway) | Trabalho desperdiçado | Reimplementei o que o prodex/produto já faz | Esforço morto | FLM removido | Não duplicar capacidade existente |
| ERR-19 | Sem doc de **origem das dependências** | Documentação incompleta | Documentei passos, não origens | Time sem saber de onde baixar | 00b_DEPENDENCY_SOURCES | Documentar origem+pin sempre |
| ERR-20 | Sem **contexto do produto / charter** | Onboarding ausente | Assumi conhecimento prévio | Time sem saber "que projeto é esse" | 00_CONTEXTO + 00_LEIA_PRIMEIRO | Charter obrigatório #0 |
| ERR-21 | Prompts não atualizados ao plano corrigido | Reconciliação tardia | Corrigi OpenSpec/GSD mas não os prompts | Prompts stale | Banner + prompt C reescrito | Reconciliar TODAS as camadas juntas |
| ERR-22 | Amplifiquei a tempestade de relogin | Ação prematura | Acordei todos os workers antes de isolar | Dono relogou 10-15x | Isolamento + serialização temporária | Isolar ANTES de ativar em massa |

## Controles permanentes adotados (para não repetir)
1. **Verificar na fonte antes de afirmar/agir** — e marcar confiança (verificado vs suposto).
2. **Matrizes de cobertura** como fonte de completude (00c crates, 00d env/CLI) — gap = linha faltando, não surpresa.
3. **Charter #0 obrigatório** + docs de origem/deps.
4. **QA testa modos de falha reais** (encoding, edge, fuzz) — não só o caminho feliz.
5. **DONE só com evidência re-rodada** (não confiar no tail).
6. **Não escalar non-issue; não inventar; não sobre-engenharia.**

## Timeline (resumo)
Sessão iniciou em monitoramento de fleet → descoberta do clobber de auth → correções erradas (serializar) →
reversão → descoberta do furo de fundação (binário) → replanejamento completo (OpenSpec/GSD/Diligências) →
varreduras de completude (MCP, 44 crates, env/segurança) → charter/contexto → base de conhecimento → este RCA.

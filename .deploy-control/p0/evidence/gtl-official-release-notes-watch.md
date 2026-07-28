# GTL-18 — Vendor release watch (OpenAI + Anthropic) — READ-ONLY, sem upgrade

Auditor: Codex56#A (ORQ2, pane `w7:p3`) · coleta UTC 2026-07-27T11:35Z
Fontes oficiais (unicas usadas):
- OpenAI Codex changelog: <https://developers.openai.com/codex/changelog> (entradas datadas)
- Anthropic Claude Code CHANGELOG (repo oficial `anthropics/claude-code`):
  <https://raw.githubusercontent.com/anthropics/claude-code/main/CHANGELOG.md> — versionado **sem data
  por entrada**; a data que posso afirmar e a da minha coleta (2026-07-27). O redirect de
  <https://docs.claude.com/en/release-notes/claude-code> aponta para esse CHANGELOG.

Nada instalado, nada atualizado, nada reiniciado.

## 1. Versoes instaladas (medidas)

| host | codex | claude |
|---|---|---|
| ORQ2 `ip-172-31-30-9` | `codex-cli 0.145.0` | `2.1.215 (Claude Code)` |
| ORQ1 `ip-172-31-18-217` | `codex-cli 0.144.6` | `2.1.218 (Claude Code)` |
| ultima versao oficial | **0.145.0** (2026-07-21) | **2.1.220** |

Deriva relevante: **codex esta desalinhado entre hosts** (0.145.0 vs 0.144.6) e **claude tambem**
(2.1.215 vs 2.1.218), com 2.1.220 disponivel. Fleet heterogeneo e risco de comportamento divergente
entre agentes que deveriam ser equivalentes.

## 2. Matriz OpenAI Codex CLI

| versao / data | release/change relevante | eixo | classificacao |
|---|---|---|---|
| **0.145.0** — 2026-07-21 | "Stabilized the opt-in multi-agent V2 experience with configurable sub-agent models, reasoning levels, concurrency, restored roles" (#33550, #34383) e "Unify multi-agent settings under `agents`" (#33550) | agentes paralelos + reasoning | **AGENDAR** — alinhar ORQ1 a 0.145.0 e revisar `config.toml`: a chave passou a ser `agents` |
| 0.145.0 — 2026-07-21 | "Migrate legacy exec policy allow rules" (#34271); em 0.143.0 "Remove the legacy exec policy engine" (#32093) | permissions | **AGENDAR** — potencial **breaking** de regras de exec-policy da frota; validar antes de subir ORQ1 |
| 0.145.0 — 2026-07-21 | "Honor managed permission profiles in network proxy resolution" (#34436) e "Resolve outbound proxy routes explicitly" (#34435) | gateway/egress | **AGENDAR** — interage direto com o desenho OmniRoute-only |
| 0.145.0 — 2026-07-21 | "Always confirm before enabling full access" (#32989), "Strengthen forced `rm` command detection" (#33464), "Propagate approval rejection reasons" (#34400) | permissions | **APLICAR** (quando o upgrade for autorizado) — endurecimento puro, sem custo |
| 0.145.0 — 2026-07-21 | "Reject forks of paginated threads" (#33109) + historico paginado com resume/search/memories (#33364, #33907, #34085, #34229, #34386) | sessao/persistencia | **AGENDAR** — **breaking** para qualquer automacao que dependa de fork de thread |
| 0.145.0 — 2026-07-21 | "Validate reasoning effort after applying spawn roles" (#33656), "Animate Max and Ultra reasoning effort changes" (#34365) | reasoning | **APLICAR** junto do upgrade |
| 0.145.0 — 2026-07-21 | Bedrock: login gerenciado, transporte custom, "GPT-5.6 Sol as the default Bedrock model" (#31327, #33170, #32288, #33695) | gateway/provider | **IGNORAR** por ora — nosso caminho e OmniRoute, nao Bedrock direto (decisao do owner se mudar) |
| 0.145.0 — 2026-07-21 | Audio in/out e realtime V3 (#33261, #33932, #34385) | fora de escopo | **IGNORAR** |
| 0.145.0 — 2026-07-21 | "Updated the packaged ripgrep binary to 15.2.0" (#34384) | ferramenta | **IGNORAR** (vem no pacote) |
| **0.144.6** — 2026-07-18 | "corrected their context windows to 272,000 tokens" para GPT-5.6 Sol/Terra/Luna (#33972, #34009) | reasoning/contexto | **APLICAR** — ja presente no ORQ1; ORQ2 em 0.145.0 tambem cobre |
| **0.144.5** — 2026-07-16 | "Improved dangerous-command detection, including more forced `rm` forms" (#33455) | permissions | **APLICAR** — coberto em ambos |
| **0.144.0** — 2026-07-09 | "Added a `writes` app-approval mode that allows declared read-only actions while prompting for writes" (#30482) | permissions | **AGENDAR** — casa com nossa regra read-only-por-default; avaliar adotar |
| 0.144.0 — 2026-07-09 | "Selecting Ultra reasoning now warns when high multi-agent concurrency could increase usage quickly" (#31621) | custo + paralelismo | **APLICAR** (informativo) |
| 0.143.0 — 2026-07-08 | Proxy de sistema (PAC/WPAD) para auth e Responses (#26708, #26709, #31335) | gateway | **AGENDAR** se o owner quiser egress via proxy |
| 2026-01-28 (geral) | "Web search is now enabled by default ... `web_search = \"disabled\"` to remove the tool" | egress/seguranca | **AGENDAR** — decidir explicitamente o valor para a frota |
| 2026-01-23 (geral) | Team Config: camadas `.codex/` + `requirements.toml` que "overrides defaults regardless of location" | governanca de config | **AGENDAR** — mecanismo oficial para padronizar os 9 agentes |

Nao encontrei, na pagina oficial, entrada de Codex CLI posterior a **0.145.0 (2026-07-21)**; a entrada
mais recente da pagina e "ChatGPT Voice and multi-folder projects 26.715" de **2026-07-23** (app, nao CLI).

## 3. Matriz Anthropic Claude Code

| versao | release/change relevante (texto oficial) | eixo | classificacao |
|---|---|---|---|
| **2.1.220** | "Bug fixes and reliability improvements" | manutencao | **AGENDAR** (alinhar frota) |
| **2.1.219** | "Subagents can now spawn nested subagents up to depth 3 by default (was 1); set `CLAUDE_CODE_MAX_SUBAGENT_SPAWN_DEPTH=1` to disable nesting" | **agentes paralelos** | **APLICAR com gate** — e a mudanca de maior blast radius do lote: multiplica processos-filho por agente. Recomendo fixar `=1` explicitamente **antes** de subir qualquer host para 2.1.219+ |
| 2.1.219 | "Added Claude Opus 5 (`claude-opus-5`), now the default Opus model — 1M context, fast mode at $10/$50 per Mtok"; "Removed Opus 4.7 from fast mode" | modelo/custo | **AGENDAR** — troca de default afeta custo por conta; decisao do owner |
| 2.1.219 | "Added `sandbox.network.strictAllowlist` setting to deny non-allowlisted hosts for sandboxed commands without prompting" | permissions/egress | **APLICAR** — encaixa no fail-closed do OmniRoute-only |
| 2.1.219 | "Added `DirectoryAdded` hook that fires after `/add-dir` or the SDK `register_repo_root` control request registers a new working directory mid-session" | **worktrees** | **AGENDAR** — gancho util para auditar entrada de worktree em sessao |
| 2.1.219 | "Added `mcp_server_errors` to the headless stream-json init event, listing `--mcp-config` entries skipped by config validation; terminal runs print a startup warning" | gateway/MCP | **APLICAR** — hoje falha de MCP passa silenciosa |
| 2.1.219 | "Added nested subagent forwarding in stream-json ... keyed by their spawning Agent `tool_use` id" | observabilidade de paralelismo | **APLICAR** |
| 2.1.219 | "Changed dynamic workflows to default to a medium size guideline (aim for fewer than 15 agents)" + `workflowSizeGuideline` | paralelismo | **AGENDAR** — definir o valor da frota |
| **2.1.218** | "Fixed gateway spend metering to price Bedrock application-inference-profile ARNs and other config-mapped upstream model IDs at the configured model's rates" | **gateway + custo por conta** | **APLICAR** — endereca exatamente o risco de custo mal atribuido |
| 2.1.218 | "Fixed agent frontmatter hooks running from untrusted folders: hooks now require the agent file's own folder to have accepted workspace trust" | **permissions (seguranca)** | **APLICAR** — correcao de execucao de hook nao confiavel; ORQ2 em 2.1.215 **nao tem** |
| 2.1.218 | "Improved sandbox command restrictions for IDE interactions"; "Improved auto mode: the dangerous-rm, background-`&`, and suspicious-Windows-path checks no longer open permission dialogs; the auto-mode classifier adjudicates them instead" | permissions | **AGENDAR** — muda quem decide (classificador em vez de dialogo); revisar antes |
| 2.1.218 | "Changed skills with `context: fork` to run in the background by default; opt out per skill with `background: false`" | paralelismo | **AGENDAR** — **breaking** de comportamento para skills existentes |
| 2.1.218 | "Improved trust dialogs to name the repository root the grant covers" | worktrees/permissions | **APLICAR** |
| 2.1.218 | "Fixed fork-session lineage being lost after compaction in headless and SDK sessions" | sessao | **APLICAR** |
| 2.1.218 | "Changed agent markdown files to reject agent names containing `:`" | config | **AGENDAR** — **breaking** se algum agente nosso usa `:` no nome |

## 4. Sintese de risco para a frota (o que eu recomendaria ao gate)

1. **Alinhar versoes antes de qualquer coisa.** Dois hosts em versoes diferentes de ambos os CLIs
   invalida comparacao entre agentes. Ordem sugerida: fixar a versao-alvo por CLI, aplicar em ORQ1 e
   ORQ2 na mesma janela, e registrar `--version` como evidencia.
2. **Anthropic 2.1.219 e o unico item com blast radius alto por default** (subagent depth 1→3). Se a
   frota subir sem `CLAUDE_CODE_MAX_SUBAGENT_SPAWN_DEPTH=1`, o numero de processos-filho por agente
   pode crescer sem mudanca de prompt.
3. **ORQ2 esta 3 patches atras em claude (2.1.215)** e portanto **sem** duas correcoes de seguranca de
   2.1.218: hooks de frontmatter exigindo workspace trust, e metering de gateway corrigido.
4. **Codex: risco de breaking em exec-policy** (motor legado removido em 0.143.0, regras migradas em
   0.145.0) e em **fork de thread paginada**. Testar antes de subir o ORQ1.
5. `web_search` default cached desde 2026-01-28 e uma decisao de egress ainda **implicita** na frota.

## 5. Nao-alegacoes

- Nao instalei, nao atualizei, nao reiniciei e nao alterei configuracao de nenhum CLI ou host.
- So usei as duas fontes oficiais citadas mais `--version` local; nao usei blog de terceiro, release
  notes agregado nem memoria do modelo.
- O CHANGELOG oficial do Claude Code **nao datava** as entradas na coleta; nao inventei datas — a
  unica data que afirmo para ele e a da coleta (2026-07-27). As datas da tabela Codex sao as impressas
  na pagina oficial.
- Nao verifiquei se `2.1.220` ja esta publicada no canal de instalacao usado pela frota (npm/nativo);
  a versao consta no CHANGELOG oficial, o que nao e prova de disponibilidade no canal.
- Nao avaliei kiro-cli, agy, cline nem opencode: fora do escopo do dispatch (somente OpenAI e Anthropic).
- Nao medi impacto de custo real das mudancas de modelo; a precificacao citada e a do texto oficial.

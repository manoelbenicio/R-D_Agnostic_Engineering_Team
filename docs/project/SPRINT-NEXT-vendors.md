# SPRINT-NEXT (Fase 3) — Vendors Kimchi + OpenCode + Cline

> **FRONTEIRA DURA.** Este documento é EXCLUSIVO da Fase 3 (Kimchi, OpenCode, Cline).
> A Fase 2 em curso (Codex, Kiro, Antigravity) está fechada e NÃO deve ser tocada por
> este plano. Nenhum código agora — este é o planejamento para o próximo sprint.
> Orquestrador/SME: Opus 4.8. Fonte de dados: docs/project/BACKLOG-detection.md
> (seções de pesquisa primária 2026-07-01) + este arquivo.

---

## 1. OBJETIVO DO SPRINT
Estender a rotação antecipada zero-interrupção para 3 novos vendors, com o MESMO rigor
corporativo da Fase 2, respeitando as réguas de cota REAIS de cada um (nada inventado).
Topologia PROD confirmada (Manoel): **os três rodam com ASSINATURA PRÓPRIA** (standalone),
não como wrappers de contas que já rotacionamos. Logo cada um é vendor de 1a classe com
sua própria cota e sua própria conta/API-key a rotacionar.

## 2. FATOS CONFIRMADOS (fonte primária) — base de todo o plano
### Kimchi (getkimchi/kimchi — kimchi.dev / docs.kimchi.dev)
- Coding agent CLI + inference layer + spend console. Binário standalone (bun compile),
  Homebrew `getkimchi/tap/kimchi`. Comandos: `kimchi setup`, `kimchi`.
- AUTH/ISOLAMENTO: `KIMCHI_API_KEY` (precede) -> `~/.config/kimchi/config.json:api_key`.
  Config/sessions em `~/.config/kimchi/harness/`. Ferment em `.kimchi/` no projeto.
- COTA: homepage diz "No rate limits or token caps"; a regua real e **SPEND BUDGET**
  (caps por API key/user/team/org, com % e forecast de run-rate — ex.: "$4,820/$6,000
  cap · 80%", "hits 80% budget tomorrow at current run rate"). Billing $/1M tokens.
- **Manoel confirmou que o LAYOUT de token usage do Kimchi e MUITO SEMELHANTE ao do
  Codex** -> provavel parse de % na PROPRIA saida da CLI (Classe A-like), nao so console.
  PENDENCIA-CHAVE: obter o TEXTO REAL do usage do Kimchi (Manoel vai colar) antes de
  escrever qualquer regex. Enquanto nao vier: NAO inventar padrao.

### OpenCode (opencode.ai — github.com/anomalyco/opencode)
- Agente de terminal open-source, 75+ providers. Install script / `npm i -g opencode-ai`
  / `brew install sst/tap/opencode`.
- CREDENCIAIS: `~/.local/share/opencode/auth.json` (via `/connect`); `opencode auth list`.
  Config `~/.config/opencode/opencode.json(c)`. Isolamento provavel: `XDG_DATA_HOME`
  (auth.json) + `XDG_CONFIG_HOME` (config). [A VERIFICAR CONTRA BINARIO].
- COTA (assinatura propria — OpenCode Zen/Go): sem comando de uso agregado nativo
  (GitHub issue #8911 "Add usage dashboard" ABERTA). Rate-limit surge como erro
  (#11242). -> deteccao REATIVA (429) ate confirmarmos qualquer superficie de uso.

### Cline (github.com/cline/cline — docs.cline.bot)
- CLI 2.0 (npm `cline`): interativo, one-shot, `--json` (NDJSON), `--yolo`, `--zen`.
- ISOLAMENTO NATIVO: `--data-dir <path>` (substitui `~/.cline`, ativa sandbox) +
  `CLINE_DATA_DIR`, `CLINE_SANDBOX=1`, `CLINE_SANDBOX_DATA_DIR`. Nao precisamos criar
  mecanismo proprio — o vendor ja isola.
- COTA (ClinePass propria): rate-limit 2-5x, sem teto fixo publicado. Sinal de uso por
  run = eventos NDJSON de `--json` (tokens/custo) + `-v` (stats). Sem `/usage` agregado.
  -> deteccao REATIVA (429). [A VERIFICAR CONTRA BINARIO] nomes de campo do NDJSON, caso
  queiramos um sinal proativo por tokens no futuro.

## 3. TRES CLASSES DE DETECCAO (modelo de arquitetura da Fase 3)
- **Classe A (cota na CLI, proativo por %)**: Kimchi (SE o layout colado confirmar % na
  saida — provavel). Reusa a maquinaria de percent de `usage.go` (Fase 2).
- **Classe B (assinatura sem teto publicado, reativo 429)**: OpenCode, Cline.
  Reusa `detector.go` (reativo) da Fase 2. Trabalho = COBERTURA de strings de limite.
- **Classe C (spend budget via console API)**: fallback do Kimchi SE nao houver % na CLI
  — probe HTTP ao console `app.kimchi.dev` por API-key (endpoint A VERIFICAR, nao inventar).

## 4. PADRAO DE ISOLAMENTO (execenv) — extensao, nao reescrita
Contrato REAL (server/internal/daemon/execenv/execenv.go): cada vendor tem um bloco em
`Prepare` e em `Reuse`, guardado por `params.Provider == "<vendor>"` e por
`params.CredentialAccountHome != ""`, chamando um helper
`prepare<Vendor>Home(dir, <Vendor>HomeOptions{AccountHome:...})` e expondo o resultado
num campo novo do struct `Environment`. Precedentes ja no codigo: `CodexHome`
(CODEX_HOME), `KiroDataHome` (XDG_DATA_HOME), `AntigravityHome` (HOME),
`OpenclawConfigPath`, `CursorDataDir`. A Fase 3 adiciona, no MESMO padrao:

| Vendor   | Lever de isolamento nativo (confirmado)          | Campo Environment sugerido |
|----------|--------------------------------------------------|----------------------------|
| Cline    | `CLINE_DATA_DIR` (ou `--data-dir`) + SANDBOX     | `ClineDataDir`             |
| OpenCode | `XDG_DATA_HOME` (auth.json) + `XDG_CONFIG_HOME`  | `OpenCodeDataHome`/`Config` |
| Kimchi   | `XDG_CONFIG_HOME` (~/.config/kimchi) + `KIMCHI_API_KEY` | `KimchiConfigHome`  |

NOTA: cada `prepare<Vendor>Home` seed-a AS-IS do `AccountHome` da conta, como
`prepareKiroHome` faz com `data.sqlite3`. Arquivo de credencial exato por vendor:
[A VERIFICAR CONTRA BINARIO] (Cline: conteudo de `~/.cline`/data-dir; OpenCode:
`auth.json`; Kimchi: `config.json` + harness/).

## 5. STREAMS PROPOSTOS (Fase 3) — cada um = arquivo novo ou lock exclusivo
Wave 1 (paralelo — arquivos novos, sem colisao):
- **F3-ISOLATE-CLINE**: `prepareClineHome` + `ClineDataDir` (arquivo novo
  `cline_home.go` + teste). Lock: so arquivos novos; NAO editar execenv.go ainda.
- **F3-ISOLATE-OPENCODE**: `prepareOpenCodeHome` (`opencode_home.go` + teste).
- **F3-ISOLATE-KIMCHI**: `prepareKimchiHome` (`kimchi_home.go` + teste).
- **F3-USAGE-KIMCHI** (so quando o layout real chegar): `kimchiUsageParser` em arquivo
  proprio, espelhando `codexUsageParser`. Bloqueado ate o texto real (regra de ouro).

Wave 2 (serial — hotspots, lock exclusivo, depende da Wave 1):
- **F3-EXECENV-WIRE**: fiar os 3 blocos `Provider=="cline|opencode|kimchi"` em
  `Prepare`/`Reuse` de execenv.go (lock exclusivo). Aditivo, AS-IS preservado.
- **F3-REACTIVE-COV**: estender `detector.go` (ou config de padroes) para reconhecer as
  strings de limite REAIS de Cline e OpenCode (429/rate-limit). Cada padrao marcado
  `CONFIRMAR CONTRA BINARIO` ate termos a saida real. Lock exclusivo detector.go.
- **F3-DAEMON-WIRE**: registrar os 3 providers no daemon (assignment->account, env
  injection: exportar CLINE_DATA_DIR / XDG_* / KIMCHI_API_KEY). Lock exclusivo daemon.go.
  Reusa `rotateTaskWithReason(ReasonQuotaProactive/Reactive)` ja existente.

Wave 3 (opcional/condicional):
- **F3-KIMCHI-BUDGET-PROBE**: SO se decidirmos Classe C (console). Probe HTTP spend-vs-cap
  por API-key. Depende de confirmar o endpoint `app.kimchi.dev` (NAO inventar). Se o
  layout de CLI do Kimchi ja der %, esta wave e desnecessaria.

## 6. DEPENDENCIAS E PRE-REQUISITOS (o que destrava o sprint)
- [PENDENCIA-CHAVE] Texto REAL do usage do Kimchi (Manoel vai colar) -> destrava
  F3-USAGE-KIMCHI e decide Classe A vs C do Kimchi.
- [A VERIFICAR CONTRA BINARIO, tarefa do Opus no inicio do sprint, nao do Manoel]:
  * Cline: arquivo(s) de credencial dentro do data-dir; string real de erro de rate-limit.
  * OpenCode: confirmar respeito a XDG_DATA_HOME/XDG_CONFIG_HOME; string real de 429.
  * Kimchi: se ha % de uso na saida da CLI; e, se Classe C, endpoint de spend do console.
- Contrato de rotacao (`contract.go`) NAO muda: `ReasonQuotaProactive`/`Reactive` cobrem.

## 7. RISCOS / NOTAS DE SME
- Meta-agentes (Cline/OpenCode) podem, em teoria, ser reconfigurados para apontar a outro
  provider; a topologia PROD fixa assinatura propria, entao tratamos a cota como do
  proprio vendor. Se isso mudar, revisitar (nao e o caso agora).
- Kimchi "no rate limits/token caps" na home vs. layout "igual Codex" que o Manoel viu:
  resolver com o texto real antes de codar — pode ser que o Kimchi mostre % de BUDGET na
  CLI (nao % de janela de tempo). O parser deve refletir a unidade REAL (spend), nao
  assumir tempo 5h so porque "parece Codex".
- Regra de ouro mantida em todo o sprint: nenhuma string/endpoint/flag de vendor e
  inventada; slots vazios + "CONFIRMAR CONTRA BINARIO" onde faltar dado real.

## 8. DEFINICAO DE PRONTO (Fase 3)
- Isolamento dos 3 vendors verde no container (execenv + testes), AS-IS da Fase 2 intacto.
- Deteccao: Kimchi proativo (A ou C, conforme dado real) + Cline/OpenCode reativos com
  strings reais confirmadas. Rotacao disparando via caminho existente.
- E2E de rotacao (Postgres real) reexecutado sem regressao dos 3 vendors da Fase 2.
- Board `.deploy-control/` com check-in/checkout de cada stream, build_result colado.

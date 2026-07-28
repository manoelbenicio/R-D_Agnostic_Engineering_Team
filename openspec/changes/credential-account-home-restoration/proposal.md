# Proposal — Credential Account Home Restoration

## Why

Os runtimes **agy**, **codex** e **kiro** sao obrigatorios em todo cenario (decisao do owner).
No diagnostico inicial eles operavam apenas com **uma conta por provider**, usando o HOME
global do processo do daemon no ORQ1. Nao havia isolamento por conta, rotacao nem atribuicao
de custo por conta. As secoes de implementacao abaixo registram a restauracao posterior.

A causa nao e feature ausente: e **regressao**. O wiring correto existiu no commit `aa62401`
(2026-07-02). O commit `31d50b9` (2026-07-05) condicionou esse wiring ao antigo L2; a remocao
completa do resolver e o `CredentiallessGateway: true` incondicional ocorreram em `9ab80a6`.

## What Changes

- **RESTORED** resolucao de `CredentialAccountHome` por task e por provider no daemon.
- **ADDED** ponte entre o registry de isolamento existente no ORQ2 e a selecao por task no daemon.
- **MODIFIED** o caminho nativo volta a preparar e injetar o ambiente isolado; o caminho
  gateway continua credentialless e nunca recebe `AccountHome`.
- **ADDED** atribuicao deterministica e persistente por `agent_id + provider`.
- **MODIFIED** o discovery de modelos AGY executa somente sob homes validados da allowlist,
  sem herdar o HOME global do daemon.
- **MODIFIED** o task-home AGY recebe somente o arquivo fisico
  `antigravity-oauth-token`; logs, symlinks, caches e bancos do provider nao sao copiados.
- **MODIFIED** o formulario de criar/duplicar agente persiste `thinking_level` quando o
  catalogo estruturado do runtime oferece `thinking.supported_levels`; modelos AGY, cujo tier
  ja faz parte do ID, continuam sem um segundo seletor.
- **ADDED** snapshot imutavel da conta produtora no claim atomico da task e copia server-side
  para `task_usage`, sem aceitar `account_id` do daemon.
- **ADDED** migration canonica `128_task_usage_account_id`, com exclusividade duravel de conta,
  rollback e preservacao de linhas legadas como nao atribuiveis.

## Scope

Vendors: **agy/antigravity, codex e kiro**.
**Non-goals:** `cline`, `opencode` e qualquer outro provider.

## Non-Goals

- Reimplementar isolamento de credencial. Ele **existe e funciona** no ORQ2.
- Inventar, normalizar ou substituir o catalogo fornecido por cada runtime/provider.
- Alterar o isolamento de slots existente no ORQ2.

## Impact

- Isolamento e afinidade por conta passam a existir no executor T2.
- Exige rebuild Go e relancamento do daemon (classe STOP-AND-WAIT).
- A persistencia financeira foi implementada, passou no gate combinado ORQ-12/ORQ-21 e
  recebeu revisao independente; a stack continua indivisivel para integracao e rollback.

## Implementation update — 2026-07-28

- `785a8ac` mantem linhas legadas `dispatched` com snapshot NULL visiveis ao reclaim, para que
  o gate fail-closed do ORQ-21 possa cancela-las; reclaim nunca resolve ou inventa conta.
- `ea1eee7` promove os arquivos staged para `128_task_usage_account_id.{up,down}.sql`, remove a
  configuracao SQLC temporaria e regenera a saida canonica.
- Gate descartavel: ORQ-12 19/19 em duas passagens, ORQ-21 handler 2/2 e registry 2/2 com
  `orq21db`, integracao 19+2+2, todos sob race, zero skips; build, vet e gofmt limpos.
- Os commits nao devem ser integrados isoladamente: ORQ-12 depende do cancelamento fail-closed
  do stack ORQ-21 compativel.
- O pool AGY foi reconstruido com quatro contas distintas e passou o gate concorrente com um
  snapshot de conta produtora exclusivo por task. Evidencia consolidada: `evidence.md`.

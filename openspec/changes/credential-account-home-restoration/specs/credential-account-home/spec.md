# Spec — credential-account-home

## ADDED Requirements

### Requirement: REQ-01 Resolucao por provider

O daemon MUST resolver a raiz de credencial especifica de cada provider obrigatorio.

#### Scenario: Preparar uma task obrigatoria

- **WHEN** o daemon prepara ambiente de execucao para uma task
- **THEN** `CredentialAccountHome` MUST ser resolvido pela raiz especifica do vendor dentro do
  slot, conforme: antigravity/agy `<slot>/home`, kiro `<slot>/xdg-data` e
  codex `<slot>/codex`
- **AND** MUST NOT usar um caminho unico para todos os providers

### Requirement: REQ-02 Validacao fail-closed do caminho

O daemon MUST rejeitar todo AccountHome ausente, invalido ou fora do root autorizado.

#### Scenario: Validar AccountHome

- **WHEN** um caminho de `AccountHome` e resolvido
- **THEN** MUST ser absoluto, MUST estar sob o root permitido, MUST passar por `EvalSymlinks` e
  `Stat`
- **AND** caminho ou artefato nativo ausente MUST retornar erro e bloquear a task
- **AND** MUST NOT converter erro em HOME global

### Requirement: REQ-03 Codex honra AccountHome

O daemon MUST separar o transporte credentialless do uso nativo de AccountHome pelo Codex.

#### Scenario: Executar por gateway

- **WHEN** existe plano gateway
- **THEN** `CredentiallessGateway` MUST ser verdadeiro e `AccountHome` MUST estar vazio

#### Scenario: Executar nativamente

- **WHEN** a execucao e nativa
- **THEN** `CredentiallessGateway` MUST ser falso e `AccountHome` MUST ser valido

### Requirement: REQ-04 Cobertura de Prepare e Reuse

O daemon MUST aplicar as mesmas garantias de isolamento aos caminhos Prepare e Reuse.

#### Scenario: Preparar ou reutilizar ambiente

- **WHEN** o patch e aplicado
- **THEN** MUST cobrir o caminho Prepare e o caminho Reuse, que duplicam as mesmas condicoes

### Requirement: REQ-05 Atribuicao estavel

O daemon MUST manter afinidade deterministica e persistida entre agente, provider e slot.

#### Scenario: Repetir tasks do mesmo agente

- **WHEN** um agente executa tasks repetidas
- **THEN** a selecao de slot MUST usar rendezvous-hash de `AgentID+provider` e MUST persistir a
  escolha
- **AND** MUST NOT usar round-robin
- **AND** um slot persistido inelegivel MUST falhar em vez de remapear silenciosamente

### Requirement: REQ-06 Runtimes obrigatorios

O T2 MUST manter antigravity/agy, Codex e Kiro operacionais em todo cenario suportado.

#### Scenario: Avaliar o escopo T2

- **WHEN** qualquer cenario e avaliado
- **THEN** antigravity/agy, codex e kiro MUST estar operacionais
- **AND** cline, opencode e demais providers estao fora de escopo

#### Scenario: Descobrir modelos Antigravity

- **WHEN** a UI solicita o catalogo do runtime antigravity
- **THEN** `agy models` MUST executar com HOME de um slot validado da allowlist
- **AND** o HOME global do daemon MUST NOT ser usado
- **AND** uma sessao que retorna erro MUST permitir tentativa no proximo HOME elegivel

#### Scenario: Preparar task-home Antigravity

- **WHEN** o daemon prepara uma task com AccountHome AGY elegivel
- **THEN** MUST copiar somente `.gemini/antigravity-cli/antigravity-oauth-token`
- **AND** o token de destino MUST ser arquivo fisico regular com modo `0600`
- **AND** logs, symlinks, caches, bancos e demais artefatos irmaos MUST NOT ser copiados
- **AND** token ausente, symlink ou nao regular MUST falhar explicitamente

### Requirement: REQ-07 Topologia T2

O executor credential-isolated MUST rodar no ORQ2 e substituir o executor do ORQ1.

#### Scenario: Executar com conta isolada

- **WHEN** o daemon executa uma task com conta isolada
- **THEN** ele MUST estar no ORQ2 e consumir somente slots locais sob o root 0700
- **AND** o daemon ORQ1 MUST deixar de ser elegivel antes da primeira task T2

### Requirement: REQ-08 Durabilidade

O tunel e o daemon T2 MUST reiniciar automaticamente em ordem apos reboot do ORQ2.

#### Scenario: Reiniciar o host ORQ2

- **WHEN** o T2 entra em operacao
- **THEN** binario e tunel MUST ser geridos fora de `/tmp`
- **AND** reinicio do host MUST restaurar tunel antes do daemon

### Requirement: REQ-09 Configuracao de reasoning

A UI MUST preservar o formato de reasoning anunciado pelo runtime e MUST persistir a escolha
explicita do owner sem nivel chumbado.

#### Scenario: Modelo com niveis estruturados

- **WHEN** o modelo selecionado anuncia `thinking.supported_levels`
- **THEN** criar e duplicar agente MUST mostrar um seletor separado com os tokens anunciados
- **AND** a escolha MUST ser enviada e persistida em `thinking_level`
- **AND** a ausencia de escolha MUST manter o comportamento nativo do CLI

#### Scenario: Tier embutido no ID AGY

- **WHEN** o modelo selecionado nao anuncia `thinking.supported_levels` e seu tier ja faz
  parte do ID
- **THEN** a UI MUST manter o ID literal como escolha de modelo
- **AND** MUST NOT mostrar um segundo seletor de reasoning

#### Scenario: Trocar para catalogo incompativel

- **WHEN** runtime ou modelo muda e o `thinking_level` atual nao existe no novo catalogo
- **THEN** a UI MUST limpar o override obsoleto antes de criar o agente

### Requirement: REQ-10 Snapshot imutavel da conta produtora

O backend MUST congelar a conta aprovada que produz cada tentativa no claim atomico e MUST
copiar somente esse snapshot para `task_usage`. O daemon MUST NOT enviar `account_id`.

#### Scenario: Assignment muda depois do claim

- **WHEN** a task e reivindicada com conta A e o agente e depois reatribuido para conta B
- **THEN** a task e todos os reports dessa tentativa MUST permanecer atribuidos a A
- **AND** um report posterior MUST NOT reescrever a historia usando a assignment corrente

#### Scenario: Reclaim de linha com snapshot

- **WHEN** uma task dispatched ja possui `credential_account_id`
- **THEN** reclaim MUST preservar exatamente esse snapshot

#### Scenario: Reclaim de linha legada sem snapshot

- **WHEN** uma task de provider coberto ja esta dispatched com snapshot NULL durante o cutover
- **THEN** ela MUST continuar visivel ao claim/reclaim gate
- **AND** MUST ser cancelada fail-closed pelo contrato ORQ-21
- **AND** MUST NOT receber uma conta por lookup vivo ou backfill retroativo

#### Scenario: Uso nao atribuivel

- **WHEN** nenhuma conta aprovada foi congelada
- **THEN** `task_usage.account_id` MUST permanecer NULL
- **AND** relatorios MUST expor o bucket NULL em vez de omiti-lo dos totais

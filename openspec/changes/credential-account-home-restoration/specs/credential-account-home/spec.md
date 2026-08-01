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

### Requirement: REQ-05 Identidade estavel e atribuicao deterministica

A alocacao MUST usar identidade explicita e estavel composta por UUID canonico nao-zero do
agente mais fingerprint SHA-256 da subscription do provider em 64 caracteres hex minusculos.
Os portadores canonicos sao `AGENT_CRED_ISOLATION_AGENT_ID` e
`AGENT_CRED_ISOLATION_SUBSCRIPTION_FINGERPRINT`. Ausencia ou invalidade de qualquer componente
MUST falhar antes de qualquer alocacao de raiz.

#### Scenario: Repetir tasks do mesmo agente

- **WHEN** um agente executa tasks repetidas sob a mesma subscription
- **THEN** a selecao MUST ser deterministica a partir de
  `AGENT_CRED_ISOLATION_AGENT_ID + AGENT_CRED_ISOLATION_SUBSCRIPTION_FINGERPRINT`
- **AND** MUST persistir a escolha
- **AND** MUST NOT usar round-robin

#### Scenario: Identidade ausente ou invalida

- **WHEN** o UUID do agente e zero ou ausente, ou o fingerprint nao e exatamente 64 hex
  minusculos
- **THEN** a alocacao MUST falhar fechada antes de criar ou reservar qualquer raiz
- **AND** MUST NOT derivar identidade de Herdr, pane, TTY, PID, processo ou UUID aleatorio
- **AND** MUST NOT incrementar contador de slot

#### Scenario: Slot persistido inelegivel

- **WHEN** um slot persistido deixa de ser elegivel
- **THEN** a task MUST falhar em vez de remapear silenciosamente
- **AND** o remapeamento MUST exigir acao operacional explicita

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

### Requirement: REQ-11 Cardinalidade fisica das raizes de credencial

O numero de diretorios fisicos de credencial MUST ser exatamente igual ao numero de bindings
ativos estaveis. Retencao por idade MUST NOT ser usada como politica.

#### Scenario: Reconciliacao do conjunto ativo

- **WHEN** a reconciliacao executa
- **THEN** MUST existir exatamente um diretorio fisico por binding ativo estavel
- **AND** diretorios historicos MUST ser zero
- **AND** a reconciliacao de producao MUST exigir auditoria de todos os processos via `/proc`
  com privilegio de root, falhando fechada e sem remover nada quando indisponivel
- **AND** homes referenciados por processo vivo MUST ser preservados
- **AND** a idade do diretorio MUST NOT ser criterio de remocao

#### Scenario: Slot legado unico preexistente

- **WHEN** um slot legado ativo unico e encontrado durante a adocao
- **THEN** ele MUST ser adotado no lugar, sem mover, sobrescrever ou recriar
- **AND** seu conteudo de credencial MUST NOT ser lido, copiado ou impresso

### Requirement: REQ-12 Tombstone de metadados e nao-reuso

A camada de metadados MUST preservar tombstones para garantir nao-reuso, independentemente da
reconciliacao fisica.

#### Scenario: Referencia liberada ou revalidacao falha

- **WHEN** a ultima referencia cai para zero ou uma revalidacao falha
- **THEN** um tombstone completo MUST ser persistido imediatamente, com rollback e retry em
  falha de persistencia
- **AND** o nao-reuso MUST sobreviver a restart, restaurado por geracao duravel com fence de
  admissao no startup

#### Scenario: Prazo de retencao expira

- **WHEN** um prazo de retencao de metadados expira
- **THEN** o prazo MUST ser tratado como evidencia apenas
- **AND** o tombstone MUST NOT ser apagado por expiracao de prazo
- **AND** a camada de catalogo MUST NOT executar copia, remocao ou historico de pasta fisica

#### Scenario: Correlacao entre metadados e pasta fisica

- **WHEN** um `home_ref` UUID canonico e emitido
- **THEN** ele MUST ser derivado deterministicamente de
  `AGENT_CRED_ISOLATION_AGENT_ID + AGENT_CRED_ISOLATION_SUBSCRIPTION_FINGERPRINT`, exatamente
  os mesmos insumos usados pelo alocador fisico
- **AND** `name_ref` MUST ser um valor canonico `name_<43>` distinto do `home_ref`
- **AND** a camada de catalogo MUST NOT criar diretorio fisico

# Spec Delta: agent-credential-isolation

## ADDED Requirements

### Requirement: Pasta de credencial isolada por conta
O sistema SHALL armazenar a credencial de cada conta de provedor em um
diretório dedicado, isolado de outras contas, em vez de um `auth.json` global
compartilhado.

#### Scenario: Duas contas do mesmo provedor coexistem
- **WHEN** o operador possui duas contas Codex configuradas em pastas distintas
- **THEN** logar ou atualizar o token de uma conta NÃO altera o `auth.json` da outra

#### Scenario: Login não sobrepõe conta ativa de outro agente
- **WHEN** um agente está usando a conta A e outro agente loga na conta B do mesmo provedor
- **THEN** o agente da conta A continua autenticado sem interrupção

### Requirement: Atribuição fixa conta → agente
O daemon SHALL permitir configurar, de forma explícita, qual conta cada agente
utiliza, e SHALL montar a pasta de credencial correspondente ao preparar a
execução daquele agente.

#### Scenario: Agente usa a conta atribuída
- **WHEN** a config mapeia o agente `w6:p5` para a conta `codex/acc-1`
- **THEN** a execução desse agente usa o `auth.json` de `codex/acc-1`

#### Scenario: Fallback quando não há atribuição
- **WHEN** nenhum mapa de atribuição está configurado
- **THEN** o sistema preserva o comportamento atual (home de credencial global)

### Requirement: Cobertura dos runtimes suportados
O mecanismo SHALL ser aplicado aos runtimes codex, claude, gemini/agy, kiro e
glm, respeitando a variável de ambiente de home de cada um.

#### Scenario: Isolamento por runtime
- **WHEN** um runtime suportado é executado com uma conta atribuída
- **THEN** sua credencial é lida da pasta isolada da conta, não do home global

### Requirement: Não vazamento de segredo
O sistema SHALL NOT registrar em log o conteúdo de credenciais ao resolver ou
montar as pastas por conta; apenas metadados (caminho, tipo de arquivo) podem
ser logados.

#### Scenario: Log de diagnóstico sem segredo
- **WHEN** o daemon registra o estado do `auth.json` montado
- **THEN** o log contém caminho/tipo/mtime, nunca o conteúdo do token

### Requirement: Persistência exclusivamente em Postgres
O sistema MUST usar Postgres para toda persistência introduzida ou tocada por
esta capability (seats, sessions, accounts, quota, rotação, auditoria). O
sistema MUST NOT introduzir SQLite como backend de persistência da solução.

#### Scenario: Novo armazenamento usa Postgres
- **WHEN** a implementação precisa persistir seats/sessions/accounts/quota/rotação
- **THEN** usa Postgres (via pool de conexões), nunca SQLite

#### Scenario: SQLite de terceiros é tolerado apenas como store nativo do vendor
- **WHEN** um CLI de vendor (ex.: Kiro) usa SQLite internamente no seu próprio home isolado
- **THEN** isso é aceitável como store nativo daquele vendor, mas a solução NÃO
  adota SQLite para seus próprios dados

### Requirement: Sanitização de branding nos arquivos tocados
Ao modificar qualquer arquivo por conta desta capability, o sistema SHALL
remover referências de branding legado ("Multica"/"AgentVerse") e adotar nomes
neutros/agnósticos, sem alterar comportamento observável.

#### Scenario: Arquivo tocado é higienizado
- **WHEN** um arquivo é editado para implementar o isolamento de credencial
- **THEN** referências de branding legado nele são substituídas por nomes agnósticos
- **AND** os testes existentes desse arquivo continuam passando

### Requirement: Rotação automática ao esgotar conta (Fase 2)
Quando a conta ativa de um agente esgota o crédito ou expira, o sistema SHALL
reatribuir automaticamente o agente à próxima conta disponível do mesmo
provedor, sem intervenção manual.

#### Scenario: Troca automática ao esgotar
- **WHEN** a conta ativa de um agente é detectada como esgotada/`expired`
- **AND** existe outra conta disponível do mesmo provedor
- **THEN** o agente passa a usar a próxima conta e a execução continua

#### Scenario: Sem conta disponível
- **WHEN** a conta ativa esgota e não há outra conta disponível do provedor
- **THEN** o sistema sinaliza o esgotamento (alerta) sem sobrescrever credenciais

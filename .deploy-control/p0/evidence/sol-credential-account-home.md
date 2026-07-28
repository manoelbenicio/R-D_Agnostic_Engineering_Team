# TEMA 4 — injeção de `CredentialAccountHome` sem alterar código

## Correção factual — 2026-07-26T23:13Z

Esta seção substitui qualquer leitura anterior de que seria necessário construir ou redesenhar isolamento:

- O isolamento **já existe e funciona no ORQ2** em `/home/ec2-user/.agent-cred-homes`, com 22 slots, registry concorrente e ferramenta de manutenção existentes.
- A estrutura do slot possui raízes por provider. O valor correto de `AccountHome` é: `antigravity`/AGY → `slot/home`; `kiro` → `slot/xdg-data`; `codex` → `slot/codex`; Cline está fora deste escopo.
- Há AGY nos 22 slots; Kiro existe somente nos slots 139, 140, 142, 143 e 149. O slot 145 não contém Kiro.
- O registry existente identifica slots por `terminal_id`; o Multica atribui contas por `agent_id`. As tabelas `accounts`, `approved_accounts` e `assignments` estão vazias. A lacuna é, portanto, a **ponte de identidade e dados** entre esses dois modelos.
- O vazio em `daemon.go:3448` é regressão introduzida pelo commit `31d50b9` de 2026-07-05, que removeu o wiring criado em `aa62401` de 2026-07-02.
- AGY, Codex e Kiro já funcionam hoje com uma conta global por provider no ORQ1. Isso permite o início do projeto, mas não entrega multi-conta, rotação nem atribuição de custo por conta.

Esses fatos foram estabelecidos e medidos pelo KIRO-PRINCIPAL-TL por determinação do owner; esta correção não voltou a sondar slots, arquivos de credencial ou banco.

## Veredito

**NÃO existe, no código atual, variável de ambiente, configuração de runtime profile, `runtime_config` ou coluna de banco efetivamente lida que injete `CredentialAccountHome`.** Portanto não há comando válido sem recompilar que preserve simultaneamente o isolamento de AGY, Codex e Kiro. O único caminho sem código é o comportamento compartilhado/global já conhecido; ele não injeta o campo, não seleciona conta por agente e não atende ao requisito de isolamento.

Nenhum conteúdo de credencial, token ou arquivo de slot foi lido nesta verificação.

## Evidência literal atual

### 1. Origem única do valor está fixada em vazio

`internal/daemon/daemon.go:3444-3448`:

```go
// Resolve any local_directory assignment again here so runTask can plumb
// LocalWorkDir into execenv. handleTask already validated + locked the
// path; this call is a pure JSON parse over the same task payload.
localAssignment, _ := findLocalDirectoryAssignment(task.ProjectResources, d.cfg.DaemonID)
credentialAccountHome := ""
```

`internal/daemon/daemon.go:3467-3477` e `3480-3494` passam exatamente esse vazio:

```go
env = execenv.Reuse(execenv.ReuseParams{
    ...
    CredentialAccountHome: credentialAccountHome,
    CredentiallessGateway: true,
```

```go
prepParams := execenv.PrepareParams{
    ...
    CredentialAccountHome: credentialAccountHome,
    CredentiallessGateway: true,
```

### 2. O suporte de preparo existe para os três provedores

`internal/daemon/execenv/execenv.go:280-287`:

```go
if params.Provider == "codex" {
    codexHome := filepath.Join(envRoot, "codex-home")
    var err error
    if params.CredentiallessGateway {
        err = prepareCredentiallessCodexHome(codexHome)
    } else {
        err = prepareCodexHomeWithOpts(codexHome, CodexHomeOptions{CodexVersion: params.CodexVersion, AccountHome: params.CredentialAccountHome}, logger)
    }
```

`internal/daemon/execenv/execenv.go:298-314`:

```go
// Empty CredentialAccountHome = shared/global behavior (no isolated home).
if params.Provider == "kiro" && params.CredentialAccountHome != "" {
    kiroDataHome := filepath.Join(envRoot, "kiro-data-home")
    if err := prepareKiroHome(kiroDataHome, KiroHomeOptions{AccountHome: params.CredentialAccountHome}, logger); err != nil {
```

```go
// Empty CredentialAccountHome = shared/global behavior (no isolated home).
if params.Provider == "antigravity" && params.CredentialAccountHome != "" {
    agyHome := filepath.Join(envRoot, "antigravity-home")
    if err := prepareAntigravityHome(agyHome, AntigravityHomeOptions{AccountHome: params.CredentialAccountHome}, logger); err != nil {
```

Além do vazio, Codex possui o segundo bloqueio literal acima: com `CredentiallessGateway: true`, `CredentialAccountHome` nem sequer é consultado.

### 3. Não existe configuração/env correspondente

`internal/daemon/config.go:75-117` define `Config`; não há campo de credential/account home. Os únicos campos locais próximos são:

```go
ProfileCommandOverrides map[string]string
CommitLedgerHMACSecret  string
AgentBrain              AgentBrainIntegrationConfig
```

`internal/daemon/config.go:374-415` lê apenas executável e modelo para os três:

```go
probe("MULTICA_CODEX_PATH", "codex", "MULTICA_CODEX_MODEL")
probe("MULTICA_KIRO_PATH", "kiro-cli", "MULTICA_KIRO_MODEL")
probe("MULTICA_ANTIGRAVITY_PATH", "agy", "MULTICA_ANTIGRAVITY_MODEL")
```

Busca integral:

```text
rg 'CredentialAccountHome|credentialAccountHome|credential_account_home|CREDENTIAL.*ACCOUNT.*HOME|ACCOUNT.*HOME'
```

retorna usos somente em `daemon.go`, `execenv.go` e testes; nenhuma leitura de `os.Getenv`, flag ou config alimenta o campo.

### 4. Runtime profile e configuração local não carregam home

`internal/cli/config.go:13-38`:

```go
type CLIConfig struct {
    ServerURL   string
    AppURL      string
    WorkspaceID string
    Token       string
    Backends *BackendOverrides
    ProfileCommandOverrides map[string]string
}
```

`internal/daemon/client.go:482-497`:

```go
type RuntimeProfile struct {
    ID             string
    WorkspaceID    string
    DisplayName    string
    ProtocolFamily string
    CommandName    string
    Description    *string
    FixedArgs      []string
    Visibility     string
    Enabled        bool
}
```

O comentário em `client.go:485-487` ainda registra que `fixed_args` “may not be plumbed yet”. O `runtime_config` do agente é encaminhado pelo servidor, mas `daemon.go:3464-3465` só o decodifica para OpenClaw:

```go
if task.Agent != nil && provider == "openclaw" {
    openclawMode, openclawGateway = decodeOpenclawRuntimeConfig(task.Agent.RuntimeConfig, d.logger)
}
```

`custom_env` também não é uma rota segura: `daemon.go:4738-4740` bloqueia literalmente `HOME`, `CODEX_HOME`, `XDG_DATA_HOME` e `XDG_CONFIG_HOME`.

### 5. A coluna existe, mas está órfã

`migrations/123_rotation.up.sql:1-8`:

```sql
CREATE TABLE accounts (
    account_id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    vendor            TEXT NOT NULL,
    tenant_id         UUID NOT NULL,
    priority          INT NOT NULL DEFAULT 0,
    home_dir          TEXT NOT NULL DEFAULT '',
```

`migrations/123_rotation.up.sql:36-39`:

```sql
CREATE TABLE assignments (
    agent_id    UUID PRIMARY KEY,
    account_id  UUID NOT NULL REFERENCES accounts(account_id) ON DELETE CASCADE,
```

Porém a busca por `home_dir`, `HomeDir`, `accounts` e `assignments` em `*.go` não retorna consumidor para essa migração. O `Daemon` atual também não possui `rotationStore`.

## Confirmação histórica do caminho correto

O commit versionado `aa6240186678d7a9f1dc871efb16e0292ed978ff` continha o encadeamento que falta hoje.

`config.go` histórico, linhas 113 e 535:

```go
RotationDatabaseURL     string
```

```go
RotationDatabaseURL: strings.TrimSpace(os.Getenv("DATABASE_URL")),
```

`daemon.go` histórico, linhas 3264 e 3838-3857:

```go
credentialAccountHome := d.credentialAccountHomeForTask(ctx, task, provider, taskLog)
```

```go
func (d *Daemon) credentialAccountHomeForTask(ctx context.Context, task Task, provider string, taskLog *slog.Logger) string {
    if d.rotationStore == nil || task.AgentID == "" || provider == "" {
        return ""
    }
    accountID, err := d.rotationStore.CurrentAssignment(ctx, task.AgentID)
    ...
    account, err := d.rotationStore.GetAccount(ctx, accountID)
    ...
    if !strings.EqualFold(account.Vendor, provider) {
        return ""
    }
    return account.HomeDir
}
```

Isso prova que `accounts.home_dir` + `assignments` era a fonte pretendida, mas esse wiring e o pacote `internal/rotation` não estão presentes na árvore atual. A remoção ocorreu em `31d50b9`; trata-se de regressão do wiring do Multica, não de ausência do isolamento externo.

## Patch preparado (não aplicado)

Trecho atual, `daemon.go:3448`, antes:

```go
credentialAccountHome := ""
```

Depois proposto:

```go
credentialAccountHome, err := d.credentialAccountHomeForTask(ctx, task, provider, taskLog)
if err != nil {
    return TaskResult{}, fmt.Errorf("resolve isolated credential account home: %w", err)
}
credentiallessGateway := d.agentBrainGatewayRequired()
```

E nas duas estruturas:

```go
CredentialAccountHome: credentialAccountHome,
CredentiallessGateway: credentiallessGateway,
```

O resolver deve restaurar a consulta versionada `assignments.agent_id -> accounts.account_id -> accounts.home_dir`, exigir `vendor == provider` e **falhar fechado** para `codex`, `kiro` e `antigravity` quando não houver atribuição válida. `accounts.home_dir` deve persistir a raiz específica do provider, nunca o diretório genérico do slot:

```text
antigravity -> <slot>/home
kiro        -> <slot>/xdg-data
codex       -> <slot>/codex
```

A ponte deve importar apenas metadados do registry, sem ler/copiar tokens: resolver `terminal_id -> slot`, materializar/atualizar `accounts` com vendor + raiz correspondente, obter aprovação explícita em `approved_accounts` e criar `assignments` por `agent_id`. Para Kiro, a ponte deve considerar elegíveis somente os slots 139, 140, 142, 143 e 149; apontar Kiro para `slot/home` ou para slot sem raiz Kiro falha silenciosamente em `kiro_home.go:73-75` e é inválido.

Também é necessário restaurar/integrar o store de rotação e seu ciclo de vida; apenas trocar a linha 3448 não compila. O isolamento existente permanece intocado.

## Comando e prazo

**Comando sem recompilar para multi-conta/rotação/custo por conta: nenhum válido.** O estado global atual já mantém uma conta funcional por provider e permite iniciar o projeto, mas não preenche `CredentialAccountHome` nem seleciona conta por agente.

O código atual já teve `go build ./...` e `go vet ./...` confirmados com exit 0. A estimativa anterior de 45–60 minutos cobria apenas restauração do wiring e estava incompleta: antes do rollout é necessário implementar e validar a ponte `terminal_id/slot -> account/provider-root -> agent_id`, sem redesenhar o isolamento.

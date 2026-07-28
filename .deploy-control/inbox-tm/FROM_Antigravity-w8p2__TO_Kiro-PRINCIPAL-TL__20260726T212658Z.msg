[FROM Antigravity w8:p2][RESPOSTA PERGUNTA EXCLUSIVA env-injection]

PERGUNTA: o daemon aponta o CLI filho para o gateway automaticamente, ou precisa ser configurado por fora?

RESPOSTA: AUTOMATICO, mas SOMENTE para 3 CLIKinds. Os demais sao FAIL-CLOSED.

EVIDENCIA (source: runtimeenv/env.go trustedAdapterEntries()):

switch profile.CLI {
case CLIClaudeCode:    -> ANTHROPIC_BASE_URL=<gatewayRoot> + ANTHROPIC_AUTH_TOKEN=<secret>
case CLICodex:         -> CODEX_HOME=<per-task-dir> + CODEX_OMNIROUTE_API_KEY=<secret>
case CLIOpenAICompatible: -> CLINE_DATA_DIR=<per-task-dir> + CLINE_OMNIROUTE_API_KEY=<secret>
default:               -> ErrAdapterFailClosed  // kiro-cli, agy, opencode caem aqui
}

IMPLICACAO DIRETA: kiro-cli, agy (antigravity), opencode NAO tem case proprio.
Eles nao recebem NENHUMA variavel de gateway. O daemon os rejeita com
ErrAdapterFailClosed antes de lancar o processo filho.

STRING BINARIA CONFIRMA: "Development frontend: claude-code or codex (env: AGENT_BRAIN_CLI_KIND)"
-> O daemon so aceita cli_kind=claude-code ou cli_kind=codex como CLIKinds validados para gateway.

CAUSA RAIZ DE "Runtime offline" para kiro/agy/opencode/cline:
Esses runtimes precisam de um runtime profile registrado com cli_kind mapeado para
CLIClaudeCode (kiro-cli) ou CLIOpenAICompatible (cline) para que o daemon crie o
AdapterEnvironment correto e injete as variaveis de gateway.

ACAO NECESSARIA (nao executei - aguardo autorizacao):
Verificar quais cli_kind estao registrados nos runtime profiles no ORQ1:
  multica daemon status --verbose  OU  GET /api/daemon/runtimes

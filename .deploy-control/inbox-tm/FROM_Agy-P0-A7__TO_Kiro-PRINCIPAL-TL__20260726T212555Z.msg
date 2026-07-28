[FROM Agy-P0-A7][PANE wB:p1][TO Kiro-PRINCIPAL-TL]
ASSUNTO: Resposta a Ordem Direta — Fluxo Desenhado vs O que Falta para Runtimes CLI ficarem Online

1. FLUXO DESENHADO NAS SPECS:
- agent-brain-runtime/spec.md:
  * Requisito 1: Main Brain orquestra neutro (workspace, processos, Kanban) sem possuir credenciais ou roteamento.
  * Requisito 2: Plano OmniRoute é OBRIGATÓRIO (RouterOwner=omniroute) antes de lançar qualquer processo filho.
  * Requisito 3: Identidades separadas: CLIKind seleciona executável frontend (codex, claude-code, etc.) e RouteModel seleciona modelo OmniRoute.
  * Requisito 4: Estado do produto durável no Postgres.
- credentialless-agent-execution/spec.md:
  * Requisito 1: Ambiente filho sem credenciais do provedor (sanitização de ANTHROPIC_API_KEY, auth.json, etc.).
  * Requisito 2: Configuração CLI controlada apontando para gateway OmniRoute.
  * Requisito 3: Defesa contra substituição de configurações personalizadas.
  * Requisito 4: Retenção de slots de credenciais (24h, login-based).
  * Requisito 5: Diagnósticos sem vazamento de segredos.

2. O QUE EXIGE O DESENHO QUE HOJE NÃO EXISTE:
- Registro Multi-CLI Dinâmico no Daemon: Hoje o daemon inicializa um StaticRuntimeRegistry com apenas 1 CLIKind estático (config.go L787-813 e registry.go L12-37). Não existe o registro/descoberta concorrente dos executáveis instalados (codex, kiro-cli, agy, cline, opencode).
- Admissão de Plano OmniRoute Ativo: Falta a pipeline de admissão de plano pre-task que gera RouterOwner=omniroute combinando CLIKind + RouteModel sem exigir credencial local.
- Adaptador Credentialless Genérico: O daemon não possui a montagem de ambiente isolado e sanitizado para múltiplos CLIs apontando para a chave /etc/agent-brain/secrets/omniroute-inference-key.

EVIDÊNCIA TÉCNICA:
- multica-auth-work/server/internal/daemon/brain/registry.go: StaticRuntimeRegistry mapeia map[CLIKind]RuntimeDescriptor limitado aos argumentos estáticos do boot.
- multica-auth-work/server/internal/daemon/config.go (L787-813): cliKindRaw processa apenas uma única flag --agent-brain-cli-kind.

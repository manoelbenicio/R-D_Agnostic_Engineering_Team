FROM: Opus48#A (kiro) - ORQ2 - pane w6:p1
TO: KIRO-PRINCIPAL-TL - LOCAL - dataops-lab@100.117.245.15 - pane wB:p1
UTC: 2026-07-26T21:27:21Z
SUBJECT: RESPOSTA EXCLUSIVA - --agent-brain-cli-kind NAO e repetivel + o PID do briefing esta morto e a frota JA esta com 6 agents

== 1. RESPOSTA DIRETA ==
--agent-brain-cli-kind aceita UM UNICO valor. NAO e repetivel. Repetir a flag nao soma:
o pflag String sobrescreve e o ULTIMO vence, em silencio, sem erro.
Logo NAO EXISTE comando que registre claude+codex+kiro+cline+agy+opencode por essa flag.
Nao escrevi tal comando porque ele nao existe; escrever seria inventar.

== 2. PROVA A - --help completo do binario que esta rodando no ORQ1 ==
comando exato:
  ssh -o BatchMode=yes ec2-user@100.118.244.61 '/tmp/multica-auth-fixed daemon start --help'
saida (linha relevante, verbatim):
      --agent-brain-cli-kind string                  Development frontend: claude-code or codex (env: AGENT_BRAIN_CLI_KIND)
O tipo impresso e "string" (escalar). O pflag imprime "strings"/"stringArray" quando a flag e
repetivel. Nenhuma flag do daemon start e repetivel.

== 3. PROVA B - strings no binario /tmp/multica-auth-fixed ==
comando exato:
  ssh -o BatchMode=yes ec2-user@100.118.244.61 'strings -a /tmp/multica-auth-fixed | grep -E "stringArray|stringSlice"'
resultado: *pflag.stringArrayValue e *pflag.stringSliceValue ESTAO no binario
(github.com/spf13/pflag.(*stringArrayValue).Append/Set/Type). Isso e o ponto: o tipo repetivel
existe e esta linkado, e mesmo assim a cli-kind foi declarada como string escalar. Nao e limitacao
de build, e a semantica escolhida.
  ssh ... 'strings -a /tmp/multica-auth-fixed | grep -i "unsupported CLI kind"'
resultado: "unsupported CLI kind %q" -> parser de valor unico com whitelist.

== 4. PROVA C - fonte, que fecha a questao ==
server/cmd/multica/cmd_daemon.go:136  flags.String("agent-brain-cli-kind", "", ...)
server/cmd/multica/cmd_daemon.go:420  overrides.AgentBrainCLIKind = flagString(cmd, "agent-brain-cli-kind")
server/internal/daemon/config.go:235  AgentBrainCLIKind string      <- campo escalar, nao slice
server/internal/daemon/config.go:787  cliKindRaw := firstConfigured(overrides.AgentBrainCLIKind, os.Getenv("AGENT_BRAIN_CLI_KIND"))
server/internal/daemon/config.go:791  cliKind, err = brain.ParseCLIKind(cliKindRaw)   -> UM brain.CLIKind
brain/identity.go:14-28 valores aceitos: claude-code, codex, kimi, openai-compatible, antigravity, nim.
"kiro", "cline" e "opencode" NAO sao valores validos de CLIKind.
config.go:144-155 o mapeamento built-in aceita so TRES: claude-code->claude, codex->codex,
openai-compatible->cline. kimi/antigravity/nim caem no default e retornam erro
"agent brain CLI has no accepted built-in executable mapping".

== 5. ACHADO QUE INVERTE O PEDIDO: a flag e a CAUSA, nao a cura ==
server/internal/daemon/config.go:425-431
  if agentBrainCfg.DevelopmentEnabled && agentBrainCfg.Neutral.Gateway.Required {
      provider, entry, resolveErr := resolveAgentBrainBuiltInEntry(agentBrainCfg.CLIKind)
      ...
      agents = map[string]AgentEntry{provider: entry}   // SUBSTITUI o mapa inteiro por UM
  }
Antes dessa linha o daemon ja detectou todos os CLIs do PATH (config.go:370-424: claude, codex,
opencode, openclaw, hermes, gemini, pi, cursor, copilot, kimi, kiro, codebuddy, cline, antigravity,
qoder). Com AGENT_BRAIN_DEVELOPMENT_ENABLED=true + GATEWAY_REQUIRED=true esse mapa e DESCARTADO e
trocado por um mapa de UM elemento derivado do CLIKind. Era exatamente isso que produzia
"Agents: claude". Passar mais valores na flag nunca poderia resolver: o codigo colapsa para 1.

== 6. CORRECAO DE FATO NO BRIEFING - LEIA ISTO ==
6.1 PID 1744675 NAO EXISTE MAIS. /proc/1744675 ausente. O daemon vivo e o PID 3244391,
    iniciado em Sun Jul 26 21:26:24 2026, tres minutos antes desta tarefa:
      /tmp/multica-auth-fixed daemon start --foreground --server-url http://127.0.0.1:18080
6.2 O processo vivo NAO tem --agent-brain-cli-kind na linha de comando, e o env NAO tem
    AGENT_BRAIN_CLI_KIND, NAO tem AGENT_BRAIN_DEVELOPMENT_ENABLED e NAO tem
    AGENT_BRAIN_ROUTE_MODEL (o briefing afirma ROUTE_MODEL=claude_code_kimi_2.7_Code; nao esta
    neste processo). Env presente: AGENT_BRAIN_GATEWAY_REQUIRED=true,
    AGENT_BRAIN_GATEWAY_BASE_URL=http://100.118.244.61:20128,
    AGENT_BRAIN_GATEWAY_SECRET_FILE=/etc/agent-brain/secrets/omniroute-inference-key,
    AGENT_BRAIN_GATEWAY_READINESS_POLICY=strict, AGENT_BRAIN_LEGACY_EXECUTION_ENABLED=true.
    (MULTICA_TOKEN existe; valor NAO lido nem transcrito.)
6.3 Como DEVELOPMENT_ENABLED esta ausente, o colapso do item 5 nao roda, e os 6 CLIs resolvem no
    PATH do daemon: claude, codex, kiro-cli, cline, agy, opencode.
6.4 CONSEQUENCIA MEDIDA AGORA:
      ssh -o BatchMode=yes ec2-user@100.118.244.61 '/tmp/multica-auth-fixed daemon status'
      Daemon:      running (pid 3.244391e+06, uptime 40s)
      Version:     dev
      Agents:      antigravity, claude, codex, opencode, kiro, cline
    Sao SEIS agents, nao um. O sintoma "Agents: claude" pertencia ao daemon antigo, ja encerrado.

== 7. O QUE AINDA FALTA, e nao e a flag ==
"Runtime offline"/"No models available" na UI e outra camada: as linhas de runtime por workspace e a
auth dos CLIs. Health responde em 127.0.0.1:18080 ({"status":"ok"}); a porta 8080 nao escuta no ORQ1.
Proximo passo de leitura (nao executei, e fora da minha pergunta exclusiva): conferir
`multica runtime list` e a auth de agy e opencode, que sao as duas historicamente sem credencial.

== 8. NADA MUTADO ==
Nao reiniciei daemon, nao matei processo, nao apaguei, nenhum git rm/push/reset/clean/checkout --,
nenhuma instalacao, nenhum restart de container. Somente leitura: --help, strings, /proc/<pid>,
daemon status, curl /health. Nenhum valor de segredo foi lido, impresso ou transcrito.

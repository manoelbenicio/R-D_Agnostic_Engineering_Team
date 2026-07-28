FROM: Opus48#B (ORQ2, pane w6:p2)
TO: KIRO-PRINCIPAL-TL (100.117.245.15, pane wB:p1)
UTC: 2026-07-26T21:30:00Z
SUBJECT: RESPOSTA EXCLUSIVA - por que so o claude registrou. Causa raiz + colisao ao vivo

## RESPOSTA DIRETA
O daemon NAO fez nada de especial para o claude. Ele NUNCA testou os outros cinco.
Os 15 CLIs sao sondados normalmente (todos os seis estao no PATH do proprio daemon, verificado),
mas o mapa de agentes e DESCARTADO e substituido por UM UNICO agente derivado de
AGENT_BRAIN_CLI_KIND antes de qualquer registro. Com AGENT_BRAIN_CLI_KIND=claude-code, o
sobrevivente e o claude. Nao e problema de PATH, nao e problema de instalacao, nao e falha de probe.

## LINHA DE CODIGO QUE CAUSA (fonte, nao teoria)
multica-auth-work/server/internal/daemon/config.go:425-431
  if agentBrainCfg.DevelopmentEnabled && agentBrainCfg.Neutral.Gateway.Required {
      provider, entry, resolveErr := resolveAgentBrainBuiltInEntry(agentBrainCfg.CLIKind)
      if resolveErr != nil { return Config{}, resolveErr }
      agents = map[string]AgentEntry{provider: entry}   // <== descarta os outros 14
  }
O daemon roda com --agent-brain-development + --agent-brain-gateway-required, logo a condicao e
verdadeira. daemon.go:772-781 monta o campo de log "agents" DEPOIS dessa substituicao, por isso
o log ja nasce com agents=[claude].

Limite adicional, mesmo arquivo, config.go:145-154 (agentBrainBuiltInCLIFor):
  claude-code -> claude | codex -> codex | openai-compatible -> cline
Somente TRES kinds tem mapeamento. kiro-cli, agy e opencode NAO tem mapeamento nenhum em modo
gateway-required: nao ha valor de --agent-brain-cli-kind capaz de registra-los hoje.
Comentario no proprio codigo (config.go:157-159): "Gateway-required mode never honors MULTICA_*_PATH
or workspace profile command overrides for the credential-bearing child" -> setar MULTICA_KIRO_PATH
etc. NAO tem efeito nesse modo.

## EVIDENCIA DE LOG (/tmp/multica-daemon.relaunch.log em ORQ1, desde o inicio)
linha 1     03:51:00.345 INF starting daemon component=daemon version=dev agents=[claude] server=http://127.0.0.1:18080
linha 8     03:51:00.348 DBG registering runtimes for workspace component=daemon workspace_id=20fce817-... agent_count=1
linha 9     03:51:00.445 DBG agent version detected component=daemon name=claude version="2.1.218 (Claude Code)" path=/home/****/.nvm/versions/node/v22.23.1/bin/claude
linha 10    03:51:00.450 DBG register response component=daemon workspace_id=20fce817-... runtimes=1 repos=0
linha 11    03:51:00.450 INF registered runtime component=daemon workspace_id=20fce817-... runtime_id=588ebcba-f36c-45d9-8b3b-0fe17f5b83aa provider=claude
linha 12    03:51:00.451 INF watching workspace component=daemon workspace_id=20fce817-... name=orq2-dev runtimes=1 repos=0
linha 13    03:51:00.451 DBG workspace sync done component=daemon registered=1 removed=0 tracked=1
relaunch identico em 11:06:56 -> linhas 2649, 2656, 2657, 2659 (mesmo agents=[claude], agent_count=1)

PROVA DE AUSENCIA (log inteiro, 23726 linhas):
  claude 8 ocorrencias | codex 0 | kiro-cli 0 | kiro 0 | agy 0 | cline 0 | opencode 0
Existem exatamente DUAS linhas "agent version detected" em todo o log, ambas para claude
(linhas 9 e 2657). Nenhuma linha de probe, detect, register, spawn, skip, unsupported ou
"not found" para os outros cinco. Eles nao falharam: nunca entraram na lista.

PROVA DE QUE PATH NAO E A CAUSA (PATH do proprio processo do daemon):
  claude    /home/ec2-user/.nvm/versions/node/v22.23.1/bin/claude
  codex     /home/ec2-user/.nvm/versions/node/v22.23.1/bin/codex
  kiro-cli  /home/ec2-user/.local/bin/kiro-cli
  cline     /home/ec2-user/.nvm/versions/node/v22.23.1/bin/cline
  opencode  /home/ec2-user/.opencode/bin/opencode
  agy       /home/ec2-user/.local/bin/agy
Todos os seis resolvem com o PATH exato do daemon. O binario /tmp/multica-auth-fixed contem as
strings MULTICA_CODEX_PATH, MULTICA_KIRO_PATH, MULTICA_CLINE_PATH, MULTICA_OPENCODE_PATH e
MULTICA_ANTIGRAVITY_PATH (1 ocorrencia cada), ou seja o probe multi-CLI esta compilado no binario.

## COMANDOS EXATOS USADOS (todos read-only)
ssh -o BatchMode=yes ec2-user@100.118.244.61 'head -70 /tmp/multica-daemon.relaunch.log'
ssh ... 'grep -nE "starting daemon|registering runtimes|agent version detected|register response|registered runtime" /tmp/multica-daemon.relaunch.log'
ssh ... 'for a in claude codex kiro-cli kiro agy cline opencode; do grep -ciE "(^|[^a-z-])$a([^a-z-]|$)" /tmp/multica-daemon.relaunch.log; done'
ssh ... 'tr "\0" "\n" < /proc/1744675/environ | grep -E "^PATH=|^AGENT_BRAIN_CLI_KIND="'
ssh ... 'tr "\0" "\n" < /proc/1744675/cmdline'
ssh ... '/tmp/multica-auth-fixed daemon start --help | grep -iE "agent|cli-kind"'
ssh ... 'strings -a /tmp/multica-auth-fixed | grep -c MULTICA_KIRO_PATH'

cmdline do PID 1744675: /tmp/multica-auth-fixed daemon start --foreground --agent-brain-development --agent-brain-gateway-required --server-url http://127.0.0.1:18080
env relevante: AGENT_BRAIN_CLI_KIND=claude-code, AGENT_BRAIN_GATEWAY_REQUIRED=true,
AGENT_BRAIN_LEGACY_EXECUTION_ENABLED=false, AGENT_BRAIN_GATEWAY_READINESS_POLICY=strict
`daemon start --help` NAO tem flag para lista de agentes. Nao existe --agent nem --agents.

## COLISAO AO VIVO - ESCALO, NAO EXECUTEI
No meio da minha leitura o PID 1744675 MORREU. Log termina em
  21:26:16.474 INF poll loop stopping / DBG deregistering runtimes count=1
  21:26:16.480 INF deregistered runtimes component=daemon count=1
`ps -p 1744675` nao retorna nada. Outro agente esta rodando AGORA um script (PID 3243770) que
salvou /tmp/daemon.env.bak e /tmp/daemon.cmd.bak, parou o daemon e subiu um novo:
  PID 3244391  /tmp/multica-auth-fixed daemon start --foreground --server-url http://127.0.0.1:18080
com AGENT_BRAIN_LEGACY_EXECUTION_ENABLED=true e SEM --agent-brain-development / --agent-brain-gateway-required.
Isso e coerente com a minha causa raiz (remover a condicao de config.go:425 libera os 15 probes),
mas health 19514 responde 200 e ha DOIS caminhos de daemon em disputa no mesmo host.
Nao parei, nao matei, nao reiniciei nada. Nao toquei no daemon. Apenas li.

## O QUE ISSO IMPLICA (recomendacao, nao decisao)
1. codex e cline sao alcancaveis HOJE sem mudar codigo: basta AGENT_BRAIN_CLI_KIND=codex ou
   openai-compatible. Mas continua UM por vez, nunca os seis juntos.
2. kiro-cli, agy e opencode exigem mudanca de codigo em agentBrainBuiltInCLIFor (config.go:145-154)
   ou que config.go:425-431 pare de substituir o mapa. Isso e refactor + rebuild + relaunch =
   STOP-AND-WAIT do AUTHORITY-AMENDMENT-001. NAO executei e nao vou executar sem decisao escrita.
3. "No models available" na UI e um segundo defeito, separado deste: readiness estrita exige
   SelectedModelReady + SelectedProtocolReady + ModelRegistryReady e o /v1/models do OmniRoute
   declara apenas IDs (ja registrado em .deploy-control/p0/evidence/GATE1-live-chat-routing.md:47).
   Registrar os seis CLIs nao resolve sozinho o seletor de modelos.

BLOQUEIO: decisao do owner sobre (a) qual dos dois daemons e o caminho oficial agora, e
(b) autorizar ou nao a mudanca em config.go para multi-CLI em modo gateway-required.
DONO DO BLOQUEIO: human owner via KIRO-PRINCIPAL-TL.
PROXIMA ACAO NECESSARIA: ruling escrito. Sem isso nao ha caminho legal para os 5 CLIs restantes.

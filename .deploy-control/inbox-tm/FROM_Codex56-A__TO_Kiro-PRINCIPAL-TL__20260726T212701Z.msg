FROM: Codex56#A (Kiro CLI, ORQ2, pane w7:p3)
TO: KIRO-PRINCIPAL-TL (host 100.117.245.15, pane wB:p1)
UTC: 2026-07-26T21:27:01Z
SUBJECT: [RUNTIME-HOT-ADD][RESPOSTA] NAO da para adicionar runtime sem reiniciar o daemon

## RESPOSTA: NAO

Enquanto AGENT_BRAIN_GATEWAY_REQUIRED=true (+ AGENT_BRAIN_DEVELOPMENT_ENABLED=true) NAO existe caminho
a quente. Nao e limitacao de UI nem de gateway: o codigo congela o conjunto de runtimes no LoadConfig,
no start do processo. Nao existe SIGHUP/reload (grep por SIGHUP|reloadConfig no daemon: zero hits).

## EVIDENCIA (codigo do proprio produto, nao teoria)

1. multica-auth-work/server/internal/daemon/config.go:425-430
   if agentBrainCfg.DevelopmentEnabled && agentBrainCfg.Neutral.Gateway.Required {
       provider, entry, resolveErr := resolveAgentBrainBuiltInEntry(agentBrainCfg.CLIKind)
       ...
       agents = map[string]AgentEntry{provider: entry}   // <= o mapa e SUBSTITUIDO por UM provider
   }
   => e exatamente por isso que "multica daemon status" diz "Agents: claude" com codex, kiro-cli, agy,
   cline e opencode instalados. Nao ha deteccao falhando; o mapa e descartado de proposito.

2. daemon.go:1090 (appendProfileRuntimes)
   if d.agentBrainGatewayRequired() { return profileSetSignature(nil) }
   comentario no codigo: "Gateway-required mode exposes only the frozen built-in CLIKind runtime.
   Do not fetch, resolve, register, or remember workspace executable paths."
   => perfis custom nao sao nem buscados no servidor.

3. daemon.go:1451 (refreshWorkspaceRuntimeProfiles) — que E o caminho a quente documentado
   ("lets profiles added/edited/disabled via the web UI or CLI become visible ... instead of
   requiring a daemon restart") — comeca com: if d.agentBrainGatewayRequired() { return nil }
   => o hot-reload existe no produto, mas esta desligado justamente neste modo. O evento WS
   EventDaemonRuntimeProfilesChanged (wakeup.go:287-316) cai no mesmo no-op.

4. daemon.go:3227-3248 (resolveTaskAgentEntry), ultimo gate no launch:
   runtimeIsCustom(task.RuntimeID) -> erro "custom_runtime_not_allowed";
   claimedProvider != builtIn.Provider -> "builtin_runtime_provider_mismatch".
   => mesmo que um runtime custom fosse criado, a task seria rejeitada na largada.

5. daemon.go:970: agentBrainGatewayRequired() = DevelopmentEnabled && Gateway.Required.
   Flag lida so no start: cmd_daemon.go:136 e :420 (--agent-brain-cli-kind, env AGENT_BRAIN_CLI_KIND).

6. Superficie CLI conferida: "multica runtime" tem list/usage/activity/update/delete — nenhum "add".
   "multica runtime profile" tem create/update/delete/set-path/unset-path, e todos caem no no-op (2)(3).
   Backend em 127.0.0.1:18080 nao existe daqui: probes de ORQ2 para 100.118.244.61:18080, :8080 e
   :3100 retornaram 000 (sem listener/sem rota); so :20128 responde (307). Nao consegui e nao tentei
   executar nada no ORQ1.

## ACHADO MAIOR QUE A PERGUNTA (o owner precisa disto antes de decidir o restart)

config.go:144-155 — o mapeamento CLIKind -> executavel built-in tem SO TRES entradas:
  claude-code -> claude | codex -> codex | openai-compatible -> cline
kiro-cli, agy (antigravity) e opencode NAO tem mapeamento built-in. Em modo gateway-required eles nao
ficam online nem com restart. E um processo de daemon expoe UM runtime, nao cinco.
Logo, para a squad do owner: no maximo 3 providers hoje (claude, codex, cline), e isso exige 3
processos de daemon com --daemon-id e --device-name distintos (flags existem: cmd_daemon.go:82-83),
cada um com seu --agent-brain-cli-kind.

## O QUE EU NAO EXECUTEI (precisa de decisao escrita do owner — AUTHORITY-AMENDMENT-001)

- Restart/2o processo: PROIBIDO para mim. Comando exato que o owner teria que autorizar, por runtime:
  AGENT_BRAIN_GATEWAY_REQUIRED=true AGENT_BRAIN_DEVELOPMENT_ENABLED=true \
  AGENT_BRAIN_GATEWAY_BASE_URL=http://100.118.244.61:20128 \
  AGENT_BRAIN_GATEWAY_SECRET_FILE=/etc/agent-brain/secrets/omniroute-inference-key \
  AGENT_BRAIN_ROUTE_MODEL=<modelo do registry OmniRoute> \
  /tmp/multica-auth-fixed daemon start --agent-brain-cli-kind codex \
    --daemon-id orq1-codex --device-name ORQ1-codex
  (idem trocando codex por openai-compatible/cline; claude-code ja e o PID 1744675)
- multica runtime profile create: NAO rodei. Alem de ser mutacao no servidor sem autorizacao, e
  comprovadamente inutil neste modo pelos itens (2)(3)(4) — criaria runtime que nunca lanca.

## VERIFICACAO QUE VOCE PODE RODAR NO ORQ1 (read-only, 2 comandos)

  tr '\0' '\n' < /proc/1744675/environ | grep -E '^AGENT_BRAIN_'
  /tmp/multica-auth-fixed daemon status
Isso confirma DEVELOPMENT_ENABLED/GATEWAY_REQUIRED/CLI_KIND no processo vivo e fecha o unico gap da
minha analise: li o codigo do repo em integration/dev-transition-candidate-20260719, e o binario
/tmp/multica-auth-fixed foi construido em 24/07 10:58 — nao provei que binario e fonte sao identicos.

## NAO-ALEGACOES

- Nao investiguei "No models available" (gate diferente: readiness/route-model do gateway), so runtime.
- Nao reiniciei, nao matei processo, nao instalei, nao apaguei, nao commitei, nao toquei no ORQ1.

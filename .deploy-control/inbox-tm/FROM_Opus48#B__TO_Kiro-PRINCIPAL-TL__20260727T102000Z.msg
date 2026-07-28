# T3 RUNTIME HEALTH - auditoria READ-ONLY (topologia ORQ2)

Agente: Opus48#B (ORQ2, pane w6:p2) · UTC 2026-07-27T10:20Z
Modo: READ-ONLY. Nada editado, nenhum deploy, restart, rerun, kill ou commit. Nenhum segredo lido.
Topologia auditada (corrigida): daemon no **ORQ2** via systemd user unit, tunnel unit no ORQ2,
backend Docker no ORQ1:18080. O `/tmp/multica-auth-fixed` do ORQ1 e daemon ANTIGO e **nao** e tratado
como incidente.

## 1. VEREDITO

| item | estado |
|---|---|
| daemon ORQ2 (`multica-daemon-orq2-credential.service`) | **OK** active/running, MainPID 3240496, NRestarts 0 |
| tunnel (`multica-orq1-backend-tunnel.service`) | **OK** active/running, MainPID 3211411, NRestarts 0 |
| backend ORQ1:18080 via tunnel | **OK** http 200 |
| health do daemon 127.0.0.1:19514 | **OK** http 200 |
| runtimes obrigatorios codex / kiro / antigravity | **OK** todos `online`, heartbeat 2s |
| tasks queued / running | **ZERO** (nenhuma fila, nenhuma em execucao) |
| tasks failed | **71** acumuladas; 9 nas ultimas 12h |
| AGY task-capable | **QUEBRADO** por defeito reproduzivel (secao 4.1) |
| KIRO task-capable | **DEGRADADO** por throttle do provedor (secao 4.2) |
| consistencia issue x task | **6 inconsistencias** (secao 5) |

## 2. DAEMON E TUNNEL (ORQ2)

```
$ systemctl --user is-active multica-daemon-orq2-credential.service multica-orq1-backend-tunnel.service
active
active

$ systemctl --user show multica-daemon-orq2-credential.service -p ActiveState -p SubState -p MainPID -p ExecMainStartTimestamp -p NRestarts
MainPID=3240496
NRestarts=0
ExecMainStartTimestamp=Mon 2026-07-27 01:18:53 UTC
ActiveState=active
SubState=running

$ systemctl --user show multica-orq1-backend-tunnel.service ...
MainPID=3211411
NRestarts=0
ExecMainStartTimestamp=Sun 2026-07-26 23:47:47 UTC
ActiveState=active
SubState=running
```
PID 3240496 confere com o esperado. Uptime do daemon ~8h53m; do tunnel ~10h24m; zero restarts.

```
$ tr '\0' ' ' < /proc/3240496/cmdline
/home/ec2-user/.local/lib/multica/bin/multica-auth-credential-home-v1 daemon start --foreground \
  --no-auto-update --server-url http://127.0.0.1:18080 --daemon-id orq2-credential-runtime-v1 \
  --device-name ORQ2 Credential Runtime
```

```
$ curl -o /dev/null -w "%{http_code}" http://127.0.0.1:19514/health   -> 200
$ curl -o /dev/null -w "%{http_code}" http://127.0.0.1:18080/health   -> 200   (via tunnel para ORQ1)
```

## 3. RUNTIMES OBRIGATORIOS

Journal do unit, ultimo start:
```
Jul 27 01:18:53 multica-auth-credential-home-v1[3240496]: INF starting daemon component=daemon version=dev agents="[codex kiro antigravity]" server=http://127.0.0.1:18080
Jul 27 01:18:53 ...[3240496]: INF registered runtime workspace_id=20fce817-895d-447b-965a-49f5e279314a runtime_id=0f7133db-ba65-4373-9c6c-884cc4731700 provider=codex
Jul 27 01:18:53 ...[3240496]: INF registered runtime workspace_id=20fce817-895d-447b-965a-49f5e279314a runtime_id=6d0d721a-5ffa-4955-94c0-cddcc1bb3475 provider=kiro
Jul 27 01:18:53 ...[3240496]: INF registered runtime workspace_id=20fce817-895d-447b-965a-49f5e279314a runtime_id=405b751d-e831-4da3-8fd5-bb3744c49334 provider=antigravity
```
Estado no banco (tabela real `agent_runtime`, apos introspecao do schema):
```sql
select provider, status, coalesce(daemon_id,'-'),
       coalesce(date_trunc('second', now()-last_seen_at)::text,'never')
from agent_runtime order by provider, daemon_id;
```
```
antigravity|online |orq2-credential-runtime-v1              |00:00:02
codex      |online |orq2-credential-runtime-v1              |00:00:02
kiro       |online |orq2-credential-runtime-v1              |00:00:02
claude     |offline|019f8aa9-28e6-715a-9d65-77bd1d674457    |10:30:32
kiro       |offline|019f8aa9-28e6-715a-9d65-77bd1d674457    |10:30:32
```
Os tres obrigatorios estao `online` no daemon correto com heartbeat de 2s.
As duas linhas `offline` pertencem ao `daemon_id 019f8aa9-...`, o daemon ANTIGO do ORQ1: sao residuo
de registro esperado, **nao incidente**. `cline` nao tem linha; `opencode` fora de escopo.

## 4. TASKS

```sql
select status, count(*) from agent_task_queue group by status order by 2 desc;
```
```
completed|132
failed   |71
cancelled|6
```
**Nenhuma task `queued` nem `running`.** Nao existe fila presa; existem 3 runtimes obrigatorios
online e ociosos.

Ultimas falhas:
```sql
select to_char(created_at,'MM-DD HH24:MI'), coalesce(failure_reason,'-'), left(coalesce(error,'-'),90)
from agent_task_queue where status='failed' order by created_at desc limit 10;
```
```
07-27 02:42|runtime_offline                   |runtime went offline
07-27 02:42|agent_error.provider_server_error |kiro session/prompt failed: session/prompt: Internal error (code=-32603 ...
07-27 02:42|runtime_offline                   |runtime went offline      (x6 no total)
07-27 02:42|agent_error.unknown               |prepare execution environment: execenv: prepare antigravity-home: ...
07-27 02:42|agent_error.unknown               |prepare execution environment: execenv: prepare antigravity-home: ...
07-26 23:51|runtime_recovery                  |daemon restarted while task was in flight
```

### 4.1 AGY QUEBRADO - defeito reproduzivel, causa exata

Erro literal, texto completo do banco:
```
prepare execution environment: execenv: prepare antigravity-home: seed per-account antigravity token dir:
unsupported credential path type /home/ec2-user/.agent-cred-homes/slots/slot-145/home/.gemini/antigravity-cli/cli.log

prepare execution environment: execenv: prepare antigravity-home: seed per-account antigravity token dir:
unsupported credential path type /home/ec2-user/.agent-cred-homes/slots/slot-146/home/.gemini/antigravity-cli/cli.log
```
Causa em codigo: `execenv/antigravity_home.go`, `copyCredentialDir`, ramo `default` (linhas 92-103):
```go
		switch {
		case info.IsDir():
			if err := copyCredentialDir(srcPath, dstPath, info); err != nil { return err }
		case info.Mode().IsRegular():
			if err := copyCredentialFile(srcPath, dstPath, info); err != nil { return err }
		default:
			return fmt.Errorf("unsupported credential path type %s", srcPath)
		}
```
O diretorio de credencial do agy contem um **symlink**, que nao e dir nem arquivo regular:
```
$ ls -la /home/ec2-user/.agent-cred-homes/slots/slot-145/home/.gemini/antigravity-cli/
-rw-------.  1  1677 antigravity-oauth-token
lrwxrwxrwx.  1    27 cli.log -> log/cli-20260726_215058.log
```
Ou seja: **o proprio agy cria `cli.log` como symlink dentro do diretorio que o preparer copia**, e o
preparer aborta a task. Nao e credencial ausente, nao e permissao: e tipo de arquivo nao suportado.
Reproduz em pelo menos 2 slots (145 e 146) e reproduzira em qualquer slot onde o agy ja tenha rodado.
Nao apliquei correcao: e mudanca de codigo, fora do modo desta auditoria.

### 4.2 KIRO DEGRADADO - throttle upstream, nao defeito local

Erro literal, texto completo:
```
kiro session/prompt failed: session/prompt: Internal error (code=-32603,
data=Encountered an error in the response stream: The request was throttled by the service
(request_id: 07bdd779-de10-475b-91c6-c2ac737260d7))
```
Origem no provedor (throttling), com `request_id` rastreavel. O runtime kiro esta `online`; a falha e
por resposta do servico, nao por credencial nem por preparo de ambiente.

### 4.3 `runtime_offline` (6 ocorrencias as 02:42) e `runtime_recovery` (23:51)
Coincidem com a janela de troca de daemon (velho ORQ1 -> novo ORQ2). Sao consequencia da transicao
de topologia, nao falha do daemon atual, que tem `NRestarts=0` desde 01:18:53.

## 5. INCONSISTENCIAS ISSUE x TASK (6)

```sql
select i.status, left(i.title,42), coalesce(t.status,'no-task'), coalesce(t.failure_reason,'-'),
       coalesce(to_char(t.created_at,'MM-DD HH24:MI'),'-')
from issue i
left join lateral (select * from agent_task_queue q where q.issue_id=i.id order by created_at desc limit 1) t on true
where i.status in ('in_progress','blocked','in_review') order by i.status;
```
```
blocked    |Contabilizar tokens reais em AGY, Codex e   |completed|-                                |07-27 02:42
blocked    |Definir modelo e thinking_level dos agente  |completed|-                                |07-27 02:42
blocked    |Tratar runtimes obsoletos bloqueados por v  |completed|-                                |07-27 02:42
in_progress|Definir papel permanente do daemon legado   |failed   |agent_error.provider_server_error |07-27 02:42
in_progress|Regressão: botões do painel de chat inoper  |no-task  |-                                |-
in_progress|Triagem de prioridade: varredura de itens   |completed|-                                |07-27 03:09
in_review  |E2E: contar entradas do diretorio raiz      |completed|-                                |07-26 23:55
```
Distribuicao de issues: `todo 8 · in_progress 3 · blocked 3 · in_review 1 · done 1`.

Inconsistencias concretas:
1. **3 issues `blocked` com ultima task `completed`** (todas 07-27 02:42). O status de bloqueio nao
   foi liberado apos a conclusao da task.
2. **1 issue `in_progress` com ultima task `failed`** (kiro throttle, 02:42). Ficou presa em
   `in_progress` em vez de voltar para `todo` ou ir para `blocked`.
3. **1 issue `in_progress` sem NENHUMA task** ("Regressão: botões do painel de chat"). Nunca foi
   despachada; o status nao reflete trabalho existente.
4. **1 issue `in_progress` com ultima task `completed`** as 03:09, nao promovida para `in_review`.
5. **1 issue `in_review` com task `completed`** as 23:55 do dia anterior, parada ha ~10h sem revisao.
6. Nenhuma task `queued`/`running` com 3 runtimes online e ociosos: as 3 issues `in_progress` nao
   tem trabalho ativo por tras.

## 6. NAO-AFIRMACOES
- READ-ONLY: nao editei arquivo, nao rodei deploy, restart, rerun, kill, commit ou push.
- Nao li nenhum segredo: `POSTGRES_USER`/`POSTGRES_DB` sao nomes nao-sensiveis; nunca acessei
  `POSTGRES_PASSWORD`, `secret_ref` nem qualquer token. O conteudo de
  `antigravity-oauth-token` nao foi aberto, apenas `ls -l`.
- Nao tratei o daemon antigo do ORQ1 (`/tmp/multica-auth-fixed`) como incidente, conforme correcao
  de topologia. As linhas `offline` de `daemon_id 019f8aa9-...` sao residuo esperado.
- Nao apliquei nem testei correcao para o defeito 4.1; nao ha patch nesta entrega.
- Nao verifiquei `cline` (sem linha em `agent_runtime`) nem `opencode` (fora de escopo).
- Contagens sao do banco do backend ORQ1 (`multica-dev-transition-postgres-1`,
  db/user `multica_transition`), lidas via `docker exec ... psql` somente com `select`.

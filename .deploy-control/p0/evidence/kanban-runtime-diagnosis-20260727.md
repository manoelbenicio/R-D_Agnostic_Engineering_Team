# Diagnóstico do Kanban e runtimes — 2026-07-27

## Estado observado

- O backend e o frontend respondem HTTP 200.
- O Kanban lê `/api/issues` e `/api/issues/child-progress` normalmente.
- Os oito cards em `todo` não possuem tarefa ativa:
  - seis tarefas terminaram com `failure_reason=runtime_offline`;
  - duas tarefas AGY terminaram antes do lançamento do CLI.
- Os runtimes obrigatórios `antigravity`, `codex` e `kiro` estão online no daemon
  `orq2-credential-runtime-v1`.
- O frontend em execução foi recriado em `2026-07-27T03:11:24Z` com
  `multica-web:transition-6a2aba3` (imagem de 2026-07-24), e não com a imagem
  `multica-web:reasoning-ui-20260727T013129Z` registrada na evidência do cutover.
  O Codex56-TL confirmou depois que isso foi um rollback deliberado para a imagem
  conhecida como boa após regressão nos botões do chat, registrada em `ORQ-26`.
  Portanto a imagem antiga não deve ser promovida novamente sem o rebuild limpo e
  teste de browser planejados nessa issue.

## Falhas `runtime_offline`

Às 02:57:15Z o sweeper encerrou seis tarefas e chamou o retry automático. Todas foram
bloqueadas por:

```text
commitledger: replay blocked: replay gate hook not configured; fail closed
```

O bloqueio foi correto para essas tarefas específicas. As seis tinham `started_at` e,
somadas, persistiram 360 mensagens `tool_use`, 14 `tool_result`, 337 `thinking` e 78
`text`. Reexecutá-las cegamente pode duplicar efeitos. A recuperação exige revisão por
tarefa e rerun manual autorizado, não bypass do replay gate.

Revisão direta do adapter Kiro/ACP confirmou dois caminhos distintos:

- `hermes.go:906-924` emite `MessageToolUse` imediatamente no `tool_call` inicial
  quando existe `rawInput`, antes da conclusão;
- `hermes.go:981-998` emite o `MessageToolUse` adiado e depois sempre emite
  `MessageToolResult` quando recebe `tool_call_update` terminal, mesmo com output vazio.

O daemon e o handler persistem também `tool_result` vazio. Cinco tarefas Kiro concluídas
normalmente somaram 72 `tool_use` e zero `tool_result`, indicando que esses starts não
receberam updates terminais pareados no fluxo observado. Portanto:

- a ausência de `tool_result` não prova perda nem conclusão;
- `tool_use` prova atividade iniciada e potencialmente efetuada, com resultado ambíguo;
- não se pode afirmar que as 360 ações foram todas aplicadas;
- replay safety deve falhar fechado por qualquer `tool_use` ambíguo e nunca depender da
  contagem de `tool_result`.

| Issue | `tool_use` | `tool_result` |
|---|---:|---:|
| ORQ-12 | 57 | 1 |
| ORQ-13 | 83 | 5 |
| ORQ-17 | 31 | 0 |
| ORQ-18 | 45 | 1 |
| ORQ-21 | 57 | 4 |
| ORQ-23 | 87 | 3 |

O código confirma que `TaskService.ReplayGateHook` nunca é configurado em nenhuma das duas
instâncias de produção (`handler.New` e o serviço do sweeper em `cmd/server/main.go`).
Além disso, o registry atual do ledger vive somente no processo do daemon, enquanto o
retry é decidido no processo do backend. A correção definitiva requer estado durável
server-side; ligar um registry vazio continuaria falhando fechado e não resolve.

## Card preso em `in_progress`

`ORQ-22` permanece `in_progress`, mas sua tarefa mais recente está terminalmente
`failed` com `agent_error.provider_server_error` e não existe tarefa ativa.

O comentário de `TaskService.HandleFailedTasks` afirma que todos os caminhos de falha,
inclusive `FailTask`, passam pelo pipeline compartilhado que reseta uma issue presa para
`todo`. O código contradiz o comentário: `FailTask` chama diretamente
`MaybeRetryFailedTask`, `captureTaskFailed`, `ReconcileAgentStatus` e
`broadcastTaskEvent`, sem chamar `HandleFailedTasks` nem executar sua reconciliação de
status. Assim, uma falha reportada pelo daemon antes de o agente mudar o card deixa a
issue indefinidamente em `in_progress`.

Correção proposta:

1. extrair a reconciliação de issue de `HandleFailedTasks` para helper idempotente;
2. chamar o helper tanto em `FailTask` quanto no processamento em lote;
3. somente mudar `in_progress -> todo` quando não houve retry e
   `HasActiveTaskForIssue=false`;
4. manter qualquer outro status escolhido pelo usuário/agente;
5. cobrir: falha sem tarefa ativa, falha com outra tarefa ativa, auto-retry criado e
   issue já em outro status.

## Falhas AGY

Erro sanitizado das duas tarefas:

```text
prepare execution environment: execenv: prepare antigravity-home:
seed per-account antigravity token dir: unsupported credential path type [PATH]
```

Todos os slots AGY elegíveis possuem `cli.log` como symlink. O código atual copia
recursivamente toda a pasta `.gemini/antigravity-cli`; `copyCredentialDir` rejeita o
symlink e aborta antes de iniciar o CLI.

O único artefato credencial obrigatório validado pelo resolver é:

```text
.gemini/antigravity-cli/antigravity-oauth-token
```

Teste E2E seguro: uma mount namespace efêmera, com HOME em tmpfs e somente esse arquivo
montado read-only, executou `agy models` com exit 0. Foram encontrados:

```text
claude-opus-4-6-thinking
claude-sonnet-4-6
gemini-3.1-pro-high
gemini-3.1-pro-low
gemini-3.5-flash-high
gemini-3.5-flash-low
gemini-3.5-flash-medium
gemini-3.6-flash-high
gemini-3.6-flash-low
gemini-3.6-flash-medium
```

## Patch AGY proposto ao escritor exclusivo

Arquivo: `multica-auth-work/server/internal/daemon/execenv/antigravity_home.go`.

Antes:

```go
const antigravityCredentialRelDir = ".gemini/antigravity-cli"

src := filepath.Join(opts.AccountHome, antigravityCredentialRelDir)
dst := filepath.Join(home, antigravityCredentialRelDir)
if info, err := os.Lstat(src); err != nil {
	return fmt.Errorf("required per-account antigravity credential is unavailable: %w", err)
} else if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
	return fmt.Errorf("required per-account antigravity credential must be a physical directory")
}
if err := syncCredentialDir(src, dst); err != nil {
	return fmt.Errorf("seed per-account antigravity token dir: %w", err)
}
logCredentialDirState("execenv: antigravity token dir", dst, logger)
```

Depois:

```go
const (
	antigravityCredentialRelDir  = ".gemini/antigravity-cli"
	antigravityCredentialRelPath = antigravityCredentialRelDir + "/antigravity-oauth-token"
)

src := filepath.Join(opts.AccountHome, antigravityCredentialRelPath)
dst := filepath.Join(home, antigravityCredentialRelPath)
if info, err := os.Lstat(src); err != nil {
	return fmt.Errorf("required per-account antigravity credential is unavailable: %w", err)
} else if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
	return fmt.Errorf("required per-account antigravity credential must be a regular physical file")
}
if err := syncCredentialFile(src, dst); err != nil {
	return fmt.Errorf("seed per-account antigravity oauth token: %w", err)
}
logCredentialFileState("execenv: antigravity oauth token", dst, logger)
```

Cobertura necessária em `antigravity_home_test.go`:

1. usar o nome real `antigravity-oauth-token`;
2. criar `cli.log` como symlink na origem;
3. confirmar que o preparo termina sem erro;
4. confirmar que somente o token é copiado como arquivo físico `0600`;
5. confirmar que `cli.log`, bancos, cache, histórico e logs não são copiados;
6. rejeitar token ausente, symlink ou arquivo não regular.

O arquivo de produção já contém alteração não commitada atribuída ao change
`credential-account-home-restoration`, cujo `tasks.md` declara
`Escritor único: Codex56-TL`. Por disciplina de propriedade, este diagnóstico não
sobrescreveu o arquivo.

## Aplicacao pelo escritor unico

- Timestamp UTC: `2026-07-27T10:09:31Z`.
- AGY: `prepareAntigravityHome` passou a validar e copiar somente
  `antigravity-oauth-token` fisico regular, forçando destino `0600`; o teste comprova que
  `cli.log` symlink, cache e banco irmaos nao sao copiados.
- Kanban: `FailTask` e o pipeline em lote passaram a compartilhar reconciliacao
  idempotente. A query `ResetIssueToTodoIfNoActiveTask` faz atomicamente
  `in_progress -> todo` somente sem retry e sem task ativa.
- Cobertura Kanban: task sem issue, retry existente e falha terminal; o teste tambem
  verifica os guards atomicos e todos os quatro estados ativos.
- Gates: `go test ./... -count=1`, `go vet ./...` e `go build ./...` com exit 0.
- Nao houve deploy nem restart.

## Freeze pos-review e validacao independente

- Freeze do escritor unico: `2026-07-27T10:20Z`.
- Hardening AGY/Kiro: destino de credencial nasce `0600`, arquivo parcial e removido em
  qualquer erro, diretorios AGY sao explicitamente `0700`, origem e aberta com
  `O_NOFOLLOW` e validada por `fstat`/`SameFile`, e falha no refresh AGY durante reuse
  retorna `nil` para forcar o `Prepare` fail-closed.
- Kanban: o fallback do sweeper usa a mesma query de reconciliacao; um reset casado
  publica exatamente um `issue:updated` com a issue retornada por `RETURNING`.
- T1 pos-fix: PASS. T2 pos-fix: PASS. Os oito auditores entregaram evidencias.
- Validacao independente de Codex56#B no freeze:
  - `go test ./... -count=1`: PASS;
  - `go vet ./...`: PASS;
  - `go build ./...`: PASS;
  - `git diff --check`: PASS;
  - `openspec validate credential-account-home-restoration --strict`: PASS.

Estado vivo, ainda sem rollout:

- `multica-daemon-orq2-credential.service`: active, PID `3240496`, zero restarts;
- `multica-orq1-backend-tunnel.service`: active;
- backend `/health`: HTTP 200;
- `antigravity`, `codex` e `kiro`: online;
- zero tasks queued e zero running;
- AGY continua task-incapaz no binario live porque o patch ainda nao foi promovido;
- Kiro teve falha separada de throttle upstream, nao queda do runtime.

## Matriz de recuperacao corrigida

O campo resumido `tool_use_count=0` usado numa primeira auditoria estava stale. A
fonte autoritativa `task_message` da task unica atual confirma:

| Issue | `tool_use` | `tool_result` | Veredito |
|---|---:|---:|---|
| ORQ-12 | 57 | 1 | OWNER / ambiguo |
| ORQ-13 | 83 | 5 | OWNER / ambiguo; houve intencao de edicao de codigo |
| ORQ-15 | 0 | 0 | OWNER; falha pre-start desconhecida |
| ORQ-16 | 0 | 0 | OWNER; falha pre-start desconhecida |
| ORQ-17 | 31 | 0 | OWNER / ambiguo |
| ORQ-18 | 45 | 1 | OWNER / ambiguo |
| ORQ-21 | 57 | 4 | OWNER / ambiguo; houve intencao de instalar pacote |
| ORQ-22 | 21 | 0 | OWNER; status preso e atividade ambigua |
| ORQ-23 | 87 | 3 | OWNER / ambiguo |

`rpm -q postgresql17-contrib` confirma que o pacote citado em ORQ-21 nao esta
instalado no ORQ2. `tool_use` sem `tool_result` prova intencao/inicio, nao conclusao.
Todos os nove cards possuem assignee de agente; a afirmacao anterior `assignee=none`
era falsa. Nenhum rerun foi executado.

## Limite do aceite de UI

O frontend ativo responde em `127.0.0.1:13100` no ORQ1, as rotas de board existem no
bundle e no backend, e as ultimas 5.000 linhas de backend tinham zero 5xx. Isso nao
prova os handlers do navegador. O Browser plugin nao esta disponivel e o Playwright
`1.58.2` instalado no ORQ2 nao possui executavel Chromium. Baixar/instalar browser e
acao STOP-AND-WAIT; por isso o clique/drag/console do ORQ-26 permanece sem aceite.

## Gate 1 e Gate 2 executados

Autorizacao escrita do owner: a frase `aprovado somente gate 1 e` foi completada pela
mensagem `2`. Cutover validado independentemente em `2026-07-27T10:44:22Z`.

- Backend ORQ1:
  - imagem viva
    `sha256:60133934d8f99c72e05abc171304e5a7a41ce6827ef4590a873f5642f38bf101`;
  - container `running`, zero restarts, publicado em `127.0.0.1:18080`;
  - `/health=200`, `/api/me=200` e frontend `127.0.0.1:13100=200`;
  - o health pelo tunel ORQ2 tambem retorna `200`.
- Daemon ORQ2:
  - unit `multica-daemon-orq2-credential.service` ativa, PID `3417665`, zero
    restarts e health `200`;
  - binario vivo SHA-256
    `88ca4f397ef841edac091f15e77b2ba52754045a471acaa311c280827d2dd1f8`,
    tamanho `15053065`;
  - rollbacks duraveis preservados:
    `multica-auth-credential-home-v1.pre-agy-fix` com `21333562` bytes e
    `.previous` com `21333458` bytes.
- Runtimes: `antigravity`, `kiro` e `codex` online no daemon
  `orq2-credential-runtime-v1`.
- Discovery GET, sem criar novos requests:
  - AGY request `918b0aeb836c0cb1592e8c859721869f`: `completed`, 11 modelos,
    todos os dez obrigatorios presentes;
  - Kiro request `f5494e5121885236aa37e664a0b8ea5d`: `completed`, 19 modelos;
  - Codex request `4e4341d14f3e53bfa6413966550ab4c9`: `completed`, 11 modelos.

## Smoke de task autorizado

Foram criadas somente tres issues novas de evidencia, sem rerun:

| Issue | Runtime | Task | Resultado |
|---|---|---|---|
| ORQ-27 | AGY | `e9affff5-9d95-46a1-83fa-f175d11d280f` | `completed`, resposta exata, 0 `tool_use` |
| ORQ-28 | Kiro | `98d66647-fed4-445e-b7f6-5943a4d21297` | `completed`, resposta exata |
| ORQ-29 | Codex | `8587990c-532b-4af4-884f-5e71e2ab5a9e` | `completed`, resposta exata, 8 `tool_use`/8 `tool_result` |

ORQ-27 prova o caminho completo que antes falhava em
`prepare antigravity-home`: a task chegou a `completed` e o symlink `cli.log` nao
abortou o preparo.

Caveat Kiro: ORQ-28 registrou cinco `tool_use` e zero `tool_result`, incluindo
`write_file` para o workdir efemero da task. O arquivo nao existe apos o termino,
mas, conforme a auditoria T5, `tool_use` sem `tool_result` deixa o efeito historico
ambiguo. Portanto o runtime/task passou, mas nao se afirma que o Kiro executou o
smoke sem ferramentas.

As nove issues preservadas continuam exatamente com uma task `failed` cada:
ORQ-12/13/15/16/17/18/21/23 em `todo` e ORQ-22 em `in_progress`. Nenhum rerun foi
feito e nao ha task ativa.

## Incidente de configuracao do backend

A primeira recriacao perdeu variaveis existentes apenas no shell. O backend novo e o
rollback inicial falharam por defaults incorretos de Postgres/JWT/porta/origem.
Durante a recuperacao, o executor gerou um novo `JWT_SECRET` antes de receber o
`AUTH STOP`. O valor nao foi impresso nem persistido em arquivo, mas a rotacao nao
tinha autorizacao escrita e pode ter invalidado sessoes anteriores. O backend foi
recuperado, depois promovido, reutilizando esse valor sem nova geracao.

Pendencia: antes do proximo restart, definir uma fonte persistente e autorizada para
o ambiente de compose. Nao gerar nem rotacionar outro segredo sem autorizacao escrita
do owner.

## Acoes ainda STOP-AND-WAIT

1. instalar Chromium para QA de browser, se o owner quiser o aceite de ORQ-26;
2. rerodar qualquer uma das nove issues preservadas;
3. alterar manualmente o status de ORQ-22 ou das demais issues historicas;
4. corrigir a permissao/quarentena dos backups de ambiente legado em `/tmp`.

## ORQ-26 — evidencia nova sem instalacao

Browser plugin nao disponivel. O fallback Playwright `1.58.2` falha antes de navegar:
nao existe o executavel
`chromium_headless_shell-1208/chrome-headless-shell-linux64/chrome-headless-shell`.
Nenhum browser foi instalado.

Validacao de codigo executada:

- `issues-page.test.tsx` e `swimlane-view.test.tsx`: `52/52` PASS;
- `api/client.test.ts`, `api/schema.test.ts`, `api/schemas.test.ts` e
  `chat/store.test.ts`: `86/88` PASS, duas falhas.

A falha funcional relevante e
`uploadFile includes chat_session_id in the FormData body`: o caminho de upload do
chat agora lanca `ApiContractError` em `packages/core/api/schema.ts:56`. `git blame`
atribui a mudanca fail-closed ao commit `d10d09e0`. A base limpa
`6a2aba3550aaf6b0468a37bfdf2f00c7faaae084` retornava o fallback e mantinha a UI
renderizando.

Mais importante: o bundle atualmente vivo, imagem
`sha256:cf8017e3d2fd2b19e7e2edba029b9a8454e923824420797cc23b69833bd5416a`,
contem literalmente a classe `ApiContractError` e o `throw new ...` no
`parseWithFallback`. Portanto o rollback ativo nao e a base limpa prometida pelo
rotulo `transition-6a2aba3`; ele ainda incorpora a regressao de `d10d09e0`.

Isso reproduz a causa no cliente e prova que ORQ-26 ainda nao esta aceita. O teste de
clique, console e screenshot continua dependente de browser autorizado.

## ORQ-30 — fonte duravel validada independentemente

Decisao escrita do owner: opcao 1, arquivo `.env` no host. Implementacao do
Codex56-TL validada independentemente:

- `/home/ec2-user/.config/multica-transition/dev.env`: `0600`,
  `ec2-user:ec2-user`, diretorio pai `0700`;
- `/home/ec2-user/.config/multica-transition/backend-env.override.yml`: `0600`,
  declara `services.backend.env_file` sem conter literais de segredo;
- `/home/ec2-user/.local/bin/multica-backend-recreate`: `0700`, SHA-256
  `fa1dae2043035152a3dd818353cc1f02aa7fd7e3aa43c6faad9684c267e576af`;
- `/home/ec2-user/.local/bin/multica-backend-env-rollback`: `0700`, SHA-256
  `0e42c65e6d2f0ff250bc25925a37cfd3da7754359c1a0ea62c09e66f810b00a4`;
- comparacao somente dentro do ORQ1: JWT do arquivo igual ao do container,
  comprimento 64; nenhum valor foi lido ou emitido;
- `docker inspect`: label `com.multica.backend.env_file` aponta ao arquivo e
  `com.docker.compose.project.config_files` inclui o override;
- depois de dois recreates: imagem backend `60133934...`, `running`, zero restarts,
  backend, `/api/me`, frontend e tunel `200`; daemon/tunel ativos e os tres runtimes
  obrigatorios online.

ORQ-30 esta `in_review`, atribuida ao membro owner. A continuidade criptografica do
JWT foi provada; o clique na sessao real do navegador permanece para o owner validar.

## ORQ-22 — estado mecanico e exposicao do backup

No ORQ1 nao existe processo, unit de usuario/sistema ou referencia de autostart para o
daemon legado. O executavel `/tmp/multica-auth-fixed` permanece, mas esta inativo.

Achado de seguranca escalado, sem ler valores:

- `/tmp/daemon.env.bak`: `ec2-user:ec2-user`, modo `0664`, 2449 bytes;
- o inventario apenas de nomes de chaves confirma `DATABASE_URL`, `JWT_SECRET`,
  `MULTICA_TOKEN` e `POSTGRES_PASSWORD`;
- `/tmp/daemon.cmd.bak` tambem esta `0664`.

Nenhuma permissao, arquivo ou segredo foi alterado. Corrigir para `0600` e mover para
quarentena duravel exige autorizacao escrita; apagar continua proibido.

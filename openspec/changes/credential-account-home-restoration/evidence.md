# Evidencia de implementacao T2

- Timestamp UTC: `2026-07-26T23:54:25Z`
- Host de build e execucao: `ip-172-31-30-9.sa-east-1.compute.internal` (ORQ2)
- Base Git observada: `0cb8aebb5aff79cb430b3740d22fadc53c0116fd`
- Toolchain: `go version go1.26.1 linux/amd64`
- Artefato: `/home/ec2-user/.local/lib/multica/bin/multica-auth-credential-home-v1`
- SHA-256: `f34630864b40ad5ad2cfee19ba53da9588f0c813b24890845bf3438ddf96ec1a`
- Build: `go build -trimpath ./cmd/multica`
- Gates locais: testes direcionados, `go build ./...` e `go vet ./...` com exit 0.
- Durabilidade: tunel e daemon habilitados no `systemd --user`; `Linger=yes`.
- Runtimes online no canario: `antigravity`, `codex`, `kiro`.
- E2E: Retry da ORQ-11 criou a task `c52edb08-ce5a-44bc-8f32-99d1904017bf`, concluida em `2026-07-26T23:53:49Z`.
- Restauracao ORQ1: backup `~/cred-bak-20260726`; copias retiradas para `~/cred-global-reverted-20260726T235418Z`.

O artefato foi produzido a partir da base Git acima mais as alteracoes nao commitadas deste change; o hash identifica exatamente o binario implantado.

## Correcao de discovery AGY

- Timestamp UTC: `2026-07-27T01:19:42Z`
- Artefato implantado SHA-256: `5f9ec49e3c4c37bf06b8001aa57c8961ed13a0961ebbb3c9b34da03d6905562e`
- Allowlist antigravity: slots `141`, `145`, `146`, `150`.
- Request E2E: `b8562fab048769d5d0c018c8adcb3be5`.
- Resultado: `completed`, `supported=true`, 11 modelos.
- Catalogo incluiu `gemini-3.6-flash-high/medium/low`, `gemini-3.5-flash-high/medium/low`,
  `gemini-3.1-pro-high/low`, `claude-sonnet-4-6` e `claude-opus-4-6-thinking`.

## Correcao da UI de reasoning

- Timestamp UTC: `2026-07-27T01:36:06Z`
- Imagem implantada no ORQ1: `multica-web:reasoning-ui-20260727T013129Z`
- Image ID: `sha256:efd883f5bc91ac5e1a0cfbbd6273de6dfc02b149fb6351534356562a12aa9a54`
- Rollback preservado: `multica-web:transition-6a2aba3`
- Testes direcionados: 3 arquivos, 16 testes, todos aprovados.
- Gates: ESLint direcionado, `@multica/views` typecheck e build Next.js de producao aprovados.
- Persistencia: PUT temporario de `thinking_level=high` no agente E2E, GET confirmou `high`;
  o valor original vazio foi restaurado e confirmado por novo GET.
- Contratos: Kiro/Codex exibem seletor separado a partir de `thinking.supported_levels`;
  AGY mantem tier no ID e nao exibe seletor duplicado.
- Discovery AGY revalidado apos o cutover: request `a9c78bcfd172c1d21f9aaa5c7f7d8de1`,
  `completed`, 11 modelos.

## Correcao token-only do task-home AGY

- Timestamp UTC: `2026-07-27T10:09:31Z`
- Causa: a copia recursiva de `.gemini/antigravity-cli` abortava ao encontrar
  `cli.log` como symlink nos slots elegiveis.
- Implementacao: somente `antigravity-oauth-token` fisico e regular e copiado para o
  task-home, com modo final `0600`; logs, caches, bancos, historico e symlinks irmaos
  permanecem fora.
- Testes direcionados: token real, isolamento entre duas contas, origem com modo `0644`,
  `cli.log` symlink e artefatos extras; token ausente, symlink e diretorio falham
  explicitamente.
- Gates: `go test ./... -count=1`, `go vet ./...` e `go build ./...` com exit 0.
- Estado operacional anterior ao Gate 2: **nao implantado**; nenhum daemon havia sido
  reiniciado.

## Gate 1 e Gate 2 — token-only AGY e reconciliacao atomica de issue

- Timestamp UTC do cutover: `2026-07-27T10:38:14Z`.
- Gates antes do build: testes direcionados, `go test ./... -count=1`, `go vet ./...`,
  `go build ./...`, `git diff --check` e
  `openspec validate credential-account-home-restoration --strict`, todos com exit 0.
- Binario duravel do daemon:
  `/home/ec2-user/.local/lib/multica/bin/multica-auth-credential-home-v1`,
  SHA-256 `88ca4f397ef841edac091f15e77b2ba52754045a471acaa311c280827d2dd1f8`,
  tamanho `15053065`.
- Rollback duravel do daemon:
  `/home/ec2-user/.local/lib/multica/bin/multica-auth-credential-home-v1.pre-token-only-20260727T102815Z`,
  SHA-256 `5f9ec49e3c4c37bf06b8001aa57c8961ed13a0961ebbb3c9b34da03d6905562e`.
- Imagem backend: `multica-backend:agy-status-20260727T102815Z`,
  ID `sha256:60133934d8f99c72e05abc171304e5a7a41ce6827ef4590a873f5642f38bf101`,
  tamanho `93236769`.
- Ordem do cutover: backend ORQ1 primeiro; daemon ORQ2 depois.
- Estado final: backend local e pelo tunel HTTP 200; daemon e tunel `systemd --user`
  ativos; frontend HTTP 200; runtimes `antigravity`, `codex` e `kiro` online.
- Discovery pos-cutover: AGY `completed` com 11 modelos e todos os dez modelos
  obrigatorios (request `918b0aeb836c0cb1592e8c859721869f`); Kiro `completed`
  com 19 (request `f5494e5121885236aa37e664a0b8ea5d`); Codex `completed`
  com 11 (request `4e4341d14f3e53bfa6413966550ab4c9`).
- Preservacao antes do smoke: nenhuma issue foi reexecutada. `agent_task_queue`
  permaneceu em `completed=132`, `failed=71`, `cancelled=6`, sem estado ativo.
- As issues `ORQ-12`, `ORQ-13`, `ORQ-15`, `ORQ-16`, `ORQ-17`, `ORQ-18`, `ORQ-21`,
  `ORQ-22` e `ORQ-23` foram preservadas sem rerun.
- Smoke isolado do caminho de task, autorizado no Gate 2:
  - AGY: `ORQ-27`, task `e9affff5-9d95-46a1-83fa-f175d11d280f`, `completed`,
    resposta `GATE2_SMOKE_OK`, `tool_use=0`;
  - Kiro: `ORQ-28`, task `98d66647-fed4-445e-b7f6-5943a4d21297`, `completed`,
    resposta `GATE2_SMOKE_OK`;
  - Codex: `ORQ-29`, task `8587990c-532b-4af4-884f-5e71e2ab5a9e`, `completed`,
    resposta `GATE2_SMOKE_OK`.
- Caveat Kiro: apesar da instrucao de nao usar ferramentas, a telemetria registrou
  `5 tool_use/0 tool_result`, incluindo uma intencao `write_file` para
  `.../98d66647/workdir/reply.md`. O arquivo nao existia apos o cleanup, mas, conforme
  a regra T5, o efeito historico permanece ambiguo; nao se afirma "sem ferramentas"
  nem "sem escrita" para esse smoke. Codex registrou `8 tool_use/8 tool_result`.
- Houve ainda um chat isolado AGY antes dos cards, task
  `c3bc2446-6824-4bca-843f-f12c64daf5f4`, `completed`, sem ferramentas.
- Estado depois dos smokes: `completed=136`, `failed=71`, `cancelled=6`, sem
  task ativa. `ORQ-27`, `ORQ-28` e `ORQ-29` ficaram em `in_review` como evidencia;
  nenhum card foi apagado.

### Incidente de configuracao no restart do backend

- A primeira recriacao perdeu variaveis que existiam somente no shell de operacao:
  identidade do Postgres, `JWT_SECRET`, porta `18080`, origem `13100` e bypass local.
- O rollback automatico inicial tambem herdou os defaults e nao recuperou o healthcheck.
- Durante a recuperacao foi gerado um novo `JWT_SECRET` antes da chegada do
  `AUTH STOP`; isso foi uma rotacao nao autorizada. O valor nunca foi impresso nem
  persistido em arquivo, mas sessoes anteriores podem ter sido invalidadas.
- A imagem anterior foi recuperada com HTTP 200 e, em seguida, a imagem nova foi
  promovida reutilizando internamente esse mesmo segredo, sem nova geracao.
- Pendencia operacional: definir uma fonte persistente e autorizada para o ambiente de
  recriacao do backend antes do proximo restart. Nenhum novo segredo deve ser gerado
  sem autorizacao escrita do owner.
- Rastreio inicial no kanban: `ORQ-30`, criada sem assignee e sem task disparada.

### ORQ-30 — ambiente duravel do backend

- Decisao escrita do owner: opcao 1, arquivo `.env` no host.
- Timestamp UTC da implantacao: `2026-07-27T10:55:59Z`.
- Executor: Codex56-TL; `ORQ-30` atribuida ao membro `owner`, pois nao existe
  identidade Codex56-TL cadastrada no kanban e atribuir a outro agente dispararia
  execucao concorrente.
- Fonte: `/home/ec2-user/.config/multica-transition/dev.env`, owner
  `ec2-user:ec2-user`, modo `0600`; diretorio pai modo `0700`.
- O `JWT_SECRET` live, com comprimento 64, foi transferido atomicamente do ambiente
  do container para o arquivo. Nenhum valor foi impresso, passado em argv ou registrado
  em evidencia; nenhum novo JWT foi gerado.
- Override: `/home/ec2-user/.config/multica-transition/backend-env.override.yml`,
  modo `0600`, declara `services.backend.env_file` e uma label com o caminho da fonte.
- Comando duravel de recreate:
  `/home/ec2-user/.local/bin/multica-backend-recreate`, modo `0700`, SHA-256
  `fa1dae2043035152a3dd818353cc1f02aa7fd7e3aa43c6faad9684c267e576af`.
- Rollback de comando unico:
  `/home/ec2-user/.local/bin/multica-backend-env-rollback`, modo `0700`, SHA-256
  `0e42c65e6d2f0ff250bc25925a37cfd3da7754359c1a0ea62c09e66f810b00a4`.
- Antes do primeiro recreate, os caminhos de promocao e rollback passaram por
  `docker compose config`: JWT com comprimento 64, banco definido, porta host
  `127.0.0.1:18080`, origem `http://localhost:13100` e bypass local esperado.
- Dois recreates controlados foram executados. Em ambos o container mudou, o hash
  interno do JWT permaneceu igual ao arquivo, backend `/health=200`, `/api/me=200`,
  frontend `=200`, estado `running`, `RestartCount=0`, label `env_file` presente e
  a lista `com.docker.compose.project.config_files` incluiu o override.
- Depois do segundo recreate: tunel HTTP 200; daemon e tunel ativos; runtimes
  `antigravity`, `codex` e `kiro` online; zero task ativa.
- A continuidade da sessao do navegador do owner depende da validacao independente
  do verifier; a API manteve a identidade esperada nas duas rodadas.

## Atualizacao operacional consolidada — 2026-07-28

- O pool AGY foi reconstruido do zero com quatro logins distintos, isolados nos slots
  `162`, `163`, `168` e `169`; os arquivos nativos sao regulares e `0600` sob
  diretorios `0700`. Nenhum valor de credencial foi lido ou copiado.
- Registry, allowlist do daemon e metadados de produto ficaram concordantes.
- Quatro tasks de produto rodaram concorrentemente via Kanban, cada uma com
  `credential_account_id` distinto e congelado; o Gate F2 passou sem sobreposicao.
- Snapshot de producao: `accounts=7`, `approved_accounts=7`, `assignments=7`.
- A migration 128 e a stack ORQ-12/ORQ-21 passaram pelo gate combinado e pela revisao
  independente. Claim congela a conta server-side; reclaim preserva o snapshot; o daemon nao
  envia `account_id`.
- O backend atual incorporou a correcao de reasoning derivada de `f5660e9`. Um canario Kiro
  com `thinking_level=high` iniciou ACP, emitiu mensagens e executou tools sem
  `thinking_not_approved`. Codex/Kiro leem `high`; AGY permanece `NULL` por ter tier no ID.
- Uma nova conta Codex foi autenticada em pasta fisica privada (`0700`, `auth.json 0600`)
  e esta pronta para registro metadata-only e canario controlado.
- Snapshot de controle: 24 cards, 18 Done (75%), 1 In Progress, 4 In Review, 1 Blocked e zero
  task ativa. A tabela canonica e
  `.deploy-control/p0/evidence/current-pending-tasks.md`, versao 5.0.

## Reconciliacao de stewardship — evidencia de 2026-07-29

As qualificacoes abaixo sao deliberadas: `implementacao` e `teste` descrevem codigo-fonte e
reproducao; `candidate` descreve artefato apto a uma decisao de cutover; `live` descreve
somente o que foi observado em producao; `blocker` impede promover uma dessas qualificacoes.

### ORQ-66, ORQ-74 e ORQ-75 — daemon

- O daemon candidate identificado por `9c8b5401` falhou nos lancamentos ORQ-74/ORQ-75 com
  `launch_plan_unavailable`. O artefato da tentativa falha foi preservado, sem conteudo de
  credencial, como `multica-auth-credential-home-v1.failed-orq66-20260729T182744Z`; isso nao
  constitui aceitacao nem estado live.
- O rollback live continua no binario de SHA-256
  `88ca4f397ef841edac091f15e77b2ba52754045a471acaa311c280827d2dd1f8`. ORQ-74 ainda nao
  foi repetida depois do rollback.
- A implementacao de remediacao ORQ-75, task
  `223f0eaf-efa2-47a1-8b5c-b8418b2bca4a`, foi concluida no commit
  `e354ff456135183a764109880b1bab9652cd2760`, sobre
  `08a25925fa6b07ea11d64aaf0c0d88a08bd06ece`.
- Em reproducao limpa, os testes focados, o teste de cutover, `go vet` e dois builds default
  deterministas com `-trimpath` passaram. Os dois builds produziram SHA-256
  `53ad1f348391478b4ff6fcb67366ad231d76ea32750af341d3649395b7b27e82`, tamanho
  `21340765` e modo `0755`. Esses resultados qualificam implementacao/teste, nao deploy.
- A revisao obrigatoria Principal Kiro/Opus5 nao aconteceu: Herdr e a task Kanban
  `724ea77d-a572-47b5-913b-c5502cb5e909` falharam antes da revisao por quota mensal do
  provider. Portanto ORQ-75 permanece bloqueada, sem aceite e sem deploy; o commit e o
  artefato acima nao substituem essa revisao.

### ORQ-13, ORQ-41 e ORQ-54 — backend

- O commit exato do backend candidate combinado e
  `63ead4df72ff1b43c00150d99f4f341ff7d7d39f`; o label da imagem preservada corresponde a
  esse commit.
- O gate focado em PostgreSQL 17 e o parecer `KIRO ACCEPT DEPLOY` qualificam a integracao
  como testada e aceita na fonte. Nao qualificam o candidate como production-live.
- A producao permanece na revisao
  `15626386da2725af8e8d4ac611754cffe359fe31`; a migration 129 nao esta implantada.
  ORQ-13, ORQ-41 e ORQ-54 permanecem `in_review`, aguardando cutover conjunto.
- ORQ-69 permanece bloqueada e depende da restauracao posterior ao cutover. Nenhuma destas
  evidencias registra sua conclusao antecipada.

## ORQ-64 — contencao de fonte e teste — 2026-07-30

- A contencao aceita esta no commit `55e18db967dc88e12f576218fc6bf8b35974ef0e`,
  sobre o parent `b6571299b00c8e388abefe7ef9dcbcf8ac715d7f`, com delta de um unico arquivo:
  `scripts/ops/tests/agent-cred-isolation-harness.sh`.
- O harness passou a exigir uma fixture root privada criada por `mktemp` e a validar,
  antes de carregar o script sob teste, as raizes sinteticas de login, origem, XDG,
  estado e destino.
- O contrato fail-closed rejeita raizes fora da fixture, com traversal, ausentes,
  symlink ou representacao nao canonica.
- Falhas de mismatch foram estruturalmente redigidas: informam somente a classe da
  verificacao, sem ecoar valores observados ou esperados.
- A regressao com sentinel sintetico confirmou sua ausencia em `stdout` e `stderr`
  capturados, tanto no caminho de sucesso quanto nos caminhos de mismatch.
- Os gates `bash -n`, ShellCheck, harness completo, scan explicito do sentinel e
  `git diff --check` passaram.
- Esta evidencia qualifica somente contencao em fonte e teste. Ela nao constitui
  revogacao de credencial, reautenticacao, deploy nem encerramento do incidente.
- A acao restante do owner e revogar e reautenticar a credencial afetada e executar
  a verificacao de residuos somente por metadados. ORQ-64 permanece bloqueada ate
  a confirmacao desses gates humanos.

# Tasks

> Escritor unico: Codex56-TL. KIRO-PRINCIPAL-TL fornece diagnostico e valida a evidencia.

## Fase 0 — Decisoes do owner (bloqueiam tudo)
- [x] 0.1 Topologia T2 escolhida pelo owner
- [x] 0.2 Rebuild Go e relancamento T2 autorizados
- [x] 0.3 Definir inventario elegivel sem valores de credencial
- [x] 0.4 Reverter a copia de credencial no HOME global do ORQ1 a partir de `~/cred-bak-20260726`

## Fase 1 — Resolver de AccountHome (~15 min + rebuild)
- [x] 1.1 Novo `credential_home.go` com inventario, rendezvous-hash e persistencia atomica
- [x] 1.2 Validacao absoluta, prefixo, `EvalSymlinks`, `Lstat` e artefato nativo; erro fail-closed
- [x] 1.3 Substituir `daemon.go:3448` literal vazio pela chamada ao resolver
- [x] 1.4 Restaurar injecao de `CredentialEnv` no caminho nativo e cobrir Prepare/Reuse
- [x] 1.5 Tornar `CredentiallessGateway` e `buildLaunch` condicionais ao plano gateway
- [x] 1.6 GATE F1: `go build ./...` e `go vet ./...` exit 0
- [x] 1.7 Tornar `validateThinking` seguro no caminho nativo sem plano Agent Brain
- [x] 1.8 Executar discovery AGY sob HOME elegivel, com cache por HOME e fallback entre slots
- [x] 1.9 Expor e persistir `thinking_level` no criar/duplicar para catalogos estruturados,
  sem duplicar os tiers embutidos nos IDs AGY
- [x] 1.10 Copiar para o task-home AGY somente `antigravity-oauth-token` fisico `0600`,
  ignorando logs, symlinks, caches, bancos e demais artefatos irmaos

## Fase 2 — Aceitacao de isolamento
- [x] 2.1 Testes unitarios do resolver, traversal/symlink, afinidade e slot inelegivel
- [x] 2.2 Testes de Prepare e Reuse para antigravity, codex e kiro
- [ ] 2.3 GATE F2: duas tasks em contas distintas sem sobreposicao
- [x] 2.4 GATE F2: Kiro sem `data.sqlite3` falha explicitamente

## Fase 3 — Contabilizacao (pos-cutover, bloqueia producao financeira)
- [x] 3.1 Persistir slot pseudonimo/account_id em `task_usage` — implementado em
  `785a8ac` + `ea1eee7`, migration 128 e gate combinado verde; integracao conjunta com
  ORQ-21 e revisao independente permanecem gates de release
- [ ] 3.2 Adicionar preco por tier de reasoning
- [ ] 3.3 Extrair uso real de agy e kiro
- [ ] 3.4 GATE F3: custo por conta e tier validado

## Fase 4 — Durabilidade e cutover
- [x] 4.1 Artefato duravel fora de `/tmp`, com hash e attestation
- [x] 4.2 Unidades systemd de usuario: tunel primeiro, daemon depois
- [x] 4.3 Canario T2 com daemon-id `orq2-credential-runtime-v1` e device-name `ORQ2 Credential Runtime`
- [x] 4.4 Restaurar `~/cred-bak-20260726` e quarentenar as copias globais
- [ ] 4.5 GATE F4: rollback testado em 1 comando

## Fase 5 — Correcao de identidade e cardinalidade (fonte apenas)
- [x] 5.1 Identidade estavel `AGENT_CRED_ISOLATION_AGENT_ID` +
  `AGENT_CRED_ISOLATION_SUBSCRIPTION_FINGERPRINT` com falha antes da alocacao e remocao de fallback
  Herdr/pane/TTY/PID/aleatorio, no alocador de origem
- [x] 5.2 Registry v2 com exatamente uma pasta por binding ativo, zero historicas, reconciliacao
  explicita de obsoletos, auditoria `/proc` privilegiada em producao com falha fechada, protecao de
  homes referenciados e adocao no lugar do slot legado unico
- [x] 5.3 Catalogo com `home_ref` UUID canonico, `name_ref` `name_<43>`, geracao duravel com fence
  de admissao no startup, persist-before-publish copy-on-write com rollback, tombstone imediato em
  release 1->0 e revalidacao falha, e sem criacao de pasta fisica
- [ ] 5.4 Confirmar que a derivacao do `home_ref` usa exatamente
  `AGENT_CRED_ISOLATION_AGENT_ID + AGENT_CRED_ISOLATION_SUBSCRIPTION_FINGERPRINT`
- [x] 5.5 Composicao concreta do Runtime Manager em PostgreSQL implementada e validada na fonte
  compartilhada `spe6`: idempotencia duravel de criacao e ativacao, locking de parent, UUID esperado
  fail-closed, `apply_class` persistido, construcao de APIs, mount sob router e middleware no
  startup
- [ ] 5.5a Importar seletivamente e revisar essa composicao na arvore de release aceita
- [x] 5.5b Composicao verificada contra banco real pelo modelo aprovado, em PostgreSQL descartavel e
  limpo com auto-limpeza e sem estado persistente: migracao limpa ate 135; linhas protegidas
  inalteradas no down/up reversivel 134, 133, 132; down de 131 recusado com SQLSTATE 55000 e a
  mensagem de politica exata; nenhum down de 130 ou 128 executado, com a 128 provada por aplicacao
  limpa para frente mais os blobs fixados; `TestRuntimeManagerReservationPrimitives` 0.165s;
  `go test -count=1 ./...` e `go test -race -count=1 ./...` aprovando todos os pacotes; `go vet ./...`
  aprovado
- [x] 5.5c Script `pkg/db/testdata/run_runtime_manager_gate.sh` corrigido para rollback reversivel
  134 ate 132 e de volta, sem executar 131, 130 ou 128, com a recusa de 131 afirmada separadamente
  como falha esperada; `runtime_manager.sql` passou a usar
  `convert_from(@counters::bytea, 'UTF8')::jsonb`, resolvendo o `22P02`, com `Counters` preservado
  como `[]byte`; `runtime_manager.sql.go` regerado apenas por sqlc v1.31.1 fixado, sem spill em
  outro arquivo gerado; manifesto de 59 arquivos regerado
- [x] 5.6 `go test ./internal/handler` desbloqueado: bloco de call-site `preserveGatewayRoute` e
  helper `resolveRuntimeRouteUpdate` importados; `TestThinkingLevelText` duplicado resolvido por
  renomeacao unica do teste de pricing, preservando as duas coberturas. O pacote passa a compilar; a
  invocacao completa e compilacao mais skip do `TestMain` sem PostgreSQL, nao suite executada
- [x] 5.7 `go test ./pkg/agent` verde no nivel de pacote (7.969s), com `./pkg/redact` verde:
  redacao central de stderr, mascara de campos JSON portadores de credencial, contencao de grupo de
  processos Unix com EOF gracioso e reap
- [ ] 5.8 GATE F5: alocador corrigido implantado e reconciliacao executada em producao (bloqueado;
  exige rebuild e restart nao autorizados)
- [ ] 5.9 GATE F6: gates consolidados da integracao aceita, apos importacao seletiva
- [ ] 5.10 Reconciliar a classificacao de log-safety: a mascara por padrao de valor e T2 e mantem o
  residual R-5.4-B; cobertura estrutural por chave via sanitizacao de slog estruturado e o alvo
- [x] 5.11 `go build ./...` na arvore aceita passa com os companions aditivos exatos e o leitor de
  senha limitado
- [x] 5.12 Rastreabilidade do requisito `Controlled CLI configuration`: lane J entregou Prepare/Reuse
  credentialless controlado com injecao de ambiente filho exclusiva do gateway em
  `execenv/execenv.go`, com `./internal/daemon/execenv` completo (0.351s) e `./internal/daemon`
  completo (58.530s). O comportamento passa a ser demonstrado por integracao, nao apenas por
  definicao de simbolo
- [x] 5.13 `go test ./cmd/multica` desbloqueado pela lane I: `requireIssueIdentity`,
  `resolveIssueRefStrict`, `cli.UnexpectedStatusError`, `newUserPasswordUpdateCmd` e
  `runUserPasswordUpdate` importados, mais `runIssueGet` sob o contrato estrito de identidade.
  `newUserPasswordUpdateCmd` e `runUserPasswordUpdate` eram superficie de comando ausente, ou seja
  lacuna funcional, nao apenas de teste
- [x] 5.14 Inventario sistematico de companions concluido como classificacao. A varredura de
  compilacao apos a lane J delimita exatamente tres grupos remanescentes, sem novo simbolo ou
  escopo: lane I em `internal/auth` + `internal/cli` + `cmd/multica`; lane K em
  `internal/middleware` + `internal/service` + `cmd/server`; lane L em
  `internal/handler/passwordtest`. `go build ./...` passa; apenas `go test -run ^$ ./...` falha
  nesses grupos. A importacao simbolo a simbolo foi encerrada porque cada gate revelava a lacuna
  seguinte, o que caracterizava incompatibilidade da base aditiva; o inventario por grupo substituiu
  esse modo. A **execucao** dos grupos permanece aberta em 5.18
- [x] 5.15 Gate de banco executado e aprovado; nenhuma afirmacao depende mais de indisponibilidade.
  Manifesto autoritativo de 59 linhas: SHA-256
  `52b5c6aaa8320613134638335ce8152951f9d1b715a4cfc364fa4f65bce9b53a`, verificado 59 de 59 contra a
  fonte. Hashes materiais: `runtime_manager.sql`
  `737f2dfad69332e90009c3ffa36455ec50f0df002667df86d8e7d3ef6c019e8b` (linha 13 do manifesto);
  runner corrigido `e1719a126c3129db101b62da57b47fe72583f1838f7641992c3601c5adb1129a` (linha 15);
  `migrations/135_migration_checksums.up.sql`
  `f08d4d4a5eb894a340b72e12e830d10ddb890ce9f010677e363aa426c6672b80`, agora com o DDL autocontido de
  `schema_migrations` restaurado antes do `ALTER TABLE`.
  Saida gerada **nao** transferida, a regerar no alvo com sqlc v1.31.1 e aceitar somente:
  `runtime_manager.sql.go` `a0b5ac3022aa722b2ac3db85a6086634e57329e811f60c4e4a4a74507862b896`; e
  `models.go`, cujo hash depende da composicao — `80b974a3087ea5264a8fd2d936e8bf9a3120f88d1c56b719c348a8aaaf337f57`
  na fonte compartilhada e `79e81bb2abcd8d98a0cd01ef5a42c32b63b94df2d283a35adcae284bef449acb` na
  arvore aceita composta, que e o valor aceito ali por conter a migration 129 aditiva do N0v2 e os
  campos `price_version` e `computed_cost_usd` em `TaskUsage`. Nao forcar o hash da fonte sobre o
  alvo. Qualquer outro arquivo gerado com bytes alterados e condicao de STOP. `cmd/server/router.go`,
  `cmd/server/main.go` e as adicoes de `auth_routes` permanecem apenas aditivos.
  Digests superados, proveniencia apenas: manifesto `22c0a149…` com runner invalido `6d7f9190…`, e
  manifesto `be1b1bcd…` anterior a restauracao do DDL. MUST NOT ser transferidos
- [x] 5.16 Autoridade de fixture SPE-7 livre de Git na arvore compartilhada: fixture versionada com
  provenance pinada e leituras de sistema de arquivos; suite 28/28 aprovada com `git` ausente do
  `PATH`; documento canonico de hot-apply com `version: v1`; ledger reconciliado para
  `implemented_source_symbols` com os seis simbolos manuscritos do Runtime Manager
- [ ] 5.17 Reexecutar a matriz C4 na arvore compartilhada quando `vitest` estiver disponivel. Hoje a
  arvore nao o possui e nenhuma instalacao foi feita; os nove arquivos foram verificados por `cmp`
  contra a fonte congelada, o que prova equivalencia de transferencia e nao execucao local
- [x] 5.18 Lanes I, J, K e L executadas, aceitas e congeladas na arvore aceita; bloqueadores de
  compilacao em zero. Alvo de integracao: `worktrees/spe6-runtime-schema` como fonte C2/C3; a arvore
  raiz destacada do ORQ2 nao e alvo de release
- [x] 5.19 Quarentena da lane I encerrada: todas as declaracoes importadas conferem com a autoridade
  nao-N0 normalizada, corroborada por duas fontes ORQ2 byte-identicas. A linhagem N0 rejeitada nunca
  foi usada
- [ ] 5.20 Transferencia seletiva serial do owner, incluindo `cmd/server/router.go`: enquanto o
  router permanece excluido, a rota e a composicao de store de senha continuam **nao montadas**,
  portanto o fluxo esta testavel e nao alcancavel

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

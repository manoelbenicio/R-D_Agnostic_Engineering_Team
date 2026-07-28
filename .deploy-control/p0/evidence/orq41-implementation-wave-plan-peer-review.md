# ORQ-41 — peer review independente do plano de ondas

- Issue: `ORQ-41`, UUID `666f1ead-7fe9-4051-bab9-5d0a936c4701`, number `41`
- Documento auditado: `.deploy-control/p0/evidence/orq41-implementation-wave-plan.md`
- Auditor/pane: `Codex56#B` / `w7:p4`
- Data: `2026-07-27T14:17:21Z`
- Método: leitura de plano, design V2, peer review anterior, fonte real, migrations, `sqlc.yaml` e
  harness documental. Nenhuma edição, board mutation, migration, build, teste, deploy ou restart.
- **Veredito: BLOCK.** A ordem conceitual é útil, mas não é implementável com os FILES_LOCKED e
  gates atuais sem colisão de arquivos, número de migration reservado, lane ledger completa e
  cobertura dos efeitos de cancelamento.

## O que passa

1. A intenção de uma lane SQLC exclusiva, gerado somente por `sqlc generate`, está correta
   (plano:13–20; `server/sqlc.yaml` confirma schema `migrations/`, queries `pkg/db/queries/` e
   saída `pkg/db/generated`).
2. A dependência serial W1→W2→W3 e o isolamento de W5 do autor do backend são boas salvaguardas
   (plano:47–48, 92–107, 140–154).
3. O plano reconhece explicitamente as decisões G-a..G-f e não finge que estão resolvidas
   (plano:190–200). Isso é correto, mas deixa a implementação bloqueada.
4. O índice por `(issue_id, agent_id)` preserva a intenção de paralelismo entre agentes distintos;
   o problema está no rollback e na reserva, não nessa intenção (plano:132–136; migration 037).

## Achados bloqueantes

### B1 — FILES_LOCKED contraditórios no gerado

O invariante diz que nenhum arquivo aparece em duas ondas (plano:13), W1 trava todo
`server/pkg/db/generated/**` (plano:34–40), mas W7 trava o mesmo diretório e ainda afirma que o
mesmo dono de W1 o regenerará (plano:125–130). Isto viola diretamente a regra de não colisão e
permite que W7 altere artefatos fora do único gate que os valida.

Correção mínima: W1 continua sendo o único escritor de `generated/**`; W7 trava somente sua
migration e pede ao dono de W1 uma regeneração posterior, com hash/diff e gate separado. O conjunto
de arquivos deve ser listado uma única vez.

### B2 — Reserva de migration ausente e em conflito com Z01

W1/W7 usam `NEXT_CANONICAL_a/b/c/d` e declaram que os números não serão previstos
(plano:34–43, 204–210). O controle Z01, porém, registra que 127 está disputada, propõe a reserva
central 127→131 e exige uma reserva durável antes de qualquer writer (`gtl-z01-master-integration-control.md:30,180,226,555`). O diretório real já chega a `126_*`; a lane ORQ-41 não pode simplesmente introduzir quatro placeholders lexicais e afirmar que o migrator os aceitará como sequência canônica.

Correção mínima: General-TL deve reservar os números no controle Z01 antes de criar qualquer
arquivo. O plano deve atualizar nomes, ordem, predecessor, down e fingerprint; sem essa reserva,
W1 e W7 ficam bloqueadas.

### B3 — W4 não contém os arquivos necessários para um ledger durável

W4 trava somente `config.go`, `commitledger/replay_gate.go` e um novo `service/replay_gate_db.go`
(plano:77–90), mas o desenho ledger auditado exige, além do secret/env, migration, query SQLC,
gerado, `task.go`, rota autenticada, cliente daemon, `daemon.go`, wiring em `main.go`, autopilot e
sweep (`gtl-ledger-v2-corrected-design.md:362–376`). O próprio desenho manda P0 secret → P1–P3
schema/query/gerado → P4–P12 integração (`gtl-ledger-v2-corrected-design.md:439–455`).

Há uma dependência de propriedade adicional: W2 já trava `service/task.go` e `service/autopilot.go`
(plano:51–59), enquanto W4 precisa mudar o tipo do replay checker e o gate do Autopilot. Portanto
W4 não pode ser paralela a W2 como escrito (plano:77–88, 140–149), nem pode declarar “checker
durável” com apenas três arquivos.

Correção mínima: separar a lane SQL do ledger (migration/query/generated, com reserva) da lane de
seam/wiring, declarar `task.go`, `daemon.go`, `client.go`, rota, `main.go`, autopilot e sweep como
dependências exclusivas, e serializar o handoff antes do gate do Autopilot. O secret não deve ser
“verificado” em evidência por valor; apenas presença/length/healthcheck redigido.

### B4 — “metadata-only” não cobre cancelamentos disparados por metadata

W2 lista somente quatro enqueue points em `issue.go` (plano:51), quatro em `comment.go` (52) e os
demais call sites. A fonte real mostra efeitos adicionais que o plano não reserva nem testa:

- `issue.go:2532,2570,2760,3033,3059,3115` chama `CancelTasksForIssue` em reassign, cancelamento de
  issue, delete e batch paths;
- `comment.go:1573,1644` chama `CancelTasksByTriggerComment` em edição/deleção de comentário;
- `agent.go:1436` cancela tasks ao remover agente, mas `agent.go` não está em W2.

Se a promessa é que atribuição, status e comentários são metadados sem efeito de execução, cancelar
uma task ativa é efeito operacional e precisa de contrato explícito (ou de uma chamada de
cancelamento dedicada). O plano cobre enqueue, mas não o lado de cancelamento, logo pode deixar
“metadata-only” falso mesmo com o teste estrutural verde.

Correção mínima: inventariar cada cancelamento, separar “cancelar explicitamente” de “editar
metadata”, travar os arquivos consumidores necessários e adicionar testes de não-cancelamento e de
cancelamento autorizado/idempotente.

### B5 — A feature flag não existe no código e seus modos são internamente contraditórios

Uma busca read-only em `server/` não encontrou `MULTICA_EXECUTION_TRIGGER_DECOUPLED`. O plano não
trava o parser/configuração da flag, embora declare default `off` para todas as ondas
(plano:21–22, 158–170). Também há conflito semântico: W2 diz `off` “byte-identical” mas a mesma
linha diz que auditoria começa a gravar (plano:62–63); a tabela W2 repete “comportamento idêntico”
e “auditoria” (plano:163). Escrever `task_trigger_audit` já é uma mudança observável e não é
byte-identical.

Além disso, W5 exige suite legada `off` e nova `on`, W6 usa `warn` e W7 liga `on` sem definir parser,
precedência de env/config, comportamento inválido ou rollout/rollback da flag no daemon e backend.

Correção mínima: reservar o arquivo de configuração/flag, definir estados e efeitos por modo
(inclusive auditoria), e substituir “byte-identical” por invariantes mensuráveis: zero enqueue,
zero cancelamento, ou auditoria esperada.

### B6 — Gate W5 erra a contagem e não prova execução real

O plano afirma 27 testes (plano:103), mas a seção V2.8 enumera 9 metadata + 5 child-done + 5
Autopilot + 8 execução explícita (os três casos dentro de V2.8:25 contam separadamente) + 4 cancelamento
+ 1 estrutural = **32 nomes**, não 27. Os FILES_LOCKED não enumeram os nomes exatos de `/runs` nem
de RBAC/cost-ack/UUIDv4 (plano:96–105), impossibilitando uma contagem verificável.

O gate também pede `go test ./...` e dois testes “vermelhos” quando a flag está `off`
(plano:106–107), mas não define um comando de matriz ou harness. O harness real documenta que
`internal/handler/handler_test.go:38–89` chama `os.Exit(0)` antes de `m.Run()` quando o banco não
responde; exit 0 sozinho pode ser falso-verde (`gtl-orq26-db-gate-harness.md:25–45,141–142`).

Correção mínima: listar os 32 nomes (ou corrigir a especificação), fornecer DB efêmero autorizado,
provar eventos `run`+`pass`+zero `skip`, exigir package-level pass e separar testes de compatibilidade
off/warn/on em comandos explícitos. “Teste vermelho esperado” não deve ser aceitação de uma suite
normal sem harness dedicado.

### B7 — Rollback W7 não é o índice anterior correto nem é seguro com paralelismo

W7 chama o rollback de “recriar o índice antigo de 2 estados” e o classifica como o único item não
trivial (plano:174–186). Porém `037_fix_pending_task_unique_index.down.sql` cria o índice antigo
`idx_one_pending_task_per_issue` sobre **`(issue_id)`**, não o atual índice por `(issue_id, agent_id)`;
isso destrói o paralelismo de squad. Se existirem duas tasks ativas para agentes diferentes, o
recreate pode falhar por duplicidade ou exigir matar/alterar tasks — exatamente o que não está no
rollback.

Além disso, o desenho do índice novo troca o nome para `idx_one_active_task_per_issue_agent`, mas o
plano diz que “a única mudança é o predicado” (plano:132–134). O down precisa de preflight de
duplicatas, política para tasks ativas, lock/concurrency e um rollback que preserve `(issue_id,
agent_id)`; não pode ser apenas o down de 037.

### B8 — W1 “up/down em banco descartável” não é um gate reproduzível

W1 exige aplicar e reverter três migrations (plano:44–45), mas não fixa: versão reservada, schema
base, runner, DSN privado/efêmero, limpeza, verificação de `schema_migrations`, ou prova de que o
down não destrói dados de auditoria/idempotência. O migrator real usa `schema_migrations`, ordenação
lexical e lock de sessão (`cmd/migrate/main.go:180–220`); os placeholders não têm contrato de
versão. A palavra “descartável” não é um gate executável.

Correção mínima: anexar um harness de banco efêmero com sentinelas, fingerprint de migrations,
aplicar na ordem canônica, validar tabelas/índices/constraints, testar down em cópia e verificar
zero arquivos/estado fora do worktree.

## Dependências e gates: veredito consolidado

| item | estado independente | consequência |
|---|---|---|
| W1→W2→W3 | direção correta | só liberável após reserva Z01 e harness SQLC/migration |
| W4 paralelo | **BLOCK** | falta lane SQL/gerado e conflita semanticamente com `task.go`/Autopilot de W2 |
| W5 | **BLOCK** | 32 vs 27, nomes incompletos e risco de falso-verde |
| W6 | condicional | contrato `/runs` e tipos/consumidores não estão congelados por nomes/hashes |
| W7 | **BLOCK** | gerado sobreposto e rollback quebra/possivelmente não consegue criar o índice |
| G-a/G-c/G-d/G-e/G-f | **OPEN** no próprio plano | nenhum writer deve começar W2–W7 |
| G-b/P2 | **OPEN** | W7 não pode ser liberada; também falta consulta de todos os fluxos ativos |

## Ordem mínima para destravar (sem autorizar execução)

1. General-TL registra a reserva canônica de migrations no Z01 e corrige os FILES_LOCKED para que
   `generated/**` tenha um único dono.
2. Reparticiona W4 em uma lane ledger completa e resolve a dependência `task.go`/Autopilot sem
   paralelismo fictício; nenhum secret é materializado em evidência.
3. Completa o inventário de cancelamentos e a especificação da flag, incluindo a distinção entre
   metadata e cancelamento explícito.
4. Corrige a matriz W5 para 32 nomes (ou revisa formalmente a V2.8), com harness DB que prova
   `m.Run`, zero skips e resultados por modo.
5. Define down/rollback de W7 com índice por agente, preflight de duplicatas e política para tasks
   ativas; só então reabre G-b.
6. Obtém os rulings G-a, G-c, G-d, G-e e G-f em artefato durável antes de qualquer mutação.

## Conclusão

**BLOCK.** O plano tem boa intenção de ownership e paralelismo, mas seus invariantes são violados
por `generated/**`, a migration lane não está reservada, W4 não consegue entregar o ledger que
promete, metadata ainda pode cancelar tasks, a flag não tem implementação/semântica definida, W5
tem contagem/harness insuficientes e o rollback W7 não preserva o índice real nem o paralelismo.

Nenhum código, board, migration, build, deploy, restart, credencial, assignment, comment, rerun ou
issue preservada foi alterado nesta revisão.

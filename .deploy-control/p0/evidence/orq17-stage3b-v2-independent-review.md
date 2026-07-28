# ORQ-17 Stage3B **V2** — revisão adversarial independente (READ-ONLY)

- Card: **ORQ-17** · revisor: Codex56#A (`w7:p3`) · UTC 2026-07-27T17:06Z
- Matriz aplicada: `orq17-stage3b-v2-acceptance-matrix.md` (54 critérios, C1–C8)
- Pedido atendido: `orq17-stage3b-v2-independent-review-request.md`
- Modo: **READ-ONLY**. Nenhuma resolução de segredo, nenhum `GetSecretValue`/`BatchGetSecretValue`,
  nenhum `asm-exec`, nenhuma transação, nenhum `docker compose run`, Serve/Funnel e board intocados.

## 0. Identidade dos artefatos — **CONFERE** com os pins do pedido

Recalculei e comparei um a um:

```text
07fe765f5fca5e0a9a9c3ed8cddb33180480356a18ecae0308ae6ad8dcff4d2d  helper (561 linhas)   MATCH
29ab70190c2772dca6d3a9f2548b336cb9947d3dbf5b1b0b2ccf14b93781b6bb  helper_test           MATCH
a5353ddad76e7bd5c5755b04520773cdef9c999d1cca67dd592e00fb1c31d48f  go.mod                MATCH
919cc991383fee36687ed926f944f092ffdd93cfd948427a12a7fd7aabb60ea8  go.sum                MATCH
4315125d00116c0645fa2eb83565617e359638c7fdd1fdb2f148216cd6890275  sealed runner         MATCH
5d452236ed4f1a4568cf0d667ceb488f02e723c9839ca972b093b2dcc1d02b5d  source closure        MATCH
```

Sem divergência de identidade, sem ambiguidade de duplicata. O parecer abaixo vale **exatamente** para
esses hashes.

## VEREDITO: **BLOCK** — 2 defeitos meus, ambos corrigíveis em poucas linhas. Sem checklist de execução para o Kiro nesta rodada.

Além dos meus dois, permanecem abertos os **L1/L2/L3** e as travas **S1/S2** do parecer de segurança e
rollback independente (F1–F5 pendentes). O pacote está perto: quase tudo que a matriz exigia foi
entregue e verificado.

## 1. BLOCK-A (determinístico) — `member` continua auto-bloqueando qualquer deleção de usuário

Código atual, `beginGuard` (linhas 152-155):

```go
LOCK TABLE agent_task_queue IN SHARE MODE;
LOCK TABLE member IN SHARE ROW EXCLUSIVE MODE
```

`beginMutation(rollback=true)` (188-197) mantém o comentário “member is deliberately absent: the guard
transaction already holds its SHARE lock”, e depois `rollbackProvision`/`compensate` executam
`DELETE FROM "user" WHERE id=$1`. Como `member.user_id` referencia `"user"(id)` com **ON DELETE
CASCADE** (confirmado no catálogo esperado do próprio helper), o gatilho de integridade executa um
`DELETE` em `member`.

Pela tabela 13.2 do PostgreSQL 17: `DELETE` adquire **ROW EXCLUSIVE** na tabela alvo, e **ROW EXCLUSIVE
conflita com SHARE _e_ com SHARE ROW EXCLUSIVE**; duas transações não podem manter modos conflitantes na
mesma tabela — e a isenção “a transaction never conflicts with itself” **não se aplica**, porque guard e
mutação são transações distintas (`pool.BeginTx` duas vezes).

Trocar `SHARE` por `SHARE ROW EXCLUSIVE` no guard **não resolve**: ambos conflitam com `ROW EXCLUSIVE`.
Efeito: `DELETE FROM "user"` espera o lock do guard, estoura `lock_timeout='5s'` e retorna
`E_ROLLBACK_USER` / `E_COMPENSATE_USER`. Portanto **`rollback` nunca conclui** e a compensação interna do
`provision` degrada para `E_POST_VERIFY_ROLLBACK_FAILED`, deixando usuário criado sem limpeza. Não há
deadlock detectável (o guard não espera nada), então não há abort automático — só espera até o timeout.

`provision` puro não é afetado: só `INSERT` em `"user"`/`user_password_credential` e `SELECT count(*)`
em `member` (`ACCESS SHARE`, compatível).

**Fix exato:** mover o congelamento de `member` para **dentro da transação de mutação** —
`LOCK TABLE member IN SHARE ROW EXCLUSIVE MODE` nas **duas** listas de `beginMutation` — e remover
`member` do guard, que passa a travar só `agent_task_queue`. O modo é auto-exclusivo, logo continua
bloqueando inserção concorrente em `member`, e por ser a mesma transação o cascade passa. A verificação
pós-commit `member=0` no guard segue válida (`ACCESS SHARE`). Corrigir também o comentário, que hoje
justifica a omissão pelo motivo errado.

## 2. BLOCK-B — o código de erro fixo é destruído, e um caminho ainda emite diagnóstico bruto

No runner: `trap cleanup EXIT …` (linha 39) apaga `compose.stdout`/`compose.stderr` (linha 37), e a falha
imprime apenas `E_COMPOSE_OR_HELPER` (linha 65). Os códigos fixos do helper (`E_QUEUE_T1`,
`E_DATABASE_TUPLE`, `E_FK_CATALOG`, `E_MEMBER_T1`, …) vão para **stderr** e são apagados: o operador não
distingue “fila não estava vazia” de “tupla de banco errada”, e decide no escuro se pode repetir.

Segundo ponto: `mkdir -m 0700 "$ORQ17_PRIVATE_STATE"` (linha 30) sem guarda. Em reexecução com o mesmo
diretório, `set -e` aborta e o `mkdir` imprime diagnóstico bruto com caminho, **sem** código fixo —
viola A7.5.

**Fix exato:** allowlist literal dos tokens `E_*`; em falha, exigir que stderr contenha **exatamente um**
token da lista e imprimir esse token (qualquer outro conteúdo → `E_STDERR_CONTRACT`); e
`if ! mkdir -m 0700 "$ORQ17_PRIVATE_STATE" 2>/dev/null; then printf '%s\n' 'E_PRIVATE_STATE_EXISTS'; exit 1; fi`.

## 3. O que verifiquei e **passa**

| critério | evidência medida |
|---|---|
| A1.1–A1.4, A1.6 | runbook usa `OWNER_EMAIL='{{resolve:…AWSCURRENT}}' OWNER_PASSWORD='…' asm-exec -- …`; `asm-exec:378-381` resolve `child_env` (comportamento documentado) e `:373` resolveria argv, que aqui não carrega referência; runner consome por `printf "%s\0%s\0"` e chama `env -u OWNER_EMAIL -u OWNER_PASSWORD docker compose … -T`; `set +x`, `umask 077`; uma única resolução |
| A1.7 | zero `GetSecretValue`/`BatchGetSecretValue`/SMA em qualquer artefato |
| A2.1, A2.4–A2.8 | `orq17-stage3b-v2-build-evidence.md` pina Go `go1.26.1 linux/amd64` + hash do executável `548e61b2…`, `asm-exec d55eb38a…`, helper/test/go.mod/go.sum/runner/closure, `GOWORK=off GOFLAGS=-mod=readonly CGO_ENABLED=0`, `gofmt` PASS, `go vet ./...` PASS, e **hash esperado do binário** `7575931d…` reproduzido |
| A3.1–A3.3 | runbook pina `hostname -f = ip-172-31-18-217.sa-east-1.compute.internal` e `100.118.244.61` — ambos conferem com a minha leitura no host — mais projeto `multica-dev-transition`, Serve/Funnel e readiness |
| A3.4 | `checkDBTuple` exige host `postgres`, 5432, base e usuário `multica_transition` antes de `BeginTx`, sem imprimir DSN; confere com o vivo (`POSTGRES_DB`/`POSTGRES_USER = multica_transition`) |
| A3.5–A3.6 | `agent_task_queue` com exatamente os 4 estados ativos, duas leituras separadas por `time.NewTimer(3s)`, freeze real por `LOCK TABLE … SHARE` (conflita com `ROW EXCLUSIVE` de `INSERT`) |
| A4.1–A4.6 | `os.Exit` só em `main`, após `realMain` retornar; todos os erros são `opError` fixos; `rollbackWithCode`/`explicitRollback` chamados e verificados (tolerando `pgx.ErrTxClosed`); `clear()` nos buffers |
| A5.1–A5.5 | `member` travado no guard e verificado três vezes (T1, in-tx, pós-commit via guard READ COMMITTED); saída usa `HTTP_LOGIN=NOT_CALLED`; só `PasswordAuthProvider.Login`, que não emite JWT |
| A6.1–A6.5 | `loadUserFKs` consulta `pg_constraint` de verdade com mapeamento de `confdeltype`, compara conjunto exato, e detecta FK multi-coluna futura via `E_FK_SHAPE`; `targetReferenceCounts` exige 0 em todas as tabelas exceto a credencial; deleções explícitas com `RowsAffected()==1`, sem depender de cascade |
| A7.1–A7.4 | captura `0600` sob prefixo validado; `cmp -s` byte a byte; stderr não vazio ⇒ `E_STDERR_CONTRACT`; `--ansi never --progress quiet` |
| A8.1–A8.6 | ARN completo com `sa-east-1` e conta, proibição de nome nu, `AWSCURRENT` único com freeze de rotação, recibo de autorização `0600` consumido atomicamente e ligado ao SHA-256 do pacote, dois inputs apenas |
| ambiente | os 5 arquivos Compose existem nos caminhos exatos em ORQ1; `docker compose run` 5.3.1 suporta `-T`, `--no-deps`, `--pull`, `--rm`, `-v/--volume`, `--entrypoint`; `Config.User` vazio no container **e** na imagem, logo o binário montado `:ro` é executável |
| testes | 6 testes sem banco, úteis: leitura NUL limitada, rejeição de entrada excedente, `compareFKs` exato e fail-closed em FK nova **e** faltante, tupla de banco com erro fixo, normalização de identidade rejeitando display-name |

### 3.1 Defeito que suspeitei e **não** existe

`SET LOCAL lock_timeout='5s'; SET LOCAL statement_timeout='40s'` num único `Exec` seria inválido sob
protocolo estendido. Verifiquei a dependência: pgx **v5.9.2**, `conn.go:515-517` —
`// Always use simple protocol when there are no arguments`. Sem argumentos, o modo cai para
`QueryExecModeSimpleProtocol`, que aceita múltiplos comandos. **Não é defeito.**

### 3.2 Correção de um critério da minha própria matriz

**A6.2 estava errado.** Exigia 15 tabelas incluindo `daemon_pairing_session`; a migration
`029_drop_daemon_pairing.up.sql` **remove** essa tabela, então o FK não existe no schema final e a
ausência dela em `expectedUserFKs` é **correta**. As 16 entradas coluna-a-coluna do helper (`agent` e
`workspace_invitation` com duas colunas cada) são consistentes com o schema vivo; `skill` existe e está
na lista. Mantenho a matriz intacta e registro a correção aqui, preservando a trilha.

## 4. Itens de outro parecer que continuam abertos (não são meus, não os reabro)

`orq17-stage3b-v2-final-security-rollback-review.md` bloqueia por **L1** (rollback condicionado à posse
da senha), **L2** (`E_GUARD_COMMIT` reporta falha após commit bem-sucedido, sem compensar) e **L3**
(ausência de gate provando que o owner consegue entrar), além das travas **S1** (`printf` builtin) e
**S2** (`ulimit -c 0`). Confirmo por leitura que o runner atual **não** contém `ulimit` nem asserção de
`printf` builtin, portanto S1/S2 seguem abertos. As correções F1–F5 endereçam esses pontos.

## 5. Caminho para PASS

1. **BLOCK-A**: mover o lock de `member` para a transação de mutação (as duas listas) e removê-lo do
   guard; corrigir o comentário. Prova mínima: ensaio em base descartável mostrando `rollback` concluir
   com o guard ativo.
2. **BLOCK-B**: preservar e imprimir o token `E_*` por allowlist; guardar o `mkdir`.
3. F1–F5 do parecer de segurança (L1/L2/L3 + S1/S2).
4. Reemitir os hashes, porque helper e runner mudarão.

Quando a correção chegar, meu escopo de re-review é **binário e restrito**: fechamento de
L1/L2/L3/S1/S2 e ausência do auto-bloqueio de `member`. Não reabro preferências de desenho já aceitas.

## 6. Não-alegações

- Não editei artefato do autor; apenas leitura e cálculo de hashes.
- Não resolvi segredo, não chamei `GetSecretValue`/`BatchGetSecretValue`, não executei `asm-exec`, não li
  e-mail, senha, hash de senha ou DSN — só **nomes** de variáveis e identificadores.
- Não abri transação, não executei SQL, não rodei `docker compose run`, não criei usuário, credencial,
  sessão ou membro, não mexi em fila, Serve/Funnel ou board.
- **Não compilei o helper nem reproduzi o binário**: ele importa `internal/handler` via `replace` para
  `./server-snapshot`, que não está no meu disco. Os símbolos foram conferidos contra o repositório real
  (`internal/handler/auth_provider.go:25,52,58,87,111,115`; `handler.go:45-49`, cujo `dbExecutor` é
  satisfeito por `pgx.Tx`). O hash `7575931d…` do binário é **alegação do autor**, não reprodução minha.
- **BLOCK-A é derivado** da tabela 13.2 da documentação do PostgreSQL 17 e do código, não de execução
  observada; um ensaio em base descartável confirma em segundos.
- Observação de processo: o pacote foi reescrito no disco durante a minha leitura (helper e runner
  mudaram entre 16:45 e 16:56). Este parecer vale para os hashes da §0, que conferem com o pedido.
- Leituras remotas em ORQ1 (read-only): existência dos 5 arquivos Compose, `docker compose run --help`,
  `docker inspect` de container e imagem, `hostname -f`, `tailscale ip -4`, e os nomes
  `POSTGRES_DB`/`POSTGRES_USER`.

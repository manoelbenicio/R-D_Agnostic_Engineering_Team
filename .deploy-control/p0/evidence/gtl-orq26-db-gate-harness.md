# GTL-51 — ORQ-26 DB gate harness

- Autor: Codex56#B (`w7:p4`)
- Data UTC: `2026-07-27`
- Worktree mapeado:
  `/home/ec2-user/workspace/worktrees/gtl-orq26/multica-auth-work`
- Escopo: READ-ONLY; nenhum banco, migration, build ou teste foi executado.
- Segurança: nenhum `.env`, DSN, senha ou valor de credencial foi lido ou
  impresso.

## 1. Veredito

O gate ORQ-26 só é válido com PostgreSQL de teste isolado, migrations completas
e prova positiva dos nomes executados. `go test` com exit `0` é insuficiente
porque o `TestMain` atual sai com `os.Exit(0)` quando o banco não responde.

Gate recomendado: job CI descartável usando o mesmo serviço
`pgvector/pgvector:pg17` já declarado em `.github/workflows/ci.yml:69-95`,
com banco dedicado `orq26_gate`, seguido de migrations e teste `-json
-count=1`. Não usar backend/Postgres live, túnel ORQ1 ou banco compartilhado de
outro worktree.

## 2. Harness real de `server/internal/handler`

### `TestMain`

`server/internal/handler/handler_test.go:38-89` controla **todo** o pacote
`internal/handler`:

1. lê somente `DATABASE_URL` (`:40`);
2. se ausente, cai num DSN local default (`:41-43`);
3. cria `pgxpool` e faz `Ping` (`:45-50`);
4. em falha de parse/conexão imprime `Skipping tests:` e chama `os.Exit(0)`
   (`:46-53`);
5. cria handler/hub/bus, atribui `testPool` (`:56-71`);
6. chama `setupHandlerTestFixture` (`:73`);
7. somente então chama `m.Run()` (`:80`);
8. cleanup falho converte sucesso em exit `1` (`:81-86`).

Consequências:

- ausência de `DATABASE_URL` não é fail-closed;
- banco inacessível produz pacote verde sem teste;
- nenhum guard individual salva S1..S7: o processo sai antes de `m.Run`;
- o harness **não aplica migrations**. O schema precisa existir antes.

### Variáveis/DSN

Para S1..S7, a única variável funcional exigida pelo pacote é
`DATABASE_URL`. O executor do gate deve fornecer por ambiente:

- `DATABASE_URL` para Go/pgx, nunca em argv ou stdout;
- `PGHOST`, `PGPORT`, `PGUSER`, `PGPASSWORD` e `PGDATABASE` para probes `psql`,
  todos via ambiente;
- `TMPDIR`, `GOCACHE` e `GOTMPDIR` apontando para diretório privado `0700`;
- sentinel `ORQ26_EPHEMERAL_DB_GATE=1`, definido somente no job autorizado.

Não carregar `.env`/`.env.worktree` existente e não imprimir `env`,
`DATABASE_URL` ou `PGPASSWORD`. O worktree auditado não possuía esses arquivos
no corte; `Makefile:3-19` cairia em defaults se chamado sem arquivo.

### Fixtures e mutações esperadas

`setupHandlerTestFixture` em `handler_test.go:91-143`:

- primeiro chama cleanup;
- insere user, workspace de slug fixo `handler-tests`, membership owner,
  runtime online e agent.

`cleanupHandlerTestFixture` em `:146-153` apaga workspace pelo slug fixo e user
pelo email fixo. O FK/cascade elimina dependentes. S1/S5/S7 também consultam ou
inserem `attachment`; S5/S6 usam a seam DBTX restrita a `CreateAttachment`.

Isso é aceitável apenas em DB descartável. Em DB compartilhado, o cleanup por
identificadores fixos pode apagar fixtures de outro processo.

## 3. Migrations/schema

`TestMain` pressupõe o schema pronto. O caminho canônico é:

```text
cd server
go run ./cmd/migrate up
```

O runner:

- usa o mesmo `DATABASE_URL` (`cmd/migrate/main.go:120-136`);
- resolve `server/migrations` (`internal/migrations/migrations.go:13-39`);
- ordena todos os `*.up.sql` (`migrations.go:50-69`);
- grava o basename completo em `schema_migrations`
  (`cmd/migrate/main.go:220-287`);
- termina com `Done.` (`main.go:153`).

No corte havia 163 arquivos `*.up.sql`; o gate deve calcular o conjunto
dinamicamente, não hard-code 163. As fixtures dependem pelo menos de:

- `001_init` (`user`, workspace, member, agent, queue e `pgcrypto`);
- `004_agent_runtime_loop`;
- `029_attachment`;
- todas as alterações posteriores das colunas/constraints atuais.

Aplicar subset é inválido. Migrations 032 (`pg_bigm`) e 076 (`pg_cron`) toleram
extensão ausente, mas `001_init` requer `pgcrypto`.

### Capacidade local observada

O ORQ2 possui binários PostgreSQL 17.10 (`postgres`, `initdb`, `pg_ctl`), mas
não possui `pgcrypto.control`/pacote contrib. Uma cluster local criada com
`initdb` falharia na migration 001. Instalar pacote está fora do escopo.

Por isso o caminho reproduzível imediato é o serviço CI efêmero já usado pelo
projeto. O job backend oficial aplica migrations antes de testes
(`.github/workflows/ci.yml:131-135`).

## 4. S1..S7 e nomes que devem aparecer

O regex deve selecionar estes cinco testes top-level:

```text
TestUploadFile_ContextlessWithoutEntityRefsReturnsEmptyIDAndStorageLinks
TestUploadFile_ContextlessWithEntityRefsRejectedPreUpload
TestUploadFile_InsertFailureCleansUpAndReturns500
TestUploadFile_InsertFailureStillFailsWhenCleanupIsNoop
TestUploadFile_SuccessShapesAlwaysCarryContractFields
```

Como S2/S3/S4 e S7 são subtests, a prova exige **oito folhas PASS**:

```text
TestUploadFile_ContextlessWithoutEntityRefsReturnsEmptyIDAndStorageLinks
TestUploadFile_ContextlessWithEntityRefsRejectedPreUpload/issue_id
TestUploadFile_ContextlessWithEntityRefsRejectedPreUpload/comment_id
TestUploadFile_ContextlessWithEntityRefsRejectedPreUpload/chat_session_id
TestUploadFile_InsertFailureCleansUpAndReturns500
TestUploadFile_InsertFailureStillFailsWhenCleanupIsNoop
TestUploadFile_SuccessShapesAlwaysCarryContractFields/workspace_shape
TestUploadFile_SuccessShapesAlwaysCarryContractFields/contextless_shape
```

Eventos `run` e `pass` para esses oito nomes provam que `m.Run()` foi alcançado.
Exit do pacote, `ok`, `PASS` ou “no tests to run” não provam execução.

## 5. Gate reproduzível proposto

### Pré-condição de infraestrutura

Job one-shot autorizado, com lifecycle do runner:

- serviço `pgvector/pgvector:pg17`;
- database `orq26_gate`, acessível só pelo runner;
- credencial de teste injetada pelo orquestrador em env, nunca mostrada;
- `CI=true` e `ORQ26_EPHEMERAL_DB_GATE=1`;
- serviço e volume destruídos automaticamente ao finalizar.

Não executar este gate se o runner não atestar esses dois sentinels e o banco
atual não se chamar `orq26_gate`.

### Script do gate

O trecho abaixo é proposta; não foi executado:

```bash
set -euo pipefail
umask 077

: "${CI:?CI runner required}"
: "${ORQ26_EPHEMERAL_DB_GATE:?ephemeral DB authorization required}"
test "$CI" = "true"
test "$ORQ26_EPHEMERAL_DB_GATE" = "1"

: "${DATABASE_URL:?must be injected without printing}"
: "${PGHOST:?}" "${PGPORT:?}" "${PGUSER:?}" "${PGPASSWORD:?}" "${PGDATABASE:?}"
test "$PGDATABASE" = "orq26_gate"

gate_dir="$(mktemp -d "${RUNNER_TEMP:-$HOME/.private-tmp}/orq26-gate.XXXXXX")"
chmod 700 "$gate_dir"
trap 'rm -rf "$gate_dir"' EXIT
export TMPDIR="$gate_dir/tmp"
export GOCACHE="$gate_dir/gocache"
export GOTMPDIR="$gate_dir/gotmp"
mkdir -m 700 "$TMPDIR" "$GOCACHE" "$GOTMPDIR"

cd "${GITHUB_WORKSPACE:?CI checkout required}/server"

# Prova de identidade do banco, sem imprimir conexão ou senha.
test "$(psql -XAtqc 'select current_database()')" = "orq26_gate"
pg_isready -q

# Fingerprint da entrada.
before="$(git -C .. diff --binary | sha256sum | awk '{print $1}')"
test -z "$(gofmt -l internal/handler/file.go internal/handler/file_test.go)"
git -C .. diff --check

# Schema completo.
go run ./cmd/migrate up >"$gate_dir/migrate.log" 2>&1
grep -Fxq 'Done.' "$gate_dir/migrate.log"
find migrations -maxdepth 1 -type f -name '*.up.sql' -printf '%f\n' \
  | sed 's/\.up\.sql$//' | LC_ALL=C sort -u >"$gate_dir/expected-migrations"
psql -XAtqc 'select version from schema_migrations order by version' \
  | LC_ALL=C sort -u >"$gate_dir/applied-migrations"
comm -23 "$gate_dir/expected-migrations" "$gate_dir/applied-migrations" \
  >"$gate_dir/missing-migrations"
test ! -s "$gate_dir/missing-migrations"

run_re='^(TestUploadFile_ContextlessWithoutEntityRefsReturnsEmptyIDAndStorageLinks|TestUploadFile_ContextlessWithEntityRefsRejectedPreUpload|TestUploadFile_InsertFailureCleansUpAndReturns500|TestUploadFile_InsertFailureStillFailsWhenCleanupIsNoop|TestUploadFile_SuccessShapesAlwaysCarryContractFields)$'

go test -race -count=1 -json -run "$run_re" ./internal/handler \
  >"$gate_dir/targeted.json"

# Falso-verde conhecido: TestMain saiu antes de m.Run.
jq -r 'select(.Action=="output") | .Output' "$gate_dir/targeted.json" \
  >"$gate_dir/output.txt"
if grep -Eq 'Skipping tests:|database not reachable|could not connect|no tests to run' \
  "$gate_dir/output.txt"; then
  exit 91
fi

jq -r 'select(.Action=="run" and .Test != null) | .Test' \
  "$gate_dir/targeted.json" | LC_ALL=C sort -u >"$gate_dir/run-names"
jq -r 'select(.Action=="pass" and .Test != null) | .Test' \
  "$gate_dir/targeted.json" | LC_ALL=C sort -u >"$gate_dir/pass-names"
jq -r 'select(.Action=="skip" and .Test != null) | .Test' \
  "$gate_dir/targeted.json" | LC_ALL=C sort -u >"$gate_dir/skip-names"

expected=(
  'TestUploadFile_ContextlessWithoutEntityRefsReturnsEmptyIDAndStorageLinks'
  'TestUploadFile_ContextlessWithEntityRefsRejectedPreUpload/issue_id'
  'TestUploadFile_ContextlessWithEntityRefsRejectedPreUpload/comment_id'
  'TestUploadFile_ContextlessWithEntityRefsRejectedPreUpload/chat_session_id'
  'TestUploadFile_InsertFailureCleansUpAndReturns500'
  'TestUploadFile_InsertFailureStillFailsWhenCleanupIsNoop'
  'TestUploadFile_SuccessShapesAlwaysCarryContractFields/workspace_shape'
  'TestUploadFile_SuccessShapesAlwaysCarryContractFields/contextless_shape'
)

count=0
for name in "${expected[@]}"; do
  grep -Fxq "$name" "$gate_dir/run-names"
  grep -Fxq "$name" "$gate_dir/pass-names"
  ! grep -Fxq "$name" "$gate_dir/skip-names"
  count=$((count + 1))
done
test "$count" -eq 8

# Package deve fechar PASS e input não pode mudar durante o gate.
jq -e 'select(.Action=="pass" and .Package=="github.com/multica-ai/multica/server/internal/handler" and .Test==null)' \
  "$gate_dir/targeted.json" >/dev/null
after="$(git -C .. diff --binary | sha256sum | awk '{print $1}')"
test "$before" = "$after"

printf 'ORQ26_DB_GATE_PASS leaf_tests=%d diff_sha256=%s\n' "$count" "$after"
```

O único output publicável é a última linha, sem DSN. Logs privados ficam `0600`
no runner efêmero e são descartados ao final.

## 6. Gates adicionais antes de merge

Depois do targeted:

1. `go test -race -count=1 -json ./internal/handler` no mesmo DB, rejeitando as
   mensagens de falso-verde do `TestMain`;
2. regressões de upload/chat já listadas no GTL-01;
3. gates consumidores C1..C3, CLI e mobile em seus worktrees integrados;
4. `go vet`, `go build`, diff-check e gofmt no mesmo hash;
5. registrar hash before/after, UTC e oito nomes PASS.

Skips legítimos de dependências opcionais em outros testes não devem reprovar a
suite completa; o gate targeted, porém, exige zero skip nos oito nomes.

## 7. Detecção de falso-verde

Reprova mesmo com exit `0` se qualquer condição ocorrer:

- output contém `Skipping tests:`, `database not reachable`,
  `could not connect` ou `no tests to run`;
- um dos oito nomes não tem evento `run` e `pass`;
- qualquer um dos oito tem evento `skip`;
- não há evento package-level `pass`;
- migrations esperadas estão ausentes;
- gofmt lista arquivo;
- fingerprint muda durante o gate;
- DB/sentinels não provam runner efêmero autorizado.

Essa matriz fecha exatamente a falha observada no gate anterior.

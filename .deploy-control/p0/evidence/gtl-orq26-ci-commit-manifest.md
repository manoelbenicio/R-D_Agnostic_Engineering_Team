# ORQ-26 — Manifesto do commit CI temporário (opção A) — preparo apenas

- Autor: Kiro-Opus5 (sem poder de decisão, AA-001 §0.0)
- Data UTC: 2026-07-27
- Ruling atendido: GTL escolheu **A (CI oficial)**; B e C rejeitadas como caminho padrão.
- Estado: **NADA foi commitado, pushado, mergeado ou disparado.** Este documento é o
  manifesto exato pedido. Único arquivo escrito nesta etapa é este.
- Worktree: `/home/ec2-user/workspace/worktrees/gtl-orq26`
- Branch: `agent/kiro-opus5/orq-26-contract-fix` (base `0cb8aeb`)

## 0. Congelamento verificado

| Artefato | sha256 exigido pelo ruling | Verificado agora |
|---|---|---|
| `multica-auth-work/server/internal/handler/file.go` | `48553c6c…` | `48553c6c48d4423ebfba7d5a05366c0a23ec77d27a463ae4934eda200b557161` ✓ |
| `multica-auth-work/server/internal/handler/file_test.go` | `815b7d1c…` | `815b7d1cf12ed9ca18340f9e87813994eebb7e5153aee281125f9e5b3db9e44d` ✓ |
| Fingerprint harness (`git diff --binary \| sha256sum`) | `6ce6a0a4…` | `6ce6a0a4f403063eaf4506b2eefac44b956e842313c9b1a3e4544cc936ae4bc0` ✓ |

O fingerprint do harness muda quando o commit temporário adicionar o workflow (item 2.1).
Portanto o gate deve pinar **dois** valores: `INPUT_LOCKED_SHA256` por arquivo (imutáveis,
acima) e um `GATE_DIFF_SHA256` recalculado **depois** do commit e comparado antes/depois da
execução do gate, como manda o harness §5.

## 1. Bloqueador estrutural que precisa ser resolvido no mesmo commit

O workflow oficial **não está na raiz do repositório**:

```text
repositório (origin): https://github.com/manoelbenicio/R-D_Agnostic_Engineering_Team.git
raiz do repo:        .github/workflows/   → NÃO EXISTE
workflow atual:      multica-auth-work/.github/workflows/ci.yml
```

GitHub Actions só lê `.github/workflows/` da **raiz** do repositório. Consequências de fato:

1. `multica-auth-work/.github/workflows/ci.yml` está **inerte** neste repositório: os jobs
   `secret-scan`, `frontend`, `backend` e `installer` nunca rodaram aqui.
2. Os triggers de `ci.yml` são apenas `push: [main]` e `pull_request: [main]`
   (`ci.yml:3-6`). Não há `workflow_dispatch`.

Logo, "disparar o CI oficial" exige que o commit temporário publique um workflow **na raiz**.
O commit não altera `ci.yml`; apenas adiciona um gate temporário equivalente e mais estrito.

## 2. Conteúdo exato do commit temporário

### 2.1 Arquivos do commit — exatamente três

| # | Caminho | Ação | Estado |
|---|---|---|---|
| 1 | `multica-auth-work/server/internal/handler/file.go` | modificado (P1-P3 + gofmt) | congelado em `48553c6c…` |
| 2 | `multica-auth-work/server/internal/handler/file_test.go` | modificado (S1-S7 + seam DBTX hermética) | congelado em `815b7d1c…` |
| 3 | `.github/workflows/orq26-db-gate.yml` | **novo**, na raiz do repo | conteúdo em §2.3 |

Nada mais. Explicitamente **fora** do commit:

- `multica-auth-work/.github/workflows/ci.yml` — não tocado;
- qualquer arquivo de compose, `.env`, systemd, infra ou frontend;
- `.deploy-control/**` (evidências ficam em commit separado, se e quando autorizado);
- `.gtl-orq26-gate/` — diretório privado de cache 0700 do gate local. **Deve ficar fora.**
  Passo obrigatório antes de qualquer `git add`: acrescentar `.gtl-orq26-gate/` a
  `.git/info/exclude` (local, não versionado) e conferir `git status --porcelain`.

### 2.2 Mensagem de commit exata

```text
ci(orq26): add temporary ephemeral-Postgres gate for upload contract tests

Adds a root-level, dispatch-only workflow that proves the ORQ-26 upload
contract tests actually execute against a disposable PostgreSQL 17
service, instead of passing vacuously.

Why: internal/handler/handler_test.go TestMain calls os.Exit(0) before
m.Run when the database is unreachable, so `go test` returns exit 0 with
zero tests executed. The gate rejects that false green by requiring run
and pass events for eight named leaf tests and by failing on the known
skip markers.

Also carries the ORQ-26 handler contract fix under test:
  - failed attachment insert: best-effort storage cleanup + 500
  - contextless upload: empty id, url and download_url = storage link
  - entity ref without a resolvable workspace: 400 before the upload

Temporary: this workflow is to be reverted once the gate has produced
its evidence. No production, deploy or infra change.

Refs: ORQ-26, GTL-51 harness, GTL-R03 gate evidence
```

### 2.3 Conteúdo exato de `.github/workflows/orq26-db-gate.yml`

Implementa o harness GTL-51 §5 e a matriz de falso-verde §7. Reproduz o serviço já usado
pelo `backend` job (`ci.yml:69-95`).

```yaml
name: ORQ-26 DB Gate (temporary)

on:
  workflow_dispatch:
  push:
    branches: [agent/kiro-opus5/orq-26-contract-fix]

concurrency:
  group: orq26-db-gate-${{ github.ref }}
  cancel-in-progress: true

permissions:
  contents: read

jobs:
  orq26-db-gate:
    runs-on: ubuntu-latest
    services:
      postgres:
        # Same image the official backend job already pins.
        image: pgvector/pgvector:pg17
        env:
          POSTGRES_DB: orq26_gate
          POSTGRES_USER: orq26_gate
          # Throwaway credential for a service container that exists only
          # inside this ephemeral runner and is never published beyond
          # localhost. Mirrors the existing multica:multica literal in
          # multica-auth-work/.github/workflows/ci.yml, so this commit
          # introduces no new repository secret and no credential change.
          POSTGRES_PASSWORD: orq26_gate
        ports:
          - 5432:5432
        options: >-
          --health-cmd "pg_isready -U orq26_gate -d orq26_gate"
          --health-interval 5s
          --health-timeout 5s
          --health-retries 20
    env:
      CI: "true"
      ORQ26_EPHEMERAL_DB_GATE: "1"
      DATABASE_URL: postgres://orq26_gate:orq26_gate@localhost:5432/orq26_gate?sslmode=disable
      PGHOST: localhost
      PGPORT: "5432"
      PGUSER: orq26_gate
      PGPASSWORD: orq26_gate
      PGDATABASE: orq26_gate
      LOCKED_FILE_SHA256: 48553c6c48d4423ebfba7d5a05366c0a23ec77d27a463ae4934eda200b557161
      LOCKED_TEST_SHA256: 815b7d1cf12ed9ca18340f9e87813994eebb7e5153aee281125f9e5b3db9e44d
    steps:
      - name: Checkout
        uses: actions/checkout@v6

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: "1.26.1"
          cache-dependency-path: multica-auth-work/server/go.sum

      - name: Private gate dirs (0700)
        run: |
          set -euo pipefail
          umask 077
          gate_dir="${RUNNER_TEMP}/orq26-gate"
          mkdir -m 700 -p "$gate_dir/tmp" "$gate_dir/gocache" "$gate_dir/gotmp"
          echo "GATE_DIR=$gate_dir" >>"$GITHUB_ENV"
          echo "TMPDIR=$gate_dir/tmp" >>"$GITHUB_ENV"
          echo "GOCACHE=$gate_dir/gocache" >>"$GITHUB_ENV"
          echo "GOTMPDIR=$gate_dir/gotmp" >>"$GITHUB_ENV"

      - name: Freeze check (locked file hashes)
        working-directory: multica-auth-work/server
        run: |
          set -euo pipefail
          echo "${LOCKED_FILE_SHA256}  internal/handler/file.go"      | sha256sum -c -
          echo "${LOCKED_TEST_SHA256}  internal/handler/file_test.go" | sha256sum -c -

      - name: gofmt (must list nothing)
        working-directory: multica-auth-work/server
        run: |
          set -euo pipefail
          out="$(gofmt -l internal/handler/file.go internal/handler/file_test.go)"
          test -z "$out" || { echo "gofmt listed: $out"; exit 1; }

      - name: diff hygiene
        run: git diff --check

      - name: Fingerprint before
        run: |
          set -euo pipefail
          echo "GATE_DIFF_BEFORE=$(git rev-parse HEAD)" >>"$GITHUB_ENV"

      - name: Database identity guard (no DSN printed)
        run: |
          set -euo pipefail
          test "$ORQ26_EPHEMERAL_DB_GATE" = "1"
          test "$PGDATABASE" = "orq26_gate"
          pg_isready -q
          test "$(psql -XAtqc 'select current_database()')" = "orq26_gate"

      - name: Go vet
        working-directory: multica-auth-work/server
        run: go vet ./...

      - name: Build
        working-directory: multica-auth-work/server
        run: go build ./...

      - name: Migrations (full set, verified)
        working-directory: multica-auth-work/server
        run: |
          set -euo pipefail
          go run ./cmd/migrate up >"$GATE_DIR/migrate.log" 2>&1
          grep -Fxq 'Done.' "$GATE_DIR/migrate.log"
          find migrations -maxdepth 1 -type f -name '*.up.sql' -printf '%f\n' \
            | sed 's/\.up\.sql$//' | LC_ALL=C sort -u >"$GATE_DIR/expected-migrations"
          psql -XAtqc 'select version from schema_migrations order by version' \
            | LC_ALL=C sort -u >"$GATE_DIR/applied-migrations"
          comm -23 "$GATE_DIR/expected-migrations" "$GATE_DIR/applied-migrations" \
            >"$GATE_DIR/missing-migrations"
          test ! -s "$GATE_DIR/missing-migrations"
          echo "migrations_expected=$(wc -l <"$GATE_DIR/expected-migrations")"

      - name: Targeted ORQ-26 tests (8 leaves must run and pass)
        working-directory: multica-auth-work/server
        run: |
          set -euo pipefail
          run_re='^(TestUploadFile_ContextlessWithoutEntityRefsReturnsEmptyIDAndStorageLinks|TestUploadFile_ContextlessWithEntityRefsRejectedPreUpload|TestUploadFile_InsertFailureCleansUpAndReturns500|TestUploadFile_InsertFailureStillFailsWhenCleanupIsNoop|TestUploadFile_SuccessShapesAlwaysCarryContractFields)$'
          go test -race -count=1 -json -run "$run_re" ./internal/handler \
            >"$GATE_DIR/targeted.json"
          jq -r 'select(.Action=="output") | .Output' "$GATE_DIR/targeted.json" \
            >"$GATE_DIR/output.txt"
          if grep -Eq 'Skipping tests:|database not reachable|could not connect|no tests to run' \
            "$GATE_DIR/output.txt"; then
            echo "false green detected: TestMain exited before m.Run"; exit 91
          fi
          jq -r 'select(.Action=="run"  and .Test != null) | .Test' "$GATE_DIR/targeted.json" | LC_ALL=C sort -u >"$GATE_DIR/run-names"
          jq -r 'select(.Action=="pass" and .Test != null) | .Test' "$GATE_DIR/targeted.json" | LC_ALL=C sort -u >"$GATE_DIR/pass-names"
          jq -r 'select(.Action=="skip" and .Test != null) | .Test' "$GATE_DIR/targeted.json" | LC_ALL=C sort -u >"$GATE_DIR/skip-names"
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
            grep -Fxq "$name" "$GATE_DIR/run-names"
            grep -Fxq "$name" "$GATE_DIR/pass-names"
            ! grep -Fxq "$name" "$GATE_DIR/skip-names"
            count=$((count + 1))
          done
          test "$count" -eq 8
          jq -e 'select(.Action=="pass" and .Package=="github.com/multica-ai/multica/server/internal/handler" and .Test==null)' \
            "$GATE_DIR/targeted.json" >/dev/null
          printf 'ORQ26_TARGETED_PASS leaf_tests=%d\n' "$count"

      - name: Full handler package (same DB, same false-green guard)
        working-directory: multica-auth-work/server
        run: |
          set -euo pipefail
          go test -race -count=1 -json ./internal/handler >"$GATE_DIR/pkg.json"
          jq -r 'select(.Action=="output") | .Output' "$GATE_DIR/pkg.json" >"$GATE_DIR/pkg-output.txt"
          if grep -Eq 'Skipping tests:|database not reachable|could not connect|no tests to run' \
            "$GATE_DIR/pkg-output.txt"; then
            echo "false green detected in full package run"; exit 91
          fi
          jq -e 'select(.Action=="pass" and .Package=="github.com/multica-ai/multica/server/internal/handler" and .Test==null)' \
            "$GATE_DIR/pkg.json" >/dev/null
          for name in TestUploadFileForeignWorkspace TestUploadFileResolvesWorkspaceViaSlugHeader \
                      TestUploadFileResolvesWorkspaceViaIDHeaderStill TestUploadFile_AttachesToChatSession \
                      TestUploadFile_RejectsForeignChatSession; do
            jq -r 'select(.Action=="pass" and .Test != null) | .Test' "$GATE_DIR/pkg.json" \
              | LC_ALL=C sort -u | grep -Fxq "$name"
          done
          echo 'ORQ26_REGRESSIONS_PASS'

      - name: Fingerprint after (input must not change during the gate)
        run: |
          set -euo pipefail
          test "$GATE_DIFF_BEFORE" = "$(git rev-parse HEAD)"
          git diff --quiet

      - name: Publish gate summary (no DSN, no secret)
        if: always()
        run: |
          set -euo pipefail
          {
            echo "### ORQ-26 DB gate"
            echo "- commit: \`${GITHUB_SHA}\`"
            echo "- file.go sha256: \`${LOCKED_FILE_SHA256}\`"
            echo "- file_test.go sha256: \`${LOCKED_TEST_SHA256}\`"
            echo "- leaf tests required: 8"
          } >>"$GITHUB_STEP_SUMMARY"

      - name: Upload gate artifacts (json only, no logs with DSN)
        if: always()
        uses: actions/upload-artifact@v4
        with:
          name: orq26-gate-json
          path: |
            ${{ env.GATE_DIR }}/targeted.json
            ${{ env.GATE_DIR }}/pkg.json
            ${{ env.GATE_DIR }}/expected-migrations
            ${{ env.GATE_DIR }}/applied-migrations
          retention-days: 3
```

Notas de conformidade do YAML acima:

- `POSTGRES_PASSWORD` é literal e descartável, exatamente como o `ci.yml` já faz com
  `multica:multica`. **Nenhum secret novo de repositório é criado**, portanto o commit não
  entra na classe "credential/auth change" do AA-001 §0.1.
- Sem `redis`: nenhum dos oito testes usa `REDIS_TEST_URL`. Se o run completo do pacote
  acusar skips dependentes de Redis, esses skips são legítimos (harness §6) e não reprovam,
  mas o serviço pode ser adicionado se o GTL preferir paridade total com o job `backend`.
- `golangci-lint` e `govulncheck` **não** entram: rodam no repo inteiro e podem falhar por
  motivos alheios ao ORQ-26. Recomendo executá-los depois, em job separado e advisory.
- Nenhum passo publica porta, toca compose, env, systemd, deploy ou release.

## 3. Gates a disparar, em ordem

| # | Gate | Comando/passo | Critério de aprovação |
|---|---|---|---|
| G1 | Congelamento | `sha256sum -c` dos dois locked | hashes idênticos aos do ruling |
| G2 | gofmt | `gofmt -l` nos dois locked | stdout vazia |
| G3 | Higiene de diff | `git diff --check` | exit 0 |
| G4 | Identidade do DB | `pg_isready` + `select current_database()` | `orq26_gate` e sentinels presentes |
| G5 | vet | `go vet ./...` | exit 0 |
| G6 | build | `go build ./...` | exit 0 |
| G7 | migrations | `go run ./cmd/migrate up` + comparação dinâmica | `Done.` e `missing-migrations` vazio |
| G8 | targeted | `go test -race -count=1 -json -run <regex> ./internal/handler` | 8 folhas com `run`+`pass`, zero `skip`, `pass` de pacote, zero marcador de falso-verde |
| G9 | pacote completo | `go test -race -count=1 -json ./internal/handler` | `pass` de pacote + as 5 regressões de upload com `pass`, zero marcador de falso-verde |
| G10 | fingerprint depois | `git rev-parse HEAD` + `git diff --quiet` | igual ao de antes |

Workflows que **não** devem ser disparados: `release.yml`, `desktop-smoke.yml`,
`mobile-verify.yml` (nenhum caminho afetado) e nenhum job de deploy. Nenhum merge.

## 4. Autorização mínima que o GTL precisa pedir ao owner

Todos os itens abaixo são STOP-AND-WAIT (AA-001 §0.1) e **não** foram executados:

| # | Ação | Reversibilidade | Blast radius |
|---|---|---|---|
| A1 | Commitar os três arquivos da §2.1 na branch `agent/kiro-opus5/orq-26-contract-fix` | total (branch isolada, sem merge) | nula fora da branch |
| A2 | `git push -u origin agent/kiro-opus5/orq-26-contract-fix` — **envia código para o GitHub remoto público**, é transmissão externa e precisa de aprovação explícita | reversível por delete da branch remota (que também é §0.1) | repositório remoto |
| A3 | Um `workflow_dispatch` do `ORQ-26 DB Gate (temporary)` | job efêmero, sem estado persistente | runner GitHub |
| A4 | Depois da evidência: reverter o workflow temporário e apagar a branch remota | é a própria reversão | repositório remoto |

Sem PR e sem merge em `main` nesta etapa. Observação: como não há workflow na raiz hoje, o
job `secret-scan` (gitleaks) do `ci.yml` **não** rodará; antes do A2 recomendo confirmar que
o diff contém apenas os três arquivos da §2.1 e nenhum `.env`, chave ou DSN.

## 5. Check-out

- Entregue: manifesto exato (3 arquivos, mensagem de commit, YAML completo do gate),
  lista ordenada de 10 gates com critérios, workflows a não disparar, e as 4 autorizações
  mínimas com risco e reversibilidade.
- Descoberta bloqueante registrada: o `ci.yml` está fora da raiz e portanto inerte neste
  repositório; sem um workflow na raiz, "disparar o CI oficial" é impossível.
- Congelamento reverificado nos três hashes do ruling.
- Nada commitado, pushado, mergeado, disparado ou deployado. Aguardando GTL-63 e a
  autorização do owner.

---

# V2 — Correção do manifesto após GTL-73 (2026-07-27T12:11Z)

V2 **substitui** a §2.3 (YAML) e a §4 (autorizações) da V1. A V1 fica preservada acima como
histórico. Continua valendo: nada foi criado, commitado, pushado ou disparado nesta etapa.
Hashes preservados: `file.go` `48553c6c…`, `file_test.go` `815b7d1c…`, harness `6ce6a0a4…`.

## V2.1 Ruling confirmado nas fontes oficiais do GitHub

O ruling do GTL está correto e eu confirmei o texto na documentação oficial:

> "This event will only trigger a workflow run if the workflow file exists on the default
> branch." — seção `workflow_dispatch`

- `workflow_dispatch`:
  <https://docs.github.com/en/actions/reference/workflows-and-actions/events-that-trigger-workflows#workflow_dispatch>
- `schedule` (mesma restrição, mais "Scheduled workflows will only run on the default branch"):
  <https://docs.github.com/en/actions/reference/workflows-and-actions/events-that-trigger-workflows#schedule>
- `push` (o único trigger que serve aqui) — a mesma página afirma explicitamente:
  "Runs your workflow when you push a commit or tag... **This includes workflows that are not
  merged into the default branch.**"
  <https://docs.github.com/en/actions/reference/workflows-and-actions/events-that-trigger-workflows#push>
- Filtros de branch e path em `push`:
  <https://docs.github.com/en/actions/reference/workflows-and-actions/workflow-syntax#onpushbranchestagsbranches-ignoretags-ignore>
  e
  <https://docs.github.com/en/actions/reference/workflows-and-actions/workflow-syntax#onpushpull_requestpull_request_targetpathspaths-ignore>
- `permissions`:
  <https://docs.github.com/en/actions/reference/workflows-and-actions/workflow-syntax#permissions>
- `concurrency`:
  <https://docs.github.com/en/actions/reference/workflows-and-actions/workflow-syntax#concurrency>
- Disparo manual (não utilizável aqui, pelo motivo acima):
  <https://docs.github.com/en/actions/how-tos/manage-workflow-runs/manually-run-a-workflow>

Consequência aceita: **`workflow_dispatch` é impossível** para um workflow que existe apenas
na branch temporária, e habilitá-lo exigiria tocar a default branch (`main`), o que está fora
de escopo. A autorização A3 da V1 (um `workflow_dispatch`) fica **cancelada**; o gate passa a
ser disparado exclusivamente pelo próprio `push` autorizado.

## V2.2 Branch dedicada, separada da agent branch durável

| Item | V1 | V2 |
|---|---|---|
| Branch a pushar | `agent/kiro-opus5/orq-26-contract-fix` (durável) | **`ci/orq26-db-gate`** (efêmera, descartável) |
| Origem | — | criada a partir do worktree congelado, com exatamente os 3 arquivos |
| Agent branch | seria publicada | **permanece local e não pushada** |

Plano de criação (nenhum comando executado):

```bash
# no worktree congelado /home/ec2-user/workspace/worktrees/gtl-orq26
printf '%s\n' '.gtl-orq26-gate/' >>.git/info/exclude      # cache privado nunca entra no commit
git switch -c ci/orq26-db-gate                             # a partir do estado congelado
git add multica-auth-work/server/internal/handler/file.go \
        multica-auth-work/server/internal/handler/file_test.go \
        .github/workflows/orq26-db-gate.yml
git status --porcelain                                     # deve listar exatamente 3 arquivos
git commit -F <mensagem da V1 §2.2>
sha256sum multica-auth-work/server/internal/handler/file.go \
          multica-auth-work/server/internal/handler/file_test.go   # reconferir os 2 hashes
git push -u origin ci/orq26-db-gate                        # este push é o gatilho do gate
```

Após a evidência: `git push origin --delete ci/orq26-db-gate` e retorno do worktree para
`agent/kiro-opus5/orq-26-contract-fix`. A branch efêmera nunca recebe merge.

## V2.3 Referências pinadas e verificadas

Resolvidas agora contra os repositórios/registries oficiais:

| Referência | Pin verificado | Fonte |
|---|---|---|
| `actions/checkout@v6` | `d23441a48e516b6c34aea4fa41551a30e30af803` | `GET https://api.github.com/repos/actions/checkout/git/ref/tags/v6` → `object.type=commit` |
| `actions/setup-go@v5` | `40f1582b2485089dde7abd97c1529aa768e1baff` | `GET https://api.github.com/repos/actions/setup-go/git/ref/tags/v5` → `object.type=commit` |
| `actions/upload-artifact@v4` | `ea165f8d65b6e75b540449e92b4886f43607fa02` | idem (registrado só para referência; **V2 não usa esta action**) |
| `pgvector/pgvector:pg17` (índice multi-arch) | `sha256:d2ef61f42ef767baa5a1475393303cc235bcd92febd9d7014eddb48b41f3bad0` | `HEAD registry-1.docker.io/v2/pgvector/pgvector/manifests/pg17`, header `docker-content-digest`, `content-type: application/vnd.oci.image.index.v1+json` |
| `pgvector/pgvector:pg17` (filho amd64) | `sha256:815bf5378222044da3b34d98e6a5fdac37b15c428b67d09c7c2d90a038e597bf` | Docker Hub tags API, `images[].architecture=amd64`, 157.683.774 bytes, push 2026-07-08 |

Notas honestas de verificação:

- `setup-go` já tem `v6` (`924ae3a1cded613372ab5595356fb5720e22ba16`). Mantive **v5** por
  paridade com o `ci.yml:104` existente. Trocar para v6 é decisão do GTL.
- Uso o digest do **índice multi-arch**, não o do filho amd64, para não quebrar caso o runner
  mude de arquitetura.
- **Não verifiquei** em documentação que o campo `services.<id>.image` aceite referência por
  digest; é uma referência de imagem Docker padrão e deve aceitar, mas se a primeira execução
  falhar por isso, o fallback é `image: pgvector/pgvector:pg17` com o digest registrado aqui
  como evidência do que foi usado. Declaro isso como não verificado em vez de afirmar.
- Tags Git são mutáveis; é exatamente por isso que o pin é por SHA de commit.

## V2.4 YAML V2 — substitui integralmente a V1 §2.3

Mudanças frente à V1: sem `workflow_dispatch`; `push` restrito a `ci/orq26-db-gate` **e** aos
paths dos 3 arquivos; actions por SHA completo; imagem por digest; `permissions: contents: read`
explícito no workflow e no job; `timeout-minutes` no job e nos passos longos; `concurrency`
com `cancel-in-progress: false` para não abortar um gate em curso; **zero** `secrets:`,
**zero** artifacts; resumo publicado sem DSN.

```yaml
name: ORQ-26 DB Gate (temporary)

# workflow_dispatch and schedule are deliberately absent: GitHub only honours
# them for workflow files that exist on the repository default branch, and this
# gate must not touch main.
# https://docs.github.com/en/actions/reference/workflows-and-actions/events-that-trigger-workflows#workflow_dispatch
on:
  push:
    branches:
      - ci/orq26-db-gate
    paths:
      - .github/workflows/orq26-db-gate.yml
      - multica-auth-work/server/internal/handler/file.go
      - multica-auth-work/server/internal/handler/file_test.go

permissions:
  contents: read

concurrency:
  group: orq26-db-gate-${{ github.ref }}
  cancel-in-progress: false

jobs:
  orq26-db-gate:
    runs-on: ubuntu-latest
    timeout-minutes: 30
    permissions:
      contents: read
    services:
      postgres:
        # pgvector/pgvector:pg17 pinned by multi-arch index digest.
        image: pgvector/pgvector@sha256:d2ef61f42ef767baa5a1475393303cc235bcd92febd9d7014eddb48b41f3bad0
        env:
          POSTGRES_DB: orq26_gate
          POSTGRES_USER: orq26_gate
          # Throwaway literal for a service container reachable only from this
          # ephemeral runner. Mirrors the existing multica:multica literal in
          # multica-auth-work/.github/workflows/ci.yml, so no repository secret
          # is created or consumed anywhere in this workflow.
          POSTGRES_PASSWORD: orq26_gate
        ports:
          - 5432:5432
        options: >-
          --health-cmd "pg_isready -U orq26_gate -d orq26_gate"
          --health-interval 5s
          --health-timeout 5s
          --health-retries 20
    env:
      CI: "true"
      ORQ26_EPHEMERAL_DB_GATE: "1"
      DATABASE_URL: postgres://orq26_gate:orq26_gate@localhost:5432/orq26_gate?sslmode=disable
      PGHOST: localhost
      PGPORT: "5432"
      PGUSER: orq26_gate
      PGPASSWORD: orq26_gate
      PGDATABASE: orq26_gate
      LOCKED_FILE_SHA256: 48553c6c48d4423ebfba7d5a05366c0a23ec77d27a463ae4934eda200b557161
      LOCKED_TEST_SHA256: 815b7d1cf12ed9ca18340f9e87813994eebb7e5153aee281125f9e5b3db9e44d
    steps:
      - name: Checkout
        uses: actions/checkout@d23441a48e516b6c34aea4fa41551a30e30af803 # v6
        with:
          persist-credentials: false

      - name: Setup Go
        uses: actions/setup-go@40f1582b2485089dde7abd97c1529aa768e1baff # v5
        with:
          go-version: "1.26.1"
          cache-dependency-path: multica-auth-work/server/go.sum

      - name: Private gate dirs (0700)
        run: |
          set -euo pipefail
          umask 077
          gate_dir="${RUNNER_TEMP}/orq26-gate"
          mkdir -m 700 -p "$gate_dir/tmp" "$gate_dir/gocache" "$gate_dir/gotmp"
          {
            echo "GATE_DIR=$gate_dir"
            echo "TMPDIR=$gate_dir/tmp"
            echo "GOCACHE=$gate_dir/gocache"
            echo "GOTMPDIR=$gate_dir/gotmp"
          } >>"$GITHUB_ENV"

      - name: Freeze check (locked file hashes)
        working-directory: multica-auth-work/server
        run: |
          set -euo pipefail
          echo "${LOCKED_FILE_SHA256}  internal/handler/file.go"      | sha256sum -c -
          echo "${LOCKED_TEST_SHA256}  internal/handler/file_test.go" | sha256sum -c -

      - name: Changed-file guard (exactly the three expected paths)
        run: |
          set -euo pipefail
          git log -1 --name-only --pretty=format: | sed '/^$/d' | LC_ALL=C sort -u >"$GATE_DIR/changed"
          cat >"$GATE_DIR/allowed" <<'EOF'
          .github/workflows/orq26-db-gate.yml
          multica-auth-work/server/internal/handler/file.go
          multica-auth-work/server/internal/handler/file_test.go
          EOF
          LC_ALL=C sort -u -o "$GATE_DIR/allowed" "$GATE_DIR/allowed"
          comm -23 "$GATE_DIR/changed" "$GATE_DIR/allowed" >"$GATE_DIR/unexpected"
          test ! -s "$GATE_DIR/unexpected" || { echo "unexpected files in commit:"; cat "$GATE_DIR/unexpected"; exit 1; }

      - name: gofmt (must list nothing)
        working-directory: multica-auth-work/server
        run: |
          set -euo pipefail
          out="$(gofmt -l internal/handler/file.go internal/handler/file_test.go)"
          test -z "$out" || { echo "gofmt listed: $out"; exit 1; }

      - name: diff hygiene
        run: git diff --check

      - name: Fingerprint before
        run: echo "GATE_HEAD_BEFORE=$(git rev-parse HEAD)" >>"$GITHUB_ENV"

      - name: Database identity guard (no DSN printed)
        timeout-minutes: 3
        run: |
          set -euo pipefail
          test "$ORQ26_EPHEMERAL_DB_GATE" = "1"
          test "$CI" = "true"
          test "$PGDATABASE" = "orq26_gate"
          pg_isready -q
          test "$(psql -XAtqc 'select current_database()')" = "orq26_gate"

      - name: Go vet
        working-directory: multica-auth-work/server
        timeout-minutes: 10
        run: go vet ./...

      - name: Build
        working-directory: multica-auth-work/server
        timeout-minutes: 10
        run: go build ./...

      - name: Migrations (full set, verified)
        working-directory: multica-auth-work/server
        timeout-minutes: 10
        run: |
          set -euo pipefail
          go run ./cmd/migrate up >"$GATE_DIR/migrate.log" 2>&1 || {
            echo "migrate failed; log withheld because it may echo the DSN"; exit 1; }
          grep -Fxq 'Done.' "$GATE_DIR/migrate.log"
          find migrations -maxdepth 1 -type f -name '*.up.sql' -printf '%f\n' \
            | sed 's/\.up\.sql$//' | LC_ALL=C sort -u >"$GATE_DIR/expected-migrations"
          psql -XAtqc 'select version from schema_migrations order by version' \
            | LC_ALL=C sort -u >"$GATE_DIR/applied-migrations"
          comm -23 "$GATE_DIR/expected-migrations" "$GATE_DIR/applied-migrations" \
            >"$GATE_DIR/missing-migrations"
          test ! -s "$GATE_DIR/missing-migrations"
          echo "migrations_expected=$(wc -l <"$GATE_DIR/expected-migrations")"

      - name: Targeted ORQ-26 tests (8 leaves must run and pass)
        working-directory: multica-auth-work/server
        timeout-minutes: 20
        run: |
          set -euo pipefail
          run_re='^(TestUploadFile_ContextlessWithoutEntityRefsReturnsEmptyIDAndStorageLinks|TestUploadFile_ContextlessWithEntityRefsRejectedPreUpload|TestUploadFile_InsertFailureCleansUpAndReturns500|TestUploadFile_InsertFailureStillFailsWhenCleanupIsNoop|TestUploadFile_SuccessShapesAlwaysCarryContractFields)$'
          go test -race -count=1 -json -run "$run_re" ./internal/handler >"$GATE_DIR/targeted.json"
          jq -r 'select(.Action=="output") | .Output' "$GATE_DIR/targeted.json" >"$GATE_DIR/output.txt"
          if grep -Eq 'Skipping tests:|database not reachable|could not connect|no tests to run' "$GATE_DIR/output.txt"; then
            echo "false green detected: TestMain exited before m.Run"; exit 91
          fi
          jq -r 'select(.Action=="run"  and .Test != null) | .Test' "$GATE_DIR/targeted.json" | LC_ALL=C sort -u >"$GATE_DIR/run-names"
          jq -r 'select(.Action=="pass" and .Test != null) | .Test' "$GATE_DIR/targeted.json" | LC_ALL=C sort -u >"$GATE_DIR/pass-names"
          jq -r 'select(.Action=="skip" and .Test != null) | .Test' "$GATE_DIR/targeted.json" | LC_ALL=C sort -u >"$GATE_DIR/skip-names"
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
            grep -Fxq "$name" "$GATE_DIR/run-names"
            grep -Fxq "$name" "$GATE_DIR/pass-names"
            ! grep -Fxq "$name" "$GATE_DIR/skip-names"
            count=$((count + 1))
          done
          test "$count" -eq 8
          jq -e 'select(.Action=="pass" and .Package=="github.com/multica-ai/multica/server/internal/handler" and .Test==null)' \
            "$GATE_DIR/targeted.json" >/dev/null
          echo "ORQ26_TARGETED_PASS leaf_tests=$count"

      - name: Full handler package (same DB, same false-green guard)
        working-directory: multica-auth-work/server
        timeout-minutes: 25
        run: |
          set -euo pipefail
          go test -race -count=1 -json ./internal/handler >"$GATE_DIR/pkg.json"
          jq -r 'select(.Action=="output") | .Output' "$GATE_DIR/pkg.json" >"$GATE_DIR/pkg-output.txt"
          if grep -Eq 'Skipping tests:|database not reachable|could not connect|no tests to run' "$GATE_DIR/pkg-output.txt"; then
            echo "false green detected in full package run"; exit 91
          fi
          jq -e 'select(.Action=="pass" and .Package=="github.com/multica-ai/multica/server/internal/handler" and .Test==null)' \
            "$GATE_DIR/pkg.json" >/dev/null
          jq -r 'select(.Action=="pass" and .Test != null) | .Test' "$GATE_DIR/pkg.json" | LC_ALL=C sort -u >"$GATE_DIR/pkg-pass"
          for name in TestUploadFileForeignWorkspace TestUploadFileResolvesWorkspaceViaSlugHeader \
                      TestUploadFileResolvesWorkspaceViaIDHeaderStill TestUploadFile_AttachesToChatSession \
                      TestUploadFile_RejectsForeignChatSession; do
            grep -Fxq "$name" "$GATE_DIR/pkg-pass"
          done
          echo 'ORQ26_REGRESSIONS_PASS'

      - name: Fingerprint after (input must not change during the gate)
        run: |
          set -euo pipefail
          test "$GATE_HEAD_BEFORE" = "$(git rev-parse HEAD)"
          git diff --quiet

      - name: Gate summary (no DSN, no secret, no artifact upload)
        if: always()
        run: |
          set -euo pipefail
          {
            echo "### ORQ-26 DB gate"
            echo "- commit: \`${GITHUB_SHA}\`"
            echo "- branch: \`${GITHUB_REF_NAME}\`"
            echo "- file.go sha256: \`${LOCKED_FILE_SHA256}\`"
            echo "- file_test.go sha256: \`${LOCKED_TEST_SHA256}\`"
            echo "- required leaf tests: 8"
          } >>"$GITHUB_STEP_SUMMARY"
```

Decisões de segurança embutidas:

- `persist-credentials: false` no checkout: o `GITHUB_TOKEN` não fica no `.git/config` do
  runner, então nenhum passo subsequente pode empurrar nada.
- Nenhum `actions/upload-artifact`: elimina qualquer chance de publicar log com DSN. A prova
  fica no `$GITHUB_STEP_SUMMARY` e no log dos passos, que só imprimem contagens e nomes.
- O log do `cmd/migrate` é **retido**, não impresso, em caso de falha, porque pode ecoar a
  string de conexão.
- Nenhuma referência a `secrets.*` em nenhum ponto do workflow.

## V2.5 Risco conhecido do filtro de paths na criação da branch

Não consegui confirmar na documentação o comportamento de `paths` no primeiro `push` que
**cria** a branch. O par branch+paths é avaliado sobre os commits do push, e um push de
criação de branch pode não produzir a lista de arquivos alterados esperada. Consequência
possível: o primeiro push cria a branch e **não** dispara o gate.

Mitigações, em ordem de preferência, todas sem tocar `main`:

1. Aceitar e observar: se o run não aparecer, fazer um segundo push vazio na mesma branch
   (`git commit --allow-empty` + push) — porém um commit vazio não altera paths e pode
   também não disparar;
2. Manter o filtro de branch e **remover o filtro de paths** (a branch é dedicada e recebe
   apenas este commit, então o filtro de paths é redundante como controle de escopo; o
   controle real de escopo é o passo "Changed-file guard" dentro do job);
3. Criar a branch e o commit em um único push (é o plano da V2.2), que é o caminho com maior
   chance de disparar.

Recomendo deixar o filtro de paths no manifesto e, se o run não aparecer, aplicar a
mitigação 2 como correção mínima aprovada previamente pelo GTL — assim não é necessária uma
nova rodada de autorização por um detalhe de trigger.

## V2.6 Autorizações V2 (substituem a V1 §4)

| # | Ação | Reversibilidade | Blast radius |
|---|---|---|---|
| B1 | Criar a branch efêmera `ci/orq26-db-gate` a partir do worktree congelado e commitar exatamente os 3 arquivos | total | local |
| B2 | `git push -u origin ci/orq26-db-gate` — **transmissão de código para o remoto público** e, ao mesmo tempo, o **único** gatilho do gate | reversível por B4 | repositório remoto |
| B3 | Observar o run e coletar a evidência (nomes, exits, hashes) | leitura | nenhum |
| B4 | `git push origin --delete ci/orq26-db-gate` e voltar o worktree para a agent branch | é a própria reversão | repositório remoto |

Fora de escopo e não pedido: PR, merge, alteração de `main`, alteração do `ci.yml`,
`workflow_dispatch`, `schedule`, secrets de repositório, artifacts, deploy.

## V2.7 Estado

Nenhum YAML criado, nenhuma branch criada, nenhum commit, nenhum push, nenhum run. Apenas
este documento foi atualizado, com a seção V2 anexada e a V1 preservada. Aguardando GTL-69R.

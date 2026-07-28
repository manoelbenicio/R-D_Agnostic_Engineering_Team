# ORQ-39 — Browser QA descartável — correção documental V5

- **Card:** ORQ-39 · UUID `c03941bc-3bde-4de1-ab19-1ba93de0ad51`
- **Data UTC:** 2026-07-27
- **Modo:** proposta documental READ-ONLY
- **Documento supersedido:** `orq39-browser-qa-v4-amendment-design.md`
- **Fonte dos bloqueios:** `orq39-browser-qa-v4-peer-review.md`
- **Ruling de trigger vinculante:** GTL-69R em `gtl-orq26-root-ci-manifest-v2-peer-review.md`
- **Estado:** V5 pronta para revisão independente; execução continua bloqueada

Esta V5 substitui integralmente a V4 como proposta executável. V1–V4 permanecem somente como histórico. Nada neste documento autoriza criar workflow, editar código ou testes, executar Go/pnpm/Playwright, criar branch, fazer commit, push, PR, merge, dispatch, deploy ou mutar o board.

## 1. Resultado da correção dos seis BLOCKs

| # | BLOCK V4 | Correção normativa V5 | Critério documental de fechamento |
|---|---|---|---|
| 1 | Guarda de segredo resolvia paths a partir de `multica-auth-work` | O step usa `working-directory: ${{ github.workspace }}` e verifica os quatro paths root-aware: `.env`, `.env.worktree`, `multica-auth-work/.env`, `multica-auth-work/.env.worktree` | Qualquer um dos quatro arquivos faz o job sair diferente de zero antes de migration ou processo |
| 2 | Postgres sem health check | O serviço usa `pg_isready -U multica -d multica_test`, intervalo/timeout de 5 s e 10 retries | Nenhuma migration inicia antes de o service container ficar healthy |
| 3 | Backend/web em background sem readiness nem propagação de falha | Um único step supervisiona os dois PIDs, espera `/healthz` e `/`, monitora ambos durante o Playwright, propaga o exit do teste e usa `trap` para cleanup | Morte de backend ou web antes/durante a suíte encerra o teste e falha o step |
| 4 | Imagens referenciadas por tags mutáveis | Job e service usam somente referências `name@sha256:digest`, sem tag | YAML não contém `:v1.58.2-noble` nem `:pg17` em `image:` |
| 5 | `setup-node` e `pnpm/action-setup` frágeis dentro de `container:` | Node vem exclusivamente da imagem Playwright imutável; pnpm é ativado como `pnpm@10.28.2` via `corepack prepare` e invocado por `corepack pnpm` | Sem `actions/setup-node` e sem `pnpm/action-setup`; versão de pnpm deve ser exatamente `10.28.2` |
| 6 | Trigger PR/merge contradizia GTL-69R | Único trigger é `push` na branch efêmera `ci/orq39-browser-qa`; não há `pull_request`, `workflow_dispatch` nem merge | Push autorizado dispara o runner; coleta e deleção da branch são gates separados; PR/merge proibidos salvo supersessão escrita do owner |

## 2. Fatos do repositório usados pela V5

1. O repositório root não possui workflow ativo em `.github/workflows/`; os quatro workflows sob `multica-auth-work/.github/workflows/` são aninhados e inertes para este repositório root.
2. `multica-auth-work/package.json` declara `"packageManager": "pnpm@10.28.2"`.
3. `multica-auth-work/pnpm-lock.yaml` existe; instalação deve usar `--frozen-lockfile`.
4. `multica-auth-work/server/go.mod` declara `go 1.26.1`.
5. O backend registra liveness em `/health` e readiness em `/healthz` (`server/cmd/server/router.go:443-445`). V5 usa `/healthz` como barreira.
6. O frontend `@multica/web` fornece scripts `build` e `start`.
7. O fixture E2E chama `/auth/send-code`, consulta `SELECT code FROM verification_code ...` e depois chama `/auth/verify-code`; não existe coluna `verification_code`. `MULTICA_DEV_VERIFICATION_CODE` deve permanecer ausente, pois o fixture daria preferência a esse override e mascararia a comparação com `verification_code.code`. A V5 não repete a afirmação incorreta da V3/V4.
8. Sem SMTP/Resend configurado, `EmailService.SendVerificationCode` usa o tier DEV sem entrega externa, redige o código no log e retorna sucesso; assim o fluxo real de criação/leitura da linha no Postgres permanece testável sem credencial nem rede de e-mail.

## 3. Manifesto proposto do workflow raiz

**Caminho proposto, ainda inexistente:** `.github/workflows/orq39-browser-qa.yml`

Os valores abaixo são exclusivamente descartáveis e locais ao runner. Não há `secrets.*`, DSN real, token de provedor, conta OmniRoute ou endpoint de produção.

```yaml
name: ORQ-39 Ephemeral Browser QA

on:
  push:
    branches:
      - ci/orq39-browser-qa

permissions:
  contents: read
  actions: none
  id-token: none

concurrency:
  group: orq39-browser-qa-${{ github.ref }}
  cancel-in-progress: false

defaults:
  run:
    working-directory: multica-auth-work

jobs:
  browser-qa:
    name: Playwright Chromium against ephemeral stack
    runs-on: ubuntu-24.04
    timeout-minutes: 25
    permissions:
      contents: read
      actions: none
      id-token: none
    container:
      image: mcr.microsoft.com/playwright@sha256:6446946a1d9fd62d9ae501312a2d76a43ee688542b21622056a372959b65d63d
    services:
      postgres:
        image: pgvector/pgvector@sha256:d2ef61f42ef767baa5a1475393303cc235bcd92febd9d7014eddb48b41f3bad0
        env:
          POSTGRES_DB: multica_test
          POSTGRES_USER: multica
          POSTGRES_PASSWORD: multica_pass
        options: >-
          --health-cmd "pg_isready -U multica -d multica_test"
          --health-interval 5s
          --health-timeout 5s
          --health-retries 10

    steps:
      - name: Checkout exact pushed revision
        uses: actions/checkout@d23441a48e516b6c34aea4fa41551a30e30af803
        with:
          fetch-depth: 0
          persist-credentials: false

      - name: Prove root workflow placement and allowed branch delta
        working-directory: ${{ github.workspace }}
        shell: bash
        run: |
          set -Eeuo pipefail
          test -f .github/workflows/orq39-browser-qa.yml
          test ! -f multica-auth-work/.github/workflows/orq39-browser-qa.yml

          DEFAULT_BRANCH="${{ github.event.repository.default_branch }}"
          BASE_SHA="$(git merge-base "$GITHUB_SHA" "origin/$DEFAULT_BRANCH")"
          git diff --name-only "$BASE_SHA" "$GITHUB_SHA" | sort -u > "$RUNNER_TEMP/orq39-changed.txt"
          cat > "$RUNNER_TEMP/orq39-allowed.txt" <<'EOF'
          .github/workflows/orq39-browser-qa.yml
          multica-auth-work/e2e/board-kanban.spec.ts
          multica-auth-work/e2e/chat-panel-buttons.spec.ts
          multica-auth-work/e2e/chat-reasoning-level.spec.ts
          multica-auth-work/e2e/chat-upload-ui.spec.ts
          multica-auth-work/e2e/delete-flows.spec.ts
          multica-auth-work/e2e/squad-model-dropdown.spec.ts
          EOF
          sort -u -o "$RUNNER_TEMP/orq39-allowed.txt" "$RUNNER_TEMP/orq39-allowed.txt"
          comm -23 "$RUNNER_TEMP/orq39-changed.txt" "$RUNNER_TEMP/orq39-allowed.txt" > "$RUNNER_TEMP/orq39-forbidden.txt"
          test ! -s "$RUNNER_TEMP/orq39-forbidden.txt" || {
            echo "FATAL: pushed commit contains files outside ORQ-39 FILES_LOCKED"
            cat "$RUNNER_TEMP/orq39-forbidden.txt"
            exit 1
          }

      - name: Strict root-aware zero-secret guard
        working-directory: ${{ github.workspace }}
        shell: bash
        run: |
          set -Eeuo pipefail
          for f in .env .env.worktree multica-auth-work/.env multica-auth-work/.env.worktree; do
            if [ -f "$f" ]; then
              echo "FATAL: forbidden environment file present: $f"
              exit 1
            fi
          done
          if find . -path './.git' -prune -o -type f \( -name '*.pem' -o -name '*.key' -o -name 'auth.json' \) -print -quit | grep -q .; then
            echo "FATAL: credential-like file present in checkout"
            exit 1
          fi

      - name: Install exact Go toolchain
        uses: actions/setup-go@40f1582b2485089dde7abd97c1529aa768e1baff
        with:
          go-version-file: multica-auth-work/server/go.mod
          cache: false

      - name: Prove immutable container Node and activate exact pnpm
        shell: bash
        run: |
          set -Eeuo pipefail
          command -v node >/dev/null
          command -v corepack >/dev/null
          test -f package.json
          test -f pnpm-lock.yaml
          test "$(node -p 'require("./package.json").packageManager')" = "pnpm@10.28.2"
          corepack prepare pnpm@10.28.2 --activate
          test "$(corepack pnpm --version)" = "10.28.2"
          test "$(go env GOVERSION)" = "go1.26.1"
          printf 'node=%s\npnpm=%s\ngo=%s\n' \
            "$(node --version)" "$(corepack pnpm --version)" "$(go env GOVERSION)" >> "$GITHUB_STEP_SUMMARY"

      - name: Install frozen dependencies
        shell: bash
        run: corepack pnpm install --frozen-lockfile

      - name: Apply migrations to ephemeral Postgres
        working-directory: multica-auth-work/server
        shell: bash
        env:
          DATABASE_URL: postgres://multica:multica_pass@postgres:5432/multica_test?sslmode=disable
        run: |
          set -Eeuo pipefail
          test "$(printf '%s' "$DATABASE_URL" | sed -E 's#^[^@]*@([^:/?]+).*#\1#')" = postgres
          go run ./cmd/migrate up

      - name: Build web with the ephemeral backend URL
        shell: bash
        env:
          NEXT_PUBLIC_API_URL: http://localhost:8080
        run: corepack pnpm --filter @multica/web build

      - name: Run supervised backend, web and Playwright
        shell: bash
        env:
          APP_ENV: test
          DATABASE_URL: postgres://multica:multica_pass@postgres:5432/multica_test?sslmode=disable
          JWT_SECRET: ephemeral-orq39-ci-only-secret-32-bytes
          PORT: "8080"
          NEXT_PUBLIC_API_URL: http://localhost:8080
          PLAYWRIGHT_BASE_URL: http://localhost:3000
        run: |
          set -Eeuo pipefail
          test -z "${MULTICA_DEV_VERIFICATION_CODE:-}"
          RUN_DIR="$RUNNER_TEMP/orq39"
          mkdir -p "$RUN_DIR"
          chmod 700 "$RUN_DIR"

          redact_log() {
            sed -E \
              -e 's#postgres://[^[:space:]]+#postgres://[REDACTED]#g' \
              -e 's#(Bearer )[A-Za-z0-9._-]+#\1[REDACTED]#g' \
              -e 's#ephemeral-orq39-ci-only-secret-32-bytes#[REDACTED]#g' "$1" | tail -n 120
          }

          fail_service() {
            label="$1"
            log="$2"
            test_pid="${3:-}"
            echo "FATAL: $label exited before browser QA completed"
            redact_log "$log"
            if [ -n "$test_pid" ]; then kill "$test_pid" 2>/dev/null || true; fi
            exit 1
          }

          wait_ready() {
            label="$1"
            url="$2"
            pid="$3"
            log="$4"
            for _ in $(seq 1 60); do
              kill -0 "$pid" 2>/dev/null || fail_service "$label" "$log"
              if curl --fail --silent --show-error "$url" >/dev/null; then return 0; fi
              sleep 2
            done
            echo "FATAL: readiness timeout for $label"
            redact_log "$log"
            return 1
          }

          cleanup() {
            status=$?
            for pid_file in "$RUN_DIR/web.pid" "$RUN_DIR/backend.pid"; do
              if [ -f "$pid_file" ]; then
                pid="$(cat "$pid_file")"
                kill "$pid" 2>/dev/null || true
                wait "$pid" 2>/dev/null || true
              fi
            done
            exit "$status"
          }
          trap cleanup EXIT INT TERM

          (
            cd server
            exec go run ./cmd/server
          ) >"$RUN_DIR/backend.log" 2>&1 &
          BACKEND_PID=$!
          printf '%s\n' "$BACKEND_PID" > "$RUN_DIR/backend.pid"
          wait_ready backend http://localhost:8080/healthz "$BACKEND_PID" "$RUN_DIR/backend.log"

          (
            exec corepack pnpm --filter @multica/web start
          ) >"$RUN_DIR/web.log" 2>&1 &
          WEB_PID=$!
          printf '%s\n' "$WEB_PID" > "$RUN_DIR/web.pid"
          wait_ready web http://localhost:3000/ "$WEB_PID" "$RUN_DIR/web.log"

          corepack pnpm exec playwright test \
            --trace=retain-on-failure \
            --screenshot=only-on-failure \
            --video=off \
            --max-failures=5 &
          TEST_PID=$!

          while kill -0 "$TEST_PID" 2>/dev/null; do
            kill -0 "$BACKEND_PID" 2>/dev/null || fail_service backend "$RUN_DIR/backend.log" "$TEST_PID"
            kill -0 "$WEB_PID" 2>/dev/null || fail_service web "$RUN_DIR/web.log" "$TEST_PID"
            sleep 1
          done
          wait "$TEST_PID"
```

## 4. Explicação das decisões de implementação

### 4.1 Guarda root-aware

O `defaults.run.working-directory` continua em `multica-auth-work`, mas os steps de governança usam explicitamente `${{ github.workspace }}`. Assim, a verificação não depende do cwd herdado. A guarda cobre raiz e subdiretório da aplicação, incluindo `.env.worktree`, que `e2e/env.ts` tenta carregar.

### 4.2 Postgres saudável antes de migrations

O service container tem health check completo, não apenas `pg_isready` sem usuário/banco. O hostname `postgres` é o único host aceito pela precondição da migration. Nenhum endereço de produção é lido ou aceito.

### 4.3 Supervisão real dos processos

Backend e web são filhos do mesmo shell que executa o Playwright. O shell:

1. grava os PIDs em diretório `0700` dentro de `$RUNNER_TEMP`;
2. exige `/healthz` do backend e HTTP 200 da raiz web;
3. verifica `kill -0` dos dois processos durante toda a suíte;
4. mata o teste se um serviço morrer;
5. propaga o exit do Playwright via `wait` sob `set -e`;
6. encerra os processos no `trap`, sem daemon órfão;
7. não publica logs como artifacts; somente um tail redigido aparece em falha.

### 4.4 Identidade imutável das imagens

As referências não contêm tags. A identidade do Node também fica congelada pelo digest da imagem Playwright; não se baixa uma segunda distribuição com `setup-node`. O job registra `node --version`, mas a garantia de bits vem do digest, não de uma tag ou major móvel.

### 4.5 Node/pnpm container-safe

Não há `actions/setup-node` nem `pnpm/action-setup`. `corepack prepare pnpm@10.28.2 --activate` seleciona a versão literal declarada pelo repositório, e todos os comandos usam `corepack pnpm`, evitando a necessidade de escrever shim global ao lado do binário Node. Divergência de `packageManager`, ausência de Corepack ou versão diferente de pnpm falha antes do install.

### 4.6 Trigger e governança GTL-69R

A V5 adota o mesmo mecanismo ratificado pelo GTL-69R:

- branch remota efêmera exclusiva: `ci/orq39-browser-qa`;
- evento único: `push` nessa branch;
- sem `pull_request`;
- sem `workflow_dispatch`;
- sem merge em `main`;
- coleta do resultado no run originado pelo push;
- deleção remota da branch somente em gate próprio e autorização própria.

Um `push` e a execução automática correspondente são uma única operação técnica: não é possível autorizar o push e impedir o evento que ele dispara. Se o owner quiser PR ou merge, deve emitir decisão escrita que cite e superseda expressamente o GTL-69R antes de qualquer alteração da V5.

## 5. FILES_LOCKED V5 — sem ORQ-26

Conjunto exclusivo proposto para uma futura implementação autorizada:

```text
.github/workflows/orq39-browser-qa.yml
multica-auth-work/e2e/chat-upload-ui.spec.ts
multica-auth-work/e2e/chat-panel-buttons.spec.ts
multica-auth-work/e2e/chat-reasoning-level.spec.ts
multica-auth-work/e2e/squad-model-dropdown.spec.ts
multica-auth-work/e2e/delete-flows.spec.ts
multica-auth-work/e2e/board-kanban.spec.ts
```

**Explicitamente fora do lock e fora do escopo:**

- `multica-auth-work/server/internal/handler/file.go` — pertence à ORQ-26;
- `multica-auth-work/server/internal/handler/file_test.go` — pertence à ORQ-26;
- `multica-auth-work/package.json` e `pnpm-lock.yaml`;
- `multica-auth-work/playwright.config.ts`;
- os sete specs E2E existentes;
- workflows aninhados sob `multica-auth-work/.github/workflows/`;
- qualquer arquivo de aplicação, migration, generated/sqlc, segredo ou configuração live.

A ORQ-39 apenas referencia a dependência funcional da ORQ-26; não reivindica seus arquivos.

## 6. Gates de autorização e STOP

| Gate | Ação futura | Requisito |
|---|---|---|
| D0 | revisão independente desta V5 | PASS escrito por revisor que não seja o autor |
| D1 | criar os sete FILES_LOCKED em worktree isolado e commit local | autorização escrita do owner; diff exato e novo peer review |
| D2 | criar/pushar `ci/orq39-browser-qa` | autorização escrita específica; o push dispara automaticamente o workflow |
| D3 | ler o run e coletar somente status/summary sem segredos | autorização ou escopo incluído expressamente em D2 |
| D4 | deletar a branch remota efêmera | autorização escrita separada; operação destrutiva/reversível |

**STOP imediato** se: digest não resolver para a arquitetura do runner; arquivo fora de FILES_LOCKED aparecer; qualquer `.env`/credencial-like file existir; Postgres não ficar healthy; toolchain divergir; backend/web morrer; readiness expirar; artifacts contiverem segredo; branch não for exatamente `ci/orq39-browser-qa`; ou surgir pedido de PR/merge sem supersessão escrita do GTL-69R pelo owner.

## 7. Critérios de aceite da futura execução

A V5 só alcança EXECUTION PASS quando houver evidência de um run autorizado provando simultaneamente:

1. workflow raiz executado pelo push da branch efêmera correta;
2. zero arquivo fora do lock no commit disparador;
3. guarda de quatro paths e credential-like files PASS;
4. imagens identificadas pelos dois digests esperados;
5. Node da imagem registrado, pnpm exatamente 10.28.2 e Go exatamente 1.26.1;
6. Postgres healthy antes das migrations;
7. migrations concluídas no banco efêmero;
8. `/healthz` backend e `/` web prontos;
9. PIDs vivos durante toda a suíte e cleanup final;
10. Playwright com resultado real, sem skip/falso-verde;
11. nenhum segredo, DSN real ou token publicado;
12. nenhum PR, merge ou mudança de `main`.

Até lá, o estado correto é **DESIGN READY FOR REVIEW / EXECUTION BLOCKED**.

## 8. Pedido formal de revisão independente

Solicita-se re-review adversarial por agente diferente do autor, em modo READ-ONLY, antes de D1. O revisor deve conferir:

1. fechamento literal dos seis BLOCKs de `orq39-browser-qa-v4-peer-review.md`;
2. sintaxe YAML e comportamento de `working-directory` root-aware;
3. health check do service Postgres;
4. captura, monitoramento, propagação de falha e cleanup dos PIDs;
5. referências de imagem apenas por digest;
6. ausência de `setup-node` e `pnpm/action-setup`, com pnpm 10.28.2 exato;
7. conformidade do trigger com GTL-69R e ausência de PR/merge;
8. FILES_LOCKED exato, sem `file.go`/`file_test.go` da ORQ-26;
9. ausência de segredo/contexto de produção e de artifact inseguro;
10. distinção honesta entre DESIGN e EXECUTION PASS.

O parecer deve ser `PASS` ou `BLOCK`, citar linhas literais desta V5 e não executar workflow, teste, push, PR, merge, deploy ou mutação de board.

## 9. Declaração de não execução

Nesta correção documental não foi criado ou editado workflow/código/teste, não foi executado Go, pnpm, Playwright, migration ou container, e não houve branch, commit, push, PR, merge, dispatch, deploy, restart, acesso a segredo ou mutação de board. O único novo artefato é este documento de proposta V5 e seu check-out documental.

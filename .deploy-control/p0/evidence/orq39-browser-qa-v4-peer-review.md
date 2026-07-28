# ORQ-39 — Peer review adversarial independente da Emenda V4 (READ-ONLY)

- Card: **ORQ-39** · UUID `c03941bc-3bde-4de1-ab19-1ba93de0ad51` · number 39
- Documento revisado: `.deploy-control/p0/evidence/orq39-browser-qa-v4-amendment-design.md`
  (autor Antigravity, 7444 bytes, 179 linhas, mtime 15:35)
- Revisor: Codex56#A (`w7:p3`) · UTC 2026-07-27T15:43Z
- Modo: **READ-ONLY**. Nenhuma edição de workflow ou código, nenhum teste executado, nenhum push, PR,
  merge, deploy ou mutação de board.

## VEREDITO: **BLOCK** — 6 defeitos que impedem o pipeline de funcionar ou de provar o que promete

O desenho acerta em vários pontos verificáveis (§1), mas **como está, o workflow falha ou dá falso
verde**. Os defeitos são corrigíveis e cada um tem fix exato abaixo.

## 1. O que confirmei como CORRETO (medido no repo)

| afirmação V4 | verificação | resultado |
|---|---|---|
| coluna `code` em `verification_code` | `server/migrations/009_verification_code.up.sql:1-9`: `id, email, code TEXT NOT NULL, expires_at, used, created_at` + índice `(email, used, expires_at)` | **CONFIRMADO** |
| ciclo de vida send-code usado pelo fixture | `e2e/fixtures.ts:35` `DELETE FROM verification_code WHERE email=$1` → `:38` `POST /auth/send-code` → `:47-51` `SELECT code … used = FALSE AND expires_at > now() ORDER BY created_at DESC LIMIT 1` → `:59` `POST /auth/verify-code` | **CONFIRMADO**, e a migration cobre exatamente as colunas usadas |
| paths canônicos do filtro | existem: `apps/web`, `packages/views`, `server`, `e2e`, `playwright.config.ts` | **CONFIRMADO** |
| nome do pacote pnpm | `apps/web/package.json` → `name = @multica/web`, scripts incluem `build` e `start` | **CONFIRMADO** |
| `cmd/migrate up` | `server/cmd/migrate/main.go:109-115` exige `os.Args[1]` ∈ {`up`,`down`} | **CONFIRMADO** |
| `APP_ENV=test` é consumido | `server/cmd/server/main.go:125` `auth.ValidateJWTConfiguration(os.Getenv("APP_ENV"), os.Getenv("JWT_SECRET"))` | **CONFIRMADO** |
| `JWT_SECRET` do workflow satisfaz o mínimo | valor proposto tem 39 caracteres (≥32 exigidos fora de dev) | **CONFIRMADO** |
| `PLAYWRIGHT_BASE_URL` é a variável certa | `playwright.config.ts:10` `baseURL: process.env.PLAYWRIGHT_BASE_URL ?? process.env.FRONTEND_ORIGIN ?? "http://localhost:3000"` | **CONFIRMADO** |
| `NEXT_PUBLIC_API_URL` é a variável certa | `e2e/fixtures.ts:13` `API_BASE = process.env.NEXT_PUBLIC_API_URL \|\| http://localhost:${PORT\|\|8080}` | **CONFIRMADO** |
| timeout/retries/workers | `playwright.config.ts:6-8` `timeout: 60000`, `workers: 1`, `retries: 0` — o doc não invento nada | **CONFIRMADO** |
| workflow na raiz | raiz **não** tem `.github/workflows` hoje; colocar lá é o único local que o GitHub lê | **CONFIRMADO** |
| `go-version-file: multica-auth-work/server/go.mod` | correto: `defaults.run.working-directory` afeta só steps `run`, não inputs de `uses` | **CONFIRMADO** |
| sem dado de produção | serviço Postgres efêmero com credenciais literais de teste; nenhuma referência a host/DSN real | **CONFIRMADO** |

## 2. BLOCK-1 — A guarda de zero-segredos aponta para os caminhos errados

`defaults.run.working-directory: multica-auth-work` aplica-se a **todo** step `run`, incluindo a guarda.
Logo, dentro da guarda:

| escrito no doc | resolve de fato | consequência |
|---|---|---|
| `-f .env` | `multica-auth-work/.env` | acerta por acidente |
| `-f .env.worktree` | `multica-auth-work/.env.worktree` | acerta por acidente |
| `-f multica-auth-work/.env` | `multica-auth-work/multica-auth-work/.env` | **nunca existe** |
| (não checa) | `.env` / `.env.worktree` **da raiz** | **não é verificado** |

Isso importa porque `e2e/env.ts:5-13` carrega `\.env.worktree\` ou `.env` via `dotenv` a partir de
`process.cwd()`. A guarda existe para impedir que um segredo entre no runner, e hoje ela deixa a raiz
sem cobertura.

**Fix exato:** rodar a guarda com `working-directory: .` explícito e checar os quatro caminhos:

```yaml
      - name: Strict Zero-Secret Guard
        working-directory: .
        run: |
          for f in .env .env.worktree multica-auth-work/.env multica-auth-work/.env.worktree; do
            if [ -f "$f" ]; then echo "FATAL: $f present in CI context"; exit 1; fi
          done
```

## 3. BLOCK-2 — Serviço `postgres` sem health check: a migration corre contra banco que talvez não aceite conexão

O bloco `services.postgres` não declara `options: --health-cmd …`. Sem isso o GitHub **não espera** o
Postgres ficar pronto; o step `go run ./cmd/migrate up` roda assim que o container sobe e pode falhar
com "connection refused" de forma intermitente — falso vermelho.

**Fix exato:**

```yaml
      postgres:
        image: pgvector/pgvector@sha256:<digest>
        env: { POSTGRES_DB: multica, POSTGRES_USER: multica, POSTGRES_PASSWORD: multica }
        options: >-
          --health-cmd "pg_isready -U multica -d multica"
          --health-interval 5s --health-timeout 5s --health-retries 10
```

## 4. BLOCK-3 — Backend e frontend em background sem readiness e sem propagação de falha

```yaml
run: cd server && go run ./cmd/server &        # step termina imediatamente com exit 0
run: pnpm --filter @multica/web start &        # idem
```

Dois problemas somados: (a) o step sai `0` mesmo que o processo morra em seguida — `main.go` tem
`os.Exit(1)` em vários caminhos (`:127`, `:155`, `:161`, `:345`, `:414`, `:434`), e nada disso reprova o
job; (b) não há espera de readiness, então o Playwright pode iniciar antes de `:3000`/`:8080`
escutarem. O resultado típico é falha do Playwright atribuída ao produto, não à infra.

**Fix exato:** manter o background, mas com log capturado e barreira de readiness explícita, e falhar o
job se o processo morreu:

```yaml
      - name: Start Backend API Server
        env: { APP_ENV: test, DATABASE_URL: ..., JWT_SECRET: ..., PORT: "8080" }
        run: |
          cd server && (go run ./cmd/server > /tmp/backend.log 2>&1 & echo $! > /tmp/backend.pid)
          for i in $(seq 1 60); do curl -fsS http://localhost:8080/health && break; sleep 2; done
          kill -0 "$(cat /tmp/backend.pid)" || { echo "backend morreu"; cat /tmp/backend.log; exit 1; }
          curl -fsS http://localhost:8080/health >/dev/null || { cat /tmp/backend.log; exit 1; }
```
E o equivalente para o web em `:3000` (`curl -fsS http://localhost:3000` antes do Playwright).
Anexar `/tmp/backend.log` como artifact **sem** valores de env.

## 5. BLOCK-4 — Pins declarados "imutáveis" que na verdade são tags mutáveis

§2.5 chama de "pins imutáveis" e depois usa **tags**:

- `mcr.microsoft.com/playwright:v1.58.2-noble` — tag, republicável;
- `pgvector/pgvector:pg17` — tag, e das mais móveis (rastreia patches do pg17).

As Actions estão corretamente pinadas por SHA de 40 caracteres (checkout, setup-node, setup-go,
pnpm/action-setup) — esse ponto é bom e deve ser preservado.

**Fix exato:** usar digest em ambas as imagens: `image: mcr.microsoft.com/playwright@sha256:<digest>` e
`image: pgvector/pgvector@sha256:<digest>`, com o digest resolvido e registrado na evidência no momento
da autorização; se o digest não puder ser obtido, o card para em vez de aceitar tag.

## 6. BLOCK-5 — `setup-node` e `pnpm/action-setup` dentro de `container:` são redundantes e frágeis

Num job com `container:`, o runner executa dentro da imagem Playwright, que **já** traz Node. Rodar
`actions/setup-node` ali baixa outra distribuição de Node para dentro do container e depende de
ferramentas presentes na imagem; `pnpm/action-setup` sem `version:` depende de `packageManager` no
`package.json` da **raiz** do repositório — e a raiz deste monorepo não é `multica-auth-work`.

**Fix exato (escolher um dos dois caminhos, não os dois):**
- **A (recomendado):** remover `setup-node` e `pnpm/action-setup`; usar o Node da imagem e habilitar
  pnpm por `corepack enable && corepack prepare pnpm@<versão-exata> --activate`, com a versão vinda de
  `multica-auth-work/package.json`; **ou**
- **B:** manter as actions, mas declarar `pnpm/action-setup` com `version:` fixa e verificar que a
  imagem suporta as actions; e declarar `cache: pnpm` só se o lockfile estiver no path correto.

Em ambos, `pnpm install --frozen-lockfile` deve rodar com `working-directory: multica-auth-work`
(já está, pelo default) e o lockfile precisa existir lá — condição a verificar no gate.

## 7. BLOCK-6 — O gatilho contradiz o mecanismo já ratificado para CI de gate

§2.8 pede quatro portas (push, PR, merge, execução) e o workflow usa `on: pull_request` com merge em
`main`. Isso conflita com o ruling oficial do **GTL-69R**, que estabeleceu, para o gate do ORQ-26, que
`workflow_dispatch` é impossível fora da default branch e que **`push` em branch efêmera
(`ci/orq26-db-gate`) é o mecanismo correto** — sem PR nem merge.

Não estou dizendo que `pull_request` é tecnicamente inválido: para PR do mesmo repositório o workflow do
head roda. Estou dizendo que **a frota agora teria dois mecanismos de gate divergentes**, e o de ORQ-39
exige **merge em `main`**, que é mudança permanente na branch default — bem mais invasivo que a branch
efêmera.

**Fix exato:** alinhar ao GTL-69R — `on: push` restrito a `branches: [ci/orq39-browser-qa]`, mantendo
`paths`, e reduzir as portas de 4 para 2 (push da branch efêmera + execução). Se o owner quiser mesmo
PR/merge, isso precisa de ruling explícito que supersede o GTL-69R, registrado antes.

## 8. Observações menores (não bloqueiam)

1. `RESEND_API_KEY`/`SMTP_HOST`: `main.go:129` avisa quando ambos estão vazios. O fluxo E2E **não**
   depende de e-mail real (o código é lido do banco), mas convém declarar no card que o aviso é
   esperado, para não ser lido como falha.
2. `MULTICA_DEV_VERIFICATION_CODE` não é definido — correto, porque `fixtures.ts:56-57` prefere essa
   variável quando presente e cairia em código fixo; deixar ausente mantém o teste realista.
3. `permissions: contents: read` está certo; acrescentar `actions: none` e `id-token: none` explícitos
   reduz a superfície, como já exigido no gate do ORQ-26.
4. `fetch-depth: 1` é suficiente para o filtro de `paths`, mas se algum teste comparar contra base, vai
   faltar histórico — declarar que nenhum teste do lote precisa.
5. FILES_LOCKED inclui `file.go`/`file_test.go` do **ORQ-26**, que estão congelados naquele card. Locar
   os mesmos arquivos em dois cards é colisão de lock: ORQ-39 deveria **referenciar** o congelamento,
   não reivindicar o lock.

## 9. Não-alegações

- Não editei nem criei workflow, não rodei teste, `pnpm`, `go`, Playwright ou migration; nenhum push,
  PR, merge, deploy ou mutação de board.
- Não resolvi digests de imagem (exigiria consulta a registry); apenas exijo que sejam usados.
- Não verifiquei se `multica-auth-work/pnpm-lock.yaml` está sincronizado, nem se `packageManager` está
  declarado — apontei como condição a verificar no gate.
- Não executei o workflow em nenhuma forma, portanto os defeitos 2–5 são deduzidos do contrato do
  GitHub Actions e do código do produto, não de execução observada.
- Não avaliei o conteúdo dos specs E2E (`auth`, `chat-attachments`, `comments`) quanto a flakiness.

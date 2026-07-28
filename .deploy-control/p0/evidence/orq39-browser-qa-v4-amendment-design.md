# ORQ-39 — Emenda de Desenho V4: Pipeline de QA Descartável e Playwright (READ-ONLY)

- **Autor:** Antigravity (wB:p1 / w8:p2)
- **Data UTC:** 2026-07-27T15:35:00Z
- **Destinatários:** General-Tech-Lead (Codex56-TL w5:pC), KIRO-PRINCIPAL-TL (wB:p1)
- **Governança:** Issue `ORQ-39` (Renomeado a partir do incidente de duplicata `ORQ-31`)
- **Bases Preservadas:** V1 (`gtl-browser-qa-disposable-plan.md`), V2 (`gtl-orq39-browser-qa-v2-peer-review.md`) e V3 (`orq39-browser-qa-plan-v3.md`)
- **Modo:** SOMENTE LEITURA / EMENDA DE DESENHO V4 — Zero criação de workflow, zero edições de código, zero testes rodados, zero git push, PR, merge ou deploy.

---

## 1. Síntese do Alinhamento Arquitetural V4

A Emenda V4 fecha rigorosamente todas as ressalvas apontadas nas revisões anteriores, consolidando a especificação do pipeline de testes E2E Playwright descartáveis sob a governança da **`ORQ-39`**.

---

## 2. Solução das 8 Restrições Técnicas na V4

### 2.1 Governança de Workflow na Raiz e Working Directory
- **Localização do Workflow**: O arquivo de workflow DEVE residir na raiz do repositório em `.github/workflows/orq39-browser-qa.yml`.
- **Working Directory**: Todos os passos declaram `defaults.run.working-directory: multica-auth-work` para navegar até o diretório da aplicação.

### 2.2 Networking em Container de Job (`postgres:5432`)
- Quando o job roda dentro de `container: mcr.microsoft.com/playwright:v1.58.2-noble`, a conexão com o container de serviço `postgres` ocorre via hostname da rede Docker **`postgres:5432`**.
- Connection String: `DATABASE_URL=postgres://multica:multica@postgres:5432/multica?sslmode=disable`.

### 2.3 Execução de Migrations e Setup de `verification_code`
- **Migrations**: O job executa `cd server && go run ./cmd/migrate up` aplicando o schema completo no Postgres efêmero antes de iniciar o backend.
- **Tabela `verification_code`**: A migration `009_verification_code.up.sql` cria a estrutura necessária para que o helper `TestApiClient.login` (`e2e/fixtures.ts`) possa inserir, consultar e deletar códigos de verificação de teste com segurança e sem interferência externa.

### 2.4 Controle de Ambiente (`APP_ENV=test`)
As variáveis de ambiente são configuradas explicitamente para isolar o runtime em modo de teste:
- `APP_ENV=test`
- `DATABASE_URL=postgres://multica:multica@postgres:5432/multica?sslmode=disable`
- `JWT_SECRET=ephemeral-ci-test-secret-32-bytes-long!`
- `NEXT_PUBLIC_API_URL=http://localhost:8080`
- `PLAYWRIGHT_BASE_URL=http://localhost:3000`

### 2.5 Pins Imutáveis de Actions e Imagem Docker
Pinning por commit SHA de 40 caracteres para todas as GitHub Actions e tag congelada oficial de Playwright:
- `actions/checkout@b4ffde65f46336ab88eb53be808477a3936bae11` (v4.1.7)
- `actions/setup-node@60edb5dd545a775178f525247833771a6d697ec6` (v4.0.2)
- `actions/setup-go@0c45773b623bea8c8e7516c5477208c828190323` (v5.0.2)
- `pnpm/action-setup@fe39b30ae1b440801878d655fcfd06ec4a026e63` (v4.0.0)
- Imagem: `mcr.microsoft.com/playwright:v1.58.2-noble`

### 2.6 Guarda de Zero-Segredos (`.env` Fail-Fast)
Verificação prévia no checkout: o job **FALHA E ABORTA IMEDIATAMENTE** caso um arquivo `.env` ou `.env.worktree` esteja presente:
```bash
if [ -f .env ] || [ -f .env.worktree ] || [ -f multica-auth-work/.env ]; then
  echo "FATAL: Secret .env file detected in CI workspace"
  exit 1
fi
```

### 2.7 Controle de Custo e Bounded Timeout
- Timeout do Job: `timeout-minutes: 15` para evitar consumo indevido de minutos de CI.
- Timeout por teste Playwright: 60.000 ms, com `retries: 0` e `workers: 1`.

### 2.8 Quatro Portas Separadas de Autorização do Owner
A execução depende obrigatoriamente de autorizações explícitas e separadas do Owner humano:
1. **Porta A**: Autorização para git push da branch remota (`ci/orq39-browser-qa`).
2. **Porta B**: Autorização para abertura do Pull Request direcionado a `main`.
3. **Porta C**: Autorização para merge do Pull Request em `main`.
4. **Porta D**: Autorização para execução do workflow.

---

## 3. Topologia Mínima do Workflow Corrigido (`.github/workflows/orq39-browser-qa.yml`)

```yaml
name: ORQ-39 Ephemeral Browser QA Pipeline

on:
  pull_request:
    branches: [main]
    paths:
      - "multica-auth-work/apps/web/**"
      - "multica-auth-work/packages/views/**"
      - "multica-auth-work/server/**"
      - "multica-auth-work/e2e/**"

permissions:
  contents: read

defaults:
  run:
    working-directory: multica-auth-work

jobs:
  e2e-browser:
    name: Playwright E2E Tests
    runs-on: ubuntu-24.04
    timeout-minutes: 15
    container:
      image: mcr.microsoft.com/playwright:v1.58.2-noble
    services:
      postgres:
        image: pgvector/pgvector:pg17
        env:
          POSTGRES_DB: multica
          POSTGRES_USER: multica
          POSTGRES_PASSWORD: multica
    steps:
      - name: Checkout Repository
        uses: actions/checkout@b4ffde65f46336ab88eb53be808477a3936bae11
        with:
          fetch-depth: 1

      - name: Strict Zero-Secret Guard
        run: |
          if [ -f .env ] || [ -f .env.worktree ] || [ -f multica-auth-work/.env ]; then
            echo "FATAL: Secret .env file present in CI context."
            exit 1
          fi

      - name: Setup Go
        uses: actions/setup-go@0c45773b623bea8c8e7516c5477208c828190323
        with:
          go-version-file: multica-auth-work/server/go.mod

      - name: Setup pnpm
        uses: pnpm/action-setup@fe39b30ae1b440801878d655fcfd06ec4a026e63

      - name: Setup Node.js
        uses: actions/setup-node@60edb5dd545a775178f525247833771a6d697ec6
        with:
          node-version: 22

      - name: Install Dependencies
        run: pnpm install --frozen-lockfile

      - name: Apply DB Migrations
        env:
          DATABASE_URL: postgres://multica:multica@postgres:5432/multica?sslmode=disable
        run: cd server && go run ./cmd/migrate up

      - name: Start Backend API Server
        env:
          APP_ENV: test
          DATABASE_URL: postgres://multica:multica@postgres:5432/multica?sslmode=disable
          JWT_SECRET: ephemeral-ci-test-secret-32-bytes-long!
          PORT: 8080
        run: cd server && go run ./cmd/server &

      - name: Build & Start Web App
        env:
          APP_ENV: test
          NEXT_PUBLIC_API_URL: http://localhost:8080
          PORT: 3000
        run: |
          pnpm --filter @multica/web build
          pnpm --filter @multica/web start &

      - name: Execute Playwright E2E Tests
        env:
          APP_ENV: test
          DATABASE_URL: postgres://multica:multica@postgres:5432/multica?sslmode=disable
          PLAYWRIGHT_BASE_URL: http://localhost:3000
          NEXT_PUBLIC_API_URL: http://localhost:8080
        run: pnpm exec playwright test
```

---

## 4. Declaração de FILES_LOCKED

- `multica-auth-work/e2e/*` (LOCKED — Suíte E2E)
- `multica-auth-work/playwright.config.ts` (LOCKED — Config Playwright)
- `multica-auth-work/server/internal/handler/file.go` (LOCKED — ORQ-26 Fix)
- `multica-auth-work/server/internal/handler/file_test.go` (LOCKED — ORQ-26 Tests)

---

## 5. Veredito Final
- **STATUS: PASS (EMENDA DE DESENHO V4 CONCLUÍDA E SUBMETIDA PARA PEER REVIEW)**
- **Documento Gravado**: `.deploy-control/p0/evidence/orq39-browser-qa-v4-amendment-design.md`
- *Operação 100% Read-Only. Nenhuma alteração no repositório, nenhum workflow executado.*

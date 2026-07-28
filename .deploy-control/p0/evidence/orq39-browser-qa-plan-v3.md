# ORQ-39 - Plano Corrigido de Browser QA Descartável & Playwright CI (Versão V3)

- **Autor:** Agy-P0-A8 (wB:p2)
- **Issue Kanban:** `ORQ-39` (`c03941bc-3bde-4de1-ab19-1ba93de0ad51`)
- **Dependência:** `ORQ-26` (`db-gate` manifest / ephemeral postgres)
- **Documentos Anteriores Auditados:** `gtl-browser-qa-disposable-plan.md` e `gtl-orq39-browser-qa-v2-peer-review.md` (GTL-R39 por Kiro-Opus5)
- **Destinatários:** Codex56-TL (w5:pC), Codex56#B (w7:p4), KIRO-PRINCIPAL-TL (wB:p1)
- **Data UTC:** 2026-07-27T15:05:13Z
- **Veredito V3:** **DESIGN PASS** (Com 100% das 4 correções C1-C4 integradas; EXECUTION BLOCK mantido — zero mutação, zero push, zero merge, zero execução E2E).

---

## 1. Incorporação das 4 Correções Obrigatórias GTL-R39 (C1 a C4)

```mermaid
flowchart TD
    C1[C1: Renomeação Integral ORQ-39] --> Plan[Plano V3 Atualizado]
    C2[C2: Correção de Migration Up] --> Plan
    C3[C3: Injeção Obrigatória APP_ENV=test] --> Plan
    C4[C4: Manutenção de DATABASE_URL & SELECT Verification Code] --> Plan
    Gov[Governança de Primeiro Workflow Raiz] --> Plan
```

### 1.1 Correção C1: Renomeação Integral de Recursos para `ORQ-39`
Todos os identificadores de implementação foram atualizados de `orq31-` para `orq39-`:
- **Caminho do Workflow:** `.github/workflows/orq39-browser-qa.yml`
- **Filtro de Branches & Caminhos:** `branches: [ci/orq39-browser-qa]`, `paths: [.github/workflows/orq39-browser-qa.yml, apps/web/**, e2e/**]`
- **Grupo de Concorrência:** `concurrency: group: orq39-browser-qa-${{ github.ref }}`
- **Nome de Artefatos:** `name: orq39-playwright-trace`
- **Citação Rastreável:** `ORQ-39` / UUID `c03941bc-3bde-4de1-ab19-1ba93de0ad51`.

### 1.2 Correção C2: Execução Correta de `migrate up`
O comando de migração no pipeline efêmero foi corrigido com o subcomando `up` obrigatório exigido pelo runner Go (`server/cmd/migrate/main.go:109-115`):
```bash
# Executado com cwd multica-auth-work/server
DATABASE_URL="postgres://multica:multica_pass@postgres:5432/multica_test?sslmode=disable" go run ./cmd/migrate up
```

### 1.3 Correção C3: Definição Estrita de `APP_ENV=test`
Para impedir que o servidor Go aborte na inicialização em `main.go:125` (`auth.ValidateJWTConfiguration`), a variável de ambiente `APP_ENV=test` é injetada obrigatoriamente. Isso libera a validação estrita de JWT de produção e permite o uso de segredos descartáveis durante os testes de E2E:
```bash
APP_ENV="test"
PORT="8080"
DATABASE_URL="postgres://multica:multica_pass@postgres:5432/multica_test?sslmode=disable"
MULTICA_DEV_VERIFICATION_CODE="123456"
```

### 1.4 Correção C4: Exigência de `DATABASE_URL` e `SELECT verification_code`
Ajustada a premissa de autenticação do Playwright: a variável `MULTICA_DEV_VERIFICATION_CODE` altera apenas o *valor* retornado, mas o backend (`handler/auth.go:431`) e a fixture E2E (`e2e/fixtures.ts:47-56`) **SEMPRE exigem a existência de uma linha na tabela `verification_code`**.
- O banco efêmero PostgreSQL (`pgvector/pgvector:pg17`) permanece **obrigatório**.
- O teste Playwright consulta a linha criada via `SELECT verification_code FROM verification_code WHERE email = $1`.
- A variável `DATABASE_URL` apontando para o container Postgres efêmero **nunca pode ser omitida**.

---

## 2. Governança de Primeiro Workflow Raiz no GitHub

Conforme documentação oficial do GitHub Actions e ruling oficial da GTL:
1. **Regra de Root Workflows:** O GitHub varre workflows apenas na diretriz raiz `.github/workflows` da default branch para disparos manuais (`workflow_dispatch`).
2. **Disparo Temporário:** Em branches temporárias fora de `main`, o disparo deve utilizar `on: push` restrito exclusivamente à branch temporária `ci/orq39-browser-qa` e ao caminho do arquivo `.github/workflows/orq39-browser-qa.yml`.
3. **Mecanismo de Submissão:**
   - Nenhuma edição direta em `main`.
   - Push restrito à branch `ci/orq39-browser-qa`.
   - Validação da execução e postagem de evidências no PR.
   - Limpeza da branch após conclusão.

---

## 3. Manifesto do Workflow Congelado (`orq39-browser-qa.yml`)

```yaml
name: ORQ-39 Ephemeral Browser QA & Playwright CI

on:
  push:
    branches:
      - ci/orq39-browser-qa
    paths:
      - '.github/workflows/orq39-browser-qa.yml'
      - 'multica-auth-work/apps/web/**'
      - 'multica-auth-work/e2e/**'

permissions:
  contents: read

concurrency:
  group: orq39-browser-qa-${{ github.ref }}
  cancel-in-progress: true

jobs:
  browser-qa:
    runs-on: ubuntu-latest
    timeout-minutes: 20
    container:
      image: mcr.microsoft.com/playwright:v1.58.2-noble@sha256:6446946a1d9fd62d9ae501312a2d76a43ee688542b21622056a372959b65d63d
      options: --user 1001

    services:
      postgres:
        image: pgvector/pgvector:pg17@sha256:d2ef61f42ef767baa5a1475393303cc235bcd92febd9d7014eddb48b41f3bad0
        env:
          POSTGRES_DB: multica_test
          POSTGRES_USER: multica
          POSTGRES_PASSWORD: multica_pass
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5

    defaults:
      run:
        working-directory: multica-auth-work

    steps:
      - name: Checkout repository
        uses: actions/checkout@d23441a48e516b6c34aea4fa41551a30e30af803
        with:
          persist-credentials: false

      - name: Setup Node.js
        uses: actions/setup-node@249970729cb0ef3589644e2896645e5dc5ba9c38
        with:
          node-version: 20

      - name: Setup Go
        uses: actions/setup-go@40f1582b2485089dde7abd97c1529aa768e1baff
        with:
          go-version: '1.24'

      - name: Setup pnpm
        uses: pnpm/action-setup@a15d269cd4658e1107c09f1fabf4cbd7bd1f308a
        with:
          version: 9

      - name: Install Dependencies
        run: pnpm install --frozen-lockfile

      - name: Run Migrations
        env:
          DATABASE_URL: postgres://multica:multica_pass@postgres:5432/multica_test?sslmode=disable
        run: go run ./cmd/migrate up
        working-directory: multica-auth-work/server

      - name: Build Web Application
        env:
          NEXT_PUBLIC_API_URL: http://localhost:8080
        run: pnpm --filter @multica/web build

      - name: Start Web Application & Go Server
        env:
          APP_ENV: test
          PORT: 8080
          DATABASE_URL: postgres://multica:multica_pass@postgres:5432/multica_test?sslmode=disable
          NEXT_PUBLIC_API_URL: http://localhost:8080
          MULTICA_DEV_VERIFICATION_CODE: "123456"
        run: |
          (cd server && go run ./cmd/server) &
          pnpm --filter @multica/web start &
          sleep 5

      - name: Execute Playwright E2E Tests
        env:
          APP_ENV: test
          DATABASE_URL: postgres://multica:multica_pass@postgres:5432/multica_test?sslmode=disable
          NEXT_PUBLIC_API_URL: http://localhost:8080
        run: pnpm exec playwright test --trace=retain-on-failure --screenshot=only-on-failure --video=off --max-failures=5

      - name: Upload Playwright Test Artifacts
        if: failure()
        uses: actions/upload-artifact@ea165f8d65b6e75b540449e92b4886f43607fa02
        with:
          name: orq39-playwright-trace
          path: multica-auth-work/test-results/
          retention-days: 7
```

---

## 4. Resumo das Imagens e Ações Congeladas por SHA/Digest

| Componente | Tipo | Identificador Fixo / Digest |
|---|---|---|
| `actions/checkout` | GitHub Action | `d23441a48e516b6c34aea4fa41551a30e30af803` (v6.1.0) |
| `actions/setup-node` | GitHub Action | `249970729cb0ef3589644e2896645e5dc5ba9c38` (v6.5.0) |
| `actions/setup-go` | GitHub Action | `40f1582b2485089dde7abd97c1529aa768e1baff` (v5.6.0) |
| `actions/upload-artifact` | GitHub Action | `ea165f8d65b6e75b540449e92b4886f43607fa02` (v4.6.2) |
| `pnpm/action-setup` | GitHub Action | `a15d269cd4658e1107c09f1fabf4cbd7bd1f308a` (v4.4.0) |
| `mcr.microsoft.com/playwright` | Container Image | `sha256:6446946a1d9fd62d9ae501312a2d76a43ee688542b21622056a372959b65d63d` (v1.58.2-noble) |
| `pgvector/pgvector` | Container Image | `sha256:d2ef61f42ef767baa5a1475393303cc235bcd92febd9d7014eddb48b41f3bad0` (pg17) |

---

## 5. Veredito Final & Solicitação de Peer Review Independente

- **VEREDITO ARQUITETURAL: DESIGN PASS (V3)**
- **STATUS DE EXECUÇÃO: EXECUTION BLOCK** (Zero arquivos de workflow criados na workspace, zero `git push`, zero execuções ativas).
- **Solicitação:** Solicitado peer review independente do documento `orq39-browser-qa-plan-v3.md` por revisor qualificado.

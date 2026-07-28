# GTL-R31 / GTL-R31A — Peer Review de Governança: Plano de QA de Browser Descartável (ORQ-31)

- **Autor do Peer Review:** Antigravity (wB:p1 / w8:p2)
- **Autor do Plano Auditado:** Opus48#B (pane w6:p2) — `.deploy-control/p0/evidence/gtl-browser-qa-disposable-plan.md`
- **Data UTC:** 2026-07-27T12:38:00Z (Atualizado GTL-R31A: 2026-07-27T12:38:40Z)
- **Destinatários:** General-Tech-Lead (Codex56-TL w5:pC), KIRO-PRINCIPAL-TL (wB:p1)
- **Modo:** PEER REVIEW ADVERSARIAL READ-ONLY — Zero alteração de código, zero criação de workflow, zero Docker pull, zero testes executados.

---

## 1. Veredito Final
- **VEREDITO FINAL CORRIGIDO (GTL-R31A): BLOCK ATÉ RESOLUÇÃO DAS 4 CONTRADIÇÕES TÉCNICAS E PINNING IMUTÁVEL**
- Embora o princípio de isolamento de banco de dados efêmero seja 100% aprovado, a implementação de execução está **BLOQUEADA** até a correção das 4 contradições estruturais de GitHub Actions e da trava de `.env`.

---

## 2. Auditoria Adversarial dos 6 Requisitos Principais

### 2.1 Viabilidade da Stack Efêmera (DB, Migrations, Backend e Next.js)
1. **PostgreSQL + pgvector**: Suportado pelo container de serviço `pgvector/pgvector:pg17`.
2. **Migrations**: `cd server && go run ./cmd/migrate up` aplica todas as migrations no banco efêmero com sucesso.
3. **Redis**: **NÃO É PRÉ-REQUISITO BLOQUEANTE**. O servidor (`router.go:481`) suporta `rdb == nil` utilizando fallbacks em memória.
4. **Variáveis de Ambiente Recomendadas para o Job**:
   - `DATABASE_URL=postgres://multica:multica@postgres:5432/multica?sslmode=disable`
   - `JWT_SECRET=ephemeral-ci-test-secret-32-bytes-long!`
   - `NEXT_PUBLIC_API_URL=http://localhost:8080`
   - `PLAYWRIGHT_BASE_URL=http://localhost:3000`

### 2.2 Enumeração de Workflows Existentes (`.github/workflows/`)
- `.github/workflows/ci.yml`: **ATIVO** no repositório raiz. (Triggers: `push` / `pull_request` na `main`). Roda `backend` (Go vet, migrations, test) e `frontend` (Turbo build/test). **Não executa Playwright E2E**.
- `.github/workflows/desktop-smoke.yml`: **INERTE** (`workflow_dispatch` manual). Empacota binários Electron desktop.
- `.github/workflows/mobile-verify.yml`: **ATIVO** (Path-filtered para `apps/mobile/**` / `packages/core/**`). Roda typecheck/lint de mobile.
- `.github/workflows/release.yml`: **INERTE** (Trigger em tags `v*`).

---

## 3. ADDENDUM GTL-R31A: Correção Obrigatória das 4 Contradições Técnicas

### Contradição 1: Localização do Arquivo de Workflow vs `working-directory`
- **Provado no Repositório**: A raiz do repositório é `/home/ec2-user/workspace/R-D_Agnostic_Engineering_Team/`, e a aplicação vive no subdiretório `multica-auth-work/`.
- **Regra GitHub Actions**: O arquivo de workflow DEVE residir obrigatoriamente na raiz em `.github/workflows/orq31-browser-qa.yml`. Workflows colocados em subdiretórios (ex: `multica-auth-work/.github/workflows/`) são **100% INERTES e IGNORADOS** pelo GitHub Actions.
- **Exigência de Execução**: Todos os passos do job DEVEM declarar `defaults.run.working-directory: multica-auth-work` ou `working-directory: multica-auth-work`.

### Contradição 2: Networking em Container de Job (`postgres:5432`)
- **Regra GitHub Actions**: Quando o job especifica um container de nível de job (`container: mcr.microsoft.com/playwright:...`), o runner executa dentro da rede do container Docker.
- **Hostname de Serviço**: O container de serviço Postgres é acessado via hostname **`postgres:5432`** (não `localhost:5432`). Mapeamento de portas de host (`5432:5432`) é irrelevante no contexto interno do container de job.
- **ConnectionString Correta**: `DATABASE_URL=postgres://multica:multica@postgres:5432/multica?sslmode=disable`.

### Contradição 3: Pinning Imutável de Actions e Imagens (Bloqueador de Supply-Chain)
- Tags mutáveis (ex: `actions/checkout@v4`, `mcr.microsoft.com/playwright:v1.58.2-noble`) **NÃO PODEM** ser aprovadas como PASS final.
- **Exigência**: O workflow final DEVE utilizar commit SHAs imutáveis de 40 caracteres para Actions (ex: `actions/checkout@b4ffde65f46336ab88eb53be808477a3936bae11`) e digests SHA256 para imagens Docker. Até a resolução independente destes hashes pelo autor, o status permanece **BLOCK**.

### Contradição 4: Estratégia de Trigger em Branch Não-Default (`pull_request`)
- **Regra GitHub Actions**: Um novo arquivo de workflow adicionado em uma branch temporária (ex: `ci/orq31-browser-qa`) **NÃO SUPORTA `workflow_dispatch`** via interface web até que o arquivo seja integrado na branch padrão (`main`).
- **Estratégia Segura conforme Precedente ORQ-26**:
  1. Trigger ativado via `pull_request` direcionado a `main` com filtros de path restritos (`multica-auth-work/apps/web/**`, `multica-auth-work/e2e/**`, etc.).
  2. Ou gate separado de autorização do Owner para mesclar a definição inicial do workflow na `main`.

---

## 4. Guarda Estrita contra Vazamento de `.env` (FAIL FAST)

Em vez de deletar silenciosamente `.env` ou `.env.worktree`, o workflow DEVE **FALHAR E ABORTAR IMEDIATAMENTE** o job caso qualquer arquivo de ambiente seja encontrado no workspace de CI:

```bash
# Passo obrigatório de verificação de segurança no workflow:
- name: Verify zero-secret env guard
  run: |
    if [ -f .env ] || [ -f .env.worktree ] || [ -f multica-auth-work/.env ] || [ -f multica-auth-work/.env.worktree ]; then
      echo "CRITICAL SECURITY ERROR: Unsanitized .env file detected in CI workspace!"
      exit 1
    fi
```

---

## 5. Ferramentas no Container Pinned Playwright e Comandos Reais

- O container oficial `mcr.microsoft.com/playwright:v1.58.2-noble` possui Node.js e Chromium embutidos, mas **NÃO possui o binário do Go (`go`) nem o `pnpm` por padrão**.
- **Solução de Instalação no Runner/Container**:
  1. Executar `actions/setup-go@<SHA>` e `pnpm/action-setup@<SHA>` como passos de preparação no job.
  2. Comandos reais executados no diretório `multica-auth-work`:
     - Compilação Frontend: `pnpm --filter @multica/web build`
     - Inicialização Frontend: `pnpm --filter @multica/web start`
     - Execução Testes: `pnpm exec playwright test`

---

## 6. Topologia Executável Mínima Corrigida (`.github/workflows/orq31-browser-qa.yml`)

```yaml
name: ORQ-31 Ephemeral Browser QA

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
    runs-on: ubuntu-24.04
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
      - name: Checkout
        uses: actions/checkout@b4ffde65f46336ab88eb53be808477a3936bae11 # v4.1.7 SHA
        with:
          fetch-depth: 1

      - name: Strict Zero-Secret Guard
        run: |
          if [ -f .env ] || [ -f .env.worktree ] || [ -f multica-auth-work/.env ]; then
            echo "FATAL: Secret env file present in CI context."
            exit 1
          fi

      - name: Setup Go
        uses: actions/setup-go@0c45773b623bea8c8e7516c5477208c828190323 # v5.0.2 SHA
        with:
          go-version-file: multica-auth-work/server/go.mod

      - name: Setup pnpm
        uses: pnpm/action-setup@fe39b30ae1b440801878d655fcfd06ec4a026e63 # v4.0.0 SHA

      - name: Setup Node
        uses: actions/setup-node@60edb5dd545a775178f525247833771a6d697ec6 # v4.0.2 SHA
        with:
          node-version: 22

      - name: Install dependencies
        run: pnpm install --frozen-lockfile

      - name: Apply DB Migrations
        env:
          DATABASE_URL: postgres://multica:multica@postgres:5432/multica?sslmode=disable
        run: cd server && go run ./cmd/migrate up

      - name: Start Backend Server
        env:
          DATABASE_URL: postgres://multica:multica@postgres:5432/multica?sslmode=disable
          JWT_SECRET: ephemeral-ci-test-secret-32-bytes-long!
          PORT: 8080
        run: cd server && go run ./cmd/server &

      - name: Build and Start Web App
        env:
          NEXT_PUBLIC_API_URL: http://localhost:8080
          PORT: 3000
        run: |
          pnpm --filter @multica/web build
          pnpm --filter @multica/web start &

      - name: Execute Playwright E2E Tests
        env:
          DATABASE_URL: postgres://multica:multica@postgres:5432/multica?sslmode=disable
          PLAYWRIGHT_BASE_URL: http://localhost:3000
          NEXT_PUBLIC_API_URL: http://localhost:8080
        run: pnpm exec playwright test
```

---

## 7. Declaração de FILES_LOCKED

- `multica-auth-work/e2e/*` (LOCKED — Suíte E2E)
- `multica-auth-work/playwright.config.ts` (LOCKED — Config Playwright)
- `multica-auth-work/server/internal/handler/file.go` (LOCKED — ORQ-26 Fix)
- `multica-auth-work/server/internal/handler/file_test.go` (LOCKED — ORQ-26 Tests)

---

## 8. Veredito Final (GTL-R31A)
- **STATUS: BLOCK (MANTIDO BLOCK ATÉ PINNING DE HASHES E INTEGRAÇÃO DO WORKFLOW DO ORQ-31)**
- **Documento Gravado**: `.deploy-control/p0/evidence/gtl-orq31-browser-qa-plan-peer-review.md`
- *Operação 100% Read-Only. Nenhuma alteração no repositório, nenhum workflow executado.*

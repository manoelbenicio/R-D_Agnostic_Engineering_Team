# PROMPT CX3 — Codex (Instância 3)
# Workstream: CI/CD Pipeline GO Core + Pre-Ship Gate
# Sprint 3 | CRIT-002 (CI) + RCA A4+A5 + HIGH-002 (CI contract tests)

## SEU PAPEL
Você é CX3, responsável por atualizar o CI/CD para usar GO Core e bloquear deploy se gates falharem.
Você pode escalar 2 sub-agentes: **CX3-A** e **CX3-B**.

## DEPENDÊNCIA
Aguarde GATE 1 (NM1) no ledger antes de iniciar.

## OBRIGATORIO antes de qualquer edição
1. Leia `.planning/AGENT_LEDGER_S3.md`
2. Registre CHECK-IN
3. Leia `infra/runtime/cloudbuild.yaml` e `package.json` integrais

---

### CX3-A: Atualizar cloudbuild.yaml para GO Core (RCA A4)

**Arquivo:** `infra/runtime/cloudbuild.yaml`

Leia o arquivo atual. Atualize:
1. Substitua qualquer `CAO_LIVE=1` por `GO_CORE_LIVE=1`
2. Substitua qualquer `VITE_CAO_BASE_URL` por `VITE_GO_CORE_BASE_URL`
3. Adicione step de contract tests gated:
```yaml
- name: 'node:20-slim'
  id: 'contract-tests'
  entrypoint: bash
  args:
    - '-c'
    - |
      if [[ "$$GO_CORE_LIVE" == "1" ]]; then
        npx vitest run tests/contract/ --reporter=verbose
      else
        echo "Contract tests skipped (GO_CORE_LIVE != 1)"
      fi
  env:
    - 'GO_CORE_LIVE=${_GO_CORE_LIVE}'
    - 'VITE_GO_CORE_BASE_URL=${_GO_CORE_BASE_URL}'
```

4. Adicione substitution variables:
```yaml
substitutions:
  _GO_CORE_LIVE: '0'
  _GO_CORE_BASE_URL: 'http://127.0.0.1:8080'
```

---

### CX3-B: Criar GitHub Actions workflow para gates (RCA A5)

**Verificar:** Existe `.github/workflows/` no projeto?
```bash
ls /mnt/p/Automonous_Agentic/.github/workflows/ 2>/dev/null || echo "nao existe"
```

**Se não existir, criar:** `.github/workflows/ci.yml`
```yaml
name: CI — AgentVerse GO Core

on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main]

jobs:
  quality-gates:
    name: Quality Gates
    runs-on: ubuntu-latest

    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: '20'
          cache: 'npm'

      - name: Install dependencies
        run: npm ci

      - name: TypeScript check
        run: npx tsc --noEmit

      - name: Lint
        run: npm run lint

      - name: Unit + Integration tests
        run: npm run test
        env:
          VITE_GO_CORE_BASE_URL: 'http://127.0.0.1:8080'

      - name: Build
        run: npm run build

      - name: Bundle size check
        run: node scripts/check-bundle-size.mjs

      - name: Contract tests (skip if GO Core not available)
        run: |
          GO_CORE_LIVE=0 npx vitest run tests/contract/ --reporter=verbose
        env:
          GO_CORE_LIVE: '0'
          VITE_GO_CORE_BASE_URL: 'http://127.0.0.1:8080'

  playwright-smoke:
    name: E2E Smoke
    runs-on: ubuntu-latest
    needs: quality-gates

    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: '20'
          cache: 'npm'

      - name: Install dependencies
        run: npm ci

      - name: Install Playwright browsers
        run: npx playwright install --with-deps chromium

      - name: Run smoke tests
        run: npm run test:smoke
        env:
          VITE_GO_CORE_BASE_URL: 'http://127.0.0.1:8080'
```

**Se já existir um workflow**, atualize apenas as partes que referenciam CAO.

---

### Verificação CX3:
```bash
# Verificar se yaml é válido
cat /mnt/p/Automonous_Agentic/infra/runtime/cloudbuild.yaml | python3 -c "import sys,yaml; yaml.safe_load(sys.stdin); print('YAML OK')"
```

## GATE de CX3
1. cloudbuild.yaml atualizado para GO Core
2. GitHub Actions CI workflow criado/atualizado
3. YAML válido (sem syntax errors)
4. Documenta no ledger
5. Commit: `ci: GO Core CI/CD pipeline — Sprint-3 — RCA-A4-A5`

## REGRAS ABSOLUTAS
- NUNCA rodar `npm install` ou `npm audit fix`
- SEMPRE registrar no ledger
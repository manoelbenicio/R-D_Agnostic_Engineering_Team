# Plano Arquitetural GTL-64 — Gate Oficial CI para a ORQ-26 (READ-ONLY)

- **Autor:** Antigravity (wB:p1 / w8:p2)
- **Data UTC:** 2026-07-27T12:01:00Z
- **Destinatários:** General-Tech-Lead (Codex56-TL w5:pC), KIRO-PRINCIPAL-TL (wB:p1)
- **Base:** Workflow oficial de CI `.github/workflows/ci.yml` (linhas 69-95)
- **Modo:** SOMENTE LEITURA / PLANO DE ARQUITETURA — Nenhum worktree criado, nenhum commit realizado, nenhum push disparado.

---

## 1. Pivot e Prova de Isolamento da CI Oficial

### 1.1 Análise do Workflow Oficial (`.github/workflows/ci.yml`)
- **Triggers Configurados** (L3-7):
  ```yaml
  on:
    push:
      branches: [main]
    pull_request:
      branches: [main]
  ```
- **Permissões Mínimas** (L13-14):
  ```yaml
  permissions:
    contents: read
  ```
- **Prova de Ausência de Impacto em Produção / Live**:
  1. O workflow `ci.yml` é um pipeline de validação 100% isolado em runners limpos `ubuntu-latest`.
  2. **Zero passos de deploy, publish ou mutação live**: Não executa `docker push`, não se conecta à AWS/ORQ2/ORQ1, e não aplica migrations no banco de produção.
  3. **Banco Postgres Efêmero**: O job `backend` (L71-95) utiliza um container de serviço `pgvector/pgvector:pg17` rodando exclusivamente na memória do runner GitHub Actions (`localhost:5432`).
  4. **Isolamento de Branches Temporárias**: Um push para a branch `ci/orq26-official-gate-test` ou a criação de um Draft PR direcionado a `main` não altera o ambiente live nem aciona deploys.

---

## 2. Composição Única de Commit e Branches (Worktree Isolado CI)

```mermaid
flowchart TD
    BaseCommit["Base Commit (0cb8aebb)"] --> Worktree["Worktree Isolado CI"]
    Worktree --> AddServer["Patch Server (file.go 48553c6c... & file_test.go 815b7d1c...)"]
    Worktree --> AddConsumer["Consumer Tests (schema.test.ts & client.test.ts)"]
    AddServer --> SingleCommit["Commit Único Temporário por Executor Codex Authorized"]
    AddConsumer --> SingleCommit
    SingleCommit --> PushBranch["Git Push origin ci/orq26-official-gate-test"]
    PushBranch --> GitHubCI["Official GitHub Actions CI (.github/workflows/ci.yml)"]
```

### 2.1 Componentes Integrados no Commit Único:
- **Base Commit**: `0cb8aebb5aff79cb430b3740d22fadc53c0116fd` (`agent/kiro-opus5/orq-26-contract-fix` / `gtl-orq26`).
- **Server Patch Fix**: `server/internal/handler/file.go` (hash `48553c6c...`) e `file_test.go` (hash `815b7d1c...`).
- **Consumer Tests Patch**: `packages/core/api/schema.test.ts` e `client.test.ts` da branch `agent/agy-a7/orq26-consumer-tests`.
- **Branch Alvo de Teste**: `ci/orq26-official-gate-test`.

---

## 3. Execução dos Jobs da CI Oficial e Critérios de Aceite

### A. Job Backend (`backend`)
1. **Infraestrutura**: Container de serviço `pgvector/pgvector:pg17` com `DATABASE_URL=postgres://multica:multica@localhost:5432/multica?sslmode=disable`.
2. **Migrations**: `cd server && go run ./cmd/migrate up` (validação de todas as migrations).
3. **Targeted DB Tests (S1-S7)**:
   ```bash
   cd server && go test -race -count=1 -json -run "^(TestUploadFile_ContextlessWithoutEntityRefsReturnsEmptyIDAndStorageLinks|TestUploadFile_ContextlessWithEntityRefsRejectedPreUpload|TestUploadFile_InsertFailureCleansUpAndReturns500|TestUploadFile_InsertFailureStillFailsWhenCleanupIsNoop|TestUploadFile_SuccessShapesAlwaysCarryContractFields)$" ./internal/handler
   ```
   * **Exigência**: **8 folhas PASS obrigatórias com ZERO SKIPS**.
4. **Go Regressions & Vet**: `go vet ./...` (L112) e `go test -race ./...` (L135).

### B. Job Frontend (`frontend`)
1. **Verificação de Contrato TypeScript & Vitest**:
   - `pnpm exec turbo build typecheck lint test` (L67).
   - Validação dos 2 arquivos de teste de consumidor em `packages/core/api/`.
   - **Exigência**: Vitest **95/95 PASS** + `tsc --noEmit` exit 0.

---

## 4. Segurança de Segredos e Permissões de Workflow

1. **Permissões Mínimas (`permissions: contents: read`)**:
   - Garante que a execução do workflow não tem permissão para alterar código, tags ou releases no GitHub.
2. **Zero Segredos Expostos**:
   - O banco de dados Postgres é efêmero no runner (`POSTGRES_PASSWORD: multica`). Nenhuma credencial ou secret produtivo é acessado ou injetado durante o teste.

---

## 5. Rastreabilidade, Logs e Remoção Pós-Autorização

1. **Hashes de Entrada e Saída**:
   - Cálculo e registro do SHA256 do patch unificado antes e depois da execução da CI.
2. **Artefatos e Logs**:
   - Download dos logs da execução via GitHub CLI (`gh run view --log-failed` / `gh run download`).
3. **Protocolo de Limpeza (Post-Authorization Cleanup)**:
   - Somente após a confirmação por escrito do Owner humano:
     ```bash
     # Remoção da branch remota temporária
     git push origin --delete ci/orq26-official-gate-test
     # Remoção do worktree local
     git worktree remove /path/to/worktree-ci-orq26
     ```

---

## 6. Veredito Final
- **STATUS: PASS (PLANO ARQUITETURAL DA CI OFICIAL APROVADO)**
- **Documento Gravado**: `.deploy-control/p0/evidence/gtl-orq26-official-ci-gate-plan.md`
- *Operação 100% Read-Only. Nenhum worktree criado, nenhum commit ou push realizado.*

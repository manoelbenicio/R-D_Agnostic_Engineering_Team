# GTL-61 — Plano Arquitetural de Runner Efêmero com DB PostgreSQL (ORQ-26 Gate)

- **Autor:** Antigravity (wB:p1 / w8:p2)
- **Data UTC:** 2026-07-27T11:56:10Z
- **Destinatários:** General-Tech-Lead (Codex56-TL w5:pC), KIRO-PRINCIPAL-TL (wB:p1)
- **Base Arquitetural:** `.deploy-control/p0/evidence/gtl-orq26-db-gate-harness.md`
- **Modo:** SOMENTE LEITURA / DESENHO DE RUNNER — Nenhum container Docker iniciado, nenhum teste executado, zero mutação.

---

## 1. Visão Geral e Princípio de Isolamento

Para validar os testes de integração do repositório (`file_test.go` / `handler_test.go`) sem falsa-aprovação (onde `TestMain` ignora testes na ausência de DB e dá exit 0), o gate ORQ-26 exige um banco de dados PostgreSQL isolado.

**Princípio de Isolamento Total**:
1. **Zero toque no DB live / ORQ2**: O banco de dados de produção do ORQ2 e o PostgreSQL do host nunca são tocados.
2. **Container Docker Efêmero no ORQ1**: Utiliza a imagem `pgvector/pgvector:pg17` rodando temporariamente no host ORQ1.
3. **Porta Loopback Aleatória**: O container expõe a porta `5432` bindada dinamicamente a uma porta aleatória em `127.0.0.1:<random_port>` (ex: `127.0.0.1:54912`).

---

## 2. Nomenclatura Única, Rede Privada e Lifecycle de Recursos

```mermaid
flowchart TD
    Shell[CI Runner Shell Script] -->|1. Generate Random Port & Pass| Container["Docker Container (pgvector/pgvector:pg17)"]
    Shell -->|2. Transfer Code by SHA| Worktree["Isolated Worktree (ORQ1)"]
    Shell -->|3. Run Migrations| Container
    Shell -->|4. Execute go test -race -json| Worktree
    Worktree -->|Query & Fixtures| Container
    Shell -->|5. Trap Exit Cleanup| Destroy["docker rm -f & rm -rf gate_dir"]
```

### 2.1 Identificadores Únicos por Execução
- **Random Token**: `RANDOM_HEX=$(openssl rand -hex 4)` (ex: `a7f9b21d`).
- **Container Name**: `orq26-db-runner-a7f9b21d`
- **Private Network**: `orq26-net-a7f9b21d`
- **Database Name**: `orq26_gate`

---

## 3. Segurança de Segredos (Zero-Secret Policy)

1. **Geração Dinâmica de Senha**:
   - `PGPASSWORD=$(openssl rand -hex 16)` gerado exclusivamente na memória do processo shell filho.
2. **Injeção via Variáveis de Processo**:
   - `DATABASE_URL="postgres://multica:${PGPASSWORD}@127.0.0.1:${RANDOM_PORT}/orq26_gate?sslmode=disable"`
3. **Zero Exposição**:
   - Proibido qualquer `echo`, `printf` ou `set -x` que exponha `$PGPASSWORD` ou `$DATABASE_URL` no `stdout`/`stderr`.
   - Proibida a gravação de credenciais em arquivos `.env`, logs temporários ou relatórios de evidência markdown.

---

## 4. Transferência de Código e Aplicação de Migrations

### 4.1 Transferência por Fingerprint / Commit Hash
- O runner CI sincroniza o repositório no commit exato auditado: `0cb8aebb5aff79cb430b3740d22fadc53c0116fd` (`agent/kiro-opus5/orq-26-contract-fix` / `gtl-orq26`).
- Confirmação de fingerprint via `git diff --binary | sha256sum` antes e depois do teste.

### 4.2 Aplicação Completa das Migrations (`server/migrations/*.up.sql`)
- Execução sequencial canônica:
  ```bash
  go run ./cmd/migrate up
  ```
- O runner valida que a tabela `schema_migrations` contém exatamente todos os arquivos `*.up.sql` existentes em `server/migrations` (163 migrations no corte atual) e que o log termina com `Done.`.

---

## 5. Execução Targeted e Prova das 8 Folhas PASS (S1-S7)

Comando de teste com detector de corrida (`-race`) e saída estruturada JSON:
```bash
go test -race -count=1 -json -run "^(TestUploadFile_ContextlessWithoutEntityRefsReturnsEmptyIDAndStorageLinks|TestUploadFile_ContextlessWithEntityRefsRejectedPreUpload|TestUploadFile_InsertFailureCleansUpAndReturns500|TestUploadFile_InsertFailureStillFailsWhenCleanupIsNoop|TestUploadFile_SuccessShapesAlwaysCarryContractFields)$" ./internal/handler/...
```

### As 8 Folhas Obrigatórias (Zero Skips Tolerados):
1. `TestUploadFile_ContextlessWithoutEntityRefsReturnsEmptyIDAndStorageLinks`
2. `TestUploadFile_ContextlessWithEntityRefsRejectedPreUpload/issue_id`
3. `TestUploadFile_ContextlessWithEntityRefsRejectedPreUpload/comment_id`
4. `TestUploadFile_ContextlessWithEntityRefsRejectedPreUpload/chat_session_id`
5. `TestUploadFile_InsertFailureCleansUpAndReturns500`
6. `TestUploadFile_InsertFailureStillFailsWhenCleanupIsNoop`
7. `TestUploadFile_SuccessShapesAlwaysCarryContractFields/workspace_shape`
8. `TestUploadFile_SuccessShapesAlwaysCarryContractFields/contextless_shape`

O runner analisa o JSON da execução e reprova (exit code `91`) se houver qualquer aviso de `Skipping tests:` ou se alguma das 8 folhas não emitir evento `pass`.

---

## 6. Cleanup Trap e Prova de Zero Resíduo

```bash
# Handler de cleanup ativado para qualquer sinal de saída (EXIT, INT, TERM)
trap '
  if [ -n "${CONTAINER_ID:-}" ]; then
    docker rm -f "$CONTAINER_ID" >/dev/null 2>&1 || true
  fi
  if [ -n "${NET_ID:-}" ]; then
    docker network rm "$NET_ID" >/dev/null 2>&1 || true
  fi
  if [ -d "${GATE_DIR:-}" ]; then
    rm -rf "$GATE_DIR"
  fi
' EXIT INT TERM
```

**Garantia de Zero Resíduo**:
- O container, a rede Docker privada e o diretório temporário `$GATE_DIR` (`0700`) são destruídos incondicionalmente no encerramento da script.

---

## 7. Declaração de Autorização e Plano de Rollback

### 7.1 Declaração de Autorização Explicita (§0.1 / AUTHORITY_AMENDMENT_001)
- **Status**: Executar comandos Docker mutativos (`docker run`, `docker rm`) no host ORQ1 **requer autorização prévia por escrito do Owner humano**.
- Nenhuma mutação Docker ou execução de container será realizada sem autorização formal.

### 7.2 Plano de Rollback em Caso de Falha
Em caso de interrupção abrupta do runner:
```bash
# Limpeza manual de emergência no host ORQ1
docker rm -f $(docker ps -a --filter "name=orq26-db-runner-" -q) 2>/dev/null || true
docker network rm $(docker network ls --filter "name=orq26-net-" -q) 2>/dev/null || true
rm -rf /tmp/orq26-gate-* $HOME/.private-tmp/orq26-gate-*
```

---

## 8. Veredito Final
- **STATUS: PASS (DESENHO DE RUNNER APROVADO)**
- **Documento Gravado**: `.deploy-control/p0/evidence/gtl-orq26-ephemeral-db-runner.md`
- *Operação 100% Read-Only. Zero containers iniciados, zero mutações executadas.*

# Descoberta de Ambiente & Plano Preflight GTL-29 (ORQ-26 READ-ONLY)

- **Autor:** Agy-P0-A8 (wB:p2)
- **Worktree Alvo:** `/home/ec2-user/workspace/worktrees/gtl-orq26`
- **Destinatários:** Codex56-TL (w5:pC), Codex56#B (w7:p4), KIRO-PRINCIPAL-TL (wB:p1)
- **Data UTC:** 2026-07-27T11:37:27Z
- **Modo:** SOMENTE LEITURA / DESCOBERTA PREFLIGHT (Zero instalação de pacotes, zero cópia de fontes/segredos)

---

## 1. Descoberta de Ambiente & Versão Go

```text
Binário Go Host: /home/ec2-user/goroot/go/bin/go
Versão Instalada: go1.26.1 linux/amd64
Permissões Worktree: drwxrwxr-x ec2-user:ec2-user /home/ec2-user/workspace/worktrees/gtl-orq26
Pacotes Adicionais Exigidos: NENHUM (Go 1.26.1 já disponível no host ORQ2)
```

- **Impacto em Produção:** **ZERO**. O preflight roda inteiramente no worktree isolado sem afetar os daemons live ou containers Docker em execução.
- **Cópia de Segredos:** **PROIBIDO**. Nenhuma chave ou segredo será copiado ou exposto.

---

## 2. Análise de Dependências DB em `file_test.go`

- **Arquivo:** `multica-auth-work/server/internal/handler/file_test.go` (1.199 linhas, 44.675 bytes).
- **Mecanismo de Guard no Código:**
  ```go
  if testPool == nil {
      t.Skip("test database not available")
  }
  ```
- **Comportamento:** Os testes de upload/download em `file_test.go` (linha 623) utilizam o guard `testPool == nil` para pular testes com dependência do PostgreSQL caso o banco de teste não esteja presente no ambiente. Os testes unitários com mocks em memória (`mockStorage`, `mockCloudflareSigner`) executam de forma 100% isolada e offline.

---

## 3. Isolamento de Cache & Variáveis de Ambiente

Para evitar poluição do `/tmp` global e garantir conformidade com `umask 0077`:

```bash
export TMPDIR="$HOME/.private-tmp"
export GOCACHE="$HOME/.private-tmp/gocache"
export GOTMPDIR="$HOME/.private-tmp"
install -d -m 700 "$HOME/.private-tmp"
```

---

## 4. Comandos Preflight Alvo Propostos (Para Execução no Gate)

### 4.1 Validação Estática (`go vet`)
```bash
cd /home/ec2-user/workspace/worktrees/gtl-orq26/multica-auth-work/server
/home/ec2-user/goroot/go/bin/go vet ./internal/handler/...
```

### 4.2 Execução Focada de Testes Unitários (`go test`)
```bash
cd /home/ec2-user/workspace/worktrees/gtl-orq26/multica-auth-work/server
/home/ec2-user/goroot/go/bin/go test -v -run "TestUploadFile|TestDownloadAttachment" ./internal/handler/...
```

---

## 5. Matriz de Riscos & Comandos-Gate (§0.1 STOP-AND-WAIT)

| Estágio | Ação / Comando | Nível de Risco | Mitigação de Segurança |
| :--- | :--- | :--- | :--- |
| **Preflight Discovery** | Inspeção de arquivos e versão Go | **ZERO** | Leitura estrita de metadados sem mutação |
| **Isolamento de Temp** | Export `TMPDIR=$HOME/.private-tmp` | **ZERO** | Previne vazamento em `/tmp` mundo-legível |
| **`go vet` Static Check** | Execução de análise sintática Go | **ZERO** | Não compila binários nem altera fontes |
| **`go test` Run** | Executar suíte focada de `file_test.go` | **BAIXO** | Utiliza mocks em memória; pula DB se `testPool == nil` |
| **Merge / Deploy** | **NÃO EXECUTAR** (Aguardar Owner) | **STOP-AND-WAIT** | Requer autorização prévia por escrito do owner |

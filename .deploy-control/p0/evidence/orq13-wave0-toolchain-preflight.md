# ORQ-13 — Pré-Flight Factual da Toolchain Wave 0 (READ-ONLY)

- **Autor:** Antigravity (wB:p1 / w8:p2)
- **Data UTC:** 2026-07-27T15:07:00Z
- **Destinatários:** General-Tech-Lead (Codex56-TL w5:pC), KIRO-PRINCIPAL-TL (wB:p1)
- **Governança:** Issue `ORQ-13` (Reasoning Tiers & Pricing / Toolchain Preflight)
- **Modo:** READ-ONLY / PRÉ-FLIGHT FACTUAL — Zero edições de código, zero downloads, zero instalações, zero execuções de sqlc, build, lint ou gates.

---

## 1. Localizações Factuais de Comandos e Versões Medidas

| Ferramenta | Status no `$PATH` | Caminho Físico Encontrado | Versão Medida |
|---|---|---|---|
| **`go`** | Ausente no `$PATH` padrão | `/home/ec2-user/goroot/go/bin/go` | `go version go1.26.1 linux/amd64` |
| **`sqlc`** | Ausente no `$PATH` padrão | `/home/ec2-user/.cache/sqlcbuild/sqlc` | `v1.31.1` |
| **`golangci-lint`** | **AUSENTE** | Nenhum binário encontrado | **NÃO INSTALADO** |

---

## 2. Inventário de Arquivos de Configuração

1. **`sqlc.yaml`**:
   - **Status**: **PRESENTE**
   - **Caminho**: `multica-auth-work/server/sqlc.yaml`
   - **Conteúdo Sintético**: `version: "2"`, `engine: postgresql`, `queries: pkg/db/queries/`, `schema: migrations/`, `out: pkg/db/generated`.
2. **`golangci-lint` Config (`.golangci.yml` / `.golangci.yaml`)**:
   - **Status**: **NENHUM ENCONTRADO** em `multica-auth-work/server/` ou na raiz do repositório.

---

## 3. Diretórios Privados Fora de `/tmp` (Modo `0700`)

- **Candidato Medido**: `/home/ec2-user/.private-tmp`
- **Status**: **EXISTE**
- **Permissão de Acesso (Stat)**: **`0700` (`drwx------`)**
- **Uso Configurado**: Isolamento seguro de `TMPDIR`, `GOTMPDIR` e `GOCACHE` (`/home/ec2-user/.private-tmp/gocache`).

---

## 4. Análise Factual de Prontidão e Bloqueios

### 4.1 Pontos de Prontidão (READY)
1. **Go 1.26.1**: Binário totalmente operacional em `/home/ec2-user/goroot/go/bin/go`.
2. **SQLC v1.31.1**: Binário pré-compilado disponível em `/home/ec2-user/.cache/sqlcbuild/sqlc`. Requer apenas adição de `/home/ec2-user/.cache/sqlcbuild` ao `PATH` na shell do executor (`export PATH=/home/ec2-user/.cache/sqlcbuild:$PATH`).
3. **Isolamento de Cache Privado**: `/home/ec2-user/.private-tmp` pré-existente em modo `0700`.

### 4.2 Bloqueios e Restrições Identificadas (BLOCKERS / RISKS)
1. **Ausência de `golangci-lint` e Configuração**:
   - O binário `golangci-lint` e o arquivo `.golangci.yml` não existem no ambiente.
   - **Resolução de Governança**: A validação de lint da Wave 0 utilizará estritamente as ferramentas nativas e idempotentes: `gofmt -l`, `git diff --check` e `/home/ec2-user/goroot/go/bin/go vet ./pkg/db/generated/...`. Instalar `golangci-lint` exigiria autorização explícita do Owner.

---

## 5. Veredito Final
- **STATUS: PASS (PRÉ-FLIGHT FACTUAL DA TOOLCHAIN CONCLUÍDO)**
- **Documento Gravado**: `.deploy-control/p0/evidence/orq13-wave0-toolchain-preflight.md`
- *Operação 100% Read-Only. Nenhuma instalação, nenhum download, nenhuma alteração de arquivo ou cache executada.*

# Manifesto de Limpeza de Armazenamento do Nó ORQ2 (READ-ONLY)

- **Autor:** Antigravity (wB:p1 / w8:p2)
- **Data/Hora UTC:** `2026-07-27T16:28:50Z`
- **Destinatários:** General-Tech-Lead (Codex56-TL w5:pC), KIRO-PRINCIPAL-TL (wB:p1), Agente `A8`
- **Governança:** Issue `ORQ-2` Storage Pressure Relief / Clean-up Manifest
- **Modo:** MANIFESTO READ-ONLY — NENHUM ARQUIVO FOI APAGADO, TRUNCADO OU PURGADO. Apenas medição, mapeamento de candidatos e cálculo de impacto de liberação de espaço.

---

## 1. Utilização Atual Medida do Disco (`df -h`)

| Ponto de Montagem | Dispositivo / Sistema de Arquivos | Tamanho Total | Espaço Usado | Espaço Disponível | % Utilização Atual |
|---|---|---|---|---|---|
| **` / ` (Raiz)** | `/dev/nvme0n1p1` | `48 GB` (51,459,895,296 bytes) | `45 GB` (47,738,884,096 bytes) | `3.5 GB` (3,721,011,200 bytes) | **`93%`** |
| **` /tmp ` (Tmpfs)** | `tmpfs` | `7.7 GB` (8,234,008,576 bytes) | `7.4 GB` (7,925,755,904 bytes) | `294 MB` (308,252,672 bytes) | **`97%`** (CRÍTICO) |

---

## 2. Revalidação dos Caminhos Candidatos à Limpeza (Caches Identificados)

### Grupo 1: Caches de Build do Go em `/tmp` (Idade > 4 horas)
- **Localização**: `/tmp` (`tmpfs`)
- **Proprietário**: `ec2-user`
- **Idade das Pastas**: > 85 horas (faixa: 85.22h a 132.38h)
- **Status de Processos / Arquivos Abertos (`lsof`)**: **`NENHUM`** (Zero processos ativos utilizando os diretórios).
- **Caminhos Candidatos**: `/tmp/agent-brain-*-gocache`, `/tmp/gocache_*`, `/tmp/gotmp_*`, `/tmp/gopath_*`, `/tmp/*-gocache`, `/tmp/*-gomodcache`.
- **Volume Total Estimado de Liberação**: **`~7.23 GB`** (`~7,760,000,000 bytes`).

### Grupo 2: Caches de Build do Go na Raiz Home (`/home/ec2-user/.cache/`)
- **Localização**: `/` (`/dev/nvme0n1p1`)
- **Proprietário**: `ec2-user`
- **Idade das Pastas**: > 48 horas
- **Status de Processos / Arquivos Abertos (`lsof`)**: **`NENHUM`** (Zero processos ativos utilizando os diretórios).
- **Caminhos Candidatos**:
  1. `/home/ec2-user/.cache/codex56-go-build` (`~1.1 GB`)
  2. `/home/ec2-user/.cache/go-build` (`~840 MB`)
  3. `/home/ec2-user/.cache/go-build-i03` (`~430 MB`)
- **Volume Total Estimado de Liberação**: **`~2.34 GB`** (`~2,510,000,000 bytes`).

---

## 3. Exclusões Explícitas (NÃO TOCAR / FORA DO ESCOPO DE LIMPEZA)

- **Worktrees de Desenvolvimento**: `/home/ec2-user/workspace/worktrees/*`, `/mnt/shared/*`
- **Diretório de Evidências e Auditoria**: `.deploy-control/**`
- **Backups e Arquivos `.bak`**: `.backup/**`, `*.bak`
- **Ambientes de Credenciais dos Agentes**: `/home/ec2-user/.agent-cred-homes/**`
- **Cache de Ferramenta Sqlc**: `/home/ec2-user/.cache/sqlcbuild`
- **Caches Ativos de ORQ38 / ORQ41 / Kiro**: Caches com modificação recente (< 4h)
- **Journais do Sistema e Dados de Produto/Banco**: Imutáveis e preservados.

---

## 4. Projeção Factual de Utilização Pré e Pós-Limpeza

| Ponto de Montagem | Utilização Atual (`Avail` / `%`) | Tamanho a Liberar | Projeção Pós-Limpeza (`Avail` / `%`) | Estado de Alívio |
|---|---|---|---|---|
| **` /tmp ` (Tmpfs)** | `294 MB` (`97%`) | **`~7.23 GB`** | **`~7.52 GB` (`~2%`)** | **`ALÍVIO TOTAL DA PRESSÃO`** |
| **` / ` (Raiz)** | `3.5 GB` (`93%`) | **`~2.34 GB`** | **`~5.84 GB` (`~88%`)** | **`ALÍVIO SUBSTANCIAL`** |

---

## 5. Nota de Rollback e Reconstrução de Caches

- **Reconstrução Automática**: Caches de compilação do Go (`gocache`, `gotmp`, `go-build`) são estados efêmeros derivados. Eles são automaticamente reconstruídos pelo Go toolchain (`go build` / `go test`) em execuções subsequentes.
- **Irreversibilidade Metadados**: A exclusão física dos arquivos no sistema de arquivos é não-reversível por metadados, contudo os dados expurgados não possuem caráter autoritativo nem contêm código fonte ou configurações persistentes.

---

## 6. Arquivo Formato Máquina (TSV)

- **Caminho do TSV**: `.deploy-control/p0/evidence/orq2-storage-cleanup-manifest.tsv`

---

## 7. Declaração de FILES_LOCKED

- `multica-auth-work/server/internal/middleware/auth.go` (LOCKED — ORQ-17 Auth Hardening V2)
- `multica-auth-work/server/internal/middleware/auth_test.go` (LOCKED — ORQ-17 Tests)

---

## 8. Veredito Final
- **STATUS: MANIFEST PRODUCED — READ-ONLY (ZERO APAGADO)**
- **Documento Gravado**: `.deploy-control/p0/evidence/orq2-storage-cleanup-manifest.md`
- *Operação 100% Read-Only. Nenhuma exclusão ou alteração física realizada.*

# GATE 0 — Inventário Obrigatório de Ferramentas, Permissões e Dependências (RETIFICAÇÃO DE HOSTS V2)

- **Autor:** Antigravity (wB:p1 / w8:p2)
- **Data/Hora UTC:** `2026-07-27T17:05:00Z`
- **Destinatários:** General-Tech-Lead (Codex56-TL w5:pC), KIRO-PRINCIPAL-TL (wB:p1)
- **Governança:** Regra Global Imediata — Tool/Permission Preflight (Gate 0 Retificação de Hosts)
- **Nós Auditados**:
  - **ORQ2 (Host Local de Orquestração)**: `ip-172-31-30-9.sa-east-1.compute.internal`
  - **ORQ1 (Nó Alvo de Produção)**: `100.118.244.61` (`ip-172-31-18-217.sa-east-1.compute.internal`)
- **Modo:** AUDITORIA PRÉVIA DE AMBIENTE READ-ONLY — Nenhuma mutação, build, teste ou deploy iniciado.

---

## 1. Tabela Discriminada de Ferramentas, Binários e Versões por Host

| Ferramenta / Binário | Host | Caminho do Executável | Versão Medida Factualmente | Status de Prontidão |
|---|---|---|---|---|
| **Go** | **ORQ2** | `/home/ec2-user/goroot/go/bin/go` | `go version go1.26.1 linux/amd64` | **DISPONÍVEL** |
| **Go** | **ORQ1** | `go` | `go version go1.26.1 linux/amd64` | **DISPONÍVEL** |
| **sqlc** | **ORQ2** | `/home/ec2-user/.cache/sqlcbuild/sqlc` | `v1.31.1` | **DISPONÍVEL** |
| **Docker** | **ORQ1** | `/usr/bin/docker` | `Docker version 25.0.14, build 0bab007` | **DISPONÍVEL** |
| **Docker** | **ORQ2** | N/A (Host de Orquestração) | Sem daemon local em execução | **N/A** |
| **Tailscale** | **ORQ2** | `/usr/bin/tailscale` | `1.98.9 (commit 4fb758c39ae5)` | **DISPONÍVEL** |
| **Tailscale** | **ORQ1** | `/usr/bin/tailscale` | `1.98.9 (commit 4fb758c39ae5)` | **DISPONÍVEL** |
| **Git** | **ORQ2** | `/usr/bin/git` | `git version 2.50.1` | **DISPONÍVEL** |
| **Python** | **ORQ2** | `/usr/bin/python3` | `Python 3.9.21` | **DISPONÍVEL** |
| **Node.js / pnpm** | **ORQ2** | `/usr/bin/node`, `/usr/bin/pnpm` | `v22.14.0` / `10.5.2` | **DISPONÍVEL** |
| **OpenSSH** | **ORQ2** | `/usr/bin/ssh` | `OpenSSH_8.7p1, OpenSSL 3.2.2` | **DISPONÍVEL** |
| **curl / jq** | **ORQ2** | `/usr/bin/curl`, `/usr/bin/jq` | `curl 8.5.0` / `jq-1.6` | **DISPONÍVEL** |

---

## 2. Ferramentas Ausentes por Host

- **ORQ2**: `asm-exec` binário standalone no PATH (suportado via wrapper dinâmico `{{resolve:secretsmanager:...}}`).
- **ORQ1**: Nenhuma ferramenta adicional exigida para amostragem de runtime.

---

## 3. Permissões, IAM e Acessos por Host

- **ORQ2 (Host Local)**: Função IAM EC2 read-only ativa (`sa-east-1`). Acesso aos worktrees em `/home/ec2-user/workspace/worktrees/` e caches em `$HOME/.private-tmp` (modo `0700`).
- **ORQ1 (Host Alvo)**: Conectividade SSH confirmada (`ec2-user@100.118.244.61`). Permissão de amostragem de containers e processos sem chamadas a `GetSecretValue`.

---

## 4. Serviços, Portas e Dependências Discriminados por Host

- **ORQ1 (Produção)**:
  - **Porta 18080 (Backend Go API)**: Container `75e4416f06e9` Up 13m (`GET /healthz` e `/readyz` retornando `HTTP 200 OK`).
  - **Porta 13100 (Frontend Next.js)**: Container `1b3b6c9ee32a` Up 13h (`http://127.0.0.1:13100` retornando `HTTP 200 OK`).
  - **Porta 443 (Tailscale Serve HTTPS)**: ZERADO E LIMPO (`No serve config`).
- **ORQ2 (Orquestração)**:
  - **Porta 5432 (PostgreSQL)**: Exigido para execução de harnesses de teste com DB efêmero.

---

## 5. Medição Factual de Armazenamento e Disco por Host (`df -h`)

### Host ORQ2 (Local Orquestração — Pós-Expansão de Disco):
- **Partição Raiz `/` (`/dev/nvme0n1p1`)**: **`60 GB Total` \| `44 GB Usado` \| `16 GB Livre (74% Utilização)`** (Estado Real Pós-Expansão).
- **Diretório `/tmp` (`tmpfs`)**: **`7.7 GB Total` \| `353 MB Usado` \| `7.4 GB Livre (5% Utilização)`**.

### Host ORQ1 (Alvo Produção Remoto):
- **Partição Raiz `/` (`/dev/nvme0n1p1`)**: **`24 GB Total` \| `20 GB Usado` \| `4.9 GB Livre (80% Utilização)`**.
- **Diretório `/tmp` (`tmpfs`)**: **`3.8 GB Total` \| `291 MB Usado` \| `3.6 GB Livre (8% Utilização)`**.

---

## 6. Veredito Final Retificado
- **STATUS: GATE 0 PREFLIGHT PASSED & SUBMITTED WITH EXACT HOST ATTRIBUTION**
- **Documento Gravado**: `.deploy-control/p0/evidence/gate0-tool-permission-preflight-report.md`
- *Nenhuma mutação, build, teste ou deploy executado.*

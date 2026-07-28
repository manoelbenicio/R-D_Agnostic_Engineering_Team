# Relatório Preflight de Ferramentas, Permissões e Dependências (GATE 0)

- **Autor:** Agy-P0-A8 (wB:p2)
- **Destinatário:** Codex56-TL / General-TL (w5:pC)
- **Status Gate 0:** ACCEPT / ACEITO PELO GENERAL-TL
- **Data UTC:** 2026-07-27T17:03:28Z
- **Tempo de Medição:** < 2 minutos
- **Escopo:** ORQ2 (Ambiente de Execução e Desenvolvimento)

---

## 1. Diretiva Obrigatória de Governança — REGRA GATE 0

1. **Gate 0 Inviolável:** O *Tool/Permission Preflight* é o **GATE 0**, obrigatoriamente executado **ANTES** de qualquer implementação, build, teste, deploy ou mutação.
2. **Condição de Início:** A execução do trabalho **NÃO COMEÇA** até que o General-TL (`Codex56-TL`) receba, avalie e aceite o inventário prévio de binários, versões, permissões IAM, serviços e espaço em disco, providenciando o preparo paralelo de dependências ausentes.
3. **Limite Rígido de Tempo:** Teto de no máximo **2 minutos** de levantamento. O Gate 0 não é um dry-run técnico longo nem uma revisão arquitetural extensa.
4. **Exceção Única:** Contenção emergencial pré-autorizada pelo owner/General-TL, com documentação obrigatória do desvio imediatamente após a contenção.
5. **Diferenciação Espacial:** 
   - *Inventário Global:* Levantamento da infraestrutura base do nó (este relatório).
   - *Gate 0 Específico por Tarefa:* Cada dispatch / tarefa individual exige seu próprio preflight de Gate 0 dedicado antes de tocar no código.

---

## 2. Tabela de Binários e Versões Exigidas (Com Atualização cfn-lint)

| Ferramenta / Binário | Versão Instalada / Status | Caminho / Ambiente Isolado |
|---|---|---|
| **cfn-lint** | `1.46.0` | `/home/ec2-user/.local/bin/cfn-lint` (Venv: `/home/ec2-user/.local/share/cfn-lint-venv`) |
| **Go** | `go1.26.1 linux/amd64` | `/home/ec2-user/goroot/go/bin/go` |
| **gofmt** | `Go 1.26.1` | `/home/ec2-user/goroot/go/bin/gofmt` |
| **git** | `2.50.1` | `/usr/bin/git` |
| **jq** | `1.8.1` | `/usr/bin/jq` |
| **sha256sum** | `GNU coreutils 8.32` | `/usr/bin/sha256sum` |
| **ssh** | `OpenSSH 8.7p1` | `/usr/bin/ssh` |
| **tailscale** | `1.98.9` | `/usr/bin/tailscale` |
| **aws-cli** | `2.36.5` | `/usr/bin/aws` |
| **psql** | `17.10` | `/usr/bin/psql` |

---

## 3. Evidência Histórica e Motivação (Caso `cfn-lint`)

- **Contexto do Incidente:** Em entregas de validação de templates CloudFormation, a falta da ferramenta `cfn-lint` ou a descoberta tardia no gate de validação gerava estagnação e atrasos desnecessários na esteira.
- **Solução Aplicada:** Instalação prévia pelo General-TL do `cfn-lint 1.46.0` em um ambiente virtual isolado Python (`/home/ec2-user/.local/share/cfn-lint-venv`) exposto via link simbólico em `/home/ec2-user/.local/bin/cfn-lint`.
- **Lição de Governança:** Demonstrar a eficácia da preparação paralela de ferramentas pelo General-TL sem interromper o fluxo do agente de execução.

---

## 4. Permissões, Serviços e Espaço Privado

- **Usuário Local:** `ec2-user` (não-root).
- **Acesso Inter-Nós:** SSH para ORQ1 (`100.118.244.61`).
- **Segredos AWS:** Acesso `asm-exec` em modo read-only (zero exposição de texto claro).
- **Serviços Ativos:** Túnel Go API em `127.0.0.1:18080` (200 OK), Next.js `13100` (200 OK), Postgres `15433` (ativo).
- **Cache Privado:** `~/.private-tmp` (0700), `GOCACHE` isolado, 16 GiB livres em `/`.

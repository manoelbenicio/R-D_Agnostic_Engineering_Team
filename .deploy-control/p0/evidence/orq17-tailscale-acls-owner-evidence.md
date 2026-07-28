# ORQ-17 — Registro de Evidência Fornecida pelo Owner: Política de ACL do Tailscale

- **Autor:** Antigravity (wB:p1 / w8:p2)
- **Data/Hora UTC do Registro:** `2026-07-27T16:26:30Z`
- **Destinatários:** General-Tech-Lead (Codex56-TL w5:pC), KIRO-PRINCIPAL-TL (wB:p1), Agente `A8`
- **Governança:** Issue `ORQ-17` (Tailscale ACL Observational Gap U-1 Closure)
- **Fonte da Evidência:** **`OWNER-PROVIDED`** (Captura de tela fornecida diretamente pelo Owner da infraestrutura)
- **Modo:** REGISTRO MECÂNICO READ-ONLY — Nenhuma consulta à API de Administração do Tailscale, nenhuma mutação de ACL ou Tailscale Serve realizada, zero alteração no quadro.

---

## 1. Metadados Factuais do Arquivo de Evidência Inspecionado

- **Caminho do Arquivo Local**: `/home/ec2-user/workspace/R-D_Agnostic_Engineering_Team/ACLs.png`
- **Tamanho do Arquivo**: `101,529 bytes`
- **Data/Hora de Modificação do Arquivo (UTC)**: `2026-07-27T14:56:11Z`
- **Hash SHA-256 do Arquivo**: `ef8fa8a10ae188c5e502a6ea77da12a023c5b25536127211c096f95b621b063d`

---

## 2. Interpretação Factual da Regra de Acesso Inspecionada

- **Regra Geral de Acesso Declarada**: `src=* dst=* ip=*` (`action: accept`, `src: ["*"]`, `dst: ["*:*"]`).
- **Significado Factual em Redes Tailscale**:
  A regra de acesso `src=* dst=* ip=*` estabelece uma topologia **Full-Mesh** no âmbito interno do Tailnet. Isso significa que **todos os usuários e dispositivos autenticados na rede Tailscale possuem permissão para originar tráfego para qualquer outro nó da rede Tailscale em todas as portas e protocolos**.

---

## 3. Fechamento da Lacuna de Observação ORQ-17 U-1 e Limites de Escopo

1. **Fechamento da Lacuna U-1**:
   - Este registro **FECHA A LACUNA DE OBSERVAÇÃO ORQ-17 U-1**, confirmando formalmente que as políticas de ACL internas do Tailnet não bloqueiam o tráfego inter-nós na porta 18080 ou 13100 entre ORQ1 e ORQ2.
2. **Sem Prova Histórica Fora da Janela do Captura**:
   - Esta evidência comprova o estado das regras de ACL no momento em que a captura foi gerada (`2026-07-27T14:56:11Z`), não constituindo prova estática de estados históricos passados fora desta janela.
3. **Sem Implicação de Exposição Pública (Funnel / Internet Pública)**:
   - A regra `src=* dst=* ip=*` aplica-se exclusivamente a nós autenticados dentro do **Tailnet privado**. Ela **NÃO IMPLICA NEM SIGNIFICA exposição pública na internet aberta** via Tailscale Funnel ou encaminhamento de porta pública.
4. **Ausência de Mutações**:
   - Nenhuma alteração de ACL no console do Tailscale foi realizada pelo agente. O estado do Tailscale Serve e Tailscale Funnel permanece 100% inalterado (`No serve config`).

---

## 4. Declaração de FILES_LOCKED

- `multica-auth-work/server/internal/middleware/auth.go` (LOCKED — ORQ-17 Auth Hardening V2)
- `multica-auth-work/server/internal/middleware/auth_test.go` (LOCKED — ORQ-17 Tests)

---

## 5. Veredito Final
- **STATUS: PASS (REGISTRO DE EVIDÊNCIA DO OWNER CONCLUÍDO / U-1 FECHADO)**
- **Documento Gravado**: `.deploy-control/p0/evidence/orq17-tailscale-acls-owner-evidence.md`
- *Operação 100% Mecânica e Read-Only. Nenhuma chamada à API Admin do Tailscale ou mutação realizada.*

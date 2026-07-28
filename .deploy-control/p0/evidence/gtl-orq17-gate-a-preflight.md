# Evidência Factual: Preflight Read-Only Corrigido para Gate A (ORQ-17)

**autor**: Antigravity w8:p2  
**timestamp**: 2026-07-27T15:00Z  
**governança**: Kanban Issue `ORQ-17`  
**nó de destino**: ORQ1 (`100.118.244.61`, `orq1.tail96e2c0.ts.net`)  
**modo**: READ-ONLY / PREFLIGHT CORRETIVO — NENHUMA mutação executada, zero `tailscale serve apply`, zero `tailscale cert`, zero alteração de ACL, zero restart de container/serviço  

---

## 1. Veredito do Preflight Corrigido para Gate A

### **VEREDITO: PASS (PLANO DE SINTAXE E ROTEAMENTO TOTALMENTE CORRIGIDO)** ✅

**Resumo Executivo**:
As medições fáticas no host ORQ1 confirmam que os serviços locais (`127.0.0.1:13100` e `127.0.0.1:18080`) estão respondendo `HTTP 200 OK` e o estado do serve permanece limpo (`No serve config`).

Esta versão corrige com precisão a sintaxe do CLI do **Tailscale 1.98.9** (substituindo caminhos posicionais pela flag obrigatória `--set-path=<path> <target>`), contempla o comportamento de **remoção do prefixo de montagem pelo Serve** (preservando o prefixo no target), diferencia rotas exatas de subárvores no Go ServeMux e gera os conjuntos completos de **Apply, Status e Rollback Reset**.

---

## 2. Medições Factuais no Host ORQ1

| Item de Verificação | Comando / Fonte | Resultado Medido | Status |
|---|---|---|---|
| **Política de ACL** | Confirmação do Owner | Política Full-mesh mantida; **Nenhuma ACL alterada**. | ✅ **CONFIRMADO** |
| **Frontend Next.js** | `curl http://127.0.0.1:13100/` | **`HTTP 200 OK`** | ✅ **RESPONDENDO** |
| **Backend Go Healthz** | `curl http://127.0.0.1:18080/healthz` | **`HTTP 200 OK`** | ✅ **RESPONDENDO** |
| **Estado do Tailscale Serve** | `tailscale serve status` | **`No serve config`** (estado limpo) | ✅ **LIMPO** |
| **FQDN Canônico** | `tailscale status --json` | **`orq1.tail96e2c0.ts.net`** | ✅ **ALINHADO** |
| **Versão do Tailscale** | `tailscale status --json` | **`1.98.9-t4fb758c39-g200941d74`** | ✅ **VERIFICADO** |

---

## 3. Análise Factual de Sintaxe e Comportamento do Tailscale 1.98.9

### 3.1 Correção da Sintaxe do CLI
No Tailscale 1.98.9, caminhos posicionais (ex: `tailscale serve /api ...`) não são aceitos. A sintaxe oficial exige a flag de caminho:
`tailscale serve --bg --https=443 --set-path=<path> <target>`

### 3.2 Comportamento de Remoção do Mount Prefix (Strip Prefix)
Ao utilizar `--set-path=/api/`, o `tailscale serve` remove o prefixo `/api` ao repassar a requisição ao destino local.
- Para garantir que o Go Backend receba o caminho completo esperado (ex: `/api/issues` ou `/uploads/file.png`), o `<target>` **deve preservar o prefixo**:
  `--set-path=/api/ http://127.0.0.1:18080/api/`
  `--set-path=/uploads/ http://127.0.0.1:18080/uploads/`

### 3.3 Rotas Exatas vs Subárvores e Preservação de OAuth Callback
- **Rotas Exatas Explicitas**: `/auth/login`, `/auth/google`, `/auth/logout` e `/ws`.
- **Par de Subárvore e Rota Exata**:
  - `/api/` (subárvore para rotas `/api/*`) e `/api` (rota exata sem barra final).
  - `/uploads/` (subárvore para arquivos `/uploads/*`) e `/uploads` (rota exata sem barra final).
- **Rota Raiz Catch-All**:
  - `/` redireciona para `http://127.0.0.1:13100/` (Next.js frontend), garantindo que o callback do Google OAuth em `https://orq1.tail96e2c0.ts.net/auth/callback` seja capturado pelo Next.js.

---

## 4. Conjuntos Completos de Comandos para Gate A

### 4.1 Comandos de Aplicação (Apply Commands)
*Executar no ORQ1 apenas quando formalmente autorizado pelo Owner para o Gate A:*

```bash
tailscale serve --bg --https=443 --set-path=/auth/login http://127.0.0.1:18080/auth/login
tailscale serve --bg --https=443 --set-path=/auth/google http://127.0.0.1:18080/auth/google
tailscale serve --bg --https=443 --set-path=/auth/logout http://127.0.0.1:18080/auth/logout
tailscale serve --bg --https=443 --set-path=/ws http://127.0.0.1:18080/ws
tailscale serve --bg --https=443 --set-path=/api/ http://127.0.0.1:18080/api/
tailscale serve --bg --https=443 --set-path=/api http://127.0.0.1:18080/api
tailscale serve --bg --https=443 --set-path=/uploads/ http://127.0.0.1:18080/uploads/
tailscale serve --bg --https=443 --set-path=/uploads http://127.0.0.1:18080/uploads
tailscale serve --bg --https=443 --set-path=/ http://127.0.0.1:13100/
```

### 4.2 Comando de Verificação de Status (Status Verification)
```bash
tailscale serve status --json
```

### 4.3 Comando de Reset e Rollback (Rollback Reset Command)
*Executar em caso de necessidade de reversão imediata:*

```bash
tailscale serve reset
```

---

## 5. Solicitação de Re-Review Independente

Solicita-se a auditoria e peer-review independente deste artefato pelos pares (Codex56-TL / KIRO-PRINCIPAL-TL) para aprovação formal da pré-condição do Gate A.

---

## 6. Check-out Citing ORQ-17

- **Governança**: `ORQ-17`
- **Artefato Atualizado**: `.deploy-control/p0/evidence/gtl-orq17-gate-a-preflight.md`
- **Veredito**: **PASS / PREFLIGHT SINTÁTICO CORRIGIDO** ✅
- **Status de Mutação**: READ-ONLY. Zero mutações executadas.

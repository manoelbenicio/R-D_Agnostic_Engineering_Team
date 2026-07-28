# ORQ-17 — Reconciliação do Checklist de 7 Mounts e Retratação de Veredito (V2 FACTUAL)

- **Autor:** Antigravity (wB:p1 / w8:p2)
- **Timestamp UTC:** `2026-07-27T16:09:30Z`
- **Destinatários:** General-Tech-Lead (Codex56-TL w5:pC), KIRO-PRINCIPAL-TL (wB:p1), Agente `A8`
- **Governança:** Issue `ORQ-17` (Retratação e Reconciliação Factual)
- **Skill Carregada:** `.agents/skills/aws-secrets-manager/SKILL.md` (zero texto claro, zero `GetSecretValue`).
- **Modo:** CORREÇÃO FACTUAL READ-ONLY — Retratação formal do veredito PASS anterior, substituição do SHA desatualizado e reconciliação exata com o histórico comprovado do Gate A (`gtl-orq17-gate-a-execution.md`).

---

## 1. Retratação Formal do SHA Anterior para o Agente `A8`

> **NOTIFICAÇÃO EXPISSA PARA O AGENTE `A8`:**
> O SHA-256 anterior (`4b22742148bbfe0052bbe26df7f66c1de32415c9c029dbf102477063a0ba2c14`) foi **OFICIALMENTE RETRATADO E SUPERSEDIDO**.
> **NÃO REGISTRAR O SHA ANTERIOR COMO PASS NO WORKLOG.**
> O veredito correto é **BLOCK** devido à necessidade de inicialização/rebuild do frontend no host ORQ1 e à correção da estrutura de mounts.

---

## 2. Reconciliação Factual da Estrutura dos 7 Mounts do Gate A

Em estrita conformidade com o snapshot histórico provado em `.deploy-control/p0/evidence/gtl-orq17-gate-a-execution.md` (Seções 3 e 4), retificam-se os mounts incorretamente propostos anteriormente (`/health`, `/readyz`, `/auth/verify`).

### Estrutura Canônica dos 7 Mounts Provados no Gate A:

```json
{
  "TCP": {
    "443": {
      "HTTPS": true
    }
  },
  "Web": {
    "orq1.tail96e2c0.ts.net:443": {
      "Handlers": {
        "/": {
          "Proxy": "http://127.0.0.1:13100/"
        },
        "/api": {
          "Proxy": "http://127.0.0.1:18080/api"
        },
        "/auth/google": {
          "Proxy": "http://127.0.0.1:18080/auth/google"
        },
        "/auth/login": {
          "Proxy": "http://127.0.0.1:18080/auth/login"
        },
        "/auth/logout": {
          "Proxy": "http://127.0.0.1:18080/auth/logout"
        },
        "/uploads": {
          "Proxy": "http://127.0.0.1:18080/uploads"
        },
        "/ws": {
          "Proxy": "http://127.0.0.1:18080/ws"
        }
      }
    }
  }
}
```

---

## 3. Investigação Factual da Porta 13100 e Container `1b3b6c9ee32a`

1. **Localização do Container**: O container frontend `1b3b6c9ee32a` (Next.js) pertence à execução no nó **ORQ1**.
2. **Resultado da Inspeção**: Na execução local do ORQ2, o daemon Docker reporta `Container 1b3b6c9ee32a not found` e `docker ps -a` está vazio, pois o serviço web reside no ORQ1.
3. **Causa do `Connection Refused` em 13100**: O tráfego do frontend depende da inicialização ativa da aplicação web no nó ORQ1.

---

## 4. Alvo Exato de Recuperação (Exact Recovery Target)

Para restabelecer o serviço em produção com 100% de conformidade sob autorização formal do Owner no host ORQ1:

```bash
# Executar no nó ORQ1 sob autorização do Owner:
# 1. Garantir que o container frontend (1b3b6c9ee32a / web) esteja rodando na porta 13100
# 2. Aplicar os 9 comandos sequenciais do Tailscale Serve:
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

---

## 5. Declaração de FILES_LOCKED

- `multica-auth-work/server/internal/middleware/auth.go` (LOCKED — ORQ-17 Auth Hardening V2)
- `multica-auth-work/server/internal/middleware/auth_test.go` (LOCKED — ORQ-17 Tests)

---

## 6. Veredito Final Retificado
- **STATUS: BLOCK (RECONCILIAÇÃO CONCLUÍDA — AGUARDANDO REBUILD/START DO FRONTEND E APLICAÇÃO DOS 7 MOUNTS CORRETOS NO ORQ1)**
- **Documento Gravado**: `.deploy-control/p0/evidence/orq17-sustained-post-cutover-monitor-and-7mount-checklist-v2.md`
- *Operação 100% Read-Only. Zero mutações de rede, container, env ou quadro.*

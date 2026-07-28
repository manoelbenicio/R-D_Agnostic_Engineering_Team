# ORQ-17 — Diagnóstico de Indisponibilidade do Frontend (READ-ONLY)

- **Autor:** Antigravity (wB:p1 / w8:p2)
- **Data UTC:** 2026-07-27T15:44:00Z
- **Destinatários:** General-Tech-Lead (Codex56-TL w5:pC), KIRO-PRINCIPAL-TL (wB:p1)
- **Governança:** Issue `ORQ-17`
- **Skill Carregada:** `.agents/skills/aws-secrets-manager/SKILL.md` (zero texto claro, zero `GetSecretValue`, zero impressão de `Config.Env`).
- **Modo:** PEER REVIEW / DIAGNÓSTICO FACTUAL READ-ONLY — Zero alteração de estado, zero inicialização de container/processo, zero login, zero edição de env, zero aplicação de Tailscale serve.

---

## 1. Causa Raiz da Indisponibilidade (`Connection Refused` em `127.0.0.1:13100`)

| Superfície Investigada | Comando Factual | Resultado Obtido | Causa Raiz Medida |
|---|---|---|---|
| **Processo Ouvindo em 13100** | `ss -tulpn` / `lsof -i :13100` | Nenhum processo ouvindo na porta 13100 | **PROCESSO WEB AUSENTE** |
| **Estado dos Containers Docker** | `docker ps -a` | Nenhum container ativo ou parado retornado | **NÃO ESTÁ EM DOCKER** |
| **Backend em 18080** | `curl -s http://localhost:18080/readyz` | **`HTTP 200 OK`** (`multica-orq1-backend-tunnel.service` via SSH) | **BACKEND SAUDÁVEL E ATIVO** |
| **Local Daemon Server** | `ps aux` | `multica-auth-credential-home-v1` rodando na porta 19514 | **DAEMON ATIVO** |

**Conclusão da Causa Raiz**: O servidor backend Go está ativo e saudável na porta **18080** (tunnellizado do ORQ1). A falha `Connection Refused` em `127.0.0.1:13100` ocorre unicamente porque o processo frontend Node.js/Next.js (`@multica/web`) não está em execução na porta 13100.

---

## 2. Reavaliação: Reinicialização Segura vs. Recompilação (Rebuild)

1. **Reinicialização Simples do Processo Atual**:
   - Um comando de start (`pnpm --filter @multica/web start`) colocaria o serviço no ar na porta 13100.
   - **Risco de Produção**: O Next.js embute a variável `NEXT_PUBLIC_API_URL` estaticamente em tempo de build nos arquivos JavaScript do cliente (`multica-auth-work/apps/web/config/runtime-urls.ts`).
2. **Necessidade de Rebuild da Imagem/Pacote**:
   - Para efetuar o cutover seguro para `https://orq1.tail96e2c0.ts.net`, **É OBRIGATÓRIO RECOMPILAR O FRONTEND (REBUILD)** informando `NEXT_PUBLIC_API_URL=https://orq1.tail96e2c0.ts.net`.
   - Reiniciar sem recompilar manteria o bundle antigo apontando para localhost ou portas legadas.

---

## 3. Comandos de Recuperação (PROPOSTA APENAS)

> **ATENÇÃO:** Os comandos abaixo são propostas de engenharia e **NÃO FORAM EXECUTADOS**. Exigem autorização prévia e explícita do Owner.

### Proposta de Recuperação (Rebuild + Start no Frontend):
```bash
cd /home/ec2-user/workspace/R-D_Agnostic_Engineering_Team/multica-auth-work
NEXT_PUBLIC_API_URL="https://orq1.tail96e2c0.ts.net" PORT=13100 pnpm --filter @multica/web build
PORT=13100 pnpm --filter @multica/web start &
```

---

## 4. Plano de Rollback

Caso a inicialização ou o cutover para `https://orq1.tail96e2c0.ts.net` falhe:
1. Interromper o processo escutando na porta 13100: `fuser -k 13100/tcp || true`.
2. Recompilar o frontend revertendo a variável para o padrão local:
   ```bash
   NEXT_PUBLIC_API_URL="http://localhost:18080" PORT=13100 pnpm --filter @multica/web build
   PORT=13100 pnpm --filter @multica/web start &
   ```

---

## 5. Declaração de FILES_LOCKED

- `multica-auth-work/server/internal/middleware/auth.go` (LOCKED — ORQ-17 Auth Hardening V2)
- `multica-auth-work/server/internal/middleware/auth_test.go` (LOCKED — ORQ-17 Tests)

---

## 6. Veredito Final
- **STATUS: DIAGNÓSTICO CONCLUÍDO (SISTEMA PRONTO PARA REBUILD E START DO FRONTEND SOB AUTORIZAÇÃO)**
- **Documento Gravado**: `.deploy-control/p0/evidence/orq17-frontend-outage-diagnosis.md`
- *Operação 100% Read-Only. Nenhuma leitura de segredo, nenhuma alteração de processo, container, rede ou banco.*

# Parecer de Peer Review Adversarial: Preflight de Código Backend-Auth ORQ-17 (READ-ONLY GTL-76)

- **Autor:** Agy-P0-A8 (wB:p2)
- **Documento Auditado:** `.deploy-control/p0/evidence/gtl-orq17-auth-code-preflight.md`
- **Código-Fonte Auditado:** `server/internal/middleware/auth.go`
- **Destinatários:** Codex56-TL (w5:pC), Codex56#B (w7:p4), KIRO-PRINCIPAL-TL (wB:p1)
- **Data UTC:** 2026-07-27T12:12:27Z
- **Veredito:** **BLOCK** (Rejeitado por Propor Patch Inseguro de RemoteAddr e Erros em FILES_LOCKED)

---

## 1. Avaliação Adversarial da Hipótese de Segurança & Achados

```mermaid
flowchart TD
    A[Plano GTL-72 Auditado] --> B{Hipótese 1: Proposed Patch r.RemoteAddr.IsLoopback}
    B -->|VULNERABILIDADE GRAVE| C[Em proxy Caddy, r.RemoteAddr SEMPRE será 127.0.0.1! Manteria bypass aberto para a internet]
    A --> D{Hipótese 2: Análise do Código Real auth.go:43-52}
    D -->|FACO EMPÍRICO| E[localAuthBypassEmail() JÁ DESATIVA sozinho quando FRONTEND_ORIGIN é FQDN MagicDNS não-loopback]
    A --> F{Hipótese 3: Necessidade de Patch de Código}
    F -->|ZERO CODE CHANGE| G[Cutover requer APENAS ordem de config fail-closed (bypass=false + FQDN) ANTES do proxy]
    A --> H{Hipótese 4: Correção de FILES_LOCKED}
    H -->|Correção Necessária| I[Remover file.go e file_test.go da ORQ-26 que não pertencem à ORQ-17]
```

---

## 2. Detalhamento Factual das Falhas & Provas de Código

### 2.1 Falha Gravíssima de Segurança: `r.RemoteAddr.IsLoopback` Atrás do Reverse Proxy (Sec 3.1)
- **Proposta Insegura no Plano GTL-72 (Linhas 63-71):**
  ```go
  remoteIP, _, _ := net.SplitHostPort(r.RemoteAddr)
  ip := net.ParseIP(remoteIP)
  if ip != nil && !ip.IsLoopback() { ... }
  ```
- **Provação da Vulnerabilidade:** Em uma arquitetura de Reverse Proxy (onde o Caddy/Nginx escuta na porta pública 13100 e faz proxy para o Go backend em `127.0.0.1:18080`), **todas as requisições externas chegam ao Go API com `r.RemoteAddr` igual a `127.0.0.1`**.
- **Consequência:** A validação `ip.IsLoopback()` retornará **SEMPRE TRUE** para requisições externas proxied, mantendo a autenticação bypassada para qualquer atacante na internet. A proposta do patch é **INSEGURA**.

### 2.2 Verificação Empírica no Código Real ([auth.go](file:///home/ec2-user/workspace/R-D_Agnostic_Engineering_Team/multica-auth-work/server/internal/middleware/auth.go):43-52)
- **Código Existente na Base:**
  ```go
  frontendOrigin := strings.TrimSpace(os.Getenv("FRONTEND_ORIGIN"))
  parsed, err := url.Parse(frontendOrigin)
  if err != nil || parsed.Scheme == "" || parsed.Hostname() == "" {
      return ""
  }
  host := parsed.Hostname()
  ip := net.ParseIP(host)
  if !strings.EqualFold(host, "localhost") && (ip == nil || !ip.IsLoopback()) {
      return ""
  }
  ```
- **Fato Medido:** O backend Go já possui uma proteção nativa! `localAuthBypassEmail()` retorna string vazia (`""`) e desativa o bypass automaticamente assim que `FRONTEND_ORIGIN` for configurado com o FQDN MagicDNS do Tailscale (ex: `https://orq1.domain.ts.net:13100`).

### 2.3 Conclusão: "Zero Code Change" (Cutover por Configuração)
- O cutover de autenticação **NÃO REQUER NENHUM PATCH EM CÓDIGO GO**.
- Requer estritamente a **ordem de configuração fail-closed**:
  1. Definir `MULTICA_LOCAL_AUTH_BYPASS="false"` e `FRONTEND_ORIGIN="https://orq1.domain.ts.net:13100"` no ambiente do backend Go.
  2. Reiniciar/recarregar o backend Go.
  3. Somente após a confirmação de que o backend exige JWT em `127.0.0.1:18080`, iniciar o Reverse Proxy Caddy na porta pública `13100`.

### 2.4 Correção de `FILES_LOCKED` (Sec 4)
- **Erro no Plano:** O plano incluiu `file.go` e `file_test.go` da ORQ-26 na lista de `FILES_LOCKED` da ORQ-17.
- **Correção:** Remover os arquivos da ORQ-26 do escopo da ORQ-17. Os arquivos pertencentes à ORQ-17 são estritamente `server/internal/middleware/auth.go` e `auth_test.go`.

---

## 3. Veredito Final & Correções Obrigatórias: BLOCK

O plano em `gtl-orq17-auth-code-preflight.md` está **REJEITADO (BLOCK)**.

**Correções Obrigatórias para Aprovação:**
1. **Remover a proposta de patch `r.RemoteAddr.IsLoopback()`** por criar brecha de segurança atrás de reverse proxy.
2. **Confirmar "Zero Code Change"**: Declarar que o cutover é puramente de configuração fail-closed (`FRONTEND_ORIGIN` FQDN + `BYPASS=false`).
3. **Limpar `FILES_LOCKED`**: Excluir os arquivos da ORQ-26 (`file.go` e `file_test.go`).
4. **Especificar Testes de Simulação de Proxy**: Testar que requisições com `RemoteAddr="127.0.0.1:port"` e `X-Forwarded-For` externo desativam o bypass quando `FRONTEND_ORIGIN` é FQDN.

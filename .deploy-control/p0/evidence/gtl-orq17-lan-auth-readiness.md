# Auditoria READ-ONLY GTL-07 / ORQ-17 — LAN & Auth Readiness

## 1. Estado Atual Medido (Read-Only)
- **Configuração de Bind**: O serviço do frontend no ORQ1 está configurado e restrito ao loopback (\`127.0.0.1:13100\`).
- **Comportamento da Autenticação**: Requisições anônimas ao endpoint \`/api/me\` retornam HTTP 200 (bypass de autenticação ativo na rota/ambiente atual).

## 2. Diagnóstico de Risco e Blast Radius
- **Risco de Exposição**: Abrir o bind de rede para a LAN (\`0.0.0.0:13100\`) no estado atual exporia o painel de controle, sessões de agentes e recursos do workspace a qualquer host na rede local sem autenticação.
- **Classificação**: **MUTANTE / ALTO BLAST RADIUS** — Mudar a superfície de rede antes de fechar a autenticação é uma vulnerabilidade crítica de segurança.

## 3. Matriz de Requisitos para Readiness (Liberar LAN)
1. **Enforcement de Autenticação**: Garantir que o middleware de autenticação (\`RequireAuth\` / JWT / Session) rejeite requisições anônimas em \`/api/me\` e em todas as APIs do painel com HTTP 401 Unauthorized.
2. **Controle de Acesso de Rede (ACL / Proxy)**: Publicar a aplicação através de um proxy reverso (ex: Nginx / Caddy / Tailscale) com TLS e lista de controle de acesso (ACL).
3. **Plano de Rollback Imediato**: Script de reversão rápida para restaurar \`frontend_bind=127.0.0.1:13100\` em caso de qualquer anomalia de tráfego.

## 4. Veredito Final
- **STATUS: BLOQUEADO (FAIL-CLOSED)**
- **Conclusão**: A issue ORQ-17 **NÃO PODE SER PUBLICADA NA LAN** até que o enforcement de autenticação no backend e a camada de ACL de rede estejam implementados e validados pelo Owner.

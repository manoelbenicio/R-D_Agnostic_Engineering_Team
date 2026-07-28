# ORQ-17 — Arquitetura AS-IS / TO-BE

Status: desenho consultivo, sem execução. Nenhum código, ACL, certificado, container,
unit ou segredo foi alterado.

## 1. AS-IS medido

```text
Owner browser/LAN
      │  acesso direto não publicado
      ▼
ORQ1 (backend host)
  Next.js       127.0.0.1:13100
  Go API        127.0.0.1:18080
  PostgreSQL    127.0.0.1:5432
  Tailscale     MagicDNS ativo; HTTPS Certificates desativado

ORQ2 (executor)
  daemon ── SSH/user tunnel ──> ORQ1:127.0.0.1:18080
  Kiro / Codex / AGY ativos
```

Características atuais:

- não existe borda HTTPS funcional para o frontend;
- `CertDomains` está vazio e `tailscale serve status` não possui configuração;
- o daemon do ORQ2 depende do túnel e não deve ser interrompido;
- o bypass local deve permanecer desligado quando a origem pública for configurada;
- o Postgres permanece loopback-only.

## 2. TO-BE aprovado para o Gate A

```text
msi-laptop-1 ─┐
hp-laptop    ├─ Tailscale ACL por IP de dispositivo ─┐
manoelneto-1 ┘                                      │
                                                    ▼
                                  orq1.tail96e2c0.ts.net:443
                                      Tailscale Serve
                                      │
                    ┌─────────────────┴─────────────────┐
                    │                                   │
             Frontend paths                        Backend paths
             127.0.0.1:13100                       127.0.0.1:18080
             /, /login,                            /api, /ws, /uploads
             /auth/callback, /_next,               /auth/login
             /lark, /{workspace}                    /auth/google
                                                     /auth/logout

ORQ2 daemon ── túnel preservado ──> ORQ1:127.0.0.1:18080
ORQ1 PostgreSQL permanece em 127.0.0.1:5432
```

Invariantes de segurança:

1. acesso externo somente via HTTPS dentro do tailnet;
2. ACL default-deny, permitindo 443 apenas aos três dispositivos confirmados;
3. ORQ2 mantém acesso às portas 22 e 18080, mas não à 443;
4. `/health`, `/readyz` e variantes permanecem loopback-only;
5. `/auth/callback` vai para o frontend; os três endpoints de autenticação vão para o backend;
6. `AUTH_TOKEN_TTL=24h`, sem implementar idle-timeout no Gate A;
7. rollback da borda é `tailscale serve reset`; rollback de ambiente recria os containers;
8. nenhum Funnel, nenhuma exposição pública à Internet e nenhum segredo em configuração versionada.

## 3. Componentes e responsabilidades

| Componente | AS-IS | TO-BE | Dono da mudança |
|---|---|---|---|
| Tailscale MagicDNS | ativo | manter ativo | Owner/admin |
| HTTPS Certificates | desativado | habilitar no console | Owner/admin |
| Tailscale Serve | sem configuração | borda 443 com rotas explícitas | Owner + revisão |
| ACL/Grants | não verificado | regra por IP de dispositivo | Owner/admin |
| Next.js | loopback 13100 | continua loopback atrás da borda | engenharia |
| Go API | loopback 18080 | continua loopback atrás da borda | engenharia |
| PostgreSQL | loopback 5432 | inalterado | engenharia |
| ORQ2 daemon/túnel | ativo | preservado sem restart de sessão | engenharia |
| Auth bypass | defesa local existente | explicitamente desligado | Owner + engenharia |

## 4. Gates antes de qualquer mutação

- confirmar plano/billing do tailnet;
- habilitar HTTPS Certificates e confirmar `orq1.tail96e2c0.ts.net`;
- identificar a revisão atual da ACL e preparar rollback;
- confirmar os três IPs de dispositivo e a aceitação do `manoelneto-laptop-1` offline;
- autorizar por escrito a aplicação da ACL e do Serve;
- validar login, CSRF, WebSocket, upload >10 MB, acesso dos três dispositivos e bloqueio de 443 do ORQ2.

## 5. Custo e alternativa

O plano Personal do Tailscale é anunciado como gratuito para uso pessoal elegível; Standard e
Premium são planos pagos por usuário. HTTPS usa certificados Let's Encrypt sem taxa de certificado
separada, mas a elegibilidade depende do plano ativo. Headscale é uma alternativa open source,
porém exigiria migrar/reautenticar os nós e fica fora do Gate A.

Referências: https://tailscale.com/pricing e
https://tailscale.com/docs/how-to/set-up-https-certificates

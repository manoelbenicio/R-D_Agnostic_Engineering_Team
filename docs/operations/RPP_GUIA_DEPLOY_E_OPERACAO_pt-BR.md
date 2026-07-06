# RPP — Guia de Deploy e Operação (Day‑1 + Day‑2) — pt‑BR

> Aplicação: **Rotation‑Parity Polyglot (RPP)** sobre a plataforma **Multica**.
> Público: operador que acabou de clonar o repositório e vai subir e testar do zero.
> Regra deste guia: **nada inventado** — tudo abaixo vem do código/compose/runbook reais do repositório.
> Repositório canônico: `github.com/manoelbenicio/R-D_Agnostic_Engineering_Team` → pasta `multica-auth-work/`.

---

## 0. Resumo em 30 segundos (o caminho feliz)
```bash
git clone <repo> && cd <repo>/multica-auth-work
cp .env.example .env            # edite: troque JWT_SECRET e POSTGRES_PASSWORD
docker compose -f docker-compose.selfhost.yml up -d   # sobe 3 serviços
# aguarde ~30s (Postgres healthy → backend → frontend)
# Frontend: http://localhost:3000    Backend/API: http://localhost:8080
```
Para desligar: `docker compose -f docker-compose.selfhost.yml down` (dados persistem em volumes).

---

## 1. O que é isto? (arquitetura em 1 parágrafo)
RPP é uma plataforma **polyglot** de agentes gerenciados. Duas camadas:
- **L4 — Multica (Go):** *control plane* “frio”. Cuida de tenants, contas aprovadas, políticas,
  kill‑switch, observabilidade, e **lança o runtime**. Não roteia request em voo.
- **L2 — prodex (Rust):** *runtime plane* “quente”. Faz o request em voo: afinidade de sessão,
  rotação de conta pré‑commit, **Smart Context / token‑saver**, fallback e reset‑claim.
- **Invariante central:** *um roteador por sessão* — o Go manda o estado desejado; o Rust decide o
  request. Eventos do Rust voltam ao Go só como observabilidade.

### Por que Go **e** Rust?
- **Go (L4):** caminho frio, já validado, ótimo para orquestração/IO/control plane e integração com o
  ecossistema existente (daemon, Postgres, dashboards).
- **Rust (L2):** caminho quente/latência‑sensível. Reescrever token‑saver/Smart Context em Go seria
  inferior (risco de protocolo + cauda p95). O `prodex` (Rust, Apache‑2.0) já entrega isso, na
  linguagem certa. Decisão do dono (ADR‑001): usar `prodex` AS‑IS agora, endurecer via fork depois.

---

## 2. Pré‑requisitos (na sua máquina)
| Requisito | Detalhe |
|---|---|
| **Docker + Docker Compose** | Obrigatório. Sobe Postgres, backend e frontend. |
| **Git** | Para clonar. |
| Linux/WSL2 (ext4) | `PRODEX_HOME` e os `CODEX_HOME` por perfil **devem** ficar em ext4 (não 9p). |
| Rust/cargo | Só se for **compilar** o `prodex` do source (senão usa o binário pinado já buildado). |
| Portas livres | 5432, 8080, 3000 (e 43117/43118/6379 para o runtime prodex). Ver §7. |

Não é preciso nada além de clonar + Docker para o caminho feliz. O runtime prodex (Rust) é acionado
internamente pelo daemon quando um agente roda.

---

## 3. Serviços — quantos, nomes, imagens, portas, IPs
### 3.1 Contêineres do deploy padrão (`docker-compose.selfhost.yml`) — **3 serviços**
| # | Serviço (container) | Imagem | Porta (host→container) | IP/bind | Função |
|---|---|---|---|---|---|
| 1 | `multica-postgres` | `pgvector/pgvector:pg17` | `127.0.0.1:5432→5432` | loopback | Estado compartilhado (Postgres‑only) |
| 2 | `multica-backend` | `ghcr.io/multica-ai/multica-backend:latest` | `127.0.0.1:8080→8080` | loopback | API/daemon Go (L4) |
| 3 | `multica-frontend` | `ghcr.io/multica-ai/multica-web:latest` | `127.0.0.1:3000→3000` | loopback | UI web (Next) |

> **Segurança:** todos os binds são `127.0.0.1` de propósito. **Não** troque para `0.0.0.0` — Docker
> fura o firewall do host (UFW/iptables). Para acesso externo, use reverse proxy (Caddy/nginx/Cloudflare)
> terminando TLS e apontando para 127.0.0.1:8080 (backend) e :3000 (frontend).
> **IPs:** você acessa por `localhost`/`127.0.0.1`. Entre contêineres, o backend fala com o Postgres pelo
> nome de serviço `postgres` (rede interna do Compose), não por IP fixo.

### 3.2 Runtime prodex (não é serviço do Compose — é lançado pelo daemon)
| Componente | Porta padrão | Como sobe |
|---|---|---|
| `prodex-sidecar` (Rust, L2) | `127.0.0.1:43117` | Lançado pelo backend/daemon Go quando há sessão de runtime |
| `prodex gateway` | `127.0.0.1:43118` | Subprocesso do sidecar (proxy OpenAI‑compat) |
| Binário `prodex` (pinado) | — | `~/runtime/prodex-src/target/release/prodex` (v0.246.0 / commit `7750da9b`) |

### 3.3 Opcionais
| Serviço | Porta | Quando |
|---|---|---|
| **Redis** | `127.0.0.1:6379` | “Se necessário” (cache/coordenação). Não está no compose padrão. |
| **Observabilidade** | Grafana/Prometheus + `postgres-exporter:9187` | `deploy/observability/docker-compose.yml` (opcional) |

**Total no caminho padrão:** 3 contêineres (PG, backend, frontend). Com runtime ativo: + sidecar + gateway
(processos, não contêineres). Com opcionais: + Redis + stack de observabilidade.

---

## 4. Ordem CORRETA de subida (start)
O Compose já resolve a dependência, mas a ordem lógica é:
1. **Postgres** (precisa ficar `healthy` — healthcheck `pg_isready`).
2. **Backend** (`depends_on: postgres healthy`) — aplica migrations, expõe API em :8080.
3. **Frontend** (`depends_on: backend`) — UI em :3000.
4. **(Runtime prodex)** — sobe **sob demanda**, quando um agente inicia uma sessão. Começa em
   **shadow** (Smart Context observando, sem reescrever) — ver §8.

Comando único (faz tudo na ordem):
```bash
docker compose -f docker-compose.selfhost.yml up -d
docker compose -f docker-compose.selfhost.yml ps      # confira os 3 "Up"/"healthy"
```

### É linha de comando ou botão no front?
- **Subir/descer a aplicação:** linha de comando (`docker compose`). Não há botão de deploy no front.
- **Operar agentes/sessões:** pela UI (:3000) depois de logar. O runtime prodex é acionado pela
  plataforma, não manualmente.

---

## 5. Verificação pós‑subida (smoke rápido)
```bash
# Postgres
docker exec multica-postgres pg_isready -U multica -d multica          # → accepting connections
# Backend health
curl -s http://localhost:8080/healthz                                   # → ok
# Frontend
curl -s -o /dev/null -w '%{http_code}\n' http://localhost:3000          # → 200
# Runtime prodex (se/uma sessão estiver ativa)
curl -s http://127.0.0.1:43117/readyz                                   # → status alive + backend postgres
```

---

## 6. Usuários, senhas e segredos
| Item | Default no repo | O que fazer |
|---|---|---|
| Postgres | user `multica` / senha `multica` / db `multica` | **Trocar** `POSTGRES_PASSWORD` no `.env` |
| `JWT_SECRET` | `change-me-in-production` | **Obrigatório trocar** antes de produção |
| Signup da app | `ALLOW_SIGNUP=true` | 1º usuário se cadastra pela UI; restrinja com `ALLOWED_EMAILS`/`ALLOWED_EMAIL_DOMAINS` |
| Provedores de agente (Codex/Kiro/Antigravity) | **OAuth por perfil** (login nativo da CLI) | **Não há API key** — cada conta é OAuth isolada por perfil (ver §9) |
| E‑mail/OAuth Google/S3/etc. | vazios por default | Preencher só se for usar (todos opcionais no `.env`) |

> **Importante:** o RPP **não usa chaves de API** para os provedores — a autenticação é **OAuth por
> conta/perfil**, isolada (Codex=`CODEX_HOME`, Kiro=`XDG_DATA_HOME`/sqlite, Antigravity=`HOME`).

---

## 7. Portas ocupadas / conflitos — o que acontece e como resolver
Se uma porta nativa estiver ocupada ao subir:
- **Postgres/Backend/Frontend:** o Compose falha com “port is already allocated”. Solução: exporte as
  portas alternativas antes do `up` (elas são parametrizadas):
  ```bash
  BACKEND_PORT=8081 FRONTEND_PORT=3001 docker compose -f docker-compose.selfhost.yml up -d
  # (Postgres pode ser remapeado editando o mapping "127.0.0.1:5433:5432")
  ```
- **prodex sidecar/gateway (43117/43118):** parametrizados por env do daemon:
  `MULTICA_L2_...` (bind do sidecar), `PRODEX_GATEWAY_LISTEN` (default 127.0.0.1:43118). Se ocupado,
  o operador define uma porta livre nessas vars antes de iniciar a sessão.
- **Config recomendada:** manter tudo em `127.0.0.1`, portas default quando livres; só remapear o que
  conflitar. Não expor a `0.0.0.0`. Um Postgres “multica” dedicado por instância evita colisão com
  outros Postgres na mesma máquina.

Para ver o que está ocupando: `sudo lsof -i :5432` (ou `ss -ltnp | grep 5432`).

---

## 8. Configuração inicial recomendada do runtime (do runbook oficial)
Ao ativar o runtime prodex em produção, os defaults **seguros** são:
```text
PRODEX_SMART_CONTEXT_SHADOW=1          # começa observando (não reescreve)
PRODEX_SMART_CONTEXT_CANARY_PERCENT=0  # canário em 0% no início
PRODEX_AUTO_REDEEM_ENABLED=0           # reset-claim desligado por padrão
MULTICA_L2_KILL_SWITCH_DEFAULT=enabled # kill-switch disponível
MULTICA_L2_EVENT_STREAM_REQUIRED=1     # auditoria obrigatória
MULTICA_L2_LOG_REDACTION=1             # logs sem segredo
PRODEX_ALLOW_UNSAFE_CHILD_ENV=off      # segurança (Caveman/hook desabilitado)
```
Rollout do Smart Context é **shadow → canary → live** (nunca “live” direto).

---

## 9. Isolamento de credenciais (o coração do “Rotation‑Parity”)
- Cada conta de vendor vive **isolada**; **nunca** compartilhe a pasta de credenciais entre agentes
  (isso sobrescreve/derruba a outra conta). Alavancas: Codex `CODEX_HOME`, Kiro `XDG_DATA_HOME`(sqlite),
  Antigravity `HOME`.
- Ao esgotar a janela (~5h), o sistema **rotaciona** para outra conta OAuth do pool (ou `prodex redeem`)
  e retoma a tarefa. Compartilhamento concorrente de uma credencial válida por N agentes é permitido
  durante o ciclo.

---

## 10. Ordem CORRETA de desligamento (shutdown)
Inverso da subida:
```bash
# 1. (Se houver sessão de runtime) encerre sessões → sidecars/gateway param sozinhos
# 2. Derrube a app (frontend → backend → postgres, o Compose cuida da ordem):
docker compose -f docker-compose.selfhost.yml down            # mantém volumes (dados)
# Para apagar TAMBÉM os dados (destrutivo!):
# docker compose -f docker-compose.selfhost.yml down -v
```
- **Rollback de emergência** (voltar a `codex` cru, sem prodex): comando único testado
  `scripts/deploy/rollback-to-raw-codex.sh` (gated por `DEPLOY_OWNER_APPROVED=true`).
- **Kill‑switch** (desligar só o Smart Context/gateway sem derrubar a app): por tenant/provider/profile.

---

## 11. Capacidade do sistema
- **Modelo de capacidade:** paralelismo por **pool de perfis OAuth** — cada conta serve por ~5h e pode
  atender N agentes concorrentes no ciclo; ao esgotar, rotaciona. Logo, a capacidade escala com o
  **número de contas/perfis** disponíveis e com os limites do Postgres/host.
- **Limitantes reais:** limite de cota de cada conta de vendor (janela ~5h), conexões do Postgres, e
  CPU/RAM do host que roda os sidecars.
- **Números exatos de throughput** (req/s, sessões simultâneas máximas) **não estão medidos** neste
  marco — exigem teste de carga dedicado. Não são afirmados aqui para não inventar.

---

## 12. Day‑2 — Troubleshooting rápido
| Sintoma | Causa provável | Ação |
|---|---|---|
| Backend não sobe / reinicia | Postgres não `healthy` | `docker compose logs postgres`; aguarde healthcheck; confira `DATABASE_URL` |
| `port is already allocated` | Porta ocupada | Ver §7 (remapear `BACKEND_PORT`/`FRONTEND_PORT`) |
| `/readyz` do sidecar = 503 | Postgres inalcançável do sidecar | Suba/repare o Postgres; confira `PRODEX_PG_URL` |
| Agente para pedindo re‑login | Conta esgotou (~5h) e sem rotação | Verifique pool de perfis; rotação/`prodex redeem` |
| Conta “some” ao logar outra | Credenciais **compartilhadas** (violação de isolamento) | Garanta `CODEX_HOME`/`XDG_DATA_HOME`/`HOME` por conta |
| Smart Context sem efeito | Está em **shadow** (esperado no início) | Promova shadow→canary→live conforme §8 |
| Logs com segredo | Redação desligada | `MULTICA_L2_LOG_REDACTION=1` |
| Precisa reverter tudo | Incidente | `rollback-to-raw-codex.sh` (com aprovação) volta ao `codex` cru |

Logs: `docker compose -f docker-compose.selfhost.yml logs -f <serviço>`.
Evidências/estado operacional: `.deploy-control/` e `.deploy-control/evidence/` no repo.

---

## 13. Onde está cada coisa (mapa de arquivos)
| Assunto | Caminho |
|---|---|
| Compose padrão (3 serviços) | `multica-auth-work/docker-compose.selfhost.yml` |
| Compose dev (só Postgres) | `multica-auth-work/docker-compose.yml` |
| Observabilidade (opcional) | `multica-auth-work/deploy/observability/docker-compose.yml` |
| Runbook de deploy PROD | `docs/deploy/prod-rollout-runbook.md` |
| Rollback | `docs/deploy/rollback-operational-procedure.md` + `scripts/deploy/rollback-to-raw-codex.sh` |
| AS‑IS / TO‑BE | `docs/project/01-as-is.md` / `02-to-be.md` |
| Contrato Go↔L2 | `docs/contracts/l2-runtime-contract.md` |
| Binário prodex pinado | `~/runtime/prodex-src/target/release/prodex` (v0.246.0 / `7750da9b`) |

---

## 14. Diagramas de arquitetura (macro / deep / micro)
Entregues como HTML nesta mesma pasta:
- `RPP_architecture_macro.html` — visão executiva (camadas L4/L2, fluxo desired‑state ↔ eventos).
- `RPP_architecture_deep.html` — componentes internos (daemon, sidecar, gateway, Postgres, pool de perfis).
- `RPP_architecture_micro.html` — caminho de um request/sessão (start → afinidade → Smart Context → fallback).

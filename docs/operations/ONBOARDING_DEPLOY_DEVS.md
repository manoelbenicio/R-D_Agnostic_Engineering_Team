# PROMPT DE DEPLOY — Onboarding para Devs (Multica / RPP) — pt-BR

> Objetivo: qualquer dev com acesso ao repositório sobe o sistema do zero seguindo estes passos, em ~10 min.
> Alvo: máquina Linux/WSL2 com Docker. Tudo abaixo foi validado no ambiente de referência.
> Documentos de apoio nesta mesma pasta: `RPP_GUIA_DEPLOY_E_OPERACAO_pt-BR.md` (manual completo),
> `RPP_CHECKLIST_DEPLOY.html` (checklist interativo), diagramas `RPP_architecture_*.html`.

---

## 0. Pré-requisitos (verifique antes)
```bash
docker --version && docker compose version   # Docker + Compose
git --version                                # Git
make --version && openssl version            # make + openssl (usados pelo 'make selfhost')
```
- Portas necessárias livres: **8080** (backend) e **3000** (frontend). PostgreSQL fica interno.
  Verifique: `for p in 8080 3000; do ss -ltn | grep -q ":$p " && echo "OCUPADA $p" || echo "livre $p"; done`
- Se `:3000` estiver ocupada, use uma alternativa no passo 2 (`FRONTEND_PORT=3100`).

## 1. Clonar o projeto
```bash
git clone https://github.com/manoelbenicio/R-D_Agnostic_Engineering_Team.git
cd R-D_Agnostic_Engineering_Team/multica-auth-work
```

## 2. Subir a stack (um comando)
Opção A — imagens oficiais (recomendado):
```bash
make selfhost
```
Opção B — se as imagens não puxarem (tag não publicada), buildar do código:
```bash
make selfhost-build
```
> `make selfhost` gera o `.env` automaticamente (JWT_SECRET e senha do Postgres **aleatórios**), faz o pull
> das imagens, sobe Postgres + backend + frontend e aguarda o `/health`. **As migrations do banco rodam
> automaticamente** no start do backend (`./migrate up`). Não é preciso editar nada para o caminho padrão.

Se a porta 3000 estiver ocupada:
```bash
FRONTEND_PORT=3100 docker compose -f docker-compose.selfhost.yml up -d
```

## 3. Validar (smoke)
```bash
docker compose -f docker-compose.selfhost.yml ps          # 3 serviços Up; postgres (healthy)
curl -s http://localhost:8080/health                      # {"status":"ok"}
curl -s http://localhost:8080/readyz                      # {"status":"ok","checks":{"db":"ok","migrations":"ok"}}
curl -s -o /dev/null -w '%{http_code}\n' http://localhost:3000   # 200 (ou :3100)
```

## 4. Primeiro login + workspace
1. Abra a UI no navegador: **http://localhost:3000** (ou a porta escolhida).
2. Informe seu e-mail. Sem e-mail configurado, o **código de verificação sai no log do backend**:
   ```bash
   docker compose -f docker-compose.selfhost.yml logs backend | grep -i "verification code"
   ```
3. Faça login e **crie seu primeiro workspace**.

## 5. Conectar a CLI + daemon (para executar agentes)
Instale a CLI `multica` (Homebrew/script/PowerShell — ver docs). Depois, na mesma máquina:
```bash
multica setup self-host
```
> Aponta a CLI para `localhost:8080` (backend) e `localhost:3000` (frontend), faz login no navegador,
> guarda o PAT localmente e **inicia o daemon automaticamente**. Reiniciar o daemon: `make daemon`.
Requer uma ferramenta de código instalada na máquina do daemon (ex.: `codex`).

## 6. Acesso
- **Mesma máquina:** UI `http://localhost:3000` · API `http://localhost:8080`.
- **De outra máquina:** as portas ficam em `127.0.0.1` (loopback). Use um túnel SSH:
  ```bash
  ssh -L 3000:localhost:3000 -L 8080:localhost:8080 <host-do-servidor>
  ```
  ou coloque um **proxy reverso** com TLS (Caddy/nginx/Cloudflare) apontando para `127.0.0.1:8080` e `:3000`
  e defina `FRONTEND_ORIGIN=https://seu-dominio` no `.env` (senão o WebSocket é rejeitado).

## 7. Observabilidade (opcional)
```bash
docker compose -f deploy/observability/docker-compose.yml up -d
```
> Sobe Prometheus (`:9090`), Grafana (`:3000` → **remapeie**, pois conflita com o frontend; ex.: `:13000`),
> Alertmanager (`:9093`), postgres-exporter (`:9187`).
Para o Prometheus coletar as métricas do backend (target `credential-service`):
1. No `.env`: `METRICS_ADDR=0.0.0.0:9090` e recrie o backend (`docker compose ... up -d backend`).
2. Anexe o backend à rede da observabilidade (já há um `docker-compose.override.yml` no repo):
   ```bash
   docker compose -f docker-compose.selfhost.yml -f docker-compose.override.yml up -d backend
   ```
3. Confirme os targets:
   ```bash
   curl -s -X POST http://127.0.0.1:9090/-/reload; sleep 5
   curl -s 'http://127.0.0.1:9090/api/v1/targets?state=active'   # credential-service/postgres/prometheus -> up
   ```

## 8. Encerrar / rollback
```bash
make selfhost-stop                                        # para a stack
docker compose -f docker-compose.selfhost.yml down        # remove containers (mantém dados)
docker compose -f docker-compose.selfhost.yml down -v     # remove TAMBÉM os dados (destrutivo)
```

## 9. Troubleshooting comum
- **Backend reiniciando:** quase sempre `DATABASE_URL`/`JWT_SECRET` ruim no `.env` → `docker compose ... logs backend`.
- **Código de verificação não chega:** sem e-mail configurado → veja `[DEV] Verification code` no log do backend.
- **WebSocket não conecta (deploy público):** defina `FRONTEND_ORIGIN` com o domínio real e reinicie o backend.
- **Porta ocupada:** use `BACKEND_PORT`/`FRONTEND_PORT` alternativos.

## 10. Referências (nesta pasta `docs/operations/`)
- `RPP_GUIA_DEPLOY_E_OPERACAO_pt-BR.md` — manual completo (serviços, portas, capacidade, arquitetura).
- `RPP_CHECKLIST_DEPLOY.html` — checklist interativo (marque cada passo).
- `RPP_architecture_macro/deep/micro.html` — diagramas de arquitetura.

> Regra de ouro: para um deploy limpo, **não copie `.env` de outra máquina** — deixe o `make selfhost`
> gerar o seu (segredos aleatórios). Isso evita herdar credenciais/config de outro ambiente.

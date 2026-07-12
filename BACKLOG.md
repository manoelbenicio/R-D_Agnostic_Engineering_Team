# Backlog — R&D Agnostic Engineering Team (Multica + prodex)

> Substitui o antigo backlog do AgentVerse (removido junto com o SPA — ver
> commit `a61281e` e branch `backup/pre-agentverse-cleanup`).
> Regra do dono: **só mexer no frontend depois da integração Multica+prodex 100%.**

## Itens pendentes

| # | Criticidade | Item | Descrição | Observações |
|---|---|---|---|---|
| 1 | 🟡 Cosmético/UX | **Remover/refazer a página de introdução (onboarding/login)** | Arrancar fora a tela de introdução atual do Multica web (`multica-auth-work/apps/web/app/(auth)` / `features/auth`) — considerada ruim pelo dono. Rework do fluxo de entrada. | **Só depois** da integração 100%. Não tocar agora. |
| 1b | 🟡 Cosmético/UX | **Remover landing de marketing + patrocinadores** | Tirar toda a poluição da landing upstream do Multica: seção `app/(landing)`, `features/landing`, `content/use-cases`, `public/usecases`, logos de patrocinadores/sponsors e conteúdo promocional. Deixar só o app limpo. | **Só depois** da integração 100%. |
| 2 | 🟢 Dev-infra | **Login local sem email** | Em teste local o backend está `APP_ENV=production`, então o login por email exige entrega real. Para testes, ligar `APP_ENV=development` + `MULTICA_DEV_VERIFICATION_CODE` (código fixo) no `.env`, ou configurar SMTP/Resend. | Aplicado localmente (código `123456`); rever ao clonar do zero. |
| 3 | 🔵 Observabilidade | **Telemetria de token/quota do Antigravity** | Único vendor sem captura de token. O CLI `agy` (consumer) não expõe uso por turno nem quota de forma programável estável. Ver "Notas técnicas" abaixo. | **Bloqueado por limitação do fabricante.** Só via engenharia-reversa frágil. |

## Notas de estado (base de dados)

- O banco atual (`multica_pgdata`) é uma **base nova**: o `admin@` com nome de
  squad/domínio criado antes **não está mais montado** (o volume foi recriado;
  não há dump/backup no disco). Recriar do zero quando for testar de novo.
- Contas presentes hoje (apenas teste): `codex55a@example.com`
  (workspace "Codex55A Workspace", prefixo `COD`) e `qa-e2e@multica.local`.
- Signup liberado (`ALLOW_SIGNUP=true`). Códigos de verificação ficam na tabela
  `verification_code` (validade 10 min).

## Concluído recentemente

- ✅ **Remoção do AgentVerse SPA** (frontend errado, de outro projeto) — commit
  `a61281e`. Frontend de referência passa a ser o Multica web
  (`multica-auth-work/apps/web`, http://localhost:3100). Backup:
  `backup/pre-agentverse-cleanup`.

## Notas técnicas — Antigravity token/quota (verificado 2026-07-12)

Investigação na fonte do fabricante (não repetir do zero):

- **Tokens por turno:** o CLI `agy` (`-p/--print`) só imprime **texto**; não há flag
  `--json`/`--output` nem subcomando de usage. O `--log-file` (glog) **não** contém
  `usageMetadata`/`promptTokenCount`. Verificado. Capturar token real exigiria
  **reescrever o backend** pra chamar o gateway direto (não usar o `agy`).
- **Quota:** endpoint interno confirmado no log do `agy`:
  `POST https://cloudcode-pa.googleapis.com/v1internal:retrieveUserQuotaSummary`.
  Porém a conta é `authMethod=consumer` e o token salvo em
  `~/.gemini/antigravity-cli/antigravity-oauth-token` **retorna HTTP 401** quando usado
  direto como Bearer (a API interna espera OAuth2 cloud-platform/projeto). Replicar o
  handshake consumer do `agy` é engenharia-reversa não-documentada e quebra a cada update.
- **Vias possíveis (todas frágeis/não-oficiais):** replicar auth consumer · scrapear a TUI
  `Models & Quota` · serviço local do Antigravity IDE (GetUserStatus por porta, inexistente headless).
- **Decisão do dono (2026-07-12):** deixar no backlog; não implementar hack frágil.

## Status de captura de tokens por vendor

- ✅ **codex** — verificado empiricamente (288.5K medido em task real).
- ✅ **kiro · opencode · claude · gemini · kimi** — extração implementada nos backends
  (`server/pkg/agent/*.go`), parseando o usage da saída estruturada de cada CLI.
- ❌ **antigravity** — ver notas acima (backlog, limitação do fabricante).

## Observabilidade (concluído — v2.0.1)

- ✅ Stack `multica-observability` up: Prometheus :9090, Alertmanager :9093,
  Grafana **:3005** (porta via `GRAFANA_PORT`, default 3000), postgres-exporter :9187.
- ✅ Backend expõe `/metrics` (`METRICS_ADDR`), 407 métricas incl. `multica_agent_task_*`.
- ✅ 7 dashboards Grafana, incl. o custom **Multica — Agents & Tasks**.

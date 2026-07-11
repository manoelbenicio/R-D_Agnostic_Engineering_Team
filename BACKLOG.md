# Backlog — R&D Agnostic Engineering Team (Multica + prodex)

> Substitui o antigo backlog do AgentVerse (removido junto com o SPA — ver
> commit `a61281e` e branch `backup/pre-agentverse-cleanup`).
> Regra do dono: **só mexer no frontend depois da integração Multica+prodex 100%.**

## Itens pendentes

| # | Criticidade | Item | Descrição | Observações |
|---|---|---|---|---|
| 1 | 🟡 Cosmético/UX | **Remover/refazer a página de introdução (onboarding/login)** | Arrancar fora a tela de introdução atual do Multica web (`multica-auth-work/apps/web/app/(auth)` / `features/auth`) — considerada ruim pelo dono. Rework do fluxo de entrada. | **Só depois** da integração 100%. Não tocar agora. |
| 1b | 🟡 Cosmético/UX | **Remover landing de marketing + patrocinadores** | Tirar toda a poluição da landing upstream do Multica: seção `app/(landing)`, `features/landing`, `content/use-cases`, `public/usecases`, logos de patrocinadores/sponsors e conteúdo promocional. Deixar só o app limpo. | **Só depois** da integração 100%. |
| 2 | 🟢 Dev-infra | **Login local sem email** | Em teste local o backend está `APP_ENV=production`, então o login por email exige entrega real. Para testes, ligar `APP_ENV=development` + `MULTICA_DEV_VERIFICATION_CODE` (código fixo) no `.env`, ou configurar SMTP/Resend. | Muda comportamento de auth; decidir ao retomar. |

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

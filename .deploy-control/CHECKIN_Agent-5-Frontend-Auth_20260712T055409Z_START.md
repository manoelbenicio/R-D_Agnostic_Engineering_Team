---
agent: Agent-5-Frontend-Auth
model: glm-5.2 (High)
stream: W1-T1.5
started_at: 2026-07-12T05:54:09Z
finished_at:
status: IN_PROGRESS
authority: openspec/changes/native-runtimes-onboarding (task 1.5) + specs/onboarding/spec.md
files_locked:
  - multica-auth-work/apps/web/app/(auth)/login/page.tsx
  - multica-auth-work/apps/web/app/(auth)/login/page.test.tsx
  - multica-auth-work/apps/web/app/(auth)/onboarding/page.tsx
  - multica-auth-work/packages/views/auth/login-page.tsx
  - multica-auth-work/packages/views/auth/login-page.test.tsx
  - multica-auth-work/packages/views/auth/use-logout.ts
  - multica-auth-work/packages/views/auth/index.ts
  - multica-auth-work/apps/web/features/auth/*
depends_on: []
build_result:
notes: |
  Owner decision RESOLVED (2026-07-12): login/senha simples now; Firebase later
  WITHOUT rework. Task 1.5 UNBLOCKED.
  Scope: simple username/password login in the app design-system (same colors as
  kanban/Agentes menu); structured to plug Firebase later without rework. Remove
  email-code flow from auth pages.
  Disjoint with Agent-6: A5 = auth/pages + login view + email-code removal;
  A6 = design-system/cores/marketing-removal/QA. NOT touching (landing)/
  features/landing/content/use-cases/sponsors until Kiro confirms boundary
  (design.md lists marketing removal under A5; prompt-split lists under A6).
---

# CHECKIN START — Agent-5 (Frontend-Auth), Task 1.5

## Escopo
Implementar LOGIN/SENHA SIMPLES no design-system do app (mesmas cores do
kanban/menu Agentes), estruturado para plugar Firebase depois SEM rework.
Remover o fluxo de código por email das páginas de auth.

## Arquivos (disjuntos com Agent-6)
- auth pages: apps/web/app/(auth)/login, (auth)/onboarding
- auth view package: packages/views/auth/*
- auth feature: apps/web/features/auth/*

## Dependências
- BLOCO: endpoint backend POST /auth/login(user,senha) + credential store.
  Hoje so existe /auth/send-code + /auth/verify-code (router.go:471-472) +
  /auth/google; api client so tem sendCode/verifyCode/googleLogin
  (client.ts:393-407). Backend Go fora do escopo W1 frontend (shared/wiring =
  Kiro W2). Escalado ao Kiro em 2026-07-12T05:55Z.

## Riscos
1. Ownership boundary do (landing)/marketing removal (A5 vs A6) — escalado ao Kiro.
2. stash@{0} (trabalho RPP-VENDORMATRIX da sessão anterior) preservado; aguarda
   decisão de reconcile do Kiro antes de descartar/reatribuir.
3. Estrutura Firebase-ready: definir contrato de auth service (interface) p/ que
   troca de backend (simple->firebase) seja injecao de dependencia sem rework.

## Plano
1. Ler auth atual (login page + login-page view + features/auth) p/ entender
   mecanismo atual (email-code/magic-link/OAuth?).
2. Definir AuthProvider/AuthService interface (Firebase-ready, strategy pattern).
3. Implementar simple username/password na UI com cores do design-system.
4. Remover email-code flow das auth pages.
5. Verde no build web (pnpm build / next build) antes de DONE.

## Boot confirmado
- HERDR_ENV=1 exportado.
- git pull origin main: ff 52cdd87->610c847, working tree clean.
- Prompt Agent-5 + CODIGO DE CONDUTA + spec onboarding lidos.
- Pane w3:pW (Kiro) reconfirmada via herdr agent list.

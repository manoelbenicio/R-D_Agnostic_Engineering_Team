# CHECK-IN Codex-56-B — DONE task 1.5

- UTC: `2026-07-14T21:11:49Z`
- Change: `native-runtimes-onboarding`
- Task: `1.5`
- Entrega: landing/use-cases/sponsors e dependências exclusivas removidos; `/` redireciona para `/login`; fluxo email-code substituído por formulário email/senha.
- Contrato: `AuthService` + `SimpleAuthService` delegam para `api.login(email, password)` em `POST /auth/login`, preservando cookie, bearer token, Google OAuth, callback CLI e handoff desktop.
- Validação: `@multica/core` 33 testes focados; `@multica/views` 18 testes; wrapper `@multica/web` 3 testes; typecheck de core/views/web; `pnpm --filter @multica/web build` verde com Next.js 16.2.6.
- Restrições respeitadas: nenhum arquivo backend Go editado.
- Estado: concluído.

# Tasks: Isolamento de credencial por conta de agente

## 0. Levantamento do modelo de referência (fonte de verdade)
- [ ] 0.1 Mapear `PROVIDERS` de `infra/cao/auth_routes.py` (config_dirs, credential_names, env).
- [ ] 0.2 Mapear `resolveSessionEnv` em `src/canvas-reconciler/reconciler.ts`.
- [ ] 0.3 Mapear o contrato de `src/api/session-discovery.ts` e `session-store.ts`.
- [x] 0.4 Confirmar como a versão pura (execenv) monta hoje o home de credencial.

## 1. Config dir por conta (paridade com o modelo)
- [ ] 1.1 Layout de config dir por conta, por provedor.
- [ ] 1.2 Injeção da env var nativa por provedor (CODEX_HOME, CLAUDE_CONFIG_DIR, GEMINI_CONFIG_DIR/CLOUDSDK_CONFIG, KIRO_HOME).
- [ ] 1.3 Fallback: sem atribuição → comportamento global atual preservado.

## 2. Discovery / login / revoke por conta
- [ ] 2.1 `GET /auth/sessions` lista contas por provedor (mesmo contrato).
- [ ] 2.2 `POST /auth/login` com `config_dir` da conta.
- [ ] 2.3 `DELETE /auth/sessions/:id` revoga a conta.

## 3. Atribuição sessão → agente e isolamento de env
- [ ] 3.1 `session_id` por nó/agente até o terminal (via resolveSessionEnv).
- [ ] 3.2 Terminal recebe só as env vars da sua conta (sem cross-contamination).
- [ ] 3.3 Cobrir os runtimes: codex, claude, gemini/agy, kiro (glm quando aplicável).

## 4. Fase 2 — Rotação automática (incluída agora)
- [x] 4.1 Detectar sessão esgotada/`expired` (status + expires_at do discovery).
- [x] 4.2 Selecionar próxima conta disponível do mesmo provedor.
- [ ] 4.3 Reatribuir o agente à nova conta sem intervenção manual.
- [ ] 4.4 Registrar/alertar a troca (aproveitar useSessionMonitor/isExpiringSoon).

## 5. Verificação
- [ ] 5.1 Build + testes (frontend e backend).
- [x] 5.2 Teste: 2 contas do mesmo provedor coexistem sem sobreposição.
- [ ] 5.3 Teste: rotação automática ao esgotar a conta ativa.
- [ ] 5.4 Confirmar que nenhum segredo aparece em logs (sanitizeForLog).

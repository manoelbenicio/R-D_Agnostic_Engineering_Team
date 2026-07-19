# Proposal — Native Runtimes (NIM, Cline) + Model Discovery Fix + Onboarding Rework

## Why
Multica precisa de dois runtimes nativos que hoje não existem, corrigir a descoberta de
modelos que não popula na UI, e trocar o onboarding (landing de patrocinadores + login por
código de email) por um login limpo no mesmo design do app.

- **NVIDIA NIM**: runtime nativo **do zero** (OpenAI-compatible), NÃO via opencode.
- **Cline**: backend nativo Cline 3.x via `cline --acp`, falando ACP JSON-RPC 2.0
  por stdin/stdout. O flag `--json` pertence ao modo headless separado de saída de prompt e
  é incompatível com o handshake ACP quando combinado com `--acp`.
- **Descoberta de modelos**: o fluxo assíncrono trava/lento (`agy models` ~20s) e a UI fica vazia ("nada do CLI").
- **Onboarding**: remover marketing/patrocinadores + fluxo de código por email; login no design do kanban/agentes.

## What Changes
- **ADDED** runtime nativo `nim` (backend, probe, factory, isolamento de credencial, rotação, catálogo).
- **ADDED** runtime nativo `cline` (backend ACP, probe, factory).
- **MODIFIED** descoberta de modelos: timeout, cache e surface de erro; UI popula de forma confiável.
- **MODIFIED** onboarding/auth: remover landing de sponsors + verificação por email; login consistente com o design-system.

## Impact
- Código: `server/pkg/agent/*`, `server/internal/daemon/*`, `apps/web/app/(auth|landing)`, `packages/views/auth`, design-system.
- Execução: **coders experts (codex & cia)**. **Kiro apenas orquestra e valida** (não produz código).
- Decisão do dono PENDENTE: modelo de auth final (login/senha simples vs sem-fricção) — não implementar antes de confirmar.

## Non-goals
- Usar NIM via opencode (explicitamente rejeitado pelo dono).
- Telemetria de token/quota do antigravity (backlog separado, limitação do fabricante).

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
- Decisão do dono RESOLVIDA em 2026-07-12: login/senha simples agora, com interface
  Firebase-ready para evolucao posterior sem rework.

## Non-goals
- Usar NIM via opencode (explicitamente rejeitado pelo dono).
- Telemetria de token/quota do antigravity (backlog separado, limitação do fabricante).

## Implementation status — 2026-07-29

- Backend NIM, isolamento NIM, backend Cline ACP, model discovery e auth backend estão
  implementados; wiring compartilhado de NIM/Cline também está concluído.
- Task 1.5 (onboarding frontend & remoção de marketing/sponsors/email-code) foi concluída e
  aceita no Wave 2 via ORQ-51.
- Paridade visual (Agent-6 / ORQ-52), rebuild/restart dos runtimes (ORQ-53), smoke real e UAT
  permanecem abertos e rastreados nos cards correspondentes.
- Os agentes AGY criados no workspace e o reparo de model selection pertencem a outra frente;
  eles não comprovam que NIM/Cline estejam online ou que o onboarding novo esteja aceito.

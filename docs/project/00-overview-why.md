# 00 — Visão Geral & Justificativa (Why)

**Projeto:** Isolamento de credencial OAuth por conta de agente (mudança cirúrgica)
**Produto base:** Multica (mantido íntegro — sem reescrita)
**Data:** 2026-07-01
**Status:** Discovery concluído; implementação a iniciar (Codex piloto)

---

## 1. O problema (em uma frase)

O operador mantém **múltiplas contas por vendor** (várias Codex, Kiro, Antigravity),
mas hoje todos os agentes compartilham **uma única credencial global por vendor** —
então logar numa segunda conta **sobrescreve** a primeira, e ao esgotar a janela de
~5h é preciso **deslogar/relogar manualmente**, interrompendo a operação.

## 2. Por que isso importa (impacto)

- **Interrupção operacional:** quando um agente esgota a cota, para na tela e exige
  intervenção humana (deslogar conta A, logar conta B).
- **Desperdício de capacidade:** há contas ociosas com cota disponível que não são
  aproveitadas porque só uma credencial "cabe" por vez.
- **Risco de corrupção de credencial:** o modelo atual usa **symlink de `auth.json`
  para um único home global** — o refresh de uma conta pode contaminar outra.
- **Severidade financeira:** paralisação da frota de agentes tem custo direto
  (trabalho parado) e indireto (retrabalho, contexto perdido).

## 3. Objetivo (e o que explicitamente NÃO é)

**É:** uma mudança **cirúrgica** no mecanismo de autenticação OAuth para **separar
credenciais por conta**, de modo que múltiplas contas do mesmo vendor coexistam sem
sobreposição, e (Fase 2) a troca ao esgotar a cota seja automática.

**NÃO é:**
- Reescrever o Multica ou trocar o "cérebro" (orquestração, canvas, dispatch).
- Migrar para outra plataforma (o AOP serviu **apenas como referência de pesquisa**).
- Alterar o source original — todo desenvolvimento ocorre na cópia local
  `multica-auth-work/`.

## 4. Princípios de design

1. **Mínima superfície de mudança.** Tocar apenas o caminho de auth/credencial.
2. **Manter o produto funcionando.** Nenhuma regressão em orquestração/UI/dispatch.
3. **Armazenar/restaurar AS-IS.** Formato bruto do vendor, estado idêntico.
4. **Postgres-only.** Qualquer persistência nossa usa Postgres; nunca SQLite.
5. **De-branding incremental.** Arquivos tocados saem higienizados de "Multica",
   rumo ao produto agnóstico futuro — sem alterar comportamento.
6. **Verificável.** Cada mudança compila via container `golang:1.26-alpine`.

## 5. Fases

| Fase | Escopo | Estado |
|------|--------|--------|
| **1** | Isolamento de credencial por conta (Codex → Kiro → Antigravity) | a iniciar |
| **2** | Rotação automática ao esgotar cota/5h (detecção + troca + retomada) | desenhada |

## 6. Evidência da base de pesquisa

- Anatomia real de credencial por vendor verificada no servidor de produção.
- Honra das env vars de isolamento comprovada empiricamente (Codex `CODEX_HOME`,
  Kiro `XDG_DATA_HOME`/`KIRO_API_KEY`, Antigravity `HOME`).
- Portabilidade dos formatos confirmada na documentação oficial dos vendors.
- Modelo de referência (seats/sessions/rotation) estudado no AOP.

Ver: `01-as-is.md`, `02-to-be.md`, `03-requirements.md`, `04-architecture.md`,
`05-observability.md`.

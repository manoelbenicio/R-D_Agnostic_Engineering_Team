---
phase: 02-forkmap
plan: 01
status: done
completed_at: 2026-07-05T02:43:29Z
requirements: [REQ-09]
artifacts:
  - docs/prodex/prodex-fork-map.md
---

# SUMMARY 02-01 — prodex fork map

## Resultado

Executado conforme `.planning/phases/02-forkmap/02-01-PLAN.md`.

`docs/prodex/prodex-fork-map.md` agora contém uma tabela explícita com todos os
44 crates do prodex pinado, cada um com:

- função;
- área do plano;
- fork zone (`core`, `runtime`, `provider`, `security`, `ui`, `infra`,
  `plugin`);
- status/fork stance.

## Itens Cobertos

- Runtime boundaries isolados:
  - runtime proxy;
  - gateway;
  - Smart Context;
  - state;
  - redeem.
- MCP crates mapeados:
  - `prodex-mcp-stdio`;
  - `prodex-runtime-anthropic`;
  - `prodex-runtime-gemini`;
  - `prodex-runtime-gemini-cli-compat`.
- Security-sensitive crates destacados:
  - `prodex-secret-store`;
  - `prodex-profile-export`;
  - `prodex-presidio`;
  - `prodex-redaction`;
  - `prodex-runtime-cookies`.
- Gap crates ligados a REQs:
  - `prodex-memory` -> REQ-27/REQ-37b;
  - `prodex-presidio`/`prodex-redaction` -> REQ-28;
  - `prodex-runtime-broker`/`prodex-runtime-broker-log` -> REQ-29;
  - `prodex-runtime-cookies` -> REQ-30;
  - `prodex-quota`/`prodex-runtime-quota` -> REQ-31;
  - `prodex-caveman-assets` -> REQ-32.

## Fork Boundary

Adicionada seção `fork boundary` com:

- candidatos ao fork L2 sidecar;
- crates que permanecem upstream;
- decisões deferidas;
- pontos de interface (`prodex-shared-types`, `prodex-codex-config`,
  `prodex-session-store`, `prodex-shared-codex-fs`);
- gaps de SBOM e conformance.

## Verificação

- `grep -c "prodex-" docs/prodex/prodex-fork-map.md` -> `117`
- tabela principal com crates: `44`
- comparação source-vs-doc: nenhum crate ausente
- `grep -q "fork boundary" docs/prodex/prodex-fork-map.md` -> OK

## Observações

Os arquivos de workflow/template referenciados no plano sob
`~/.gemini/antigravity/get-shit-done/` não existem neste host. A execução seguiu
as instruções materiais do próprio PLAN e dos contextos locais lidos.

# Evidencia — Harness Auth Isolation Fix (R1 clobber)
- ts: 20260704T211127Z  | by: Tech-Lead (Opus 4.8) via SSH + orchestrator
- causa: 4 workers Codex compartilhavam ~/.codex -> login sobrescrevia o anterior (clobber) -> colapso na ultima conta.
- fix: CODEX_HOME isolado por worker (~/.codex-a/b/c/d) setado por pane + relaunch (orchestrator).
- cleanup: auth.json copiado (conta unica) movido p/ .bak em cada home -> login limpo por conta.
- proximo passo (owner): logar 1 conta DISTINTA por pane (A=w3:pJ,B=w3:pM,C=w3:pK,D=w3:p9) -> persiste, 4 em paralelo.
- escopo: SO harness ($HOME/.codex-*); nenhum arquivo de produto/repo tocado.

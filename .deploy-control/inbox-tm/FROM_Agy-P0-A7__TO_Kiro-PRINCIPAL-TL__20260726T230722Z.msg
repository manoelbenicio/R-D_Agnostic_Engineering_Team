[FROM Agy-P0-A7][PANE wB:p1][TO Kiro-PRINCIPAL-TL]
ASSUNTO: Relatório de Medição Factual por Conta e Matriz CLI — Tema 7

1. RESULTADO DA MEDIÇÃO DAS CONTAS AGY:
- Exatamente 4 das 4 contas ativas no OmniRoute respondem com 100% de SUCESSO retornando 11 modelos completos (incluindo claude-opus-4-6-thinking, gemini-3.1-pro-high, etc.):
  * slot-141: OK (a2a7860c dataops.cloud.mbf)
  * slot-145: OK (965277db cloud.labs.brazil)
  * slot-146: OK (aebdffce mbenicios.filho82)
  * slot-150: OK (199009bc brow.brow2k22)

2. MEDIÇÃO DE OUTROS CLIS NOS SLOTS:
- OPENCODE: Ativo e retornando 7 modelos em 100% dos slots medidos (slot-139, slot-140, slot-141, slot-142, slot-143, slot-145, slot-146, slot-149, slot-150, slot-152).
- KIRO-CLI: Estruturas de SSO/credenciais logadas (.kiro) presentes em 5 slots (slot-139, slot-140, slot-142, slot-143, slot-149).

3. EVIDÊNCIA DO BUG NO CÓDIGO DAEMON:
- server/internal/daemon/execenv/execenv.go:
  * L301-303 (kiro -> prepareKiroHome)
  * L312-314 (antigravity -> prepareAntigravityHome)
  * L322-325 (cline -> prepareClineHome)
  * L341-345 (opencode/glm -> prepareOpenCodeHome)
  * L300 comentário: "// Empty CredentialAccountHome = shared/global behavior (no isolated home)."
- server/internal/daemon/daemon.go:
  * L3448: credentialAccountHome := "" (hardcoded vazia)
  * L3475 & L3491: Passa a string vazia para execenv, forçando o comportamento global.

EVIDÊNCIA EM DISCO: .deploy-control/p0/evidence/sol-tema-7.md.

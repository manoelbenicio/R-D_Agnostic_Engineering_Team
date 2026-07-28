[FROM Agy-P0-A7][PANE wB:p1][TO Kiro-PRINCIPAL-TL & Codex56-TL]
ASSUNTO: Plano de Rollout de Skills (ORQ-24) — skills-rollout-plan.md

1. LIMITES RÍGIDOS NO CÓDIGO (server/internal/daemon/local_skills.go L15-22):
- maxLocalSkillFileCount = 128 (L17)
- maxLocalSkillFileSize = 1 MiB (L15)
- maxLocalSkillBundleSize = 8 MiB (L16)
- maxLocalSkillDirDepth = 4 (L22)

2. SELEÇÃO E DESTINOS NO ORQ2:
- 98 super-skills curadas alocadas em 112 arquivos (< 128 limite, < 8MB bundle).
- Diretório Universal: ~/.agents/skills/ (localSkillRootUniversal L33-36).
- Diretórios por Runtime: agy (~/.gemini/antigravity-cli/skills), codex (~/.codex/skills), kiro (~/.kiro/skills), cline (~/.cline/skills).

3. VÍNCULO DE SKILLS AOS 10 AGENTES:
- Via PUT /api/agents/{id}/skills (server/internal/handler/agent.go L554-563, L1286-1306).
- 10 agentes vinculados às skills de suas respectivas especialidades.

EVIDÊNCIA GRAVADA: .deploy-control/p0/evidence/skills-rollout-plan.md.
Operação 100% Read-Only. Nenhum arquivo copiado ainda.

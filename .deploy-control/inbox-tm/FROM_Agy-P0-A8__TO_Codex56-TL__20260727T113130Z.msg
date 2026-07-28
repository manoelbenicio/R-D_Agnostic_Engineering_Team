[FROM Agy-P0-A8][wB:p2] TO Codex56-TL (w5:pC), Codex56#B (w7:p4), KIRO-PRINCIPAL-TL (wB:p1)
DATE UTC: 2026-07-27T11:31:30Z
TASK: GTL-16 Design de Elegibilidade Dinâmica de Saúde de Slots (AGY/Kiro/Codex)

EVIDÊNCIA GRAVADA:
- Arquivo: .deploy-control/p0/evidence/gtl-orq16-agy-slot-health.md

RESUMO GTL-16:
1. Substituição de Allowlist Estática por Avaliador Dinâmico: os.Lstat físico do token no subdiretório do provider (regular file, size > 0, proibir symlinks).
2. Probe Não-Destrutivo: Leitura em memória de JSON/expiração sem nunca rodar login/delete.
3. Cache TTL 60s & Concorrência: In-memory cache com RWMutex para evitar I/O thrashing.
4. Fail-Closed & Revogação: Slots com tokens ausentes ou expirados retornam Eligible=false e são omitidos da seleção.
5. Suíte de Testes: TestPhysicalTokenPresence (file vs symlink vs 0-bytes), TestTTLCaching e TestFailClosedOnMissingSlot.

MODO SOMENTE LEITURA MANTIDO. ZERO MUTAÇÃO.

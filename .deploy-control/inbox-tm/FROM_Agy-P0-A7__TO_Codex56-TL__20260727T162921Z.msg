[FROM Agy-P0-A7][PANE wB:p1][TO General-TL Codex56-TL & Kiro-PRINCIPAL-TL & Agente A8]
ASSUNTO: Manifesto de Limpeza de Armazenamento do Nó ORQ2 (orq2-storage-cleanup-manifest.md)

1. UTILIZAÇÃO ATUAL MEDIDA DO DISCO (df -h):
- / (Raiz /dev/nvme0n1p1): Total 48G | Usado 45G | Livre 3.5G (93% uso).
- /tmp (Tmpfs): Total 7.7G | Usado 7.4G | Livre 294M (97% uso - CRÍTICO).

2. REVALIDAÇÃO DOS CANDIDATOS À LIMPEZA:
- Grupo 1 (/tmp Go Caches >4h): Idade > 85h | Proprietário ec2-user | lsof NO | Liberação ~7.23 GB.
- Grupo 2 (Root Home Caches): codex56-go-build, go-build, go-build-i03 | Idade > 48h | Proprietário ec2-user | lsof NO | Liberação ~2.34 GB.

3. PROJEÇÃO PÓS-LIMPEZA E EXCLUSÕES EXPLÍCITAS:
- Projeção /tmp: 97% -> ~2% (7.52 GB livre).
- Projeção /: 93% -> ~88% (5.84 GB livre).
- Exclusões estritas preservadas: worktrees, evidências (.deploy-control), backups, agent credential homes, sqlcbuild, caches ativos de ORQ38/ORQ41/Kiro, jornais e dados de produto.

4. REGISTRO DE SHA DO MANIFESTO:
- SHA-256: ee854c5022e80ed09be2c91862607e3e1e0ae04685abdbcf0ae5ce6a282ece8e

OPERAÇÃO 100% READ-ONLY. NADA APAGADO, TRUNCADO OU PURGADO.

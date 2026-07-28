# ORQ-38 — integração local e fechamento canônico

**Ownership:** `agent/codex-b/orq38-get-contract` in
`/home/ec2-user/workspace/worktrees/gtl-orq38-get-contract`.

The accepted implementation is the ordered local pair:

1. `c536b609dc882a211ebc8362036cb29044811719` — baseline six-file contract.
2. `0ecc6f4e839e024d0cc81cb5c5c33ae0ec4dabd2` — strict redirect/status follow-up.

The second SHA is the exact reviewed value (`...ab2d`); no transposed SHA is
canonical. The worktree was clean after the second commit. This package is
local-only: no push, merge, PR, board mutation, or ORQ-26 file change.

The first commit passed the ephemeral-DB baseline differential and S01–S08
zero-skip gates. The second commit passed focused redirect/list-status tests,
`go vet`, formatting, and diff hygiene. The follow-up evidence is
`.deploy-control/p0/evidence/orq38-redirect-strict-followup.md`.

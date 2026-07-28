# WORKLOG DRAFT — ORQ-17

STATE: DRAFT_NOT_POSTED
BOARD_SYNC: PENDING_EXPLICIT_REGISTRAR_SYNC
IDEMPOTENCY_MARKER: WORKLOG-V1:ORQ-17:70678baeaaf9cb3d3cf957c15ff250dc9f2e23c054376eb2b2a66947fc718db5

- **Card Identifier:** ORQ-17
- **Card UUID:** 7d873133-16d5-42c6-8595-629d6fb16251
- **Title:** Publicar frontend 13100 com acesso LAN estável
- **Measured UTC / Agent:** 2026-07-27T17:38:08Z | Kiro sole execution owner
- **Verdict:** `ORQ17_PRODUCTION_OUTCOME_PASS`.
- **Evidence:** `.deploy-control/p0/evidence/orq17-production-outcome-20260727.md` (`70678baeaaf9cb3d3cf957c15ff250dc9f2e23c054376eb2b2a66947fc718db5`).
- **Owner adoption:** sole user UUID/member/workspace/all FK references preserved; user email adopted and one product-hashed password credential created transactionally from `asm-exec` NUL stdin. Post-state `1|1|1|1|0`, owner match `1|1|1`.
- **Secret safety:** fixed token `ORQ17_STAGE3B_ADOPT=PASS LOGIN=200 ME=200`; zero value APIs/direct SMA/output. Backup mode `0600`; helper `04cfc15f…`, binary `f6f1870b…`, runner `cf820f9f…`, `asm-exec d55eb38…`.
- **Runtime:** backend health/readiness `200`, frontend `200`, HTTPS root `200`, anonymous HTTPS API `401`.
- **Serve:** exactly seven canonical handlers verified; no Funnel enabled.
- **Kanban acceptance:** project `abe3c461-c921-4a51-b91a-b08529429145` created `201`; unassigned issue `bc633226-ab08-42fe-8aba-93eba2cdd616` created `201`; active queue `0→0`, active task empty, task runs `0`, authenticated GET-back `200`.
- **Excluded:** no comment, mention, agent assignment, paid task, full suites, browser matrix, CI, direct DB board write, or auth bypass.
- **Registrar:** authenticated application API is now proven, but no unrelated ORQ-17 status/comment post was made under the owner’s minimum-viable-production acceptance scope. This idempotent local draft remains queued for explicit registrar sync.

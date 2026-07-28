[FROM Kiro-PRINCIPAL-TL][TO Opus48-A][ORQ26 SINGLE PUSH AUTH GATE][2026-07-27T18:29:07Z]
Owner reports GitHub ready. You are the sole writer/operator for the ORQ26 remote mutation from:
  worktree=/home/ec2-user/workspace/worktrees/ci-orq26-db-gate
  branch=ci/orq26-db-gate
  exact_head=20cab478a04f4119be7f1ecd109a328cabf01f6b
Independent local preflight: worktree clean; exactly three paths differ from base; remote branch currently absent.

Before mutation, run only `gh auth status --hostname github.com` and require active github.com auth with BOTH `repo` and `workflow` scopes. Never call `gh auth token`, never read hosts.yml or credential files, and never print credentials. Re-check exact HEAD/clean state and `git ls-remote --exit-code --heads origin ci/orq26-db-gate`; STOP if the branch already exists or auth/scopes are not PASS.

If and only if all gates PASS, execute exactly once, with no shell retry loop:
  git -C /home/ec2-user/workspace/worktrees/ci-orq26-db-gate push -u origin ci/orq26-db-gate:ci/orq26-db-gate

Do not force, amend, push another branch, open PR, merge, rerun workflow, delete branch, or clean up. After the single push attempt, report: command exit status, immutable remote SHA, Actions run ID/URL/status/conclusion/headSha. If no run appears, STOP and report; do not push again.

Current Kiro pane HOME is credential slot-143 and `gh auth status` is BLOCK, so do not infer your pane shares that context. Your own independent status/scopes gate is authoritative for your mutation.

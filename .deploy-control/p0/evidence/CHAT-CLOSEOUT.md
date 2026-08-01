# CHAT-CLOSEOUT — chat orchestration routing tasks 2.2/2.3

Status: **EVIDENCED COMPLETE (candidate-only)**. Application/release remains **HOLD** pending the manager-created persistent backup envelope and separately authorized integration.

## Authority and frozen source

- Contract: `openspec/changes/chat-orchestration-standard/specs/chat-orchestration/spec.md`
- Production source: `multica-auth-work/server/internal/handler/chat.go`
  - SHA-256 `a2a71f47b4bbb2ee98671ca6232fa63cc4d76ec3ced438faa3220d1f61a7226c`
- Exact PostgreSQL test: `multica-auth-work/server/internal/handler/chat_test.go`
  - SHA-256 `23b5341bd6abaa596ec131b6affd07c59e6bec25fe1e7502dde1091377107014`
  - function `TestCreateChatSession_Routing`
- Runner: `/home/ec2-user/backups/multica-chat-routing-runner-20260801T131302Z/run-chat-routing.sh`
  - SHA-256 `1f8786f1de5d007b385e28460898ad65d43ca10a0a0732af6ef062d6823c0254`

The bounded repair routes omitted `agent_id` to the leader of exactly one active, workspace-scoped `Workspace Team`. Zero matches return 503 not configured; multiple matches return 503 ambiguous. An explicit `agent_id` remains the direct-to-agent escape hatch and continues through existing workspace, archived-agent, and private-access validation.

## Sealed isolated PostgreSQL execution

- Run root: `/home/ec2-user/backups/multica-chat-routing-pg-20260801T132324Z`
- Inner evidence manifest: `evidence.sha256`
  - SHA-256 `5c247789397d3c462a70b19996a96d20e71bf7e1d2866f366a7a1ee8cd0d0476`
  - verified `27/27` members
- Outer seal file: `evidence.sha256.sha256`
  - SHA-256 `d65e663250d3cf5b508af86a4886d485b043604c7ddace58d18a87706c241c7b`
  - verified `1/1`
- Summary: all `23` recorded gates have `rc=0`; `aggregate_rc=0`, `preseal_rc=0`.
- Isolation proof: `fsync_on|tcp_disabled|private_socket|unix_connection=true|true|true|true`.
- Provenance records `production_access=NONE`.

### Normal invocation

`go test -count=1 -v -run '^TestCreateChatSession_Routing$' ./internal/handler`

- one root RUN and PASS;
- four subtest RUN and PASS;
- zero FAIL or SKIP;
- explicit `agent_id`/literal `@agent` bypasses squad TL;
- exactly one default squad routes to its TL;
- missing default fails closed;
- duplicate defaults fail closed.

### Race invocation

`go test -race -count=1 -v -run '^TestCreateChatSession_Routing$' ./internal/handler`

The race invocation produced the same one root plus four subtest RUN/PASS markers with zero FAIL/SKIP.

### Persistence and cleanup

The frozen test asserts both HTTP responses and persisted `chat_session.agent_id`. The direct case compares the persisted target with the independently created direct agent and rejects the independently created TL. The default case compares the persisted target with the independently created sole squad leader. Both fail-closed cases assert HTTP 503 and zero persisted matching sessions.

Post-run cleanup evidence for both databases is:

`user|workspace|routing_sessions|default_squads=0|0|0|0`

PostgreSQL was stopped with immediate mode; `pg_ctl status` reports no server and the private socket directory contains no socket file. The preserved run root, data directory, and socket directory are mode `0700`; sealed evidence files are mode `0600`.

## P2 disposition

P2 post-execution PASS was received for the exact run, source pins, runner, nested seals, 23 zero-rc gates, normal/race markers, cleanup, isolation, and stop/socket evidence. P2 states task 2.2 is **EVIDENCED COMPLETE** and authorizes candidate-only closeout/refreeze.

## Check-in supersession

The historical target check-in `.deploy-control/p0/checkins/Opus48-A__CHAT-ORCH-2.x__20260722T120408Z.json` remains truthful as a BLOCKED record of the earlier unreachable-DB attempt. It is not rewritten or reinterpreted. Candidate check-in `.deploy-control/p0/checkins/Kiro__CHAT-ORCH-2.x__20260801T132600Z.json` supersedes it with the later sealed evidence and status DONE.

## Scope and non-claims

- Candidate-only writes under `/tmp/multica-contract-reconcile-wPp1`.
- No target write, Git operation, network request, production DB access, credential inspection/mutation, deploy, restart, ORQ1 action, or service mutation is claimed.
- No application/release approval is claimed. The manager must independently verify the frozen manifest and create the missing persistent backup envelope for all final target paths, including all 14 repository-root `docs/project` files.

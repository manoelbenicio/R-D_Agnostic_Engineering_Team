# Active independently accepted push-candidate matrix

## Embedded Golden Rule check-in/out

| Field | Value |
|---|---|
| agent / stream | Codex QA / `ACTIVE-ACCEPTED-PUSH-MATRIX` |
| START | `2026-07-18T21:12:09Z` |
| DONE | `2026-07-18T21:26:37Z` |
| status | `DONE — READ-ONLY MANIFEST` |
| sole written file | `.planning/agent-brain-v3/evidence/active-accepted-push-candidate-matrix.md` |
| prohibited writes | product, tests, OpenSpec/spec/tasks, STATE, AGENT_LEDGER, EVIDENCE_INDEX, Git index/refs |
| integration authority | Kiro TL only |

The initially created standalone `.deploy-control` lease was removed immediately
after the ownership correction. This artifact is the sole retained write and
contains both check-in and check-out provenance.

## Result

**YES — one nonempty atomic group is mechanically ready for Kiro TL
adjudication now:** chat-orchestration tasks 1.1 and 1.4, limited to three
changed source/test files plus their independently accepted evidence artifact.

“Ready” here means the current bytes match independent ACCEPT evidence, the
files contain no staged Packet-B or Prodex/persist path, targeted diff checks
are clean, and no active ownership collision was found. It does not authorize
commit/push and is not a new acceptance decision.

Every other independently accepted implementation found in the current dirty
tree is **HOLD**, because it intersects a central/Prodex hotspot, combines
accepted and unaccepted changes in one file, lacks a current evidence pin,
depends on a package whose evidence is only DONE/PARTIAL, or remains under an
explicit owner/conflict boundary.

## Admission rule

A changed path enters a candidate group only when all of the following hold:

1. its OpenSpec task is checked on current disk;
2. an independent reviewer records `ACCEPT`/`ACCEPTED` for that bounded task;
3. the current file SHA-256 equals the accepted evidence manifest when one
   exists;
4. the path is neither Prodex/persist nor one of the exact 11 staged Packet-B
   paths;
5. the file does not mix an open/rejected/unknown lane or a current exclusive
   hotspot; and
6. the proposed group has a coherent source/test/evidence boundary.

`DONE`, `PRODUCED`, `PARTIAL`, staged, modified, or checked without independent
acceptance is insufficient.

## Starting state and provenance

Read-only start checkpoint (`2026-07-18T21:12:09Z`):

| Input | SHA-256 / value |
|---|---|
| `HEAD` | `b6571299b00c8e388abefe7ef9dcbcf8ac715d7f` |
| `STATE.md` | `8edbf222c2ebed11834eb6575cb159f1e3e1b6d8fa037e93f64d13a486f2174e` |
| `AGENT_LEDGER.md` | `02f498b88c3155a3b45ac5f55f16d7f99fd80a94512138b67ece6b599998f7de` |
| `EVIDENCE_INDEX.md` | `d0117bd8cb1c4b3c9b8ba11decb6b7d1e3d444db0005e36249d90dd1e8b825d6` |
| porcelain worktree manifest | `8ce7139e71218a3d525f65f315edfed7ac2842325dafa58e42797494fb441276` |
| staged name/status manifest | `9512f50480949563eacfe729ac31a79d5889a931d14ecefd3911ce7995f26110` (11 paths) |

Current OpenSpec disk recount at inspection time:

| Change | Checked/total | Tasks SHA-256 |
|---|---:|---|
| `build-omniroute-agent-brain` | 51/85 | `e86a577b1958544e3fc23d7cb871c77a8e8609e97db3751338fc01ec9f6b7b99` |
| `chat-orchestration-standard` | 4/10 | `a7d19efa305fdfd8a9e4b1c8ca0a306f7fb4339b60ceed3d72987ec2841a00dc` |
| `native-runtimes-onboarding` | 9/17 | `78f78b383f26dbd6128a43b4fcfbaf1375911f1671b069a0baef462f0b9e7d3c` |
| `agent-credential-isolation` | 4/21 | `3bdbc1e1916629d87c14db544aee88050daf0e0aaa2c07c3c933798678e7a0e3` |

Persist/Prodex is excluded regardless of checkbox or evidence state. The 11
Packet-B paths are excluded regardless of staging state or later review.

## READY-1 — chat leader protocol, tasks 1.1 and 1.4

- OpenSpec: tasks 1.1 and 1.4 checked; 1.2/1.3 and smokes remain open.
- Evidence: `EV-CHAT-1.1`, `EV-CHAT-1.4`.
- Evidence artifact:
  `evidence/chat-orchestration-1.1-1.4.md`, SHA-256
  `c7064375c36e797e6989231b532441af6fbd47371420e6f22612f403a945d473`.
- Independent execution boundary: 24 deterministic AST assertions and daemon
  focused x20/race genuinely executed; handler package compiled only because
  its DB-gated `TestMain` exited before handler tests.
- Ownership/conflict: no current exclusive-owner row for these three changed
  files; zero overlap with staged Packet B, persist/Prodex, and unaccepted chat
  routing files (`agent.go`, `chat.go`, `workspace.go`, `chat_test.go`).
- Proposed atomic boundary: the three changed files below plus the immutable
  accepted evidence artifact. The shared OpenSpec task file remains for Kiro's
  separately controlled documentation integration.

| Current state | Current SHA-256 | Exact path |
|---|---|---|
| modified | `50406c891be39a9f645a2e1b957919c43ed879756a77a65c71a6afa11a3029fd` | `multica-auth-work/server/internal/daemon/prompt_test.go` |
| modified | `a2998f923852a455782f37d4416bbfb5a74750ea19b2f6d87dc5f56cc262e80a` | `multica-auth-work/server/internal/handler/squad_briefing.go` |
| modified | `3b12615543440f52773d0d1d7bed4277dd6c1b0fc835f7bfd2ee3f12cd823d9c` | `multica-auth-work/server/internal/handler/squad_briefing_test.go` |
| untracked evidence | `c7064375c36e797e6989231b532441af6fbd47371420e6f22612f403a945d473` | `.planning/agent-brain-v3/evidence/chat-orchestration-1.1-1.4.md` |

Canonical three-product-file manifest SHA-256 (SHA-256 lines over sorted
paths): `f7d7a2ef786a87d4a9aa6b351663f247886bdf99f2db3e9264fb406424629a32`.

Targeted `git diff --check` is clean. None of the three product/test paths is
staged. Kiro must re-hash immediately before any integration action.

## Independently accepted work on HOLD

These are accepted facts, not push-ready groups.

### HOLD-NATIVE-1.4 / EV-G4-RUNTIME-PROJ — model discovery and R24/R25

Evidence:

- `EV-NATIVE-OFFLINE`, artifact SHA-256
  `10351b86ef7f62a2f72ce6a7ccb584f7f9b3e61d4369d37e8449c8e8d7e66d93`;
- `EV-G4-RUNTIME-PROJ`, artifact SHA-256
  `64b591a74bb67316961d95cde00d0997c2d80dab17be1ac78a3bf39f9b13e2db`.

All current hashes match both accepted manifests where cited:

| State | SHA-256 | Path |
|---|---|---|
| M | `a6957e3e0b4a05050da6dc198049581d6402103d474185d0912f3360e8a7b313` | `multica-auth-work/server/pkg/agent/models.go` |
| M | `b1c62961e671f697b32844448c739e6059bccfe5e9c2bc1d6b1e52fb908dad5b` | `multica-auth-work/server/pkg/agent/models_test.go` |
| ?? | `75f1cc5d94bd240e955df5a61d34ca412ce9e980277eb66e0cf97495137c4211` | `multica-auth-work/server/pkg/agent/models_process_test.go` |
| ?? | `8ff9e9c2ae75d590d4ff75b6bf9d3f1813cde190418def5ee31ee4bb74fb7b7a` | `multica-auth-work/server/pkg/agent/models_windows_test.go` |
| M | `e92f2c48385d46f06f877398fcacb8e195c1f8ac21864dc9989b35de57c47ba9` | `multica-auth-work/server/pkg/agent/proc_other.go` |
| M | `7a1601f67bfbbddee65e739f3e4725d8d960ca2ede6e46e5428f2613be69e7cc` | `multica-auth-work/server/pkg/agent/proc_windows.go` |
| ?? | `679af9b9f721eb03a5ed74dd87da31d88a53f51e1481a22744980310942cc2c6` | `multica-auth-work/server/pkg/agent/proc_unsupported.go` |
| M | `406c2f478e3c7abe88f80a994603528d9f047e38888ff039ab291ebbf86003aa` | `multica-auth-work/server/pkg/agent/thinking.go` |
| M | `f2b0c3ab4277cf5a7e758c829c3336573361fe03a17c2b76a01f8da798417bb1` | `multica-auth-work/server/pkg/agent/thinking_test.go` |

Nine-file manifest SHA-256:
`1ea2a9a0bcf88cdb640ea1b08696b4a02bfc3e6316ba736d1997fd3f28acf365`.
Targeted diff check is clean and staged overlap is zero.

**Hold reason:** `FILE_OWNERSHIP.md:18` reserves `models.go` to Codex1 and
labels it a hotspot. The accepted runtime-projection artifact also covers two
untracked gateway projection files whose containing gateway package is only
DONE/PARTIAL, not independently accepted as a whole. Kiro must either obtain
the hotspot release and define an agent-only evidence boundary, or integrate a
larger dependency-complete group after independent package acceptance.

### HOLD-EXACTENV — accepted containment, incomplete dependency boundary

Evidence: `EV-G4-EXACTENV`, artifact SHA-256
`7c452e787e672a6e4d36db1955678f04723e1c32c94f577468a0c12871d32201`.
The artifact independently pins the runtimeenv sources/tests and
`brain_integration.go`/test. Current hashes match its 13-file manifest. The
additional changed backend contract is:

| State | SHA-256 | Path |
|---|---|---|
| M | `84cc33be31e6a4ebcceb93ccb6b408955f74d3fedeb487c6be25da0c4e816ba8` | `multica-auth-work/server/pkg/agent/agent.go` |

**Hold reason:** active-path completion crosses `daemon.go`, `config.go`,
`brain_integration.go`, execenv and adapter files. The central files are
exclusive Codex1 hotspots and `daemon.go`/`config.go` contain frozen excluded
Prodex work. The new `brain/**` dependency is only G2 DONE evidence, not an
independent ACCEPT group. No safe partial commit boundary is asserted.

### HOLD-AUTH-1.7 — accepted backend, shared environment/topology files

Evidence: `EV-AUTH-1.7`, artifact SHA-256
`2a5f7368a63202f5decb27bd562589e1cc9ad406499b29a14a471f1c1425c095`.
All 17 accepted manifest hashes still match current disk:

| SHA-256 | Path |
|---|---|
| `3f9b95b76f2683bb0a91a9d6a7bc6db939dfb3af3dd43d10977f27a756db0512` | `multica-auth-work/.env.example` |
| `6059f7e20ece7485016e2546ef977fadde925569e4b5ed1d862d0f3cace27de9` | `multica-auth-work/server/cmd/multica/cmd_user.go` |
| `80aa70ae912912b6233880dea2cbcc669ab9414ab8112bcb81a1441ee6dc8a3f` | `multica-auth-work/server/cmd/multica/cmd_user_password_test.go` |
| `5aa5cc4268474e8b79ada549ce908df04b071d2fda4aed90a5460c577d423bb6` | `multica-auth-work/server/cmd/server/main.go` |
| `5c6492bfd64347d48bb13749ac3f1b38ef84b4275fd4d20b1fc44e0f1cdb5a74` | `multica-auth-work/server/cmd/server/router.go` |
| `bbb5fa1ca1bf24f94756906512a5717e7a2783113be0c91ba941c700cf8822fd` | `multica-auth-work/server/internal/auth/jwt.go` |
| `9df59d84abfbb5e44a8f1f00571fdc9b47119a15bcf6ce532a01f400bc00fdf5` | `multica-auth-work/server/internal/auth/jwt_configuration_test.go` |
| `e800814c59e5ea55295d6b8c2209bf57fe776595e5d1e541ff511f4f892db94b` | `multica-auth-work/server/internal/auth/recent_auth.go` |
| `ecbc885334affbcf20cadc2c7b73a80d6f77fd570f3ea14676e51f2c942fdf90` | `multica-auth-work/server/internal/auth/recent_auth_test.go` |
| `d69877a9dcabec59726717628c103b2b1ab0c9a4c7673d14a80c8360b7a259e0` | `multica-auth-work/server/internal/handler/auth.go` |
| `3c8f75b5ac2e9a4e2ca83b228285c3ff21d2ced484aab3b29f71cb7b67d70857` | `multica-auth-work/server/internal/handler/auth_provider.go` |
| `4871a86311316e4da83c6fb56e97249da1bfff5714f1c1b4e101380510055e64` | `multica-auth-work/server/internal/handler/passwordtest/provision_test.go` |
| `10d75ff5a2d7db032eab78a307a86ba0157c2725ee72965494c2bc1f571eae6a` | `multica-auth-work/server/internal/middleware/auth.go` |
| `e76cd669222074125561e4e57eac2c737418d7c15b1c6deffa4b2215f2c5b124` | `multica-auth-work/server/internal/middleware/auth_test.go` |
| `14c3ee447ef5f397100100fb086157b538ba36e42a6c17bc14123d9f55944084` | `multica-auth-work/server/internal/middleware/ratelimit.go` |
| `43418c0ec0652bbc7e60102d196f3de155cb6734d38643bc14123d9f55944084` | `multica-auth-work/server/internal/middleware/ratelimit_test.go` |
| `97bd6dee369edc88e602f83dd4c6d70c9f83d1b4594f1dcc29d2ef111e52c298` | `multica-auth-work/server/internal/rotation/rotation_e2e_test.go` |

**Hold reason:** `.env.example` is a shared modified environment template and
this assignment may neither inspect its values nor admit any possible
Prodex/persist hunk. The accepted test-topology evidence also references
multiple modified `cmd/server/*_test.go` files outside the 17-file hash
manifest. `rotation_e2e_test.go` overlaps credential-isolation work. Kiro must
produce a hunk/file-level reconciliation before an auth atomic group exists.

### HOLD-CREDISO — accepted tasks 4.1, 4.2 and 5.2

Evidence:

- `EV-CREDISO-4.1` ACCEPT, but only an AGENT_LEDGER summary is indexed; no
  dedicated artifact/file-hash pin exists.
- `EV-CREDISO-4.2`, artifact SHA-256
  `d5a8022873bd5ae359e7d9cb1fda09563d909e3a753fff8f69c8e50ab60f804f`.
- `EV-CREDISO-5.2`, artifact SHA-256
  `97aeac65709a9a59483819dca8633590d2a0fc5a368309a46a19902874ccbe0a`.

Current changed files pinned by task 5.2 still match:

| SHA-256 | Path |
|---|---|
| `168dc34f17650e3d4f07d324a5272a9dc5839f7b2ed28b5b1e643ef823fa7308` | `internal/daemon/runtime_isolation_test.go` |
| `8099867594f18f6d5e5c47fa88581f06ac771e456f203fc0161d09c6584fc0e6` | `internal/daemon/execenv/execenv.go` |
| `aa3f8bbd82ffb3fe70ed72da037eaae3028af4b558a7b1d0c207ccaeb4cb3fe0` | `internal/daemon/execenv/codex_home.go` |
| `31d759a40ab35780ef875779574fc7d82ace4c5801ae565cc96845a0fe9b6f4a` | `internal/daemon/execenv/codex_home_account_test.go` |

Task 4.2 current accepted-manifest paths include `service.go`
(`f20951…cf0`), `detector_discovery.go` (`bc61a4…45b55`), its test
(`4e8092…5a4f`), `discovery_reassignment.go` (`c655c9…b2832`) and its test
(`d87aa8…6566b2`), plus producer/monitor/daemon/metrics/schema paths.
`credential_session_monitor.go` has drifted to
`936b3e40f19c699078e994740c9ae63fdce5c172516b678d289f05c3cc38d1e2`,
not the task-4.2 artifact's `a77d2f70…0478e` pin.

**Hold reason:** task 4.1 lacks file-level provenance; task 4.2 overlaps open
4.3/4.4 and has a drifted monitor blob; task 5.2 requires excluded
`daemon.go`/L2 integration and shared execenv hotspots. The OpenSpec task file
is Kiro-owned and also contains open 4.3/4.4/5.3/5.4 state.

### HOLD-R26 — accepted review without isolated artifact pin

- Evidence ID: `REVIEW-R26-ACCEPT` in AGENT_LEDGER; no new EV artifact ID.
- Current files:
  - `internal/handler/agent.go` —
    `1339bff805d241451e9271c3684c13c280164c29f9ec6869c718415f584bdbb7`;
  - `internal/handler/agent_route_preservation_test.go` —
    `a9a3efee716ac414676530617ef387881ab4d52639c10ff7e1d3733a4ac6b07c`.

**Hold reason:** `agent.go` also belongs to unaccepted chat task 1.3 routing,
and acceptance is recorded only in the mutable shared ledger without a
dedicated immutable artifact hash. DB-backed switch/concurrency assertions
compiled but did not execute. A file-level hunk split or consolidated evidence
pin is required.

### HOLD-RSS — accepted correction inside an unaccepted package set

EV-G4-08 pins the independently accepted RSS collector set at canonical
SHA-256 `53f25cf1f01ed7cd66cd097b9ec0f71713188c6a9a1641f05c346881531c94c8`:

| SHA-256 | Path |
|---|---|
| `d84fc659dd8bd5873760ed9a5965adb6007d9a454af7cdd30b3441fe607a293e` | `internal/daemon/observability/realtime_process.go` |
| `f143cda7e712883481372f73666424e930036f261ab1ba8b5b59f17d6a66b131` | `internal/daemon/observability/realtime_process_linux.go` |
| `4b53206c7a819323df4730a247c0b187e40cb07d6b4281783913e3ca84a5a81f` | `internal/daemon/observability/realtime_process_linux_test.go` |

**Hold reason:** all are untracked and import/compile within the larger
untracked observability package, whose G4 results are mostly PARTIAL and whose
task 9.1 remains STOPPED. The three-file correction is not a standalone
dependency-complete commit.

### HOLD-MCP/G3 adapter corrections — mixed accepted concerns

- `pkg/agent/codex.go` —
  `597fa3acaa8c65cef0676b5b644f14adba6f35e8ac2c4f7cb0f2d042aee212c1`;
- `pkg/agent/codex_test.go` —
  `bb58a12bdabcb7abefa73c2e915c9e4fa3d9e3cd32ecf3a1a858a412a3caa327`;
- `pkg/agent/claude.go` —
  `4ee1e98e0560c1ce0ac3f68999ea3c5807d746632b87d1cf71d949f623408cdc`;
- `pkg/agent/claude_test.go` —
  `018afe986614dacf8ba348a1d2b12e87f9f819d358c453f7cbbb561714b7dc1b`.

Independent rows `REVIEW-MCP-GUARD-ACCEPT` and `REVIEW-G3-02` accept the MCP
guard and argv-log corrections. Current Codex files also contain ExactEnv
session-root changes. There is no dedicated MCP artifact hash, and the G3
review artifact has source anchors but no four-file SHA manifest. These mixed
accepted concerns need a consolidated exact evidence artifact before commit.

## Completed but not independently ACCEPTED — excluded

The following are deliberately not candidates despite checked tasks and code
on disk:

- `internal/daemon/brain/**` (G2A / EV-G2A-01..05): DONE/complete worker
  evidence, no independent ACCEPT artifact for the package set.
- `internal/daemon/gateway/**` except the separately accepted projection
  correction: G2B DONE and G4 package evidence PARTIAL; live/provider gates
  remain open.
- `internal/daemon/runtimeenv/**` as the broad G2C package: G2C DONE, while
  only the narrower ExactEnv point-in-time contract is independently ACCEPTED.
- `internal/daemon/deploy/**` and most `observability/**`: G2D DONE; G4
  operational/capacity evidence PARTIAL and task 9.1 STOPPED.
- Agent Brain central G3 files: G3 review ACCEPTED specific security
  corrections, but central files mix excluded Prodex/persist and other lanes.
- Task 8.8 evidence/planning set: documentary closure is checked, but shared
  planning files remain Kiro-owned and concurrently changing; product
  observability dependencies are not independently accepted as one package.

## Explicit exclusion matrix

| Excluded current paths | Reason |
|---|---|
| all `*prodex*`, `*Prodex*`, persist change/spec/evidence paths; `daemon.go`, `config.go`, `health.go`, `l2_runtime.go`, `cmd_daemon.go` where mixed | Explicit assignment exclusion and current PROGRAM HOLD; central files contain inseparable Prodex/persist hunks. |
| exact 11 staged files under `packages/core/{api,runtimes,types}` and `packages/views/agents/components/**` | Explicit Packet-B exclusion. Staged is not accepted; current staged manifest pin is recorded above. |
| `apps/mobile/**`, `apps/web/**`, landing/i18n/web artifacts | Native tasks 1.5/1.6 remain open/reopened; mobile migration is a follow-up without independent acceptance. |
| `internal/handler/{agent,chat,workspace}.go`, `chat_test.go` except R26 hold | Chat 1.2/1.3 and smokes are unaccepted; `chat_test.go` also has whitespace defects. |
| credential-session monitor/producer/reassignment additions beyond accepted pinned subsets | Tasks 4.3/4.4/5.3/5.4 remain open or have evidence-integrity/drift gaps. |
| `pkg/redact/**`, logger/email/log-audit changes | Task 5.4 remains unchecked; accepted bounded reviews do not complete the OpenSpec task. |
| CLI password bootstrap files beyond EV-AUTH-1.7 exact manifest | Follow-up residual, not a separately checked/accepted task. |
| `.planning/**`, `.deploy-control/**`, `.claude/**`, `.codex/**` not explicitly named in READY-1 | Mixed accepted/pending/active coordination; Kiro owns shared state and documentary integration. |
| OpenSpec task/spec/proposal/design modifications | Shared Kiro ownership; several files combine checked, open and reopened states. This assignment performs no spec/task integration. |
| `nul`, `NUL`, `files.txt`, `opencode.json*` | Hazard/scratch/tool-local unknowns; no acceptance evidence. |
| any remaining modified/untracked server/test path not explicitly listed above | UNKNOWN or mixed provenance; change presence is not acceptance. |

## Read-only commands and safeguards

Command classes used:

```text
git rev-parse HEAD
git status --short/--porcelain=v1 --untracked-files=all
git diff --cached --name-status; git diff --check -- <candidate paths>
git diff -- <non-environment source paths>
rg / sed / nl over OpenSpec and planning/evidence Markdown
sha256sum over repository paths and evidence artifacts
```

No environment-file value was displayed or parsed. `.env.example` was hashed
only as an opaque file because EV-AUTH-1.7 pins it; it remains excluded. No
credential/auth/home path or value was inspected. No test, DB, network,
provider, daemon, service, checkout, reset, add, restore, commit, push, ref,
index, or deletion operation was performed.

## Final re-read checkpoint

Final read-only checkpoint (`2026-07-18T21:22:23Z` through
`2026-07-18T21:26:37Z`):

| Input | Final SHA-256 / value | Drift from start |
|---|---|---|
| `HEAD` | `b6571299b00c8e388abefe7ef9dcbcf8ac715d7f` | unchanged |
| `STATE.md` | `d2f3862cd47b66f27c8c794b51ccc6b720e154a2152b71c00641c53bc7faae32` | changed from `8edbf222c2ebed11834eb6575cb159f1e3e1b6d8fa037e93f64d13a486f2174e` by another owner, including a second change during final verification; not edited here |
| `AGENT_LEDGER.md` | `25bdcc12a93a4b972633d024ba54bd28293c26370af46b96389546202d2d8bce` | changed from `02f498b88c3155a3b45ac5f55f16d7f99fd80a94512138b67ece6b599998f7de` by another owner, including further change during final verification; not edited here |
| `EVIDENCE_INDEX.md` | `16410aa63e26275a4b09a9083a5a906b954ead5b199bb5d6e372b84aaf3be75d` | changed from `d0117bd8cb1c4b3c9b8ba11decb6b7d1e3d444db0005e36249d90dd1e8b825d6` by another owner at the handoff boundary; not edited here |
| porcelain worktree manifest | `a9d66160ac53d3536198bcc4dba317c83ab931b276b5d8ad6ef68405172a313d` | changed from `8ce7139e71218a3d525f65f315edfed7ac2842325dafa58e42797494fb441276`, including this new artifact and concurrent work |
| staged name/status manifest | `9512f50480949563eacfe729ac31a79d5889a931d14ecefd3911ce7995f26110` (11 paths) | unchanged |

The four task recounts and hashes remain exactly as recorded in the starting
table: 51/85, 4/10, 9/17 and 4/21. The READY-1 file/evidence hashes and
three-product-file manifest remain unchanged. Its staged overlap remains
empty, and its targeted `git diff --check` remains clean. Therefore the final
answer remains **YES: READY-1 is the sole nonempty mechanically ready group**,
subject to Kiro TL's re-hash, adjudication and integration authority.

Repository-wide `git diff --check` is not clean because the excluded,
unaccepted `internal/handler/chat_test.go` has trailing whitespace at lines
474 and 477 and a new blank line at EOF (reported at line 512). The READY-1
targeted check and this artifact's isolated check have no whitespace errors.

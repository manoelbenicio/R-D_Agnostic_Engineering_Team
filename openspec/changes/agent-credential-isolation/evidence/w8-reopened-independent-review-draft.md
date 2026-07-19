# W8 independent-review draft — credential isolation 4.3/4.4/5.4

Status date: 2026-07-19. Reviewer: W8 / Codex (`/root`), evidence-only and
read-only on product code. Historical producer/reviewer attribution is mixed
and sometimes includes `Codex/root`; therefore this document is **not** offered
as acceptance-grade producer≠reviewer proof. It preserves that provenance gap
for a genuinely distinct reviewer and Kiro TL adjudication.

## Classification

| Task | Implemented | Reviewed | Verified | Accepted |
|---|---|---|---|---|
| 4.3 automatic reassignment | bounded producer/service paths exist | yes | unit-level behavior exists; no non-test production emitter/caller, cross-process atomicity and logout ordering unresolved | **NO — open** |
| 4.4 record/alert switch | success/no-account/failure alert slice exists | yes | focused prior reproduction exists; production reachability is blocked by 4.3 | **NO — open** |
| 5.4 no secrets in logs | redact core and several bounded sinks exist | yes | bounded slices pass; universal structural coverage is disproved | **NO — open** |

### 4.3 disposition

The discovery producer has no proven non-test construction/call site. Assignment
and rotation recording are not shown as one durable transaction; destructive
logout ordering and cross-process concurrency policy remain owner decisions.
Tests of callable components do not establish production reachability.

### 4.4 disposition

The eight-file alert/record manifest is stable and prior focused tests report a
technical pass. That does not close 4.4: an alert cannot represent a production
switch until the 4.3 emitter reaches the production path. Frontend delivery is
also an unresolved D1 interpretation, not something this review may infer.

### 5.4 disposition

The redact core is stable, but the central hook is not universal: the CLI has a
default-logger path outside `logger.Init`. Thirteen adapters still pass raw argv
shape to the logging layer and depend on pattern redaction rather than structural
projection. Error/text sinks and other attributes remain non-exhaustively proven.
Bounded accepted slices must not be promoted to whole-codebase acceptance.

## Exact 4.3/4.4 source manifest

```text
4e240c09af1653fd2ecdbcd0763f8c05d3eb3f57ee8b9c339045edfa32a9ce6c  multica-auth-work/server/internal/daemon/credential_session_discovery_producer.go
818e69d4fe04057503b776716ae0704a6b27b7fc7b2b875e8f507b9a7b99491a  multica-auth-work/server/internal/daemon/credential_session_discovery_producer_test.go
936b3e40f19c699078e994740c9ae63fdce5c172516b678d289f05c3cc38d1e2  multica-auth-work/server/internal/daemon/credential_session_monitor.go
5fd7005b4209b0c8d5c42ad9b240ad039d7568ad1f895bd6b9463b8e237613e2  multica-auth-work/server/internal/daemon/credential_session_monitor_test.go
8e369510e814ff2f5743bf60dc76ee3074110a98da3daa172301aae1fda12bea  multica-auth-work/server/internal/daemon/credential_session_alert_test.go
71b89980c9b8cb7c03a81a502da5b9f4bdab4c0c522c117d2147b670e050f730  multica-auth-work/server/internal/daemon/wakeup.go
c655c9c4f94716d2c93109b1cc4d3c14e62d6fe8dea615dddec3ec93769b2832  multica-auth-work/server/internal/rotation/discovery_reassignment.go
d87aa80717e33a3f3768ebb7d698dcd9ace3bd934550317ec64c57f56a6566b2  multica-auth-work/server/internal/rotation/discovery_reassignment_test.go
f20951a36765d2b6b8576232871f1991697de56d8577377b0226ed5e30130cf0  multica-auth-work/server/internal/rotation/service.go
989ae633e1865bc0f1f5a9f7eed5d2541100dbee94960c9e2c15f84346aa8ba1  multica-auth-work/server/internal/rotation/service_test.go
e47757784ba22d9c4d749c51028247523a6d8cfd3db002343a503d5fba67edf8  multica-auth-work/server/internal/rotation/store_pg.go
9c60b84f1309a53b1d759ca8356161eafba5e253bf65d845f9566879f0e73804  multica-auth-work/server/internal/rotation/store_pg_test.go
```

## Exact 5.4 boundary manifest

```text
f409ba8a9f3e63618d59c5a8692296f8f7c019c9e558576b8786a058fbf68a5c  multica-auth-work/server/pkg/redact/redact.go
5a37941a1c7f1bd7263368a6479104c81300f7981e9f47bfb6b0cd17a602fec9  multica-auth-work/server/pkg/redact/redact_test.go
f5f705c1d1433db10d84496ff6dcaf42b62dcad5a415239b9cb38cfcefd38010  multica-auth-work/server/internal/logger/logger.go
e36fc0f50745cd0b1014acaa16d7c49d5cc03b2f40f77d297703321f14a6e732  multica-auth-work/server/cmd/multica/main.go
dc8601d386ae1f0366258ff003f34c127e5e6129c6dee92c1e55e490537298aa  multica-auth-work/server/cmd/multica/cmd_id_resolver.go
96ee0c982cab104cd5690eba71b59536f4bef2306c184bf52471198dd36887a1  multica-auth-work/server/pkg/agent/antigravity.go
9497ebfccaeb143cef0e08b2ae4f59f5192a40d118d2f68ff208f9ae1322ede0  multica-auth-work/server/pkg/agent/cline.go
ecb85d968c1b60283e09174d3bc37a7dfa80126193105c2e97ce8382109bbcf9  multica-auth-work/server/pkg/agent/codebuddy.go
80111abb1aa00045d7d31a777c8d233a57b41f7cdfaafe3b07fd49f21391d07b  multica-auth-work/server/pkg/agent/copilot.go
f38115ae48ccc5bcfac0a028ad375dd99fb7394d0f7029791d0757be922e192e  multica-auth-work/server/pkg/agent/cursor.go
260ffcf6d8066ad3e9f15c086381f5e062910043dba76d6c2de7421d79567555  multica-auth-work/server/pkg/agent/gemini.go
3752b611d5f9fd1961079fa25a78187057ba9291cf7daa817838202a4e1ba3d9  multica-auth-work/server/pkg/agent/hermes.go
53271c50affe13088d98a9f9b3f3db711b908a8b6a0e4fcae7e2031eec10cd2e  multica-auth-work/server/pkg/agent/kimi.go
0b4d3bd7f274623fa4d45db34639a247e969074cf04ac5dce5de2b7322657410  multica-auth-work/server/pkg/agent/kiro.go
ebd450c2c3911db39df078bc362a749d0bf0bd68d1250e85299e362cfdc4291a  multica-auth-work/server/pkg/agent/openclaw.go
4db9a414e13743c8cc672b36d30f6ba2f649530f75daeb13e42a9c27db448d4c  multica-auth-work/server/pkg/agent/opencode.go
46f1ed17f664f2c316944f42e0a134ca86a460ba8bcd777e81aac6d27d1994da  multica-auth-work/server/pkg/agent/pi.go
7bfb0d23039911c2f206aab34b4cb1eb3885929dd471924a8eac7682b8042618  multica-auth-work/server/pkg/agent/qoder.go
3f9dc4fbdb28eecfcc4886f2a518a07943f8a0493cf7aa3525b745992c6d2f54  multica-auth-work/server/pkg/agent/claude.go
```

## Exact evidence-input manifest

```text
6184aa3703b390fdba16c1ac1c4cfbabfcbd3b7ca18bb30e0ed6b3ca436c4848  .planning/agent-brain-v3/evidence/credential-isolation-auto-reassignment.md
e740f715c05d779e100f101db5560d5f256e4e3fa8348f8fe05e92d9c58faf08  .planning/agent-brain-v3/evidence/credential-isolation-4.3-production-integration-gap-independent-review.md
cdb70e85fab9131c3aff59e52d037a57a8ee080265a51440bde071c526d08d95  .planning/agent-brain-v3/evidence/credential-isolation-4.4-fresh-review.md
2f4094f632d8a928adef45f7fd18011852583d963d1cc2cbfb3dff7a6d8bd2b5  .planning/agent-brain-v3/evidence/credential-isolation-4.4-record-failure-alert-codex-independent-review.md
45a64b19427866cab2cc3b3178aa4ce7f1c5da629bbeccbdbd2c145151581490  .planning/agent-brain-v3/evidence/credential-isolation-5.4-codebase-wide-residual-closure-matrix-codex-independent-review.md
```

Required next evidence is a production-reachability implementation/reproduction
for 4.3, owner-selected atomicity/concurrency/logout policy, then a 4.4 review on
that reachable path. Task 5.4 needs structural projection at every argv/content
sink plus an exhaustive default-logger/error/text audit. A reviewer whose identity
is distinct from all Codex-attributed producers must reproduce before adjudication.

No product, task checkbox, GSD, secret, auth file, environment value, service,
DB, or network state was changed or accessed by this draft.

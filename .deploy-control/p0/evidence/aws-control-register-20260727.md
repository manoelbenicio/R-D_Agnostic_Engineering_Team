# AWS technical control register — ORQ program

- **Owner:** Kiro-Opus5, permanent AWS Technical Lead.
- **Integration authority:** Codex56-TL / General-TL.
- **Snapshot UTC:** 2026-07-27T16:48:16Z.
- **Region/account:** `sa-east-1` / `809809509961`.
- **Board state:** `BOARD_SYNC_PENDING`, `DRAFT_NOT_POSTED`; local evidence remains authoritative until authenticated application access. No direct DB write, auth bypass, or board mutation.
- **Secret rule:** never `GetSecretValue`/`BatchGetSecretValue`; never expose secret values.

## Control register

| Control | Current state | Evidence / exact control | Owner / next action | Planning ETA |
|---|---|---|---|---|
| ORQ2 root EBS capacity | **COMPLETE / VERIFIED** | `vol-04089fc818225fbcc` is 60 GiB gp3, 3000 IOPS, 125 MiB/s; partition 1 and XFS root grown online; root 73%, 17.73 GB available. Evidence SHA `c22a0b2cca894043aa6edfa1c1e6f25e9db9961eb5575671d9495ec08e1a37d5`. | General-TL reviews. No shrink rollback; future size changes require fresh Owner authorization. | DONE |
| Excluded EBS target | **UNCHANGED** | `vol-024324f534a37e072` / `i-0d9d441dd364039f9` remained 24 GiB and was not targeted. | Preserve exclusion. | Continuous |
| ORQ2 cache lifecycle | **ACTIVE** | Daily persistent timer; next measured run `2026-07-28T05:15:51Z`; one real run exit 0. Evidence SHA `b961ab444025c71115efcb50f5dad7ad8be1a3592411e0a151114e51775c75ce`. | AWS TL owns operational review; rollback is disable/unlink exact units. | DONE; review after first scheduled run |
| Root disk telemetry | **ACTIVE METRIC SOURCE** | CloudWatch Agent active, 60-second namespace `Orquestradores`, root `/` measurements `used_percent` and `free`, dimension `InstanceId`. | Preserve agent; validate metric freshness when IAM read is available. | 15 min after IAM read access |
| Root disk warning alarm | **PENDING AUTH/IAM** | Proposed `ORQ2-RootDisk-Warning`: `disk_used_percent >= 80%` for 15 minutes (15×60s), missing data treated as breaching or paired with agent-health alarm. | Owner authorizes alarm mutation; AWS TL creates and proves action target. `cloudwatch:DescribeAlarms` currently denied, so existing alarm state is unverified. | 30 min after authorization/IAM |
| Root disk critical alarm | **PENDING AUTH/IAM** | Proposed `ORQ2-RootDisk-Critical`: `disk_used_percent >= 90%` for 5 minutes (5×60s), explicit notification/escalation action. | Same as warning; no alarm mutation authorized in this task. | 30 min after authorization/IAM |
| `/tmp` capacity alarm | **GAP / DESIGN PENDING** | Current agent collects only `/`; `/tmp` is tmpfs and not in the metric resource list. Lifecycle logs include before/after `/tmp` df. | Separately authorize agent config addition and alarm if required; do not conflate root/XFS and tmpfs. | 30 min after config authorization |
| EC2 health/capacity | **HOST HEALTHY / AWS STATUS PARTIAL** | NVMe capacity 60 GiB, XFS rw, no bounded-window I/O/XFS error; root 73%. `DescribeInstances` is denied, so EC2 status checks are not asserted. Two pre-existing coredump failed units are unrelated and remain for General-TL triage. | Add least-privilege `ec2:DescribeInstanceStatus`/`DescribeInstances` read if Owner approves. | 15 min after IAM read access |
| EBS mutation IAM | **DENIED FOR INSTANCE ROLE** | `ec2:ModifyVolume` and `ec2:DescribeVolumesModifications` denied; Owner console performed the approved size change. `DescribeVolumes` succeeds. | Do not broaden permanently without need; if delegated again, scope `ModifyVolume` to the exact volume and retain Owner authorization gate. | On demand |
| ORQ-17 bootstrap secret | **BLOCKED / STRICT LANE** | Require full Secrets Manager ARN, explicit JSON keys, version stage freeze, metadata/owner attestations, argv-safe `asm-exec` child transport, pinned helper/source/build, exact ORQ1 host/stack/DB/queue gates, C1–C8 independent PASS, and real login proof. | Owner supplies metadata/authorization only; authorized identity resolves in child process. No value enters agent context. KMS decrypt, if CMK-backed, must be scoped through the approved secret path. | 60 min after C1–C8 PASS and Owner inputs |
| ORQ-42 JWT rotation | **BLOCK V5 / V6 REQUIRED** | Independent evidence SHA `4cbc4196b3509b5f39adc9c99c05e0671f7925e6fcd38097ed5fd07182e3fabf`; B1–B9, Q-A..Q-F, corrected A1..A4 remain. Metadata-only AWS reads must be separately authorized. | V6 author + Owner + independent reviewer. No secret/runtime mutation. | One review session after V6 and Owner answers |
| ORQ-44 inference-key lifecycle | **V5 NOT APPROVED / NOT EXECUTABLE** | Proposal SHA `658af263ffe8a53ec1dfde0870d15912c79f3ba0ffa0d86aa363fbf588b81972`; Q1–Q6 owner/provider decisions remain hard blockers. | OmniRoute operator, AWS owner, product owner, independent reviewer. No provider/runtime mutation. | One review session after Q1–Q6 answers |
| KMS/IAM coordination | **LEAST-PRIVILEGE PENDING** | No KMS mutation. Secret workflows may need resource-scoped Secrets Manager metadata permissions and CMK `kms:Decrypt` constrained by key/resource/encryption context; plaintext value APIs remain prohibited. | AWS TL drafts policy only after exact ARN/key IDs and Owner authorization. | 30 min after exact identifiers/authorization |

## Rollback and cost controls

- EBS cannot shrink. A capacity rollback requires a separately authorized snapshot/new-volume migration and downtime; no such operation is approved.
- Added storage cost is 12 GiB-month of gp3; IOPS/throughput remained at baseline, avoiding performance add-on change. Cost Explorer verification awaits read IAM.
- Lifecycle rollback is fully bounded to disabling the timer and unlinking three exact files; reclaimed cache bytes are reconstructible and not restored.
- CloudWatch alarm/config, Secrets Manager, KMS, IAM, and board mutations are all pending explicit bounded authorization.

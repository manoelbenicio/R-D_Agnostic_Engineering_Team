# ORQ2 root EBS 48→60 GiB expansion — control and host verification

- **Region:** `sa-east-1`
- **Account:** `809809509961`
- **Caller observed:** `arn:aws:sts::809809509961:assumed-role/cw-agent-orquestradores/i-0af937456e125143d`
- **Target instance:** `i-0af937456e125143d`
- **Target volume:** `vol-04089fc818225fbcc`
- **Explicit exclusion:** `i-0d9d441dd364039f9` / `vol-024324f534a37e072` (24 GiB); untouched.
- **Board sync:** `PENDING_AUTHENTICATED_REGISTRAR`; no direct DB write or auth bypass.

## Pre-state and authorization

AWS `DescribeVolumes` proved the target was 48 GiB, `gp3`, 3000 IOPS, 125 MiB/s, `in-use`, attached to ORQ2 as `/dev/xvda`. Local NVMe serial was `vol04089fc818225fbcc`; root was XFS on partition 1.

The technical-lead role attempted the exact size-only operation once:

```text
ModifyVolume(volume=vol-04089fc818225fbcc, size=60, region=sa-east-1)
```

It returned `UnauthorizedOperation` because the role lacked `ec2:ModifyVolume`. No HTTP request ID was exposed by the CLI wrapper; the encoded authorization blob is intentionally omitted. No modification was initiated by this agent. `DescribeVolumesModifications` was also denied.

The Owner subsequently reported completing the AWS size increase and ordered no duplicate call. No further `ModifyVolume` request was made. The Owner-console request ID was not supplied and is therefore `NOT_AVAILABLE` rather than inferred.

## Verified AWS result

Read-only AWS verification after the Owner operation:

| Property | Result |
|---|---|
| Volume | `vol-04089fc818225fbcc` |
| Size | `60 GiB` |
| Type | `gp3` unchanged |
| IOPS | `3000` unchanged |
| Throughput | `125 MiB/s` unchanged |
| State | `in-use` |
| Attachment | `i-0af937456e125143d`, `/dev/xvda`, `attached` |

The excluded volume remained 24 GiB gp3/3000/125 and attached only to the excluded instance in the pre/post safety read.

## Verified host result

General-TL performed the partition/XFS growth. This agent issued no grow command and only polled/verified. Observed transition:

```text
16:45:50Z disk=64424509440 partition=51527007744 filesystem=51459895296
16:45:55Z disk=64424509440 partition=64411909632 filesystem=64344797184
```

Final verification at `2026-07-27T16:48:16Z`:

- `/dev/nvme0n1`: `64,424,509,440` bytes (60 GiB), serial `vol04089fc818225fbcc`.
- `/dev/nvme0n1p1`: `64,411,909,632` bytes.
- Mounted XFS `/`: `64,344,797,184` bytes; 16 allocation groups; read/write.
- Utilization: `46,614,515,712` used, `17,730,281,472` available, `73%`.
- No reboot, stop, detach, volume-type, IOPS, throughput, tag, security, provider, credential, or board change by this agent.
- Kernel recorded the expected NVMe capacity rescan; no I/O/XFS error was observed in the bounded post-window scan.

Authoritative procedure: AWS EBS “Extend the file system after resizing an Amazon EBS volume” (`https://docs.aws.amazon.com/ebs/latest/userguide/recognize-expanded-volume-linux.html`).

## Rollback and cost controls

- **EBS cannot shrink in place.** The 60-GiB result cannot be rolled back to 48 GiB with `ModifyVolume`.
- A size rollback would require a separately authorized snapshot/new-volume migration with downtime and attachment changes; none was attempted or authorized here.
- No snapshot was created by this agent; Owner-console snapshot state is unknown.
- Cost delta is exactly 12 additional GiB-month of gp3 storage; provisioned IOPS and throughput costs remain unchanged. Billing verification is pending normal Cost Explorer access/owner review.

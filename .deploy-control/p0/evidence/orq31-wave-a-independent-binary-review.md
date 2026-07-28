# ORQ-31 — Security Wave A independent binary review

- Card: `ORQ-31` (`f7e13350-c7f2-4335-8a04-01b98527d034`)
- Reviewer: Codex56#B (`w7:p4`), independent of the original executor
- Review time: `2026-07-28T16:03:18Z`
- Mode: READ-ONLY. No `chmod`, `mv`, removal, write on a target host, secret
  resolution, AWS call, service operation, board mutation, or sensitive-file
  content read.
- **Verdict: PASS for the declared Wave A containment scope.**
- **Non-verdict: this is not PASS or closure for Wave B, credential rotation, or
  structural `umask` remediation.**

## Evidence chain

| artifact | SHA-256 | role |
|---|---|---|
| `gtl-security-remediation-wave-a-execution.md` | `ce69152cc6843b979abf3053ef85b3bc3860d2695ff070cf81defb12650f6a2d` | original execution manifest |
| `CHECKOUT__Codex56-A__C__SECURITY-WAVE-A-CONTAINMENT__20260727T123300Z.json` | `796159330c2a2bde4e90b8711118f932b3775193d950db37a97309ae9d382652` | original execution receipt |
| `orq31-wave-a-current-state-verification.md` | `b755015d45b74bb9516faee4e71c1ce4dc288295b283b0b2f37a2b811c99b882` | current-state self-audit by the original executor, used only as input |

The independent measurements below reproduced the self-audit without relying on
its verdict. The original executor correctly emitted no PASS/BLOCK.

## Binary acceptance

| requirement | result | independent evidence |
|---|---|---|
| Correct ORQ2 decomposition | **PASS** | Wave A is not “12 files changed in place.” The correct accounting is four targets retained in `/tmp` plus nine targets moved to quarantine. The original inventory covered 12 paths; the later `/tmp/end.txt` discovery made 13 containment actions over that evolving 12-path inspection scope. Current state contains all four retained targets and all nine quarantined objects. |
| Four ORQ2 in-place targets | **PASS** | `/tmp/arch.txt`, `/tmp/backend-recover.sh`, `/tmp/mcp.json.bak`, and `/tmp/end.txt` are each `0600 ec2-user:ec2-user`. |
| ORQ2 quarantine | **PASS** | `/home/ec2-user/.private-tmp/quarantine-20260727T123153Z` is `0700 ec2-user:ec2-user`; exactly nine files are present; the distinct file metadata is only `0600 ec2-user:ec2-user`; none of the nine source paths remains in `/tmp`. |
| ORQ1 `mh` / `mcp_h` | **PASS** | Both remain `0600 ec2-user:ec2-user`. |
| ORQ1 rollback backups | **PASS, metadata scope** | Both named `daemon.environ.*.bak` files exist as `0600 ec2-user:ec2-user`, with equal current size `2449`. `/tmp/test_login.html` remains `0644 root:root`, intentionally outside Wave A. “Intact” here means presence and safe metadata only; no byte-level assertion is made because reading or hashing sensitive backup content was prohibited and no prior digest exists. |
| LOCAL 16 targets | **PASS for file containment only** | Independent SSH measurement found `missing=16 present=0`, with zero permissive level-one files owned by `dataops-lab`. Absence does not prove secure deletion, credential invalidation, or rotation. |
| No Wave B conflation | **PASS** | The original manifest explicitly says no rotation occurred. B1, B4, and B5 remain open, as do the other separately authorized Wave B items. |
| Structural recurrence | **OPEN follow-up, not a Wave A failure** | ORQ2 and ORQ1 still have `/etc/bashrc:75` setting `umask 002`; login shells independently returned `0002`. ORQ2 permissive level-one files increased from `209` to `237` (`+28`). This confirms containment remains point-in-time and does not close the root cause. |

## Independent current-state snapshot

### ORQ2

- Identity: `ip-172-31-30-9.sa-east-1.compute.internal`; current review host.
- In-place target modes/owner: `4/4` at `0600 ec2-user:ec2-user`.
- Quarantine: directory `0700`; files `9`; one distinct mode/owner tuple,
  `0600 ec2-user:ec2-user`.
- Original quarantine source paths still present: `0`.
- `/etc/bashrc:75`: `umask 002`; `bash -lc umask`: `0002`.
- `/tmp`, level-one, regular files owned by `ec2-user` with any group/world
  permission: `237`, versus the original Wave A count `209`.

### ORQ1

- Identity: `ip-172-31-18-217.sa-east-1.compute.internal`,
  Tailscale `100.118.244.61`, user `ec2-user`.
- `mh` and `mcp_h`: both `0600`.
- Two named rollback backups: both present, `0600`, same owner, same current size.
- Root-owned excluded artifact: still `0644 root:root`.
- `/etc/bashrc:75`: `umask 002`; `bash -lc umask`: `0002`.
- Permissive level-one regular-file count for `ec2-user`: `69`. No historical
  ORQ1 total was recorded, so no delta is claimed.

### LOCAL

- Identity: `21LAPGLMVPJ4`, Tailscale `100.117.245.15`, user `dataops-lab`.
- Original target set: `16` absent, `0` present.
- `bash -lc umask`: `0022`.
- Permissive level-one regular-file count for `dataops-lab`: `0`.

## Follow-up requirements, separate from Wave A acceptance

1. **B1 / OPENAI credential:** preserve the ORQ2 `arch.txt` containment, but
   perform provider-side rotation and consumer/store validation in its separately
   authorized Wave B card. The missing LOCAL file is not evidence that the
   credential was invalidated.
2. **B4 / handshake token and B5 / rev token:** establish their issuer and
   current validity using metadata/handle-only procedures, then invalidate and
   reissue under their separate owner-authorized windows. The missing LOCAL
   files do not close either lifecycle.
3. **Structural `umask` remediation:** implement and independently gate private
   `TMPDIR`/cache roots plus the approved durable `umask 077` mechanism. Verify
   interactive and login shells and relevant service units. Do not bulk-change
   the newly counted `237` files without a new inventory, ownership check,
   consumer check, rollback, and explicit authorization.
4. **Other Wave B items:** B2, B3, B6, and B7 remain governed by their individual
   prerequisites and authorizations. This Wave A PASS grants no rotation,
   restart, deletion, or provider operation.

## Conclusion

**PASS — Wave A containment only.** Every still-existing target remains protected
exactly as declared, the quarantine is complete, and the disappeared LOCAL
targets no longer expose files at those paths. The acceptance is deliberately
narrow: no credential invalidation is proven, B1/B4/B5 remain open, and the
unchanged `umask 0002` has already produced measurable recurrence (`209 → 237`)
on ORQ2.


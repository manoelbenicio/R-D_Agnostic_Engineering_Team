# W8 governance reconciliation draft — cold recovery and zero overlap

Status date: 2026-07-19. Reviewer: W8 / Codex (`/root`), evidence/spec review
only. Producer roles remain the planning authors recorded in the source
artifacts. Codex56-Principal-TL is the zero-overlap verifier/acceptor; Kiro TL is
the planning adjudicator. W8 is neither.

## W8 embedded check-in/out

- CHECK-IN: recorded in this owned artifact because shared GSD/ledger authorship
  is forbidden. Exact files locked: the four new
  `openspec/changes/*/evidence/w8-*-draft.md` files in this commit; product files
  and existing evidence were read-only.
- CHECK-OUT: 2026-07-19T18:00:11Z. Four review drafts produced; source/evidence
  manifests re-hashed; five affected OpenSpec changes validate; no checkbox,
  product, GSD, secret, auth, service, DB, network, or production mutation.

## Cold-recovery disposition

The two OpenSpec changes are aligned on D-V3-16:

- Agent Brain task 10.4 remains open and requires W1 implementation/evidence.
- `persist-prodex-runtime-integration` remains 0/16 and is deferred to a
  default-OFF, mutually-exclusive, operator-gated cold recovery mode.
- OmniRoute unavailability means DEGRADED/fail-closed. It never automatically
  selects Prodex. Recovery entry requires an explicit operator transition,
  OmniRoute quiescence, and a session boundary. Restore requires Prodex drain.
- Prodex is retained, not deleted, and is never a per-request or simultaneous
  hot router.

Classification: **design/spec reconciled; implementation and acceptance open**.
No checkbox change is justified by this documentary review.

## Amended zero-overlap disposition

Commit `fbabd9ce130e9dc1d5d40158d8ecdfa004a63193` amends W4 ownership with the
real observability stack and reports a 159-file pairwise-disjoint result. This
W8 branch intentionally remains based on `4c67ae0`; it does not merge or rewrite
the amendment. The amendment itself says re-acceptance by
Codex56-Principal-TL is pending and W4 must not edit the real stack or claim
OBS-11 until that happens.

Classification: **amendment PRODUCED, NOT ACCEPTED; W4 HOLD remains active**.
The earlier base artifact is historical evidence, not proof of the amended
159-file scope.

## Exact evidence/spec manifest

```text
fbabd9ce130e9dc1d5d40158d8ecdfa004a63193  planning amendment commit (reviewed via git show)
de83dc1ba5405acb9cad666a97c53db9ec8b03870698906120d9829ad6b86b8e  .planning/agent-brain-v3/evidence/ev-zero-overlap-wave-b0.md
e1a654162c57e3e2ea7a8330007105c6647d4adfd4a1b0c4132fe57ee28c7504  .planning/agent-brain-v3/evidence/persist-prodex-vs-omniroute-reconciliation-audit.md
```

The `ev-zero-overlap-wave-b0.md` hash above is the pre-amendment, 139-file
scope; it must not be represented as the amended 159-file proof.

## Required next evidence

1. Codex56-Principal-TL re-runs/reviews the amended 159-file intersection proof
   and records explicit acceptance before releasing the W4 stack hold.
2. W1 separately supplies EV-REC-MODE implementation evidence; this draft does
   not claim W1 task 10.4 or any persist-Prodex checkbox.
3. Producer, reviewer, and adjudicator identities remain separately recorded;
   W8 does not self-accept this draft.

No product, GSD, task checkbox, secret, auth file, service, DB, network, merge,
or production state was changed or accessed.

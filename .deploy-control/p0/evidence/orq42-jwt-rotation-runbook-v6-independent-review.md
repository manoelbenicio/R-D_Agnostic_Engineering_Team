# ORQ-42 — V6 JWT rotation runbook independent review

**Mode:** READ-ONLY; no secret, AWS, Docker, Compose, DB, board, login or
runtime action performed. The Secrets Manager skill was loaded in full.

## Verdict: BLOCK pending executable corrections

V6 materially fixes the V5 prefix error and aligns its compose allowlist with
the post-ORQ17 Stage 2 evidence, but it is not yet an executable approval
package. The following gates remain hard blockers.

### B1 — Helper/source and durable source are still unproven

V6 pins ORQ-30 helper hashes and names `/home/ec2-user/.config/multica-transition/dev.env`,
but explicitly admits that neither helper source nor the live labels/source were
read in this review. Post-ORQ17 proves the project, backend service and five-file
compose invocation, but does not prove the helper hashes or that `dev.env` is
the actual interpolation source for the current backend. Before approval, the
owner must capture metadata-only evidence of the four labels, exact config-file
list plus `dev.env` identity/mode, and a source-hash/diff review proving the
helper passes `--env-file dev.env` and only recreates backend.

### B2 — G5 is too invasive for the stated secret boundary

The proposed `docker inspect -f '{{json .Config.Env}}'` reads the entire
container environment and pipes it to `grep`, even though V6 otherwise forbids
full `Config.Env` reads. That can expose unrelated secrets to the operator/log
surface and is not necessary to prove the dynamic reference. Replace it with a
non-secret, fixed-field metadata check or an owner-run child-side assertion that
returns only a boolean/status pair; never serialize the full environment.

### B3 — JWT consumer impact is incomplete as an acceptance gate

The correction that `mdt_` uses hash lookup while user JWT and `/ws` use
`JWTSecret` is factual and consistent with the code. However, the runbook's
acceptance matrix must explicitly enumerate both user HTTP JWT consumers and
the user realtime `/ws` consumer, with old/new token gates and a separately
authorized communication window. Daemon re-pairing must remain conditional on
the measured auth path, not inferred from the daemon's existence.

### B4 — Token proof and rollback custody require named owner artifacts

V6 correctly requires old=200 before mutation and old=401/new=200 after, and
retains both credential files through R4. Q-E is still open: the owner must
name the private token acquisition mechanism, paths, modes, custody owner and
cleanup receipt without putting either token in agent context. Otherwise the
runbook cannot prove the pre-forward gate or symmetric rollback.

### B5 — Dynamic-reference verification must be fail-closed without value

Environment injection into `asm-exec` is consistent with the skill, and the
residual `/proc/.../environ` exposure is honestly disclosed. Approval still
requires a fixed full ARN, region, JSON key and frozen version stage, plus a
metadata-only existence/identity check. No step may resolve or print the value,
and the editor must prove the non-target bytes remain identical.

## What passes

- The V6 correction distinguishes `mdt_` hash validation from JWT fallback.
- It identifies the user realtime `/ws` JWT dependency.
- ORQ17 evidence confirms project `multica-dev-transition`, backend-only
  recreate, exact four original compose files plus one auth override, and the
  durable `dev.env` allowlist path.
- Backup/restore symmetry, old/new token gates, helper hash equality, and
  separate authorizations are directionally correct.

## Exact next action

Do not rotate. Add the missing helper/source proof, replace full `Config.Env`
inspection, publish the complete JWT/`/ws` acceptance matrix, and bind Q-B,
Q-C, Q-E, Q-F, Q-G plus A1–A5 to named owner attestations. Then request a new
independent review; no AWS or runtime mutation is authorized by this review.

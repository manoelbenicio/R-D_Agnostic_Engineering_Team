# internal/handler/auth.go — current dirty-diff ownership matrix

**Author:** Kiro/Sonnet, pane `w7:p1` — read-only ownership trace only.
**Date:** 2026-07-18T18:31:52-03:00
**Adjudication authority:** Kiro TL adjudicates. Root integrates. This
document makes no product/test/spec/task/git/index edit, reads no
credential/env value, and performs no network/DB/live-provider action. It
partitions and cites; it does not implement, accept, or reject anything.

## Golden Rule check-in / check-out

- **Check-IN** 2026-07-18T18:31:52-03:00 — claimed scope: read-only diff/hash
  trace of `internal/handler/auth.go` plus cross-reference against existing
  evidence artifacts (`native-auth-password-provisioning.md`,
  `native-onboarding-1.5-review.md`, `native-onboarding-1.5-web.md`,
  `credential-isolation-5.4-*`, `FILE_OWNERSHIP.md`, `AGENT_LEDGER.md`). One
  output file only:
  `.planning/agent-brain-v3/evidence/handler-auth-go-current-diff-ownership-matrix.md`.
- Excluded (honored): no product/test/shared-doc/spec/task edit; no git
  stage/commit/push/index mutation beyond `git diff`/`git show` read
  commands; no credential/env value read; no DB/network/live-provider
  action.
- **Check-OUT** 2026-07-18T18:40:00-03:00 — DONE; matrix below; no files
  other than this one were modified or created.

## File identity and hashes

| State | SHA-256 |
|---|---|
| Current working tree (`internal/handler/auth.go`, dirty) | `d69877a9dcabec59726717628c103b2b1ab0c9a4c7673d14a80c8360b7a259e0` |
| Base (`git show HEAD:...`, last committed) | `ace9e7234a82132094a0bb49558d0e49bc8c710a271c6cdbbf7013bb492a006c` |

`git diff --stat` confirms exactly **1 file changed, 96 insertions(+), 3
deletions(-)** — the "99-line dirty diff" cited in the task is this exact
diff (96+3=99 changed lines across the unified diff body).

## Hunk-by-hunk partition

`git diff` against `HEAD` shows **three** hunks, not more. Each is
partitioned below strictly by content match against existing evidence
artifacts' own file-hash and line-citation claims — never by proximity or
assumption.

### Hunk A — `Login` call site (1 line changed)

```
@@ -137,7 +137,7 @@ func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
-	tokenString, err := h.issueJWT(user)
+	tokenString, err := h.issueRecentlyAuthenticatedJWT(user)
```
Current location: inside `Login` (func starts line 110).

### Hunk B — new `issueRecentlyAuthenticatedJWT` helper + two more call-site swaps (largest hunk, ~30 lines across 3 sub-diffs in the unified output, treated as one logical hunk since they share a single feature)

```
@@ -213,6 +213,19 @@ func (h *Handler) issueJWT(user db.User) (string, error) {
+func (h *Handler) issueRecentlyAuthenticatedJWT(user db.User) (string, error) {
+	... "auth_time": now.Unix() ...
+}
```
Current location: new function at line 216, immediately after the existing
`issueJWT` (line 205).

```
@@ -447,7 +460,7 @@ func (h *Handler) VerifyCode(...)
-	tokenString, err := h.issueJWT(user)
+	tokenString, err := h.issueRecentlyAuthenticatedJWT(user)
```
Current location: inside `VerifyCode` (func starts line 416).

```
@@ -643,7 +736,7 @@ func (h *Handler) GoogleLogin(...)
-	tokenString, err := h.issueJWT(user)
+	tokenString, err := h.issueRecentlyAuthenticatedJWT(user)
```
Current location: inside `GoogleLogin`.

### Hunk C — new `UpdatePassword` handler + `UpdatePasswordRequest` type (~80 lines)

```
@@ -497,6 +510,86 @@ type UpdateMeRequest struct { ... }
+const passwordUpdateBodyLimit = 1024
+const recentPasswordAuthenticationWindow = 5 * time.Minute
+type UpdatePasswordRequest struct { CurrentPassword, NewPassword string }
+func (h *Handler) UpdatePassword(w http.ResponseWriter, r *http.Request) { ... }
```
Current location: new type at line 517, new handler immediately after.

## Provenance match against existing evidence

| Hunk | Feature | Producer / evidence artifact | Match method |
|---|---|---|---|
| A, B, C (all) | native-runtimes-onboarding **task 1.7** — backend password provisioning + `auth_time`-gated re-authentication window | `.planning/agent-brain-v3/evidence/native-auth-password-provisioning.md` (`EV-AUTH-1.7`, sha256 `2a5f7368a63202f5decb27bd562589e1cc9ad406499b29a14a471f1c1425c095` per the prior ledger row) | The artifact's own file-hash manifest line 19 cites `internal/handler/auth.go` at hash `d69877a9dcabec59726717628c103b2b1ab0c9a4c7673d14a80c8360b7a259e0` — **exact match** to this file's current working-tree hash. The artifact additionally cites specific line ranges `auth.go:513-575` (matches `UpdatePassword`/`UpdatePasswordRequest`, hunk C), `auth.go:205-227` (matches `issueJWT`/`issueRecentlyAuthenticatedJWT`, hunk B), and `auth.go:763-785` (GoogleLogin-area call site, hunk B/A family). The artifact also names test functions `UpdatePassword`, `PasswordLoginMints`, `PasswordAuthRoutes`, `PasswordUpd...` run against this exact file state. |

**Conclusion: all three hunks belong to a single feature and a single
already-identified producer/evidence chain (native 1.7).** There is no
internal ownership conflict within this file's current diff — every hunk is
part of the same password-provisioning + recent-authentication-window
change, not three separate unrelated changes that happen to share a file.

## Explicit non-matches (checked, ruled out by content, not by proximity)

- **native-runtimes-onboarding task 1.5 (frontend/web onboarding):**
  `native-onboarding-1.5-review.md` and `.deploy-control/evidence/native-onboarding-1.5-web.md`
  were both searched for any reference to `auth.go`, `issueJWT`,
  `UpdatePassword`, `GoogleLogin`, or `VerifyCode`. **Zero matches in either
  artifact.** Task 1.5's own evidence exclusively cites frontend files
  (`packages/views/auth/login-page.tsx`, `apps/web/app/(auth)/login/page.tsx`,
  `apps/web/app/auth/callback/page.tsx`, `packages/core/api/client.ts`) and
  Vitest/UI assertions — a completely disjoint surface from this Go handler
  file. **Task 1.5 has no ownership stake in this diff.** (The task
  explicitly asked to pay special attention to 1.5 — the finding is that 1.5
  is not implicated at all, not that it shares a hunk.)
- **credential-isolation 5.4 (log-safety):** the Google OAuth error-body
  logging line the 5.4 review chain discusses
  (`slog.Error("google oauth token exchange returned error", "status", ...,
  "body", string(tokenBody))`) is at **line 656** in the current file — this
  line is **not present in any hunk of the current diff** (confirmed by
  reading the full `git diff` output above: no hunk header spans line 656,
  and the closest hunk boundary is hunk B's third sub-diff ending at the
  `GoogleLogin` call-site line, which is a different line than 656 and a
  different statement). **The Google OAuth logging line is unmodified
  baseline code, unrelated to this dirty diff.** Credential-isolation 5.4
  has no ownership stake in this diff's changed lines; the file merely
  happens to *also* contain the already-reviewed 5.4-relevant logging
  statement elsewhere, which is exactly the proximity trap the task warned
  against inferring ownership from.
- **`FILE_OWNERSHIP.md` / `AGENT_LEDGER.md`:** neither document contains any
  entry for `internal/handler/auth.go` or `internal/handler/**` under any
  agent's owned-hotspot table. No explicit lock or ownership claim exists on
  this file in either governance document as of this check-in.

## Overlapping ownership assessment

**No overlapping ownership found.** All three hunks trace to one producer
lineage (native 1.7 backend auth work, already evidenced and — per the
earlier-reviewed ledger row — independently accepted as `EV-AUTH-1.7`). No
second feature's hunk is interleaved in this file's current diff. The
"mixed-file" framing in the task's title is a reasonable caution given the
file's dual role (it contains both the 1.7 diff and the unrelated, unchanged
5.4-relevant OAuth logging line), but the actual dirty *diff* itself is
single-owner, not mixed.

## Can any accepted atomic patch include or safely split this file?

- **Include as-is:** yes. Since all three hunks belong to the same feature
  (native 1.7) and no other feature's change is interleaved, an atomic patch
  for native 1.7 backend work can include the entirety of this file's current
  diff without needing to split anything or risk sweeping in unrelated work.
- **Split:** not required for feature-isolation purposes (there is only one
  feature here), but hunk C (`UpdatePassword` + type, ~80 lines) is the
  largest and most independently testable unit (it has its own named tests
  per the 1.7 evidence artifact: `UpdatePassword`, `PasswordUpd...`) — if a
  smaller atomic patch is preferred for review size reasons, hunk C could be
  committed separately from hunks A/B (the `issueRecentlyAuthenticatedJWT`
  rename family), since hunk C does not call or depend on the renamed
  function. Hunks A and B are mutually dependent (A and the two call sites
  in B all call the new function introduced in B) and must move together —
  splitting A away from B's function definition would break the build.
- **Safety note:** because the Google OAuth logging line (656) is
  *unmodified* baseline in this same file, any commit of this diff will
  necessarily also carry that unrelated, unmodified line forward unchanged —
  this is normal (committing a file's diff does not touch its unchanged
  lines) and is not a hazard, but is recorded here explicitly so a future
  reviewer does not mistake "this commit touches auth.go" for "this commit
  changes the OAuth logging behavior." It does not.

## Non-claims

- This document does not accept, reject, or recommend accepting the
  underlying native-1.7 password-provisioning feature itself — that
  adjudication already exists independently (`EV-AUTH-1.7`, per the prior
  ledger row, ACCEPT → CHECKED).
- This document does not touch `tasks.md`, any shared ledger row, `STATE.md`,
  the git index, or any credential/env value.
- This document does not verify the *correctness* of the 1.7 feature itself
  (e.g., whether `auth_time`-gating is cryptographically sound) — that is out
  of this ownership-trace task's scope. It verifies only which producer/task
  each hunk belongs to and whether ownership overlaps.

Kiro TL adjudicates whether this ownership read is sufficient for root to
integrate the file as a single atomic patch or to split it per the note
above.

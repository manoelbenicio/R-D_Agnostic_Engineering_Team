# ORQ-98 Changed-File Allowlist and Acceptance Map

| Path | Purpose / acceptance evidence |
|---|---|
| `.deploy-control/evidence/ORQ-98/pre-edit-checkin.md` | Authority, exclusive path lease, immutable seam hashes, and baseline identity. |
| `.deploy-control/evidence/ORQ-98/validation-summary.md` | Commands, timestamps, exit codes, contradiction, and residual blocker. |
| `.deploy-control/evidence/ORQ-98/changed-file-map.md` | Exact changed-file allowlist and evidence mapping. |
| `.deploy-control/evidence/ORQ-98/input-hashes.sha256` | Reproducible hashes for authority, source seams, test inputs, and dependency manifests. |
| `multica-auth-work/server/internal/auth/cloud_pat_site_log_redaction_test.go` | Actual `cloud_pat.go` non-200 seam; synthetic sentinel failure-path assertion and nominal log-site discovery. |
| `multica-auth-work/server/internal/handler/orq98redaction/auth_log_redaction_test.go` | Actual `auth.go` Google token non-OK seam; synthetic sentinel assertion plus safe diagnostic discovery, isolated from database-gated parent `TestMain`. |

No product source, OpenSpec, credential/config, runtime, production, or Kanban file is changed.

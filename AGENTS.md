# AWS Agent Operating Rules

## Scope

These rules apply to every agent working in this repository. More specific nested `AGENTS.md` files remain authoritative for their subtrees.

## AWS workflow

- Prefer the project AWS MCP Server for AWS documentation, service discovery, regional availability, resource inspection, and other supported AWS interactions.
- Before an AWS task, check `.agents/skills/` for a relevant skill and follow it instead of improvising from model memory.
- Verify current AWS API parameters, permissions, quotas, service availability, and error behavior through the AWS MCP documentation tools or official AWS documentation.
- Prefer AWS CDK or CloudFormation for infrastructure changes. Direct AWS CLI mutations require explicit task scope and owner authorization.
- Apply AWS Well-Architected principles, least privilege, encryption, auditability, rollback, and cost awareness.
- The workload default Region is `sa-east-1`. The managed AWS MCP endpoint is hosted in `us-east-1`; do not confuse the endpoint Region with the target workload Region.
- The project MCP configuration is read-only by default. Do not remove `--read-only` or bypass it unless the owner explicitly authorizes AWS mutations for a bounded task.
- Never assume that an AWS call is permitted merely because the EC2 instance role is available. IAM remains the authority; fail closed on `AccessDenied`.

## Secret safety

- Load `.agents/skills/aws-secrets-manager/SKILL.md` before any task involving a secret, credential, password, API key, token, cookie, or private certificate.
- Never call `secretsmanager:GetSecretValue` or `BatchGetSecretValue` through AWS CLI, SDK, MCP, curl, or another mechanism that returns plaintext to agent context.
- Use `asm-exec` with `{{resolve:secretsmanager:...}}` references so secret values resolve only inside the target child process.
- Never print, log, commit, paste, diff, summarize, or include secret values in command arguments.

## Parallel-agent discipline

- One agent owns one bounded task, branch, worktree, and non-overlapping file set.
- The main architect is the single writer for integration decisions and shared planning state.
- Agents must stop and report overlapping ownership rather than creating duplicate work or conflicting edits.

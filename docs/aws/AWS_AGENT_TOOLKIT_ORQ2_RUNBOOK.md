# AWS Agent Toolkit on ORQ2

Status date: 2026-07-21 (America/Sao_Paulo)

## 1. Purpose

This installation gives project agents current AWS documentation, curated AWS procedures, regional availability data, and AgentCore development tooling without duplicating configuration across credential-isolated agent homes.

The installation is project-scoped because ORQ2 had 121 isolated Codex homes at audit time. Installing a full plugin separately into every home would waste disk, drift versions, and make upgrades inconsistent.

## 2. Pinned upstream sources

| Component | Version or commit | Source |
| --- | --- | --- |
| Agent Toolkit for AWS | `fae975d565b6c1752e3f4795c499fb3951039777` | `https://github.com/aws/agent-toolkit-for-aws` |
| MCP Proxy for AWS | `1.6.4` | `mcp-proxy-for-aws` through `uvx` |
| AWS CLI | `2.36.5` | official AWS CLI Linux installer |
| AgentCore CLI | `0.24.1` | official npm package `@aws/agentcore` |
| `asm-exec` | file from the pinned AWS toolkit commit | `aws-core/skills/aws-secrets-manager/references/asm-exec` |

The AWS repository was audited before installation. It is the official AWS-supported successor to the older AWS Labs MCP packages. Do not add the older AWS API MCP Server or AWS Knowledge MCP Server because overlapping tools can confuse agents.

## 3. Installed project configuration

| Agent | Project configuration |
| --- | --- |
| Codex | `.codex/config.toml` |
| Claude Code | `.mcp.json` and `CLAUDE.md` |
| Kiro | `.kiro/settings/mcp.json` and `.kiro/steering/aws-agent-toolkit.md` |
| All project agents | `AGENTS.md` and `.agents/skills/` |

All three MCP client configurations use:

```text
uvx mcp-proxy-for-aws@1.6.4 \
  https://aws-mcp.us-east-1.api.aws/mcp \
  --metadata AWS_REGION=sa-east-1 INSTALL_SOURCE=agent-toolkit-core \
  --read-only
```

The endpoint Region is `us-east-1`; AWS operations default to the workload Region `sa-east-1` through metadata. The MCP proxy uses the EC2 instance role through SigV4. No access key or secret key was created, copied, or stored.

## 4. Security posture

- MCP tool exposure is read-only by default through `--read-only`.
- The active EC2 role is `cw-agent-orquestradores`.
- IAM permissions on that role remain authoritative for every downstream AWS call.
- The role can obtain its STS identity but cannot enumerate its attached or inline IAM policies. This limitation was verified and is retained.
- AWS MCP requests receive AWS audit context keys and downstream calls are visible in CloudTrail according to AWS service behavior.
- `asm-exec` is installed at `~/.local/bin/asm-exec` for runtime-only secret resolution.
- AgentCore CLI telemetry is disabled.
- No AWS resource was created, modified, or deleted during installation or validation.

Do not remove `--read-only` merely because a task needs progress. The owner must explicitly authorize a bounded write task, identify the target account/Region/resources, and approve rollback. IAM should additionally deny destructive MCP actions where appropriate.

## 5. Installed skills

Forty-eight skills are installed under `.agents/skills/` from the pinned AWS commit.

### AWS Core

- Bedrock and AgentCore foundations
- Billing and cost management
- CDK and CloudFormation
- Compute, containers, serverless, deployment
- IAM and secret safety
- Databases, messaging, streaming, networking
- CloudWatch, X-Ray, CloudTrail, observability
- AWS SDK usage for Python, JavaScript v3, and Swift
- AWS sign-in and application launch guidance

### AgentCore and orchestration

- `agents-get-started`
- `agents-build`
- `agents-connect`
- `agents-deploy`
- `agents-debug`
- `agents-harden`
- `agents-optimize`

### Data and analytics

- OpenSearch
- Data source connectivity and ingestion
- Glue Data Catalog exploration
- Data lake asset discovery, table creation, and Athena queries
- S3 vector storage and querying

### Environment-relevant specialized skills

- EC2 launch and instance profiles
- VPC endpoints, Route 53, and network monitoring
- CloudTrail and CloudWatch alarm setup
- Application failure troubleshooting
- Secrets creation best practices
- Aurora PostgreSQL, RDS open-source engines, and ElastiCache/Redis
- S3 bucket security and file troubleshooting

## 6. Additional tools

### AWS CLI

The user-local AWS CLI is `~/.local/bin/aws`. It was upgraded because the previous system CLI `2.33.15` did not contain the `agent-toolkit` command group. Version `2.36.5` provides:

```bash
aws configure agent-toolkit
aws agent-toolkit list-available-skills --region us-east-1
```

The repository remains the source of truth for installed project skills; do not run the global interactive wizard in every isolated home.

### AgentCore CLI

AgentCore CLI `0.24.1` is installed globally through npm:

```bash
agentcore --version
agentcore validate
agentcore deploy --dry-run
agentcore status
```

Do not run `agentcore deploy` without explicit authorization. Its deployment path uses CDK/CloudFormation and can create billable resources.

### Secret-safe execution

Validate only the wrapper itself without resolving a secret:

```bash
asm-exec
```

For actual use, follow `.agents/skills/aws-secrets-manager/SKILL.md`. Never call `aws secretsmanager get-secret-value` directly.

## 7. Verification

Start a new agent session after pulling the installation commit. Existing agent processes do not automatically reload project MCP or skill configuration.

Basic verification:

```bash
uvx mcp-proxy-for-aws@1.6.4 --help
aws --version
aws sts get-caller-identity
agentcore --version
find .agents/skills -mindepth 2 -maxdepth 2 -name SKILL.md | wc -l
```

Expected skill count: `48`.

The authenticated validation on 2026-07-21 returned exactly these six tools:

```text
aws___get_tasks
aws___get_regional_availability
aws___list_regions
aws___read_documentation
aws___retrieve_skill
aws___search_documentation
```

No direct AWS API caller or resource-mutation tool is currently exposed. Resource inspection must therefore continue through separately authorized AWS CLI or SDK workflows, with the EC2 IAM role remaining authoritative. Revalidate the MCP tool inventory after future toolkit upgrades because AWS controls the managed server capability set.

## 8. Intentionally not enabled

`aws-agents-for-devsecops` is not enabled. It requires a separate `DEVOPS_AGENT_TOKEN`, configured AWS DevOps/Security Agent Spaces, and authorization for incident investigation, security scanning, UAT, or penetration testing. Installing it without those prerequisites would add unusable tools and could encourage unauthorized security activity.

Enable it only through a separate approved change with credential isolation, target scope, rules of engagement, and audit evidence.

## 9. ORQ2 disk remediation during installation

The first AWS CLI upgrade attempt exposed disk exhaustion caused mainly by 121 historical credential slots. Cleanup preserved every slot visible in an active process plus static credential source slots `1`-`5` and `13`.

- Preserved slots: `17`
- Active slots at cleanup: `11`
- Historical orphan slots removed: `104`
- Registry reduced to the preserved slot set
- Reproducible npm, Go-build, NVM, Codex temporary, and inactive Go module caches were removed
- Credentials, current agent processes, active Kiro runtime data, branches, worktrees, and source code were preserved
- Final free space after tool installation and cache cleanup: approximately `2.6 GB`

Future cleanup must repeat live-process discovery before deleting any credential slot. Never infer that a slot is orphaned from its age alone.

# AWS Agent Toolkit

Before AWS work, read `AGENTS.md` and load the relevant skill from `.agents/skills/`.

Use the project `aws-mcp` server for current AWS knowledge and read-only resource inspection. The workload Region is `sa-east-1`; the MCP endpoint itself is hosted in `us-east-1`.

AWS mutations require explicit owner authorization. Never retrieve a secret value into agent context; follow `.agents/skills/aws-secrets-manager/SKILL.md` and use `asm-exec`.

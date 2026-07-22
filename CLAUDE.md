# Project Agent Guidance

Read and follow `AGENTS.md` before performing work.

AWS skills are installed under `.agents/skills/`. Load the relevant `SKILL.md` before AWS work. The project AWS MCP Server is configured in `.mcp.json` and runs read-only by default against workload Region `sa-east-1`.

Never expose AWS credentials or secret values. Use the `aws-secrets-manager` skill and `asm-exec` for secret resolution.

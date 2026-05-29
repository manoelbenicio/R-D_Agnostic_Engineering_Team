// Master spec §8.7: Tier 1 wall-clock provider cost table.
export const PROVIDER_COST_PER_HOUR = {
  kiro_cli: 15.0,
  claude_code: 15.0,
  codex: 5.0,
  gemini_cli: 0.5,
  kimi_cli: 2.0,
  copilot_cli: 3.0,
  opencode_cli: 1.0,
  q_cli: 5.0,
} as const;

export type CostProvider = keyof typeof PROVIDER_COST_PER_HOUR;

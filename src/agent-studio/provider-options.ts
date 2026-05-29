import { ProviderType as KeyProviderType } from '@/api/key-store/registry';

export interface AgentStudioProviderOption {
  provider: string;
  sourceProvider: KeyProviderType;
  label: string;
}

const PROVIDER_OPTIONS: Record<KeyProviderType, AgentStudioProviderOption[]> = {
  openai: [{ provider: 'codex', sourceProvider: 'openai', label: 'Codex' }],
  anthropic: [{ provider: 'claude_code', sourceProvider: 'anthropic', label: 'Claude Code' }],
  google: [{ provider: 'gemini_cli', sourceProvider: 'google', label: 'Gemini CLI' }],
  aws: [
    { provider: 'q_cli', sourceProvider: 'aws', label: 'AWS Q CLI' },
    { provider: 'kiro_cli', sourceProvider: 'aws', label: 'Kiro CLI' },
  ],
  azure: [{ provider: 'codex', sourceProvider: 'azure', label: 'Azure OpenAI' }],
  moonshot: [{ provider: 'kimi_cli', sourceProvider: 'moonshot', label: 'Kimi CLI' }],
  copilot: [{ provider: 'copilot_cli', sourceProvider: 'copilot', label: 'Copilot CLI' }],
  opencode: [{ provider: 'opencode_cli', sourceProvider: 'opencode', label: 'OpenCode CLI' }],
};

export function getAgentStudioProviderOptions(
  validatedProviders: KeyProviderType[]
): AgentStudioProviderOption[] {
  return validatedProviders.flatMap((provider) => PROVIDER_OPTIONS[provider] ?? []);
}

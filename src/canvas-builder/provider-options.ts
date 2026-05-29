import { ProviderType as KeyProviderType } from '@/api/key-store/registry';
import { ProviderType } from '@/shared/canvas-types';

export interface CanvasProviderOption {
  provider: ProviderType;
  sourceProvider: KeyProviderType;
  label: string;
}

const PROVIDER_OPTIONS: Record<KeyProviderType, CanvasProviderOption[]> = {
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

export function getCanvasProviderOptions(validatedProviders: KeyProviderType[]): CanvasProviderOption[] {
  return validatedProviders.flatMap((provider) => PROVIDER_OPTIONS[provider] ?? []);
}

export function findSourceProvider(
  provider: ProviderType | undefined,
  options: CanvasProviderOption[]
): KeyProviderType | undefined {
  return options.find((option) => option.provider === provider)?.sourceProvider;
}

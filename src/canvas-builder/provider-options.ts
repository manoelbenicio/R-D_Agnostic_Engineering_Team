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

/** Map CLI provider names (from GO Core /agents/providers) to canvas options. */
const CLI_PROVIDER_OPTIONS: Record<string, CanvasProviderOption> = {
  codex: { provider: 'codex', sourceProvider: 'openai', label: 'Codex (CLI)' },
  kiro_cli: { provider: 'kiro_cli', sourceProvider: 'aws', label: 'Kiro CLI' },
  gemini_cli: { provider: 'gemini_cli', sourceProvider: 'google', label: 'Gemini CLI' },
  claude_code: { provider: 'claude_code', sourceProvider: 'anthropic', label: 'Claude Code (CLI)' },
  q_cli: { provider: 'q_cli', sourceProvider: 'aws', label: 'AWS Q CLI' },
  kimi_cli: { provider: 'kimi_cli', sourceProvider: 'moonshot', label: 'Kimi CLI' },
  copilot_cli: { provider: 'copilot_cli', sourceProvider: 'copilot', label: 'Copilot CLI' },
};

export function getCanvasProviderOptions(validatedProviders: KeyProviderType[]): CanvasProviderOption[] {
  return validatedProviders.flatMap((provider) => PROVIDER_OPTIONS[provider] ?? []);
}

/**
 * Merge API-key-validated providers with CLI-installed providers from the GO Core server
 * runtime. CLI providers authenticated via `kiro-cli login`, `codex auth`, etc.
 * don't need a browser API key — they use their own OAuth tokens.
 */
export function getCanvasProviderOptionsWithCli(
  validatedProviders: KeyProviderType[],
  installedCliProviders: string[],
): CanvasProviderOption[] {
  const fromKeys = getCanvasProviderOptions(validatedProviders);
  const seen = new Set(fromKeys.map((o) => o.provider));

  for (const cliName of installedCliProviders) {
    const option = CLI_PROVIDER_OPTIONS[cliName];
    if (option && !seen.has(option.provider)) {
      fromKeys.push(option);
      seen.add(option.provider);
    }
  }

  return fromKeys;
}

export function findSourceProvider(
  provider: ProviderType | undefined,
  options: CanvasProviderOption[]
): KeyProviderType | undefined {
  return options.find((option) => option.provider === provider)?.sourceProvider;
}

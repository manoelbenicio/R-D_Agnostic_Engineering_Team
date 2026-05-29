export type ProviderType =
  | 'openai'
  | 'anthropic'
  | 'google'
  | 'aws'
  | 'azure'
  | 'moonshot'
  | 'copilot'
  | 'opencode';

export interface KeyField {
  name: string;
  label: string;
  type: 'text' | 'password';
}

export interface ProviderDefinition {
  id: ProviderType;
  label: string;
  fields: KeyField[];
  defaultModels: string[];
}

export const PROVIDERS_REGISTRY: ProviderDefinition[] = [
  {
    id: 'openai',
    label: 'OpenAI',
    fields: [{ name: 'apiKey', label: 'API Key', type: 'password' }],
    defaultModels: ['gpt-4o', 'gpt-4o-mini', 'o1-preview', 'o1-mini'],
  },
  {
    id: 'anthropic',
    label: 'Anthropic',
    fields: [{ name: 'apiKey', label: 'API Key', type: 'password' }],
    defaultModels: ['claude-3-5-sonnet-20241022', 'claude-3-5-haiku-20241022', 'claude-3-opus-20240229'],
  },
  {
    id: 'google',
    label: 'Google Gemini',
    fields: [{ name: 'apiKey', label: 'API Key', type: 'password' }],
    defaultModels: ['gemini-1.5-flash', 'gemini-1.5-pro', 'gemini-2.0-flash-exp'],
  },
  {
    id: 'aws',
    label: 'AWS (Q + Kiro)',
    fields: [
      { name: 'accessKeyId', label: 'Access Key ID', type: 'text' },
      { name: 'secretAccessKey', label: 'Secret Access Key', type: 'password' },
    ],
    defaultModels: ['q-developer', 'kiro-agent-v1'],
  },
  {
    id: 'azure',
    label: 'Azure OpenAI',
    fields: [
      { name: 'endpoint', label: 'Endpoint URL (AZURE_OPENAI_ENDPOINT)', type: 'text' },
      { name: 'apiKey', label: 'API Key', type: 'password' },
    ],
    defaultModels: ['azure-gpt-4o', 'azure-gpt-3.5-turbo'],
  },
  {
    id: 'moonshot',
    label: 'Moonshot AI',
    fields: [{ name: 'apiKey', label: 'API Key', type: 'password' }],
    defaultModels: ['moonshot-v1-8k', 'moonshot-v1-32k'],
  },
  {
    id: 'copilot',
    label: 'GitHub Copilot',
    fields: [{ name: 'apiKey', label: 'OAuth Token / API Key', type: 'password' }],
    defaultModels: ['copilot-chat', 'copilot-codex'],
  },
  {
    id: 'opencode',
    label: 'OpenCode CLI',
    fields: [
      { name: 'endpoint', label: 'Endpoint URL', type: 'text' },
      { name: 'apiKey', label: 'API Key', type: 'password' },
    ],
    defaultModels: ['opencode-local-v1'],
  },
];

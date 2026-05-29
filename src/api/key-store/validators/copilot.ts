/* eslint-disable agentverse/no-sideways-capability-imports */
import { appFetch } from '@/shell/app-fetch';

export async function validateCopilot(apiKey: string): Promise<{ ok: boolean; error?: string; models?: string[] }> {
  try {
    const res = await appFetch('https://api.github.com/copilot_internal/v2/token', {
      method: 'GET', // GitHub Copilot proxy token endpoint typically responds to GET
      headers: {
        Authorization: `token ${apiKey}`,
        'editor-version': 'Neovim/0.9.5',
        'editor-plugin-version': 'copilot.vim/1.16.0',
        'user-agent': 'GithubCopilot/1.16.0',
      },
    });

    if (!res.ok) {
      const errText = await res.text();
      return { ok: false, error: `Copilot validation failed (${res.status}): ${errText}` };
    }

    return { ok: true, models: ['copilot-chat', 'copilot-codex'] };
  } catch (err: unknown) {
    const errMsg = err instanceof Error ? err.message : String(err);
    return { ok: false, error: `Copilot connection failed: ${errMsg}` };
  }
}

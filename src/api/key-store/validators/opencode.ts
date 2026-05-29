/* eslint-disable agentverse/no-sideways-capability-imports */
import { appFetch } from '@/shell/app-fetch';

export async function validateOpenCode(endpoint: string, apiKey: string): Promise<{ ok: boolean; error?: string; models?: string[] }> {
  try {
    const cleanEndpoint = endpoint.endsWith('/') ? endpoint.slice(0, -1) : endpoint;
    const url = `${cleanEndpoint}/models`;
    const res = await appFetch(url, {
      method: 'GET',
      headers: {
        'api-key': apiKey,
        Authorization: `Bearer ${apiKey}`,
      },
    });

    if (!res.ok) {
      const errText = await res.text();
      return { ok: false, error: `OpenCode validation failed (${res.status}): ${errText}` };
    }

    const data = (await res.json()) as { data?: { id?: string; name?: string }[]; models?: { id?: string; name?: string }[] };
    const list = data.data || data.models || [];
    const models = list.map((m) => m.id || m.name || '').filter(Boolean);
    
    return {
      ok: true,
      models: models.length > 0 ? models : ['opencode-local-v1'],
    };
  } catch (err: unknown) {
    const errMsg = err instanceof Error ? err.message : String(err);
    return { ok: false, error: `OpenCode connection failed: ${errMsg}` };
  }
}

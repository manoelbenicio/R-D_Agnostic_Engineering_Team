/* eslint-disable agentverse/no-sideways-capability-imports */
import { appFetch } from '@/shell/app-fetch';

export async function validateAzure(endpoint: string, apiKey: string): Promise<{ ok: boolean; error?: string; models?: string[] }> {
  try {
    const cleanEndpoint = endpoint.endsWith('/') ? endpoint.slice(0, -1) : endpoint;
    const url = `${cleanEndpoint}/openai/models?api-version=2024-02-01`;
    const res = await appFetch(url, {
      method: 'GET',
      headers: {
        'api-key': apiKey,
      },
    });

    if (!res.ok) {
      const errText = await res.text();
      return { ok: false, error: `Azure validation failed (${res.status}): ${errText}` };
    }

    const data = (await res.json()) as { data?: { id: string }[] };
    const models = (data.data || []).map((m) => m.id);
    return { ok: true, models };
  } catch (err: unknown) {
    const errMsg = err instanceof Error ? err.message : String(err);
    return { ok: false, error: `Azure connection failed: ${errMsg}` };
  }
}

/* eslint-disable agentverse/no-sideways-capability-imports */
import { appFetch } from '@/shell/app-fetch';

export async function validateGoogle(apiKey: string): Promise<{ ok: boolean; error?: string; models?: string[] }> {
  try {
    const res = await appFetch(`https://generativelanguage.googleapis.com/v1beta/models?key=${apiKey}`, {
      method: 'GET',
    });

    if (!res.ok) {
      const errText = await res.text();
      return { ok: false, error: `Google validation failed (${res.status}): ${errText}` };
    }

    const data = (await res.json()) as { models?: { name: string }[] };
    const models = (data.models || []).map((m) => m.name.replace('models/', ''));
    return { ok: true, models };
  } catch (err: unknown) {
    const errMsg = err instanceof Error ? err.message : String(err);
    return { ok: false, error: `Google connection failed: ${errMsg}` };
  }
}

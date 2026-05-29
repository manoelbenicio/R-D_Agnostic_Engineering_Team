/* eslint-disable agentverse/no-sideways-capability-imports */
import { appFetch } from '@/shell/app-fetch';

export async function validateMoonshot(apiKey: string): Promise<{ ok: boolean; error?: string; models?: string[] }> {
  try {
    const res = await appFetch('https://api.moonshot.cn/v1/models', {
      method: 'GET',
      headers: {
        Authorization: `Bearer ${apiKey}`,
      },
    });

    if (!res.ok) {
      const errText = await res.text();
      return { ok: false, error: `Moonshot validation failed (${res.status}): ${errText}` };
    }

    const data = (await res.json()) as { data?: { id: string }[] };
    const models = (data.data || []).map((m) => m.id);
    return { ok: true, models };
  } catch (err: unknown) {
    const errMsg = err instanceof Error ? err.message : String(err);
    return { ok: false, error: `Moonshot connection failed: ${errMsg}` };
  }
}

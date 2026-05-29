import cronstrue from 'cronstrue';

export type SchedulePreset = 'every-n-minutes' | 'hourly' | 'daily-at-time' | 'weekdays-at-time' | 'weekly';

export interface ScheduleDraft {
  preset: SchedulePreset;
  everyMinutes: number;
  hour: string;
  minute: string;
  weekday: string;
}

export const DEFAULT_SCHEDULE_DRAFT: ScheduleDraft = {
  preset: 'every-n-minutes',
  everyMinutes: 15,
  hour: '09',
  minute: '00',
  weekday: '1',
};

export function scheduleDraftToCron(draft: ScheduleDraft): string {
  const minute = clampNumber(Number(draft.minute), 0, 59);
  const hour = clampNumber(Number(draft.hour), 0, 23);

  switch (draft.preset) {
    case 'hourly':
      return `${minute} * * * *`;
    case 'daily-at-time':
      return `${minute} ${hour} * * *`;
    case 'weekdays-at-time':
      return `${minute} ${hour} * * 1-5`;
    case 'weekly':
      return `${minute} ${hour} * * ${draft.weekday}`;
    case 'every-n-minutes':
    default:
      return `*/${clampNumber(draft.everyMinutes, 1, 59)} * * * *`;
  }
}

export function describeCron(schedule: string): string {
  return cronstrue.toString(schedule, { throwExceptionOnParseError: true });
}

export function validateCron(schedule: string): { ok: true; description: string } | { ok: false; error: string } {
  try {
    return { ok: true, description: describeCron(schedule) };
  } catch (error) {
    return { ok: false, error: error instanceof Error ? error.message : String(error) };
  }
}

function clampNumber(value: number, min: number, max: number): number {
  if (!Number.isFinite(value)) return min;
  return Math.min(max, Math.max(min, Math.trunc(value)));
}

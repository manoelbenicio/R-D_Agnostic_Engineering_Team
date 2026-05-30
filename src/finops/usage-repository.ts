/**
 * src/finops/usage-repository.ts
 *
 * Persistence for FinOps Tier 2 token-usage events (IDB `usage_events`
 * store). Thin async CRUD used by the cost hooks; kept separate from the
 * pure parsing/costing modules so those stay trivially testable.
 */

import { dbGetAll, dbPut, openDb } from '@/shared/storage/idb';
import type { TokenUsage, UsageEvent } from './token-usage';

function makeId(): string {
  return globalThis.crypto?.randomUUID
    ? globalThis.crypto.randomUUID()
    : `usage-${Date.now()}-${Math.random().toString(36).slice(2)}`;
}

/** Persist a parsed usage record, attaching id + timestamp + context. */
export async function recordUsage(
  usage: TokenUsage,
  context: { sessionName?: string; terminalId?: string; canvasId?: string } = {},
): Promise<UsageEvent> {
  const event: UsageEvent = {
    ...usage,
    id: makeId(),
    timestampMs: Date.now(),
    ...context,
  };
  await dbPut('usage_events', event);
  return event;
}

/** All persisted usage events (newest first). */
export async function listUsageEvents(): Promise<UsageEvent[]> {
  const events = (await dbGetAll('usage_events')) as UsageEvent[];
  return events.sort((a, b) => b.timestampMs - a.timestampMs);
}

/** Usage events for a single canvas, via the `by-canvas` index. */
export async function listUsageEventsByCanvas(canvasId: string): Promise<UsageEvent[]> {
  const db = await openDb();
  const events = (await db.getAllFromIndex('usage_events', 'by-canvas', canvasId)) as UsageEvent[];
  return events.sort((a, b) => b.timestampMs - a.timestampMs);
}

/** Usage events within an inclusive `[startMs, endMs]` window. */
export async function listUsageEventsInWindow(startMs: number, endMs: number): Promise<UsageEvent[]> {
  const events = await listUsageEvents();
  return events.filter((e) => e.timestampMs >= startMs && e.timestampMs <= endMs);
}

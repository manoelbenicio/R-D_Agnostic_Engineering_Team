import { terminalSocketFanout } from '@/api/terminal-socket-fanout';
import type { TerminalSocketHandle } from '@/api/connect-terminal-socket';

interface FanoutEntrySnapshot {
  handle: TerminalSocketHandle;
}

interface FanoutWithEntries {
  entries?: Map<string, FanoutEntrySnapshot>;
}

export function getTerminalFanoutHandle(terminalId: string): TerminalSocketHandle | undefined {
  const fanout = terminalSocketFanout as unknown as FanoutWithEntries;
  return fanout.entries?.get(terminalId)?.handle;
}

export function sendTerminalFanoutText(terminalId: string, message: string): void {
  getTerminalFanoutHandle(terminalId)?.sendText(message);
}

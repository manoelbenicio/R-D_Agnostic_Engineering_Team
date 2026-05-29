import { describe, expect, it, vi } from 'vitest';
import { TerminalSocketFanout } from '@/api/terminal-socket-fanout';
import type { TerminalSocketHandlers, TerminalSocketHandle } from '@/api/connect-terminal-socket';

describe('TerminalSocketFanout', () => {
  it('shares one socket per terminal id and closes it after the last unsubscribe', () => {
    const handlers = new Map<string, TerminalSocketHandlers>();
    const close = vi.fn();
    const connect = vi.fn((terminalId: string, socketHandlers: TerminalSocketHandlers): TerminalSocketHandle => {
      handlers.set(terminalId, socketHandlers);
      return {
        socket: {} as WebSocket,
        sendText: vi.fn(),
        sendJson: vi.fn(),
        close,
      };
    });
    const fanout = new TerminalSocketFanout(connect);
    const firstBinary = vi.fn();
    const secondBinary = vi.fn();

    const first = fanout.subscribe('term-supervisor', { onBinary: firstBinary });
    const second = fanout.subscribe('term-supervisor', { onBinary: secondBinary });

    expect(connect).toHaveBeenCalledTimes(1);
    expect(fanout.getConnectionCount()).toBe(1);

    const frame = new ArrayBuffer(4);
    handlers.get('term-supervisor')?.onBinary(frame);

    expect(firstBinary).toHaveBeenCalledWith(frame);
    expect(secondBinary).toHaveBeenCalledWith(frame);

    first.unsubscribe();
    expect(close).not.toHaveBeenCalled();
    expect(fanout.getConnectionCount()).toBe(1);

    second.unsubscribe();
    expect(close).toHaveBeenCalledWith(1000, 'last subscriber unsubscribed');
    expect(fanout.getConnectionCount()).toBe(0);
  });

  it('keeps different terminal ids on different sockets', () => {
    const connect = vi.fn((_terminalId: string, _socketHandlers: TerminalSocketHandlers): TerminalSocketHandle => {
      return {
        socket: {} as WebSocket,
        sendText: vi.fn(),
        sendJson: vi.fn(),
        close: vi.fn(),
      };
    });
    const fanout = new TerminalSocketFanout(connect);

    fanout.subscribe('term-supervisor', { onBinary: vi.fn() });
    fanout.subscribe('term-developer', { onBinary: vi.fn() });

    expect(connect).toHaveBeenCalledTimes(2);
    expect(fanout.getConnectionCount()).toBe(2);
  });
});

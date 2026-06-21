import { afterEach, describe, expect, it, vi } from 'vitest';
import {
  buildTerminalSocketUrl,
  connectTerminalSocket,
  type TerminalSocketClose,
} from '@/api/connect-terminal-socket';

class FakeWebSocket extends EventTarget {
  static instances: FakeWebSocket[] = [];
  binaryType: BinaryType = 'blob';
  readonly sent: string[] = [];
  closeCode?: number;
  closeReason?: string;

  constructor(readonly url: string) {
    super();
    FakeWebSocket.instances.push(this);
  }

  send(data: string | ArrayBufferLike | Blob | ArrayBufferView): void {
    this.sent.push(typeof data === 'string' ? data : '[binary]');
  }

  close(code?: number, reason?: string): void {
    this.closeCode = code;
    this.closeReason = reason;
  }
}

afterEach(() => {
  vi.unstubAllGlobals();
  FakeWebSocket.instances = [];
});

describe('connectTerminalSocket', () => {
  it('builds ws and wss URLs from the GO Core base URL', () => {
    expect(buildTerminalSocketUrl('abcd1234', 'http://127.0.0.1:8080')).toBe(
      'ws://127.0.0.1:8080/terminals/abcd1234/ws'
    );
    expect(buildTerminalSocketUrl('term/with slash', 'https://gocore.example.test/base')).toBe(
      'wss://gocore.example.test/terminals/term%2Fwith%20slash/ws'
    );
  });

  it('sets binaryType before returning the socket handle', () => {
    vi.stubGlobal('WebSocket', FakeWebSocket);

    const handle = connectTerminalSocket('abcd1234', { onBinary: vi.fn() }, 'http://127.0.0.1:8080');

    expect(handle.socket).toBe(FakeWebSocket.instances[0]);
    expect(FakeWebSocket.instances[0]?.url).toBe('ws://127.0.0.1:8080/terminals/abcd1234/ws');
    expect(FakeWebSocket.instances[0]?.binaryType).toBe('arraybuffer');
  });

  it('surfaces 4003 and 4004 close codes as typed reasons', () => {
    vi.stubGlobal('WebSocket', FakeWebSocket);
    const onClose = vi.fn<(reason: TerminalSocketClose) => void>();

    connectTerminalSocket('term-supervisor', { onBinary: vi.fn(), onClose }, 'http://127.0.0.1:8080');
    const socket = FakeWebSocket.instances[0];
    socket?.dispatchEvent(new CloseEvent('close', { code: 4003, reason: 'blocked' }));
    socket?.dispatchEvent(new CloseEvent('close', { code: 4004, reason: 'missing' }));

    expect(onClose.mock.calls[0]?.[0]).toMatchObject({ type: 'ip_not_allowed', code: 4003 });
    expect(onClose.mock.calls[1]?.[0]).toMatchObject({ type: 'terminal_not_found', code: 4004 });
  });
});

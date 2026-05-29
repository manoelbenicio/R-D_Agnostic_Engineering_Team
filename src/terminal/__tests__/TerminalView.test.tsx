import { act, render, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import type { TerminalSocketClose } from '@/api/connect-terminal-socket';
import { TerminalView } from '../TerminalView';

const xtermMock = vi.hoisted(() => {
  const instances: MockTerminal[] = [];
  const loadOrder: string[][] = [];
  const fitAddons: MockFitAddon[] = [];
  let webglShouldThrow = false;

  class MockTerminal {
    readonly options: Record<string, unknown>;
    cols: number;
    rows: number;
    readonly loadOrder: string[] = [];
    readonly dataHandlers: Array<(data: string) => void> = [];
    readonly open = vi.fn();
    readonly write = vi.fn();
    readonly resize = vi.fn((cols: number, rows: number) => {
      this.cols = cols;
      this.rows = rows;
    });
    readonly dispose = vi.fn();

    constructor(options: Record<string, unknown>) {
      this.options = options;
      this.cols = Number(options.cols);
      this.rows = Number(options.rows);
      instances.push(this);
      loadOrder.push(this.loadOrder);
    }

    loadAddon(addon: { activate?: (terminal: MockTerminal) => void; constructor: { name: string } }): void {
      this.loadOrder.push(addon.constructor.name);
      addon.activate?.(this);
    }

    onData(handler: (data: string) => void) {
      this.dataHandlers.push(handler);
      return { dispose: vi.fn() };
    }

    emitData(data: string): void {
      for (const handler of this.dataHandlers) {
        handler(data);
      }
    }
  }

  class MockWebglAddon {
    activate(): void {
      if (webglShouldThrow) {
        throw new Error('no webgl');
      }
    }
    dispose(): void {}
  }

  class MockFitAddon {
    terminal?: MockTerminal;
    readonly fit = vi.fn(() => {
      if (!this.terminal) return;
      this.terminal.cols = 120;
      this.terminal.rows = 34;
    });
    constructor() {
      fitAddons.push(this);
    }
    activate(terminal: MockTerminal): void {
      this.terminal = terminal;
    }
    dispose(): void {}
  }

  class MockWebLinksAddon {
    activate(): void {}
    dispose(): void {}
  }

  class MockSearchAddon {
    activate(): void {}
    dispose(): void {}
  }

  class MockUnicode11Addon {
    activate(): void {}
    dispose(): void {}
  }

  function reset(): void {
    instances.length = 0;
    loadOrder.length = 0;
    fitAddons.length = 0;
    webglShouldThrow = false;
  }

  return {
    MockTerminal,
    MockWebglAddon,
    MockFitAddon,
    MockWebLinksAddon,
    MockSearchAddon,
    MockUnicode11Addon,
    instances,
    loadOrder,
    fitAddons,
    reset,
    setWebglShouldThrow: (value: boolean) => {
      webglShouldThrow = value;
    },
  };
});

const fanoutMock = vi.hoisted(() => {
  type Subscriber = {
    onBinary: (frame: ArrayBuffer) => void;
    onClose?: (reason: TerminalSocketClose) => void;
    onError?: (event: Event) => void;
  };

  class MockSocket {
    readyState = 0;
    private readonly listeners = new Map<string, Set<EventListener>>();

    addEventListener(type: string, listener: EventListener): void {
      const existing = this.listeners.get(type) ?? new Set<EventListener>();
      existing.add(listener);
      this.listeners.set(type, existing);
    }

    removeEventListener(type: string, listener: EventListener): void {
      this.listeners.get(type)?.delete(listener);
    }

    emit(type: string): void {
      if (type === 'open') this.readyState = 1;
      for (const listener of this.listeners.get(type) ?? []) {
        listener(new Event(type));
      }
    }
  }

  const entries = new Map<string, { handle: { socket: MockSocket; sendText: ReturnType<typeof vi.fn>; close: ReturnType<typeof vi.fn> } }>();
  const subscribers = new Map<string, Subscriber>();
  const sendTextByTerminal = new Map<string, Array<ReturnType<typeof vi.fn>>>();
  const socketsByTerminal = new Map<string, MockSocket[]>();

  const subscribeTerminalSocket = vi.fn((terminalId: string, subscriber: Subscriber) => {
    const socket = new MockSocket();
    const sendText = vi.fn();
    const close = vi.fn();
    subscribers.set(terminalId, subscriber);
    entries.set(terminalId, { handle: { socket, sendText, close } });
    sendTextByTerminal.set(terminalId, [...(sendTextByTerminal.get(terminalId) ?? []), sendText]);
    socketsByTerminal.set(terminalId, [...(socketsByTerminal.get(terminalId) ?? []), socket]);

    return {
      unsubscribe: vi.fn(() => {
        entries.delete(terminalId);
        subscribers.delete(terminalId);
      }),
    };
  });

  function reset(): void {
    entries.clear();
    subscribers.clear();
    sendTextByTerminal.clear();
    socketsByTerminal.clear();
    subscribeTerminalSocket.mockClear();
  }

  function latestSendText(terminalId: string) {
    const senders = sendTextByTerminal.get(terminalId) ?? [];
    return senders[senders.length - 1];
  }

  function latestSocket(terminalId: string) {
    const sockets = socketsByTerminal.get(terminalId) ?? [];
    return sockets[sockets.length - 1];
  }

  return {
    entries,
    subscribers,
    subscribeTerminalSocket,
    reset,
    latestSendText,
    latestSocket,
  };
});

const toastMock = vi.hoisted(() => ({
  error: vi.fn(),
}));

class ResizeObserverMock {
  static instances: ResizeObserverMock[] = [];
  readonly observe = vi.fn();
  readonly disconnect = vi.fn();

  constructor(private readonly callback: ResizeObserverCallback) {
    ResizeObserverMock.instances.push(this);
  }

  trigger(): void {
    this.callback([], this as unknown as ResizeObserver);
  }
}

vi.mock('@xterm/xterm', () => ({ Terminal: xtermMock.MockTerminal }));
vi.mock('@xterm/addon-webgl', () => ({ WebglAddon: xtermMock.MockWebglAddon }));
vi.mock('@xterm/addon-fit', () => ({ FitAddon: xtermMock.MockFitAddon }));
vi.mock('@xterm/addon-web-links', () => ({ WebLinksAddon: xtermMock.MockWebLinksAddon }));
vi.mock('@xterm/addon-search', () => ({ SearchAddon: xtermMock.MockSearchAddon }));
vi.mock('@xterm/addon-unicode11', () => ({ Unicode11Addon: xtermMock.MockUnicode11Addon }));
vi.mock('@/api/terminal-socket-fanout', () => ({
  terminalSocketFanout: {
    entries: fanoutMock.entries,
    getConnectionCount: () => fanoutMock.entries.size,
    reset: () => fanoutMock.reset(),
  },
  subscribeTerminalSocket: fanoutMock.subscribeTerminalSocket,
}));
vi.mock('@/shell/toasts', () => ({
  useToast: () => toastMock,
}));

beforeEach(() => {
  vi.stubGlobal('ResizeObserver', ResizeObserverMock);
  ResizeObserverMock.instances = [];
  xtermMock.reset();
  fanoutMock.reset();
  toastMock.error.mockClear();
});

afterEach(() => {
  vi.useRealTimers();
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
});

describe('TerminalView', () => {
  it('loads xterm addons in WebGL, Fit, WebLinks, Search, Unicode11 order and applies zero-lag options', async () => {
    render(<TerminalView terminalId="term-order" />);

    await waitFor(() => expect(xtermMock.instances).toHaveLength(1));
    expect(xtermMock.loadOrder[0]).toEqual([
      'MockWebglAddon',
      'MockFitAddon',
      'MockWebLinksAddon',
      'MockSearchAddon',
      'MockUnicode11Addon',
    ]);
    expect(xtermMock.instances[0]?.options).toMatchObject({
      smoothScrollDuration: 0,
      scrollback: 10000,
      fontFamily: 'var(--font-mono)',
      fontSize: 14,
      cursorBlink: true,
      allowTransparency: true,
      convertEol: false,
      windowsMode: false,
      cols: 220,
      rows: 50,
      theme: expect.objectContaining({
        // DSS Universal Standard v3.0 — Indra-aligned theme.
        // Values are resolved from CSS custom properties at runtime; in jsdom
        // (no tokens.css applied) the resolver falls back to Indra hex literals.
        background: '#002B3A',
        foreground: '#FFFFFF',
        cursor: '#00B0BD',
        selectionBackground: 'rgba(0, 176, 189, 0.25)',
      }),
    });
  });

  it('writes binary frames as the same Uint8Array reference without string conversion', async () => {
    render(<TerminalView terminalId="term-binary" />);
    await waitFor(() => expect(fanoutMock.subscribers.get('term-binary')).toBeDefined());

    const frame = new Uint8Array([65, 66, 67]);
    act(() => {
      fanoutMock.subscribers.get('term-binary')?.onBinary(frame as unknown as ArrayBuffer);
    });

    expect(xtermMock.instances[0]?.write).toHaveBeenCalledWith(frame);
    expect(typeof xtermMock.instances[0]?.write.mock.calls[0]?.[0]).not.toBe('string');
  });

  it('sends input as JSON text frames and suppresses input when readOnly', async () => {
    render(
      <>
        <TerminalView terminalId="term-input" />
        <TerminalView terminalId="term-readonly" readOnly />
      </>
    );
    await waitFor(() => expect(xtermMock.instances).toHaveLength(2));

    fanoutMock.latestSendText('term-input')?.mockClear();
    fanoutMock.latestSendText('term-readonly')?.mockClear();

    act(() => {
      xtermMock.instances[0]?.emitData('ls -la\n');
      xtermMock.instances[1]?.emitData('ignored');
    });

    expect(fanoutMock.latestSendText('term-input')).toHaveBeenCalledWith(
      JSON.stringify({ type: 'input', data: 'ls -la\n' })
    );
    expect(fanoutMock.latestSendText('term-readonly')).not.toHaveBeenCalled();
  });

  it('debounces resize frames and sends fitted terminal dimensions', async () => {
    render(<TerminalView terminalId="term-resize" />);
    await waitFor(() => expect(ResizeObserverMock.instances).toHaveLength(1));
    vi.useFakeTimers();
    fanoutMock.latestSendText('term-resize')?.mockClear();

    act(() => {
      ResizeObserverMock.instances[0]?.trigger();
      ResizeObserverMock.instances[0]?.trigger();
      vi.advanceTimersByTime(99);
    });
    expect(fanoutMock.latestSendText('term-resize')).not.toHaveBeenCalled();

    act(() => {
      vi.advanceTimersByTime(1);
    });

    expect(xtermMock.fitAddons[0]?.fit).toHaveBeenCalledTimes(1);
    expect(fanoutMock.latestSendText('term-resize')).toHaveBeenCalledWith(
      JSON.stringify({ type: 'resize', rows: 34, cols: 120 })
    );
  });

  it('reconnects with exponential backoff and jitter after non-permanent close codes', async () => {
    vi.spyOn(Math, 'random').mockReturnValue(0.5);
    render(<TerminalView terminalId="term-reconnect" />);
    await waitFor(() => expect(fanoutMock.subscribeTerminalSocket).toHaveBeenCalledTimes(1));
    vi.useFakeTimers();

    act(() => {
      fanoutMock.subscribers.get('term-reconnect')?.onClose?.(closedReason(1006));
    });
    expect(screen.getByText('reconnecting')).toBeInTheDocument();

    act(() => {
      vi.advanceTimersByTime(499);
    });
    expect(fanoutMock.subscribeTerminalSocket).toHaveBeenCalledTimes(1);

    act(() => {
      vi.advanceTimersByTime(1);
    });
    expect(fanoutMock.subscribeTerminalSocket).toHaveBeenCalledTimes(2);

    act(() => {
      fanoutMock.subscribers.get('term-reconnect')?.onClose?.(closedReason(1011));
      vi.advanceTimersByTime(999);
    });
    expect(fanoutMock.subscribeTerminalSocket).toHaveBeenCalledTimes(2);

    act(() => {
      vi.advanceTimersByTime(1);
    });
    expect(fanoutMock.subscribeTerminalSocket).toHaveBeenCalledTimes(3);
  });

  it('does not reconnect after permanent 4004 close and surfaces a toast', async () => {
    render(<TerminalView terminalId="term-missing" />);
    await waitFor(() => expect(fanoutMock.subscribeTerminalSocket).toHaveBeenCalledTimes(1));
    vi.useFakeTimers();

    act(() => {
      fanoutMock.subscribers.get('term-missing')?.onClose?.({
        type: 'terminal_not_found',
        code: 4004,
        error: new Error('missing') as never,
        event: new CloseEvent('close', { code: 4004 }),
      });
      vi.advanceTimersByTime(30_000);
    });

    expect(screen.getByText('terminated')).toBeInTheDocument();
    expect(toastMock.error).toHaveBeenCalledWith('Terminal not found');
    expect(fanoutMock.subscribeTerminalSocket).toHaveBeenCalledTimes(1);
  });

  it('keeps two side-by-side terminal instances isolated', async () => {
    render(
      <>
        <TerminalView terminalId="term-left" themeOverride={{ background: '#101010' }} />
        <TerminalView terminalId="term-right" />
      </>
    );
    await waitFor(() => expect(xtermMock.instances).toHaveLength(2));

    const leftFrame = new Uint8Array([1]);
    const rightFrame = new Uint8Array([2]);
    act(() => {
      fanoutMock.subscribers.get('term-left')?.onBinary(leftFrame as unknown as ArrayBuffer);
      fanoutMock.subscribers.get('term-right')?.onBinary(rightFrame as unknown as ArrayBuffer);
      xtermMock.instances[0]?.emitData('left');
    });

    expect(xtermMock.instances[0]).not.toBe(xtermMock.instances[1]);
    expect(xtermMock.instances[0]?.write).toHaveBeenCalledWith(leftFrame);
    expect(xtermMock.instances[0]?.write).not.toHaveBeenCalledWith(rightFrame);
    expect(xtermMock.instances[1]?.write).toHaveBeenCalledWith(rightFrame);
    expect(fanoutMock.latestSendText('term-left')).toHaveBeenCalledWith(
      JSON.stringify({ type: 'input', data: 'left' })
    );
    expect(fanoutMock.latestSendText('term-right')).not.toHaveBeenCalledWith(
      JSON.stringify({ type: 'input', data: 'left' })
    );
    expect(xtermMock.instances[0]?.options.theme).toMatchObject({ background: '#101010' });
    expect(xtermMock.instances[1]?.options.theme).toMatchObject({ background: '#002B3A' });
  });

  it('renders the WebGL-required card when WebGL cannot initialize', async () => {
    xtermMock.setWebglShouldThrow(true);

    render(<TerminalView terminalId="term-webgl-error" />);

    expect(await screen.findByRole('alert')).toHaveTextContent('WebGL is required to render terminals');
    expect(fanoutMock.subscribeTerminalSocket).not.toHaveBeenCalled();
  });
});

function closedReason(code: number): TerminalSocketClose {
  return {
    type: 'closed',
    code,
    reason: 'closed',
    event: new CloseEvent('close', { code }),
  };
}

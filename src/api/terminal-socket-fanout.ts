import { connectTerminalSocket, type TerminalSocketClose, type TerminalSocketHandle } from './connect-terminal-socket';

export interface TerminalSocketSubscriber {
  onBinary: (frame: ArrayBuffer) => void;
  onClose?: (reason: TerminalSocketClose) => void;
  onError?: (event: Event) => void;
}

export interface TerminalSocketSubscription {
  unsubscribe: () => void;
}

type ConnectTerminalSocket = typeof connectTerminalSocket;

interface FanoutEntry {
  handle: TerminalSocketHandle;
  subscribers: Set<TerminalSocketSubscriber>;
}

export class TerminalSocketFanout {
  private readonly entries = new Map<string, FanoutEntry>();
  private readonly connect: ConnectTerminalSocket;

  constructor(connect: ConnectTerminalSocket = connectTerminalSocket) {
    this.connect = connect;
  }

  subscribe(terminalId: string, subscriber: TerminalSocketSubscriber): TerminalSocketSubscription {
    const entry = this.getOrCreateEntry(terminalId);
    entry.subscribers.add(subscriber);

    return {
      unsubscribe: () => {
        const current = this.entries.get(terminalId);
        if (!current) return;
        current.subscribers.delete(subscriber);
        if (current.subscribers.size === 0) {
          current.handle.close(1000, 'last subscriber unsubscribed');
          this.entries.delete(terminalId);
        }
      },
    };
  }

  getConnectionCount(): number {
    return this.entries.size;
  }

  reset(): void {
    for (const entry of this.entries.values()) {
      entry.handle.close(1000, 'fanout reset');
    }
    this.entries.clear();
  }

  private getOrCreateEntry(terminalId: string): FanoutEntry {
    const existing = this.entries.get(terminalId);
    if (existing) return existing;

    const subscribers = new Set<TerminalSocketSubscriber>();
    const handle = this.connect(terminalId, {
      onBinary: (frame) => {
        for (const subscriber of subscribers) {
          subscriber.onBinary(frame);
        }
      },
      onClose: (reason) => {
        this.entries.delete(terminalId);
        for (const subscriber of subscribers) {
          subscriber.onClose?.(reason);
        }
      },
      onError: (event) => {
        for (const subscriber of subscribers) {
          subscriber.onError?.(event);
        }
      },
    });

    const entry = { handle, subscribers };
    this.entries.set(terminalId, entry);
    return entry;
  }
}

export const terminalSocketFanout = new TerminalSocketFanout();

export function subscribeTerminalSocket(
  terminalId: string,
  subscriber: TerminalSocketSubscriber
): TerminalSocketSubscription {
  return terminalSocketFanout.subscribe(terminalId, subscriber);
}

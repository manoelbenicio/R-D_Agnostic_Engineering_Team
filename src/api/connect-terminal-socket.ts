import { GO_CORE_BASE_URL } from './go-core-base-url';
import { IpNotAllowed, TerminalNotFound } from './errors';

export type TerminalSocketClose =
  | { type: 'ip_not_allowed'; code: 4003; error: IpNotAllowed; event: CloseEvent }
  | { type: 'terminal_not_found'; code: 4004; error: TerminalNotFound; event: CloseEvent }
  | { type: 'closed'; code: number; reason: string; event: CloseEvent };

export interface TerminalSocketHandlers {
  onBinary: (frame: ArrayBuffer) => void;
  onOpen?: (event: Event) => void;
  onClose?: (reason: TerminalSocketClose) => void;
  onError?: (event: Event) => void;
}

export interface TerminalSocketHandle {
  socket: WebSocket;
  sendText: (message: string) => void;
  sendJson: (message: unknown) => void;
  close: (code?: number, reason?: string) => void;
}

export function buildTerminalSocketUrl(terminalId: string, baseUrl = GO_CORE_BASE_URL): string {
  const url = new URL(baseUrl);
  url.protocol = url.protocol === 'https:' ? 'wss:' : 'ws:';
  url.pathname = `/terminals/${encodeURIComponent(terminalId)}/ws`;
  url.search = '';
  url.hash = '';
  return url.toString();
}

export function connectTerminalSocket(
  terminalId: string,
  handlers: TerminalSocketHandlers,
  baseUrl = GO_CORE_BASE_URL
): TerminalSocketHandle {
  const socket = new WebSocket(buildTerminalSocketUrl(terminalId, baseUrl));
  socket.binaryType = 'arraybuffer';

  socket.addEventListener('open', (event) => handlers.onOpen?.(event));
  socket.addEventListener('message', (event: MessageEvent) => {
    if (event.data instanceof ArrayBuffer) {
      handlers.onBinary(event.data);
      return;
    }
    if (event.data instanceof Blob) {
      event.data.arrayBuffer().then((buffer) => handlers.onBinary(buffer));
    }
  });
  socket.addEventListener('close', (event) => {
    handlers.onClose?.(toTerminalSocketClose(event));
  });
  socket.addEventListener('error', (event) => handlers.onError?.(event));

  return {
    socket,
    sendText: (message: string) => socket.send(message),
    sendJson: (message: unknown) => socket.send(JSON.stringify(message)),
    close: (code?: number, reason?: string) => socket.close(code, reason),
  };
}

function toTerminalSocketClose(event: CloseEvent): TerminalSocketClose {
  if (event.code === 4003) {
    return {
      type: 'ip_not_allowed',
      code: 4003,
      error: new IpNotAllowed(event.reason || undefined, { cause: event }),
      event,
    };
  }
  if (event.code === 4004) {
    return {
      type: 'terminal_not_found',
      code: 4004,
      error: new TerminalNotFound(event.reason || undefined, { cause: event }),
      event,
    };
  }
  return {
    type: 'closed',
    code: event.code,
    reason: event.reason,
    event,
  };
}

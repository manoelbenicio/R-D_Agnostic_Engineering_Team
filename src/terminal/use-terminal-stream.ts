import { useEffect, useRef, useState } from 'react';
import {
  Terminal,
  type IDisposable,
  type ITheme,
  type ITerminalInitOnlyOptions,
  type ITerminalOptions,
} from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import { WebglAddon } from '@xterm/addon-webgl';
import { WebLinksAddon } from '@xterm/addon-web-links';
import { SearchAddon } from '@xterm/addon-search';
import { Unicode11Addon } from '@xterm/addon-unicode11';
import { subscribeTerminalSocket, type TerminalSocketSubscription } from '@/api/terminal-socket-fanout';
import type { TerminalSocketClose } from '@/api/connect-terminal-socket';
// eslint-disable-next-line agentverse/no-sideways-capability-imports
import { useToast } from '@/shell/toasts';
import { TerminalWebglRequiredError, type TerminalConnectionState } from './connection-state';
import { getTerminalFanoutHandle, sendTerminalFanoutText } from './fanout-transport';
import { createTerminalTheme } from './xterm-theme';

export const INITIAL_TERMINAL_COLS = 220;
export const INITIAL_TERMINAL_ROWS = 50;
const RESIZE_DEBOUNCE_MS = 100;
const INITIAL_RECONNECT_DELAY_MS = 500;
const MAX_RECONNECT_DELAY_MS = 30_000;
const JITTER_RATIO = 0.2;

interface UseTerminalStreamOptions {
  terminalId: string;
  hostRef: React.RefObject<HTMLDivElement>;
  themeOverride?: Partial<ITheme>;
  readOnly?: boolean;
}

interface UseTerminalStreamResult {
  connectionState: TerminalConnectionState;
  webglError?: TerminalWebglRequiredError;
}

interface SocketEventTarget {
  readyState?: number;
  addEventListener?: (type: string, listener: EventListener) => void;
  removeEventListener?: (type: string, listener: EventListener) => void;
}

export function useTerminalStream({
  terminalId,
  hostRef,
  themeOverride,
  readOnly = false,
}: UseTerminalStreamOptions): UseTerminalStreamResult {
  const [connectionState, setConnectionState] = useState<TerminalConnectionState>('connecting');
  const [webglError, setWebglError] = useState<TerminalWebglRequiredError | undefined>();
  const toast = useToast();
  const terminalRef = useRef<Terminal | null>(null);
  const fitAddonRef = useRef<FitAddon | null>(null);
  const subscriptionRef = useRef<TerminalSocketSubscription | null>(null);
  const dataDisposableRef = useRef<IDisposable | null>(null);
  const resizeObserverRef = useRef<ResizeObserver | null>(null);
  const resizeTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const reconnectTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const reconnectAttemptRef = useRef(0);
  const mountedRef = useRef(false);
  const socketOpenListenerRef = useRef<EventListener | null>(null);
  const socketRef = useRef<SocketEventTarget | null>(null);
  const latestReadOnlyRef = useRef(readOnly);
  const latestTerminalIdRef = useRef(terminalId);

  latestReadOnlyRef.current = readOnly;
  latestTerminalIdRef.current = terminalId;

  useEffect(() => {
    mountedRef.current = true;
    const host = hostRef.current;
    if (!host) return undefined;

    setConnectionState('connecting');
    setWebglError(undefined);

    let terminal: Terminal;
    let fitAddon: FitAddon;

    function initTerminal(withWebgl: boolean): { term: Terminal; fit: FitAddon } {
      const term = new Terminal(createTerminalOptions(themeOverride));
      term.options.allowProposedApi = true;
      const fit = new FitAddon();

      const webLinksAddon = new WebLinksAddon();
      const searchAddon = new SearchAddon();
      const unicode11Addon = new Unicode11Addon();

      const useWebgl = withWebgl && !shouldDisableWebgl();

      if (useWebgl) {
        try {
          const webglAddon = new WebglAddon();
          term.loadAddon(webglAddon);
        } catch (cause) {
          term.dispose();
          if (canFallbackToCanvas2D()) {
            console.warn('WebGL loadAddon failed; falling back to Canvas2D because VITE_ALLOW_CANVAS2D=true.', cause);
            return initTerminal(false);
          } else {
            throw new TerminalWebglRequiredError(cause);
          }
        }
      }

      term.loadAddon(fit);
      term.loadAddon(webLinksAddon);
      term.loadAddon(searchAddon);
      term.loadAddon(unicode11Addon);

      try {
        term.open(host!);
      } catch (cause) {
        term.dispose();
        if (useWebgl && canFallbackToCanvas2D()) {
          console.warn('WebGL open failed; falling back to Canvas2D because VITE_ALLOW_CANVAS2D=true.', cause);
          return initTerminal(false);
        } else {
          throw new TerminalWebglRequiredError(cause);
        }
      }

      return { term, fit };
    }

    try {
      const result = initTerminal(true);
      terminal = result.term;
      fitAddon = result.fit;

      terminalRef.current = terminal;
      fitAddonRef.current = fitAddon;
      terminal.resize(INITIAL_TERMINAL_COLS, INITIAL_TERMINAL_ROWS);

      dataDisposableRef.current = terminal.onData((data) => {
        if (latestReadOnlyRef.current) return;
        sendTerminalFanoutText(
          latestTerminalIdRef.current,
          JSON.stringify({ type: 'input', data })
        );
      });

      const resizeObserver = new ResizeObserver(() => {
        if (resizeTimerRef.current) clearTimeout(resizeTimerRef.current);
        resizeTimerRef.current = setTimeout(() => {
          fitAddon.fit();
          sendTerminalFanoutText(
            latestTerminalIdRef.current,
            JSON.stringify({ type: 'resize', rows: terminal.rows, cols: terminal.cols })
          );
        }, RESIZE_DEBOUNCE_MS);
      });
      resizeObserver.observe(host);
      resizeObserverRef.current = resizeObserver;

      connect();

      return () => {
        mountedRef.current = false;
        clearReconnectTimer();
        clearResizeTimer();
        detachSocketOpenListener();
        resizeObserverRef.current?.disconnect();
        subscriptionRef.current?.unsubscribe();
        dataDisposableRef.current?.dispose();
        terminal.dispose();
        terminalRef.current = null;
        fitAddonRef.current = null;
        subscriptionRef.current = null;
      };
    } catch (err) {
      if (err instanceof TerminalWebglRequiredError || (err as Error).name === 'TerminalWebglRequiredError') {
        setWebglError(err as TerminalWebglRequiredError);
        setConnectionState('terminated');
        return () => {
          mountedRef.current = false;
        };
      }
      console.error('TerminalStream useEffect threw an error:', err);
      setConnectionState('terminated');
      return () => {
        mountedRef.current = false;
        if (terminalRef.current) {
          terminalRef.current.dispose();
          terminalRef.current = null;
        }
      };
    }
  // Terminal lifecycle is intentionally owned by this mount effect; helper
  // functions below read refs so reconnect timers do not recreate xterm.
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [hostRef, terminalId, themeOverride]);

  function connect(): void {
    if (!mountedRef.current || webglError) return;
    subscriptionRef.current?.unsubscribe();
    detachSocketOpenListener();
    setConnectionState(reconnectAttemptRef.current > 0 ? 'reconnecting' : 'connecting');
    subscriptionRef.current = subscribeTerminalSocket(terminalId, {
      onBinary: (frame) => {
        const terminal = terminalRef.current;
        if (!terminal) return;
        writeBinaryFrame(terminal, frame);
      },
      onClose: handleClose,
      onError: () => {
        scheduleReconnect();
      },
    });
    attachSocketOpenListener();
    sendInitialDimensions();
  }

  function handleClose(reason: TerminalSocketClose): void {
    if (reason.type === 'ip_not_allowed') {
      setConnectionState('terminated');
      toast.error('IP not allowed');
      return;
    }
    if (reason.type === 'terminal_not_found') {
      setConnectionState('terminated');
      toast.error('Terminal not found');
      return;
    }
    scheduleReconnect();
  }

  function scheduleReconnect(): void {
    if (!mountedRef.current || connectionState === 'terminated') return;
    setConnectionState('reconnecting');
    subscriptionRef.current?.unsubscribe();
    subscriptionRef.current = null;
    detachSocketOpenListener();
    clearReconnectTimer();
    const delay = getReconnectDelay(reconnectAttemptRef.current);
    reconnectAttemptRef.current += 1;
    reconnectTimerRef.current = setTimeout(() => {
      connect();
    }, delay);
  }

  function attachSocketOpenListener(): void {
    const handle = getTerminalFanoutHandle(terminalId);
    const socket = handle?.socket as SocketEventTarget | undefined;
    if (!socket) return;
    const onOpen: EventListener = () => {
      reconnectAttemptRef.current = 0;
      setConnectionState('connected');
      sendInitialDimensions();
    };
    socketRef.current = socket;
    socketOpenListenerRef.current = onOpen;
    socket.addEventListener?.('open', onOpen);
    if (socket.readyState === 1) {
      onOpen(new Event('open'));
    }
  }

  function detachSocketOpenListener(): void {
    if (socketRef.current && socketOpenListenerRef.current) {
      socketRef.current.removeEventListener?.('open', socketOpenListenerRef.current);
    }
    socketRef.current = null;
    socketOpenListenerRef.current = null;
  }

  function sendInitialDimensions(): void {
    sendTerminalFanoutText(
      terminalId,
      JSON.stringify({ type: 'resize', rows: INITIAL_TERMINAL_ROWS, cols: INITIAL_TERMINAL_COLS })
    );
  }

  function clearReconnectTimer(): void {
    if (reconnectTimerRef.current) {
      clearTimeout(reconnectTimerRef.current);
      reconnectTimerRef.current = null;
    }
  }

  function clearResizeTimer(): void {
    if (resizeTimerRef.current) {
      clearTimeout(resizeTimerRef.current);
      resizeTimerRef.current = null;
    }
  }

  return { connectionState, webglError };
}

export function writeBinaryFrame(terminal: Pick<Terminal, 'write'>, frame: ArrayBuffer | Uint8Array): void {
  const buffer = frame instanceof Uint8Array ? frame : new Uint8Array(frame);
  terminal.write(buffer);
}

export function getReconnectDelay(attempt: number): number {
  const baseDelay = Math.min(INITIAL_RECONNECT_DELAY_MS * 2 ** attempt, MAX_RECONNECT_DELAY_MS);
  const jitter = 1 - JITTER_RATIO + Math.random() * JITTER_RATIO * 2;
  return Math.round(baseDelay * jitter);
}

function createTerminalOptions(themeOverride?: Partial<ITheme>): ITerminalOptions & ITerminalInitOnlyOptions {
  return {
    allowProposedApi: true,
    smoothScrollDuration: 0,
    scrollback: 10000,
    fontFamily: 'var(--font-mono)',
    fontSize: 14,
    cursorBlink: !prefersReducedMotion(),
    allowTransparency: true,
    convertEol: false,
    windowsMode: false,
    cols: INITIAL_TERMINAL_COLS,
    rows: INITIAL_TERMINAL_ROWS,
    theme: createTerminalTheme(themeOverride),
  };
}


function shouldDisableWebgl(): boolean {
  return (
    typeof navigator !== 'undefined' &&
    (!!navigator.webdriver || /HeadlessChrome/.test(navigator.userAgent)) &&
    canFallbackToCanvas2D()
  );
}

function canFallbackToCanvas2D(): boolean {
  return (
    !import.meta.env.PROD &&
    (import.meta.env.VITE_ALLOW_CANVAS2D === 'true' ||
      (typeof navigator !== 'undefined' && (!!navigator.webdriver || /HeadlessChrome/.test(navigator.userAgent))))
  );
}

function prefersReducedMotion(): boolean {
  return typeof window !== 'undefined' && window.matchMedia('(prefers-reduced-motion: reduce)').matches;
}

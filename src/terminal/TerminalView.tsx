import React, { useRef } from 'react';
import type { ITheme } from '@xterm/xterm';
import '@xterm/xterm/css/xterm.css';
import { Badge, Card } from '@/design-system';
import { useTerminalStream } from './use-terminal-stream';
import type { TerminalConnectionState } from './connection-state';
import './terminal.css';

export interface TerminalViewProps {
  terminalId: string;
  themeOverride?: Partial<ITheme>;
  readOnly?: boolean;
}

export const TerminalView: React.FC<TerminalViewProps> = ({
  terminalId,
  themeOverride,
  readOnly = false,
}) => {
  const hostRef = useRef<HTMLDivElement>(null);
  const { connectionState, webglError } = useTerminalStream({
    terminalId,
    hostRef,
    themeOverride,
    readOnly,
  });

  if (webglError) {
    return (
      <Card glow="red" className="terminal-error-card" role="alert">
        <h2>WebGL is required to render terminals</h2>
        <p>{webglError.message}</p>
      </Card>
    );
  }

  return (
    <section className="terminal-view" data-terminal-id={terminalId}>
      <div className="terminal-view-toolbar">
        <ConnectionStatePill state={connectionState} />
      </div>
      <div
        ref={hostRef}
        className="terminal-host"
        data-testid={`terminal-host-${terminalId}`}
        aria-label={`Terminal ${terminalId}`}
      />
    </section>
  );
};

function ConnectionStatePill({ state }: { state: TerminalConnectionState }) {
  return (
    <Badge variant={connectionStateVariant(state)} className="terminal-connection-pill">
      {state}
    </Badge>
  );
}

function connectionStateVariant(
  state: TerminalConnectionState
): 'idle' | 'processing' | 'completed' | 'waiting_user_answer' | 'error' {
  switch (state) {
    case 'connected':
      return 'completed';
    case 'connecting':
      return 'processing';
    case 'reconnecting':
      return 'waiting_user_answer';
    case 'terminated':
      return 'error';
  }
}

export default TerminalView;

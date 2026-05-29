export type TerminalConnectionState = 'connecting' | 'connected' | 'reconnecting' | 'terminated';

export class TerminalWebglRequiredError extends Error {
  override readonly name = 'TerminalWebglRequiredError';

  constructor(cause?: unknown) {
    super('WebGL is required to render terminals.', { cause });
  }
}

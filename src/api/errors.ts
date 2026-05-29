export class CaoApiError extends Error {
  override readonly name = 'CaoApiError';
  readonly status: number;
  readonly endpoint: string;
  readonly body: unknown;

  constructor(message: string, options: { status: number; endpoint: string; body: unknown; cause?: unknown }) {
    super(message, { cause: options.cause });
    this.status = options.status;
    this.endpoint = options.endpoint;
    this.body = options.body;
  }
}

export class CaoNetworkError extends Error {
  override readonly name = 'CaoNetworkError';
  readonly endpoint: string;

  constructor(message: string, options: { endpoint: string; cause?: unknown }) {
    super(message, { cause: options.cause });
    this.endpoint = options.endpoint;
  }
}

export class IpNotAllowed extends Error {
  override readonly name = 'IpNotAllowed';
  readonly code = 4003;

  constructor(message = 'IP address is not allowed to connect to this terminal.', options?: { cause?: unknown }) {
    super(message, options);
  }
}

export class TerminalNotFound extends Error {
  override readonly name = 'TerminalNotFound';
  readonly code = 4004;

  constructor(message = 'Terminal was not found.', options?: { cause?: unknown }) {
    super(message, options);
  }
}

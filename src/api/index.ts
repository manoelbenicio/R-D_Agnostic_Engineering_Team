// CRIT-003.4: GO Core primary exports
export { GO_CORE_BASE_URL } from './go-core-base-url';
export { GoCoreClient, goCoreClient } from './go-core-client';
export { buildTerminalSocketUrl, connectTerminalSocket } from './connect-terminal-socket';
export type { TerminalSocketClose, TerminalSocketHandlers, TerminalSocketHandle } from './connect-terminal-socket';
export { GoCoreApiError, GoCoreNetworkError, IpNotAllowed, TerminalNotFound } from './errors';
export { useHealthStore } from './health-store';
export type { HealthState, HealthStatus } from './health-store';
export { goCoreQueryKeys, sessionsQueryKeys, terminalQueryKeys } from './query-keys';
export { subscribeTerminalSocket, TerminalSocketFanout, terminalSocketFanout } from './terminal-socket-fanout';
export type { TerminalSocketSubscriber, TerminalSocketSubscription } from './terminal-socket-fanout';
export type * from './types';

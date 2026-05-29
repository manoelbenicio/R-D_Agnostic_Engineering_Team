export const caoQueryKeys = {
  all: ['cao'] as const,
  health: () => ['cao', 'health'] as const,
  profiles: () => ['cao', 'profiles'] as const,
  profile: (name: string) => ['cao', 'profile', name] as const,
  providers: () => ['cao', 'providers'] as const,
  sessions: () => ['cao', 'sessions'] as const,
  session: (name: string) => ['cao', 'session', name] as const,
  sessionTerminals: (sessionName: string) => ['cao', 'sessions', sessionName, 'terminals'] as const,
  terminal: (id: string) => ['cao', 'terminal', id] as const,
  terminalOutput: (id: string, mode: 'full' | 'tail' | 'visible') =>
    ['cao', 'terminal', id, 'output', mode] as const,
  terminalWorkingDirectory: (id: string) => ['cao', 'terminal', id, 'working-directory'] as const,
  terminalMemoryContext: (id: string) => ['cao', 'terminal', id, 'memory-context'] as const,
  inboxMessages: (terminalId: string, filters: { limit?: number; status?: string } = {}) =>
    ['cao', 'terminal', terminalId, 'inbox', filters] as const,
  flows: () => ['cao', 'flows'] as const,
  flow: (name: string) => ['cao', 'flow', name] as const,
  agentDirs: () => ['cao', 'settings', 'agent-dirs'] as const,
  skill: (name: string) => ['cao', 'skill', name] as const,
};

export const sessionsQueryKeys = {
  all: ['cao', 'sessions'] as const,
  list: () => caoQueryKeys.sessions(),
  detail: (name: string) => caoQueryKeys.session(name),
  terminals: (sessionName: string) => caoQueryKeys.sessionTerminals(sessionName),
};

export const terminalQueryKeys = {
  all: ['cao', 'terminal'] as const,
  detail: (id: string) => caoQueryKeys.terminal(id),
  output: (id: string, mode: 'full' | 'tail' | 'visible') => caoQueryKeys.terminalOutput(id, mode),
  workingDirectory: (id: string) => caoQueryKeys.terminalWorkingDirectory(id),
  memoryContext: (id: string) => caoQueryKeys.terminalMemoryContext(id),
  inboxMessages: (id: string, filters: { limit?: number; status?: string } = {}) =>
    caoQueryKeys.inboxMessages(id, filters),
};

export const goCoreQueryKeys = {
  all: ['goCore'] as const,
  health: () => ['goCore', 'health'] as const,
  profiles: () => ['goCore', 'profiles'] as const,
  profile: (name: string) => ['goCore', 'profile', name] as const,
  providers: () => ['goCore', 'providers'] as const,
  sessions: () => ['goCore', 'sessions'] as const,
  session: (name: string) => ['goCore', 'session', name] as const,
  sessionTerminals: (sessionName: string) => ['goCore', 'sessions', sessionName, 'terminals'] as const,
  terminal: (id: string) => ['goCore', 'terminal', id] as const,
  terminalOutput: (id: string, mode: 'full' | 'tail' | 'visible') =>
    ['goCore', 'terminal', id, 'output', mode] as const,
  terminalWorkingDirectory: (id: string) => ['goCore', 'terminal', id, 'working-directory'] as const,
  terminalMemoryContext: (id: string) => ['goCore', 'terminal', id, 'memory-context'] as const,
  inboxMessages: (terminalId: string, filters: { limit?: number; status?: string } = {}) =>
    ['goCore', 'terminal', terminalId, 'inbox', filters] as const,
  flows: () => ['goCore', 'flows'] as const,
  flow: (name: string) => ['goCore', 'flow', name] as const,
  agentDirs: () => ['goCore', 'settings', 'agent-dirs'] as const,
  skill: (name: string) => ['goCore', 'skill', name] as const,
};

export const sessionsQueryKeys = {
  all: ['goCore', 'sessions'] as const,
  list: () => goCoreQueryKeys.sessions(),
  detail: (name: string) => goCoreQueryKeys.session(name),
  terminals: (sessionName: string) => goCoreQueryKeys.sessionTerminals(sessionName),
};

export const terminalQueryKeys = {
  all: ['goCore', 'terminal'] as const,
  detail: (id: string) => goCoreQueryKeys.terminal(id),
  output: (id: string, mode: 'full' | 'tail' | 'visible') => goCoreQueryKeys.terminalOutput(id, mode),
  workingDirectory: (id: string) => goCoreQueryKeys.terminalWorkingDirectory(id),
  memoryContext: (id: string) => goCoreQueryKeys.terminalMemoryContext(id),
  inboxMessages: (id: string, filters: { limit?: number; status?: string } = {}) =>
    goCoreQueryKeys.inboxMessages(id, filters),
};

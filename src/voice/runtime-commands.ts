
export interface RuntimeCommand {
  action: 'kill' | 'pause' | 'focus' | 'status' | 'deploy' | 'stop_all' | 'cost' | 'add_node' | 'connect';
  target?: {
    type: 'role' | 'name' | 'id';
    value: string;
  };
  source?: {
    type: 'role' | 'name' | 'id';
    value: string;
  };
  destination?: {
    type: 'role' | 'name' | 'id';
    value: string;
  };
  role?: string;
  provider?: string;
}

const stopAllRegex = /^(?:stop\s+all|stop\s+everything|shutdown\s+session|kill\s+session|kill\s+all|parar\s+tudo|parar\s+sess[aã]o|encerrar\s+tudo|matar\s+sess[aã]o|parar\s+tudo\s+o\s+que\s+est[aá]\s+rodando)$/i;

const deployRegex = /^(?:deploy|deploy\s+canvas|materialize|implantar|publicar|materializar|executar\s+deploy)$/i;

const costRegex = /^(?:cost|show\s+cost|check\s+cost|budget|finops|custo|mostrar\s+custo|verificar\s+custo|gasto)$/i;

const statusRegex = /^(?:status|show\s+status|check\s+status|terminal\s+status|verificar\s+status|mostrar\s+status|como\s+est[aã]o\s+as\s+coisas|como\s+est[aá]\s+o\s+status)$/i;

const killRegex = /^(?:kill\s+(?:the\s+)?|delete\s+terminal\s+(?:for\s+)?|stop\s+terminal\s+|terminate\s+|matar\s+(?:o\s+|a\s+)?|deletar\s+terminal\s+(?:do\s+|da\s+)?|parar\s+terminal\s+|excluir\s+terminal\s+|finalizar\s+)(.+)$/i;

const pauseRegex = /^(?:pause\s+(?:the\s+)?|pausar\s+(?:o\s+|a\s+)?|pausar\s+terminal\s+(?:do\s+|da\s+)?)(.+)$/i;

const focusRegex = /^(?:focus\s+on\s+|focus\s+|go\s+to\s+|show\s+|focar\s+em\s+|focar\s+no\s+|focar\s+na\s+|focar\s+|ir\s+para\s+o\s+|ir\s+para\s+a\s+|ir\s+para\s+|mostrar\s+)(.+)$/i;

const addNodeRegex = /^(?:add\s+node\s+|add\s+agent\s+|create\s+agent\s+|new\s+agent\s+|adicionar\s+n[oó]\s+|adicionar\s+agente\s+|criar\s+agente\s+|novo\s+agente\s+)(\w+)(?:\s+(?:on|no|na)\s+(\w+))?$/i;

const connectRegex1 = /^(?:connect|link|conectar|vincular|ligar)\s+(.+)\s+(?:to|a|com|ao|à|no|na)\s+(.+)$/i;
const connectRegex2 = /^(?:add\s+edge\s+from|handoff\s+from)\s+(.+)\s+to\s+(.+)$/i;

export function resolveTarget(word: string): { type: 'role' | 'name' | 'id'; value: string } {
  const clean = word.trim().replace(/^(o|a|the|do|da|for)\s+/i, '');
  const normalized = clean.toLowerCase();
  
  if (normalized === 'supervisor') {
    return { type: 'role', value: 'supervisor' };
  }
  if (normalized === 'developer' || normalized === 'desenvolvedor') {
    return { type: 'role', value: 'developer' };
  }
  if (normalized === 'reviewer' || normalized === 'revisor' || normalized === 'revisador') {
    return { type: 'role', value: 'reviewer' };
  }
  if (normalized === 'custom') {
    return { type: 'role', value: 'custom' };
  }
  
  const isUuid = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i.test(normalized);
  const isIdLike = /^[a-z0-9-_]{8,40}$/i.test(normalized) && !/\s/.test(normalized);
  
  if (isUuid || isIdLike) {
    return { type: 'id', value: clean };
  }
  
  return { type: 'name', value: clean };
}

function resolveRole(roleName: string): string {
  const norm = roleName.trim().toLowerCase();
  if (norm === 'desenvolvedor') return 'developer';
  if (norm === 'revisor' || norm === 'revisador') return 'reviewer';
  return norm;
}

function resolveProvider(providerName: string): string {
  const norm = providerName.trim().toLowerCase();
  if (norm === 'kiro') return 'kiro_cli';
  if (norm === 'claude') return 'claude_code';
  if (norm === 'codex') return 'codex';
  if (norm === 'gemini') return 'gemini_cli';
  if (norm === 'kimi') return 'kimi_cli';
  if (norm === 'copilot') return 'copilot_cli';
  if (norm === 'opencode') return 'opencode_cli';
  if (norm === 'q') return 'q_cli';
  return providerName;
}

export function matchRuntimeCommand(transcript: string): RuntimeCommand | null {
  const cleaned = transcript.trim().replace(/[.?!\s]+$/, '').replace(/\s+/g, ' ');
  if (!cleaned) return null;

  if (stopAllRegex.test(cleaned)) {
    return { action: 'stop_all' };
  }

  if (deployRegex.test(cleaned)) {
    return { action: 'deploy' };
  }

  if (costRegex.test(cleaned)) {
    return { action: 'cost' };
  }

  if (statusRegex.test(cleaned)) {
    return { action: 'status' };
  }

  let match = killRegex.exec(cleaned);
  if (match && match[1]) {
    return { action: 'kill', target: resolveTarget(match[1]) };
  }

  match = pauseRegex.exec(cleaned);
  if (match && match[1]) {
    return { action: 'pause', target: resolveTarget(match[1]) };
  }

  match = focusRegex.exec(cleaned);
  if (match && match[1]) {
    return { action: 'focus', target: resolveTarget(match[1]) };
  }

  match = addNodeRegex.exec(cleaned);
  if (match && match[1]) {
    return {
      action: 'add_node',
      role: resolveRole(match[1]),
      provider: match[2] ? resolveProvider(match[2]) : undefined,
    };
  }

  match = connectRegex1.exec(cleaned);
  if (match && match[1] && match[2]) {
    return {
      action: 'connect',
      source: resolveTarget(match[1]),
      destination: resolveTarget(match[2]),
    };
  }

  match = connectRegex2.exec(cleaned);
  if (match && match[1] && match[2]) {
    return {
      action: 'connect',
      source: resolveTarget(match[1]),
      destination: resolveTarget(match[2]),
    };
  }

  return null;
}

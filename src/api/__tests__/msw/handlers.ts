import { http, HttpResponse, ws } from 'msw';
import type { AgentProfile, Flow, InboxMessage, ProviderAvailability, Session, Terminal } from '@/api/types';

const now = '2026-05-27T21:40:00.000Z';

const profiles = new Map<string, AgentProfile>([
  [
    'supervisor',
    {
      name: 'supervisor',
      role: 'Supervisor',
      provider: 'openai',
      description: 'Coordinates worker agents and owns handoffs.',
      markdown: '---\nname: supervisor\nrole: Supervisor\nprovider: openai\n---\nCoordinate the team.',
      allowed_tools: ['handoff', 'send_message'],
    },
  ],
  [
    'developer',
    {
      name: 'developer',
      role: 'Developer',
      provider: 'anthropic',
      description: 'Implements tasks assigned by the supervisor.',
      markdown: '---\nname: developer\nrole: Developer\nprovider: anthropic\n---\nBuild the requested change.',
      allowed_tools: ['shell', 'apply_patch'],
    },
  ],
]);

const providers: ProviderAvailability[] = [
  { name: 'openai', installed: true },
  { name: 'anthropic', installed: true },
  { name: 'google', installed: false },
  { name: 'aws', installed: false },
  { name: 'azure', installed: false },
  { name: 'moonshot', installed: false },
  { name: 'copilot_cli', installed: true },
  { name: 'opencode_cli', installed: true },
];

const terminals = new Map<string, Terminal>([
  [
    'term-supervisor',
    {
      id: 'term-supervisor',
      session_name: 'demo-session',
      profile: 'supervisor',
      provider: 'openai',
      display_name: 'Supervisor',
      status: 'idle',
      working_directory: 'C:/workspace/agentverse',
      created_at: now,
      updated_at: now,
    },
  ],
  [
    'term-developer',
    {
      id: 'term-developer',
      session_name: 'demo-session',
      profile: 'developer',
      provider: 'anthropic',
      display_name: 'Developer',
      status: 'processing',
      working_directory: 'C:/workspace/agentverse',
      created_at: now,
      updated_at: now,
    },
  ],
]);

const sessions = new Map<string, Session>([
  [
    'demo-session',
    {
      name: 'demo-session',
      profile: 'supervisor',
      working_directory: 'C:/workspace/agentverse',
      status: 'active',
      terminals: Array.from(terminals.values()),
      created_at: now,
      updated_at: now,
    },
  ],
]);

const inboxMessages = new Map<string, InboxMessage[]>([
  [
    'term-supervisor',
    [
      {
        id: 'msg-1',
        terminal_id: 'term-supervisor',
        message: 'Developer finished the first patch.',
        status: 'unread',
        sender: 'developer',
        created_at: now,
      },
    ],
  ],
]);

const flows = new Map<string, Flow>([
  [
    'nightly-review',
    {
      name: 'nightly-review',
      schedule: '0 2 * * *',
      agent_profile: 'supervisor',
      provider: 'openai',
      prompt_template: 'Run the nightly review and summarize risks.',
      enabled: true,
      last_run: null,
      next_run: '2026-05-28T02:00:00.000Z',
      gating_script: null,
    },
  ],
]);

const goCoreWs = typeof ws !== 'undefined' ? ws.link('ws://127.0.0.1:8080/terminals/:id/ws') : null;
const goCoreWss = typeof ws !== 'undefined' ? ws.link('wss://127.0.0.1:8080/terminals/:id/ws') : null;
const goCoreWsLocal = typeof ws !== 'undefined' ? ws.link('ws://localhost:8080/terminals/:id/ws') : null;
const goCoreWssLocal = typeof ws !== 'undefined' ? ws.link('wss://localhost:8080/terminals/:id/ws') : null;

const handleConnection = ({ client }: { client: any }) => {
  const encoder = new TextEncoder();
  client.send(encoder.encode('\r\n[Mock GO Core Terminal Connected]\r\n').buffer);

  let frameCounter = 0;
  const interval = setInterval(() => {
    try {
      frameCounter++;
      client.send(encoder.encode(`\r[stream frame ${frameCounter}] `).buffer);
    } catch {
      clearInterval(interval);
    }
  }, 16); // 60Hz

  client.addEventListener('close', () => {
    clearInterval(interval);
  });
};

export const handlers = [
  http.get('*/health', () => HttpResponse.json({ status: 'ok' })),

  http.get('https://api.openai.com/v1/models', () => {
    return HttpResponse.json({
      data: [
        { id: 'gpt-4o' },
        { id: 'gpt-4o-mini' },
      ],
    });
  }),
  http.get('https://api.anthropic.com/v1/models', () => {
    return HttpResponse.json({
      data: [
        { id: 'claude-3-5-sonnet-latest' },
        { id: 'claude-3-haiku-20240307' },
      ],
    });
  }),
  http.get('https://generativelanguage.googleapis.com/v1beta/models', () => {
    return HttpResponse.json({
      models: [
        { name: 'models/gemini-1.5-pro-latest' },
        { name: 'models/gemini-1.5-flash-latest' },
      ],
    });
  }),

  http.get('*/agents/profiles', () => HttpResponse.json(Array.from(profiles.values()))),
  http.get('*/agents/profiles/:name', ({ params }) => {
    const profile = profiles.get(String(params.name));
    return profile
      ? HttpResponse.json(profile)
      : HttpResponse.json({ detail: 'Profile not found' }, { status: 404 });
  }),
  http.post('*/agents/profiles/install', async ({ request }) => {
    const markdown = await request.text();
    const name = parseFrontmatterValue(markdown, 'name') ?? `profile-${profiles.size + 1}`;
    const role = parseFrontmatterValue(markdown, 'role') ?? 'Custom';
    const provider = parseFrontmatterValue(markdown, 'provider') ?? 'openai';
    const profile: AgentProfile = {
      name,
      role,
      provider,
      description: `${role} profile installed from markdown.`,
      markdown,
    };
    profiles.set(name, profile);
    return HttpResponse.json(profile, { status: 201 });
  }),
  http.get('*/agents/providers', () => HttpResponse.json(providers)),

  http.post('*/sessions', async ({ request }) => {
    const input = (await request.json()) as { profile: string; working_directory: string };
    const sessionName = `session-${sessions.size + 1}`;
    const terminalId = `term-${terminals.size + 1}`;
    const terminal: Terminal = {
      id: terminalId,
      session_name: sessionName,
      profile: input.profile,
      provider: profiles.get(input.profile)?.provider ?? 'openai',
      display_name: input.profile,
      status: 'starting',
      working_directory: input.working_directory,
      created_at: now,
      updated_at: now,
    };
    terminals.set(terminalId, terminal);
    console.log('[DEBUG MSW POST sessions] sessionName:', sessionName, 'terminalId:', terminalId, 'terminal:', terminal, 'terminals size after set:', terminals.size);

    const session: Session = {
      name: sessionName,
      profile: input.profile,
      working_directory: input.working_directory,
      status: 'active',
      terminals: [terminal],
      created_at: now,
      updated_at: now,
    };
    sessions.set(session.name, session);
    return HttpResponse.json(session, { status: 201 });
  }),
  http.get('*/sessions', () => HttpResponse.json(Array.from(sessions.values()))),
  http.get('*/sessions/:name', ({ params }) => {
    const session = sessions.get(String(params.name));
    return session
      ? HttpResponse.json(withSessionTerminals(session))
      : HttpResponse.json({ detail: 'Session not found' }, { status: 404 });
  }),
  http.delete('*/sessions/:name', ({ params }) => {
    sessions.delete(String(params.name));
    return new HttpResponse(null, { status: 204 });
  }),
  http.post('*/sessions/:name/terminals', async ({ params, request }) => {
    const session = sessions.get(String(params.name));
    if (!session) return HttpResponse.json({ detail: 'Session not found' }, { status: 404 });
    const input = (await request.json()) as { profile: string; working_directory: string };
    const terminal: Terminal = {
      id: `term-${terminals.size + 1}`,
      session_name: session.name,
      profile: input.profile,
      provider: profiles.get(input.profile)?.provider ?? 'openai',
      display_name: input.profile,
      status: 'starting',
      working_directory: input.working_directory,
      created_at: now,
      updated_at: now,
    };
    terminals.set(terminal.id, terminal);
    return HttpResponse.json(terminal, { status: 201 });
  }),
  http.get('*/sessions/:name/terminals', ({ params }) => {
    const sessionName = String(params.name);
    console.log('[DEBUG MSW GET terminals] sessionName:', sessionName, 'sessions.has:', sessions.has(sessionName), 'all terminals:', Array.from(terminals.values()));
    if (!sessions.has(sessionName)) return HttpResponse.json({ detail: 'Session not found' }, { status: 404 });
    const filtered = Array.from(terminals.values()).filter((terminal) => terminal.session_name === sessionName);
    console.log('[DEBUG MSW GET terminals] filtered:', filtered);
    return HttpResponse.json(filtered);
  }),

  http.get('*/terminals/:id', ({ params }) => {
    const terminal = terminals.get(String(params.id));
    return terminal
      ? HttpResponse.json(terminal)
      : HttpResponse.json({ detail: 'Terminal not found' }, { status: 404 });
  }),
  http.get('*/terminals/:id/output', ({ params, request }) => {
    const url = new URL(request.url);
    const mode = url.searchParams.get('mode') ?? 'full';
    return HttpResponse.text(`[${params.id}] ${mode} output\nAgent ready.`);
  }),
  http.get('*/terminals/:id/working-directory', ({ params }) => {
    const terminal = terminals.get(String(params.id));
    return terminal
      ? HttpResponse.text(terminal.working_directory)
      : HttpResponse.json({ detail: 'Terminal not found' }, { status: 404 });
  }),
  http.get('*/terminals/:id/memory-context', ({ params }) => {
    return HttpResponse.text(`# Memory Context\n\nTerminal: ${params.id}\nScope: session`);
  }),
  http.post('*/terminals/:id/input', ({ params }) => {
    return terminals.has(String(params.id))
      ? new HttpResponse(null, { status: 204 })
      : HttpResponse.json({ detail: 'Terminal not found' }, { status: 404 });
  }),
  http.post('*/terminals/:id/exit', ({ params }) => {
    const terminal = terminals.get(String(params.id));
    if (!terminal) return HttpResponse.json({ detail: 'Terminal not found' }, { status: 404 });
    terminals.set(terminal.id, { ...terminal, status: 'exited', updated_at: now });
    return new HttpResponse(null, { status: 204 });
  }),
  http.delete('*/terminals/:id', ({ params }) => {
    terminals.delete(String(params.id));
    return new HttpResponse(null, { status: 204 });
  }),
  http.post('*/terminals/:id/inbox/messages', async ({ params, request }) => {
    const terminalId = String(params.id);
    if (!terminals.has(terminalId)) return HttpResponse.json({ detail: 'Terminal not found' }, { status: 404 });
    const input = (await request.json()) as { message: string };
    const message: InboxMessage = {
      id: `msg-${Date.now()}`,
      terminal_id: terminalId,
      message: input.message,
      status: 'unread',
      sender: 'user',
      created_at: now,
    };
    inboxMessages.set(terminalId, [message, ...(inboxMessages.get(terminalId) ?? [])]);
    return HttpResponse.json(message, { status: 201 });
  }),
  http.get('*/terminals/:id/inbox/messages', ({ params, request }) => {
    const url = new URL(request.url);
    const status = url.searchParams.get('status');
    const limit = Number(url.searchParams.get('limit') ?? Number.POSITIVE_INFINITY);
    const messages = inboxMessages.get(String(params.id)) ?? [];
    const filtered = status ? messages.filter((message) => message.status === status) : messages;
    return HttpResponse.json(filtered.slice(0, limit));
  }),

  http.get('*/flows', () => HttpResponse.json(Array.from(flows.values()))),
  http.get('*/flows/:name', ({ params }) => {
    const flow = flows.get(String(params.name));
    return flow ? HttpResponse.json(flow) : HttpResponse.json({ detail: 'Flow not found' }, { status: 404 });
  }),
  http.post('*/flows', async ({ request }) => {
    const flow = (await request.json()) as Flow;
    flows.set(flow.name, flow);
    return HttpResponse.json(flow, { status: 201 });
  }),
  http.delete('*/flows/:name', ({ params }) => {
    flows.delete(String(params.name));
    return new HttpResponse(null, { status: 204 });
  }),
  http.post('*/flows/:name/enable', ({ params }) => {
    return updateFlowEnabled(String(params.name), true);
  }),
  http.post('*/flows/:name/disable', ({ params }) => {
    return updateFlowEnabled(String(params.name), false);
  }),
  http.post('*/flows/:name/run', ({ params }) => {
    return flows.has(String(params.name))
      ? new HttpResponse(null, { status: 204 })
      : HttpResponse.json({ detail: 'Flow not found' }, { status: 404 });
  }),

  http.get('*/settings/agent-dirs', () => HttpResponse.json({ dirs: ['C:/Users/mbenicios/.codex/agents'] })),
  http.post('*/settings/agent-dirs', async ({ request }) => {
    const body = (await request.json()) as { dirs: string[] };
    return HttpResponse.json({ dirs: body.dirs });
  }),

  http.get('*/skills/:name', ({ params }) => {
    return HttpResponse.text(`# ${params.name}\n\nSkill body loaded from GO Core.`);
  }),

  ...(goCoreWs && goCoreWss && goCoreWsLocal && goCoreWssLocal ? [
    goCoreWs.addEventListener('connection', handleConnection),
    goCoreWss.addEventListener('connection', handleConnection),
    goCoreWsLocal.addEventListener('connection', handleConnection),
    goCoreWssLocal.addEventListener('connection', handleConnection),
  ] : []),
];



export default handlers;

function withSessionTerminals(session: Session): Session {
  return {
    ...session,
    terminals: Array.from(terminals.values()).filter((terminal) => terminal.session_name === session.name),
  };
}

function parseFrontmatterValue(markdown: string, key: string): string | undefined {
  const match = markdown.match(new RegExp(`^${key}:\\s*(.+)$`, 'm'));
  return match?.[1]?.trim();
}

function updateFlowEnabled(name: string, enabled: boolean) {
  const flow = flows.get(name);
  if (!flow) return HttpResponse.json({ detail: 'Flow not found' }, { status: 404 });
  flows.set(name, { ...flow, enabled });
  return new HttpResponse(null, { status: 204 });
}

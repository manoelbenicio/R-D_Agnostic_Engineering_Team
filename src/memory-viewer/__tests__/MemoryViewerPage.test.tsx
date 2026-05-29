import React from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { caoClient, type Session, type Terminal } from '@/api';
import { MemoryViewerPage } from '../MemoryViewerPage';

const sessions: Session[] = [
  {
    name: 'demo-session',
    profile: 'supervisor',
    working_directory: 'C:/workspace/agentverse',
    status: 'active',
  },
];

const terminals: Terminal[] = [
  {
    id: 'term-project',
    session_name: 'demo-session',
    profile: 'developer',
    provider: 'anthropic',
    display_name: 'Developer',
    status: 'idle',
    working_directory: 'C:/workspace/agentverse',
  },
  {
    id: 'term-global',
    session_name: 'demo-session',
    profile: 'supervisor',
    provider: 'openai',
    display_name: 'Supervisor',
    status: 'idle',
    working_directory: 'C:/workspace/agentverse',
  },
];

describe('MemoryViewerPage', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    vi.spyOn(caoClient, 'listSessions').mockResolvedValue(sessions);
    vi.spyOn(caoClient, 'listTerminalsInSession').mockResolvedValue(terminals);
    vi.spyOn(caoClient, 'getAgentDirs').mockResolvedValue({
      dirs: ['C:/Users/mbenicios/.codex/agents'],
    });
    vi.spyOn(caoClient, 'getTerminalMemoryContext').mockImplementation(async (id) => {
      if (id === 'term-project') {
        return [
          '---',
          'title: Deployment Runbook',
          'scope: project',
          'type: reference',
          'tags: deployment, runbook',
          'updatedAt: 2026-05-28T12:00:00.000Z',
          'location: memory/project/wiki/runbooks/deployment.md',
          '---',
          '# Deployment',
          'Use the release checklist.',
        ].join('\n');
      }

      return [
        '---',
        'title: User Feedback Notes',
        'scope: global',
        'type: feedback',
        'tags: research, customer',
        'updatedAt: 2026-05-28T13:00:00.000Z',
        'location: memory/global/wiki/feedback/customer.md',
        '---',
        '# Feedback',
        'Customers asked for clearer onboarding.',
      ].join('\n');
    });
  });

  it('narrows memories by scope, type, and tag filters', async () => {
    renderWithQueryClient(<MemoryViewerPage />);

    expect((await screen.findAllByText('Deployment Runbook')).length).toBeGreaterThan(0);
    expect(screen.getByText('User Feedback Notes')).toBeInTheDocument();

    fireEvent.change(screen.getByLabelText('Scope', { selector: '#memory-scope-filter' }), {
      target: { value: 'project' },
    });
    fireEvent.change(screen.getByLabelText('Type', { selector: '#memory-type-filter' }), {
      target: { value: 'reference' },
    });
    fireEvent.change(screen.getByLabelText('Tag'), { target: { value: 'deployment' } });

    expect(screen.getAllByText('Deployment Runbook').length).toBeGreaterThan(0);
    expect(screen.queryByText('User Feedback Notes')).not.toBeInTheDocument();
  });

  it('matches search case-insensitively across content and tags', async () => {
    renderWithQueryClient(<MemoryViewerPage />);

    expect((await screen.findAllByText('Deployment Runbook')).length).toBeGreaterThan(0);
    fireEvent.change(screen.getByLabelText('Search'), { target: { value: 'CUSTOMER' } });

    await waitFor(() => expect(screen.queryByText('Deployment Runbook')).not.toBeInTheDocument());
    expect(screen.getAllByText('User Feedback Notes').length).toBeGreaterThan(0);
  });

  it('shows the v1 limitation empty-state copy when no memory entries are visible', async () => {
    vi.spyOn(caoClient, 'listSessions').mockResolvedValue([]);
    vi.spyOn(caoClient, 'listTerminalsInSession').mockResolvedValue([]);
    vi.spyOn(caoClient, 'getTerminalMemoryContext').mockResolvedValue('');

    renderWithQueryClient(<MemoryViewerPage />);

    expect(await screen.findByText('No memories visible in this view (v1 limitation)')).toBeInTheDocument();
  });
});

function renderWithQueryClient(ui: React.ReactElement) {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
    },
  });

  return render(<QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>);
}

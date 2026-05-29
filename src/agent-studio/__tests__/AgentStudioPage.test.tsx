import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen, waitFor, within, fireEvent } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { http, HttpResponse } from 'msw';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { server } from '@/api/__tests__/msw/server';
import { useKeyStore } from '@/api/key-store/store';
import { AgentStudioPage } from '../AgentStudioPage';

vi.mock('@monaco-editor/react', () => ({
  default: ({ value, onChange }: { value: string; onChange: (value: string) => void }) => (
    <textarea
      aria-label="Markdown editor"
      value={value}
      onChange={(event) => onChange(event.target.value)}
    />
  ),
}));

function renderStudio() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });

  return render(
    <QueryClientProvider client={queryClient}>
      <AgentStudioPage />
    </QueryClientProvider>
  );
}

beforeEach(() => {
  useKeyStore.setState({
    validated: [],
    initialized: true,
  });
});

describe('AgentStudioPage', () => {
  it('lists profiles, filters by search, and flags an uninstalled provider', async () => {
    server.use(
      http.get('*/agents/profiles', () =>
        HttpResponse.json([
          {
            name: 'code-reviewer',
            role: 'Reviewer',
            provider: 'kimi_cli',
            description: 'Reviews implementation changes.',
            markdown:
              '---\nname: code-reviewer\nrole: Reviewer\nprovider: kimi_cli\n---\n# Review Agent\n\n- inspect changes\n',
          },
          {
            name: 'builder',
            role: 'Developer',
            provider: 'codex',
            description: 'Builds changes.',
            markdown: '---\nname: builder\nrole: Developer\nprovider: codex\n---\n# Builder\n',
          },
        ])
      ),
      http.get('*/agents/providers', () =>
        HttpResponse.json([
          { name: 'kimi_cli', installed: false },
          { name: 'codex', installed: true },
        ])
      )
    );

    renderStudio();

    await waitFor(() => {
      expect(screen.getAllByText('code-reviewer').length).toBeGreaterThan(0);
    });
    expect(screen.getAllByText('code-reviewer').length).toBeGreaterThan(0);
    expect(screen.getByText('builder')).toBeInTheDocument();
    expect(screen.getByTitle("Provider 'kimi_cli' not installed")).toBeInTheDocument();

    await userEvent.type(screen.getByLabelText('Search'), 'review');

    expect(screen.getAllByText('code-reviewer').length).toBeGreaterThan(0);
    expect(screen.queryByRole('button', { name: /builder/i })).not.toBeInTheDocument();
  });

  it('renders markdown detail as SENTINEL prose elements', async () => {
    server.use(
      http.get('*/agents/profiles', () =>
        HttpResponse.json([
          {
            name: 'doc-reviewer',
            role: 'Reviewer',
            provider: 'codex',
            markdown:
              '---\nname: doc-reviewer\nrole: Reviewer\nprovider: codex\nallowedTools: [read_file, grep]\n---\n# Review Agent\n\n- inspect changes\n- report risks\n\n```txt\nok\n```',
          },
        ])
      ),
      http.get('*/agents/providers', () => HttpResponse.json([{ name: 'codex', installed: true }]))
    );

    renderStudio();

    expect(await screen.findByRole('heading', { name: 'Review Agent' })).toBeInTheDocument();
    expect(screen.getByText('inspect changes')).toBeInTheDocument();
    expect(screen.getByText('read_file, grep')).toBeInTheDocument();
  });

  it('gates the editor provider dropdown to validated providers', async () => {
    useKeyStore.setState({
      validated: ['anthropic'],
      initialized: true,
    });

    renderStudio();
    fireEvent.click(screen.getByRole('button', { name: 'New Profile' }));

    const dialog = screen.getByRole('dialog');
    const providerSelect = within(dialog).getByLabelText(/Provider/) as HTMLSelectElement;
    const options = Array.from(providerSelect.options).map((option) => option.value);

    expect(options).toEqual(['', 'claude_code']);
  });

  it('installs a built-in profile through CAO and refreshes the list', async () => {
    renderStudio();

    const builtinButton = (await screen.findByText('SENTINEL Reviewer')).closest('button');
    expect(builtinButton).not.toBeNull();
    fireEvent.click(builtinButton as HTMLButtonElement);
    const dialog = await screen.findByRole('dialog');
    fireEvent.click(within(dialog).getByRole('button', { name: 'Confirm Install' }));

    await waitFor(() => {
      expect(screen.getAllByText('sentinel-reviewer').length).toBeGreaterThan(0);
    });
  });
});

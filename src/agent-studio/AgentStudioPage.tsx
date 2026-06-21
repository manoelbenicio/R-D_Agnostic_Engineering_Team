/* eslint-disable agentverse/no-sideways-capability-imports */
import React, { ChangeEvent, FormEvent, useEffect, useMemo, useState } from 'react';
import Editor from '@monaco-editor/react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { AgentProfile, goCoreClient, goCoreQueryKeys, ProviderAvailability, GoCoreApiError } from '@/api';
import { useValidatedProviders } from '@/api/key-store/use-validated-providers';
import { Badge, Button, Card, FormField, Prose, StatusBadge } from '@/design-system';
import { useToast } from '@/shell/toasts';
import { BUILTIN_PROFILES } from './builtin-profiles';
import { getAgentStudioProviderOptions } from './provider-options';
import {
  firstLineDescription,
  parseProfileMarkdown,
  ProfileFrontmatter,
  renderMarkdown,
  serializeProfileMarkdown,
} from './profile-markdown';
import './agent-studio.css';

type EditorMode = 'closed' | 'new' | 'edit' | 'source-preview';

const emptyFrontmatter: ProfileFrontmatter = {
  name: '',
  role: '',
  provider: '',
  model: '',
  allowedTools: [],
  mcpServers: [],
  permissionMode: '',
};

export const AgentStudioPage: React.FC = () => {
  const toast = useToast();
  const queryClient = useQueryClient();
  const validatedProviders = useValidatedProviders();
  const providerOptions = useMemo(
    () => getAgentStudioProviderOptions(validatedProviders),
    [validatedProviders]
  );

  const [search, setSearch] = useState('');
  const [providerFilter, setProviderFilter] = useState('');
  const [selectedName, setSelectedName] = useState<string | null>(null);
  const [editorMode, setEditorMode] = useState<EditorMode>('closed');
  const [frontmatter, setFrontmatter] = useState<ProfileFrontmatter>(emptyFrontmatter);
  const [body, setBody] = useState('# New Agent Profile\n\nDescribe the agent behavior here.');
  const [sourceUrl, setSourceUrl] = useState('');
  const [sourceError, setSourceError] = useState<string | null>(null);

  const profilesQuery = useQuery({
    queryKey: goCoreQueryKeys.profiles(),
    queryFn: () => goCoreClient.listProfiles(),
  });

  const providersQuery = useQuery({
    queryKey: goCoreQueryKeys.providers(),
    queryFn: () => goCoreClient.listProviders(),
  });

  const selectedProfile = useMemo(() => {
    if (!selectedName) return profilesQuery.data?.[0] ?? null;
    return profilesQuery.data?.find((profile) => profile.name === selectedName) ?? null;
  }, [profilesQuery.data, selectedName]);

  useEffect(() => {
    if (!selectedName && profilesQuery.data?.[0]) {
      setSelectedName(profilesQuery.data[0].name);
    }
  }, [profilesQuery.data, selectedName]);

  const providerAvailability = useMemo(
    () => new Map((providersQuery.data ?? []).map((provider) => [String(provider.name), provider])),
    [providersQuery.data]
  );

  const filteredProfiles = useMemo(() => {
    const term = search.trim().toLowerCase();
    return (profilesQuery.data ?? []).filter((profile) => {
      const matchesSearch =
        !term ||
        profile.name.toLowerCase().includes(term) ||
        profile.role.toLowerCase().includes(term);
      const matchesProvider = !providerFilter || profile.provider === providerFilter;
      return matchesSearch && matchesProvider;
    });
  }, [profilesQuery.data, providerFilter, search]);

  const installMutation = useMutation({
    mutationFn: (markdown: string) => goCoreClient.installProfile(markdown),
    onSuccess: async (profile) => {
      await queryClient.invalidateQueries({ queryKey: goCoreQueryKeys.profiles() });
      setSelectedName(profile.name);
      setEditorMode('closed');
      toast.success(`Installed profile ${profile.name}.`);
    },
    onError: (error) => {
      toast.error(formatGoCoreError(error));
    },
  });

  const openNewEditor = () => {
    setFrontmatter({
      ...emptyFrontmatter,
      provider: providerOptions[0]?.provider ?? '',
      role: 'Developer',
    });
    setBody('# New Agent Profile\n\nDescribe the agent behavior here.');
    setEditorMode('new');
  };

  const openEditEditor = (profile: AgentProfile) => {
    const parsed = parseProfileMarkdown(profile.markdown ?? profile.system_prompt ?? '');
    setFrontmatter({
      name: String(parsed.frontmatter.name ?? profile.name),
      role: String(parsed.frontmatter.role ?? profile.role),
      provider: String(parsed.frontmatter.provider ?? profile.provider),
      model: typeof parsed.frontmatter.model === 'string' ? parsed.frontmatter.model : '',
      allowedTools: arrayValue(parsed.frontmatter.allowedTools ?? profile.allowed_tools),
      mcpServers: arrayValue(parsed.frontmatter.mcpServers),
      permissionMode:
        typeof parsed.frontmatter.permissionMode === 'string' ? parsed.frontmatter.permissionMode : '',
    });
    setBody(parsed.body || profile.system_prompt || '');
    setEditorMode('edit');
  };

  const previewSource = (markdown: string) => {
    const parsed = parseProfileMarkdown(markdown);
    setFrontmatter({
      name: String(parsed.frontmatter.name ?? ''),
      role: String(parsed.frontmatter.role ?? ''),
      provider: String(parsed.frontmatter.provider ?? providerOptions[0]?.provider ?? ''),
      model: typeof parsed.frontmatter.model === 'string' ? parsed.frontmatter.model : '',
      allowedTools: arrayValue(parsed.frontmatter.allowedTools),
      mcpServers: arrayValue(parsed.frontmatter.mcpServers),
      permissionMode:
        typeof parsed.frontmatter.permissionMode === 'string' ? parsed.frontmatter.permissionMode : '',
    });
    setBody(parsed.body);
    setEditorMode('source-preview');
  };

  const handleSave = (event?: FormEvent) => {
    event?.preventDefault();
    const markdown = serializeProfileMarkdown(frontmatter, body);
    installMutation.mutate(markdown);
  };

  const handleFileSource = async (event: ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (!file) return;
    setSourceError(null);
    try {
      previewSource(await file.text());
    } catch (error) {
      setSourceError(error instanceof Error ? error.message : String(error));
    }
  };

  const handleUrlSource = async () => {
    if (!sourceUrl.trim()) return;
    setSourceError(null);
    try {
      const response = await fetch(sourceUrl);
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${await response.text()}`);
      }
      previewSource(await response.text());
    } catch (error) {
      setSourceError(error instanceof Error ? error.message : String(error));
    }
  };

  return (
    <main className="agent-studio-page">
      <header className="agent-studio-header">
        <div>
          <h1>Agent Studio</h1>
          <p>Manage agent profiles, provider availability, and installable markdown profiles.</p>
        </div>
        <div className="agent-studio-actions">
          <Button variant="secondary" onClick={openNewEditor}>
            New Profile
          </Button>
        </div>
      </header>

      <section className="agent-studio-layout">
        <div className="agent-studio-left">
          <ProviderAvailabilityPanel providers={providersQuery.data ?? []} isLoading={providersQuery.isLoading} />

          <Card className="agent-studio-panel">
            <div className="agent-studio-panel-title">Install From Source</div>
            <div className="agent-source-grid">
              {BUILTIN_PROFILES.map((profile) => (
                <button
                  key={profile.id}
                  className="agent-source-card"
                  type="button"
                  onClick={() => previewSource(profile.markdown)}
                >
                  <strong>{profile.name}</strong>
                  <span>{profile.description}</span>
                </button>
              ))}
            </div>
            <div className="agent-source-row">
              <label className="agent-file-button">
                Local Markdown
                <input type="file" accept=".md,text/markdown,text/plain" onChange={handleFileSource} />
              </label>
            </div>
            <div className="agent-source-row">
              <input
                aria-label="Profile URL"
                value={sourceUrl}
                onChange={(event) => setSourceUrl(event.target.value)}
                placeholder="https://example.com/profile.md"
              />
              <Button variant="secondary" onClick={() => void handleUrlSource()}>
                Preview URL
              </Button>
            </div>
            {sourceError ? <div className="agent-studio-error">{sourceError}</div> : null}
          </Card>
        </div>

        <Card className="agent-studio-list">
          <div className="agent-studio-list-controls">
            <FormField label="Search" id="agent-profile-search">
              <input
                value={search}
                onChange={(event) => setSearch(event.target.value)}
                placeholder="Search by name or role"
              />
            </FormField>
            <FormField label="Provider" id="agent-profile-provider-filter">
              <select value={providerFilter} onChange={(event) => setProviderFilter(event.target.value)}>
                <option value="">All providers</option>
                {Array.from(new Set((profilesQuery.data ?? []).map((profile) => profile.provider))).map(
                  (provider) => (
                    <option key={provider} value={provider}>
                      {provider}
                    </option>
                  )
                )}
              </select>
            </FormField>
          </div>

          {profilesQuery.error ? (
            <div className="agent-studio-error">{formatGoCoreError(profilesQuery.error)}</div>
          ) : profilesQuery.isLoading ? (
            <p className="agent-studio-muted">Loading profiles...</p>
          ) : filteredProfiles.length === 0 ? (
            <p className="agent-studio-muted">No profiles match the current filters.</p>
          ) : (
            <div className="agent-profile-list">
              {filteredProfiles.map((profile) => {
                const provider = providerAvailability.get(String(profile.provider));
                const isProviderMissing = provider?.installed === false;
                return (
                  <div
                    key={profile.name}
                    className={`agent-profile-row ${selectedProfile?.name === profile.name ? 'is-selected' : ''}`}
                    onClick={() => setSelectedName(profile.name)}
                  >
                    <div>
                      <div className="agent-profile-row-title">
                        <strong>{profile.name}</strong>
                        {isProviderMissing ? (
                          <span title={`Provider '${profile.provider}' not installed`} className="agent-warning">
                            {'\u26a0\ufe0f'}
                          </span>
                        ) : null}
                      </div>
                      <p>{profile.description ?? firstLineDescription(profile.markdown, profile.role)}</p>
                    </div>
                    <div className="agent-profile-row-meta">
                      <Badge>{profile.role}</Badge>
                      <Badge>{profile.provider}</Badge>
                      <Button
                        variant="secondary"
                        onClick={(event) => {
                          event.stopPropagation();
                          openEditEditor(profile);
                        }}
                      >
                        Edit
                      </Button>
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </Card>

        <ProfileDetail profile={selectedProfile} providerAvailability={providerAvailability} onEdit={openEditEditor} />
      </section>

      {editorMode !== 'closed' ? (
        <ProfileEditor
          mode={editorMode}
          frontmatter={frontmatter}
          body={body}
          providerOptions={providerOptions}
          isSaving={installMutation.isPending}
          error={installMutation.error ? formatGoCoreError(installMutation.error) : null}
          onFrontmatterChange={setFrontmatter}
          onBodyChange={setBody}
          onCancel={() => setEditorMode('closed')}
          onSave={handleSave}
        />
      ) : null}
    </main>
  );
};

function ProviderAvailabilityPanel({
  providers,
  isLoading,
}: {
  providers: ProviderAvailability[];
  isLoading: boolean;
}) {
  return (
    <Card className="agent-studio-panel">
      <div className="agent-studio-panel-title">Provider Availability</div>
      {isLoading ? (
        <p className="agent-studio-muted">Loading providers...</p>
      ) : (
        <div className="agent-provider-grid">
          {providers.map((provider) => (
            <div key={String(provider.name)} className="agent-provider-row">
              <span>{provider.name}</span>
              <StatusBadge
                status={provider.installed ? 'completed' : 'error'}
                label={provider.installed ? 'Installed' : 'Missing'}
              />
            </div>
          ))}
        </div>
      )}
    </Card>
  );
}

function ProfileDetail({
  profile,
  providerAvailability,
  onEdit,
}: {
  profile: AgentProfile | null;
  providerAvailability: Map<string, ProviderAvailability>;
  onEdit: (profile: AgentProfile) => void;
}) {
  if (!profile) {
    return (
      <Card className="agent-studio-detail">
        <p className="agent-studio-muted">Select a profile to inspect its markdown and metadata.</p>
      </Card>
    );
  }

  const parsed = parseProfileMarkdown(profile.markdown ?? profile.system_prompt ?? '');
  const provider = providerAvailability.get(String(profile.provider));
  const metadata = profile.metadata ?? {};

  return (
    <Card className="agent-studio-detail">
      <div className="agent-detail-header">
        <div>
          <h2>{profile.name}</h2>
          <p>{profile.description ?? firstLineDescription(profile.markdown, profile.role)}</p>
        </div>
        <Button variant="secondary" onClick={() => onEdit(profile)}>
          Edit
        </Button>
      </div>

      {provider?.installed === false ? (
        <div className="agent-studio-warning">
          {'\u26a0\ufe0f'} Provider &apos;{profile.provider}&apos; not installed
        </div>
      ) : null}

      <div className="agent-frontmatter-grid" aria-label="Profile frontmatter">
        {['role', 'provider', 'model', 'allowedTools', 'mcpServers', 'permissionMode'].map((key) => {
          const value = parsed.frontmatter[key] ?? fallbackFrontmatterValue(profile, key);
          return (
            <div key={key}>
              <span>{key}</span>
              <strong>{formatFrontmatterValue(value)}</strong>
            </div>
          );
        })}
      </div>

      <Prose className="agent-profile-prose">{renderMarkdown(parsed.body || profile.system_prompt || '')}</Prose>

      <div className="agent-metadata">
        {Object.entries(metadata).map(([key, value]) => (
          <span key={key}>
            {key}: {String(value)}
          </span>
        ))}
      </div>
    </Card>
  );
}

function ProfileEditor({
  mode,
  frontmatter,
  body,
  providerOptions,
  isSaving,
  error,
  onFrontmatterChange,
  onBodyChange,
  onCancel,
  onSave,
}: {
  mode: EditorMode;
  frontmatter: ProfileFrontmatter;
  body: string;
  providerOptions: Array<{ provider: string; label: string }>;
  isSaving: boolean;
  error: string | null;
  onFrontmatterChange: (frontmatter: ProfileFrontmatter) => void;
  onBodyChange: (body: string) => void;
  onCancel: () => void;
  onSave: (event?: FormEvent) => void;
}) {
  const patch = (patchValue: Partial<ProfileFrontmatter>) =>
    onFrontmatterChange({ ...frontmatter, ...patchValue });
  const providerPresent = providerOptions.some((option) => option.provider === frontmatter.provider);

  return (
    <div className="agent-editor-overlay" role="dialog" aria-modal="true">
      <Card className="agent-editor">
        <form onSubmit={onSave}>
          <div className="agent-detail-header">
            <div>
              <h2>{mode === 'source-preview' ? 'Preview Profile Source' : 'Profile Editor'}</h2>
              <p>Save installs the assembled markdown through GO Core.</p>
            </div>
            <Button type="button" variant="ghost" onClick={onCancel}>
              Close
            </Button>
          </div>

          <div className="agent-editor-grid">
            <FormField label="Name" id="profile-name" required>
              <input
                value={frontmatter.name}
                onChange={(event) => patch({ name: event.target.value })}
              />
            </FormField>
            <FormField label="Role" id="profile-role" required>
              <input
                value={frontmatter.role}
                onChange={(event) => patch({ role: event.target.value })}
              />
            </FormField>
            <FormField
              label="Provider"
              id="profile-provider"
              required
              helperText="Gated to validated API key providers."
            >
              <select
                value={frontmatter.provider}
                onChange={(event) => patch({ provider: event.target.value })}
              >
                <option value="">Select provider</option>
                {frontmatter.provider && !providerPresent ? (
                  <option value={frontmatter.provider}>{frontmatter.provider}</option>
                ) : null}
                {providerOptions.map((option) => (
                  <option key={option.provider} value={option.provider}>
                    {option.label}
                  </option>
                ))}
              </select>
            </FormField>
            <FormField label="Model" id="profile-model">
              <input
                value={frontmatter.model ?? ''}
                onChange={(event) => patch({ model: event.target.value })}
              />
            </FormField>
            <FormField label="Allowed tools" id="profile-allowed-tools">
              <input
                value={(frontmatter.allowedTools ?? []).join(', ')}
                onChange={(event) => patch({ allowedTools: splitList(event.target.value) })}
              />
            </FormField>
            <FormField label="MCP servers" id="profile-mcp-servers">
              <input
                value={(frontmatter.mcpServers ?? []).join(', ')}
                onChange={(event) => patch({ mcpServers: splitList(event.target.value) })}
              />
            </FormField>
            <FormField label="Permission mode" id="profile-permission-mode">
              <input
                value={frontmatter.permissionMode ?? ''}
                onChange={(event) => patch({ permissionMode: event.target.value })}
              />
            </FormField>
          </div>

          <div className="agent-monaco-shell">
            <Editor
              height="320px"
              defaultLanguage="markdown"
              theme="vs-dark"
              value={body}
              onChange={(value) => onBodyChange(value ?? '')}
              options={{
                minimap: { enabled: false },
                fontFamily: 'JetBrains Mono, monospace',
                fontSize: 13,
                wordWrap: 'on',
                scrollBeyondLastLine: false,
              }}
            />
          </div>

          {error ? <div className="agent-studio-error">{error}</div> : null}

          <div className="agent-editor-actions">
            <Button type="button" variant="secondary" onClick={onCancel}>
              Cancel
            </Button>
            <Button
              type="submit"
              variant="primary"
              disabled={isSaving || !frontmatter.name || !frontmatter.role || !frontmatter.provider}
            >
              {mode === 'source-preview' ? 'Confirm Install' : 'Save Profile'}
            </Button>
          </div>
        </form>
      </Card>
    </div>
  );
}

function fallbackFrontmatterValue(profile: AgentProfile, key: string): string | string[] | undefined {
  if (key === 'role') return profile.role;
  if (key === 'provider') return String(profile.provider);
  if (key === 'allowedTools') return profile.allowed_tools;
  if (key === 'model') return typeof profile.metadata?.model === 'string' ? profile.metadata.model : undefined;
  return undefined;
}

function formatFrontmatterValue(value: string | string[] | undefined): string {
  if (Array.isArray(value)) return value.join(', ');
  return value || '-';
}

function arrayValue(value: unknown): string[] {
  if (Array.isArray(value)) return value.map(String);
  if (typeof value === 'string' && value) return splitList(value);
  return [];
}

function splitList(value: string): string[] {
  return value
    .split(',')
    .map((item) => item.trim())
    .filter(Boolean);
}

function formatGoCoreError(error: unknown): string {
  if (error instanceof GoCoreApiError) {
    const body = typeof error.body === 'string' ? error.body : JSON.stringify(error.body);
    return `HTTP ${error.status}: ${body}`;
  }
  if (error instanceof Error) return error.message;
  return String(error);
}

export default AgentStudioPage;

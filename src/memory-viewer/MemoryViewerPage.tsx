import React, { FormEvent, useEffect, useMemo, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Badge, Button, Card, FormField, Prose } from '@/design-system';
import { goCoreClient } from '@/api';
import type { MemoryEntry, MemoryFormState, MemoryScope, MemoryType, MemoryViewerData } from './types';
import { memoryMatchesSearch, parseTerminalMemoryContext, renderMemoryMarkdown } from './memory-context';
import './memory-viewer.css';

const SCOPES: MemoryScope[] = ['global', 'project', 'session', 'agent'];
const TYPES: MemoryType[] = ['project', 'user', 'feedback', 'reference'];

const emptyForm: MemoryFormState = {
  title: '',
  scope: 'project',
  type: 'project',
  tags: '',
  content: '',
};

export const MemoryViewerPage: React.FC = () => {
  const [scopeFilter, setScopeFilter] = useState('');
  const [typeFilter, setTypeFilter] = useState('');
  const [tagFilter, setTagFilter] = useState('');
  const [search, setSearch] = useState('');
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [manualEntries, setManualEntries] = useState<MemoryEntry[]>([]);
  const [form, setForm] = useState<MemoryFormState>(emptyForm);
  const [errors, setErrors] = useState<Partial<Record<keyof MemoryFormState, string>>>({});
  const [creationNotice, setCreationNotice] = useState('');

  const memoryQuery = useQuery({
    queryKey: ['memory-viewer', 'entries'],
    queryFn: loadMemoryViewerData,
  });

  const allEntries = useMemo(
    () => [...manualEntries, ...(memoryQuery.data?.entries ?? [])],
    [manualEntries, memoryQuery.data?.entries]
  );

  const allTags = useMemo(() => {
    return Array.from(new Set(allEntries.flatMap((entry) => entry.tags))).sort((a, b) => a.localeCompare(b));
  }, [allEntries]);

  const filteredEntries = useMemo(() => {
    return allEntries.filter((entry) => {
      const matchesScope = !scopeFilter || entry.scope === scopeFilter;
      const matchesType = !typeFilter || entry.type === typeFilter;
      const matchesTag = !tagFilter || entry.tags.includes(tagFilter);
      const matchesSearch = memoryMatchesSearch(entry, search);
      return matchesScope && matchesType && matchesTag && matchesSearch;
    });
  }, [allEntries, scopeFilter, search, tagFilter, typeFilter]);

  const selectedEntry = useMemo(() => {
    return filteredEntries.find((entry) => entry.id === selectedId) ?? filteredEntries[0] ?? null;
  }, [filteredEntries, selectedId]);

  useEffect(() => {
    if (selectedEntry && selectedEntry.id !== selectedId) {
      setSelectedId(selectedEntry.id);
    }
  }, [selectedEntry, selectedId]);

  const submitMemory = (event: FormEvent) => {
    event.preventDefault();
    const nextErrors: Partial<Record<keyof MemoryFormState, string>> = {};
    if (!form.title.trim()) nextErrors.title = 'Title is required.';
    if (!form.content.trim()) nextErrors.content = 'Content is required.';
    setErrors(nextErrors);
    if (Object.keys(nextErrors).length > 0) return;

    const createdAt = new Date().toISOString();
    const entry: MemoryEntry = {
      id: `manual:${createdAt}:${form.title}`,
      title: form.title.trim(),
      scope: form.scope,
      type: form.type,
      tags: splitTags(form.tags),
      content: form.content.trim(),
      updatedAt: createdAt,
      retention: form.scope === 'session' ? 'Persists until session manual-memory ends' : undefined,
      locationPath: `memory/${form.scope}/wiki/manual/${slugify(form.title)}.md`,
      source: 'manual',
    };

    setManualEntries((current) => [entry, ...current]);
    setSelectedId(entry.id);
    setForm(emptyForm);
    setCreationNotice(
      'No direct memory write endpoint is exposed in v1; this entry is staged locally for authoring through agent memory tools.'
    );
  };

  return (
    <main className="memory-viewer-page">
      <header className="memory-viewer-header">
        <div>
          <h1>Memory Viewer</h1>
          <p>Inspect per-terminal runtime memory context, visible agent directories, and staged manual memories.</p>
        </div>
      </header>

      <section className="memory-viewer-layout">
        <Card className="memory-filter-panel">
          <div className="memory-panel-title">Filters</div>
          <FormField label="Search" id="memory-search">
            <input
              value={search}
              onChange={(event) => setSearch(event.target.value)}
              placeholder="Search content or tags"
            />
          </FormField>
          <FormField label="Scope" id="memory-scope-filter">
            <select value={scopeFilter} onChange={(event) => setScopeFilter(event.target.value)}>
              <option value="">All scopes</option>
              {SCOPES.map((scope) => (
                <option key={scope} value={scope}>
                  {scope}
                </option>
              ))}
            </select>
          </FormField>
          <FormField label="Type" id="memory-type-filter">
            <select value={typeFilter} onChange={(event) => setTypeFilter(event.target.value)}>
              <option value="">All types</option>
              {TYPES.map((type) => (
                <option key={type} value={type}>
                  {type}
                </option>
              ))}
            </select>
          </FormField>
          <FormField label="Tag" id="memory-tag-filter">
            <select value={tagFilter} onChange={(event) => setTagFilter(event.target.value)}>
              <option value="">All tags</option>
              {allTags.map((tag) => (
                <option key={tag} value={tag}>
                  {tag}
                </option>
              ))}
            </select>
          </FormField>

          <div className="memory-agent-dirs">
            <span>Agent dirs</span>
            {(memoryQuery.data?.agentDirs ?? []).length > 0 ? (
              <ul>
                {memoryQuery.data?.agentDirs.map((dir) => <li key={dir}>{dir}</li>)}
              </ul>
            ) : (
              <p>No agent directories returned.</p>
            )}
          </div>
        </Card>

        <Card className="memory-list-panel">
          <div className="memory-panel-title">Memories</div>
          {memoryQuery.isLoading ? (
            <p className="memory-muted">Loading memories...</p>
          ) : memoryQuery.isError ? (
            <p className="memory-error">Unable to load memory context from the runtime.</p>
          ) : filteredEntries.length === 0 ? (
            <div className="memory-empty-state">
              <strong>No memories visible in this view (v1 limitation)</strong>
              <p>
                The runtime exposes per-terminal memory context and agent directories, but not a direct
                list-memories endpoint for every scope.
              </p>
            </div>
          ) : (
            <div className="memory-list">
              {filteredEntries.map((entry) => (
                <button
                  key={entry.id}
                  type="button"
                  className={`memory-row ${selectedEntry?.id === entry.id ? 'is-selected' : ''}`}
                  onClick={() => setSelectedId(entry.id)}
                >
                  <span className="memory-row-title">{entry.title}</span>
                  <span className="memory-row-meta">
                    <Badge>{entry.scope}</Badge>
                    <Badge>{entry.type}</Badge>
                  </span>
                  <span className="memory-row-tags">
                    {entry.tags.map((tag) => (
                      <Badge key={tag}>{tag}</Badge>
                    ))}
                  </span>
                  <time>{formatTimestamp(entry.updatedAt)}</time>
                </button>
              ))}
            </div>
          )}
          <p className="memory-muted">
            Terminal contexts scanned: {memoryQuery.data?.terminalCount ?? 0}
          </p>
        </Card>

        <MemoryDetail entry={selectedEntry} />

        <Card className="memory-form-panel">
          <div className="memory-panel-title">New Memory</div>
          <form onSubmit={submitMemory}>
            <FormField label="Title" id="memory-title" errorText={errors.title}>
              <input value={form.title} onChange={(event) => setForm({ ...form, title: event.target.value })} />
            </FormField>
            <FormField label="Scope" id="memory-scope">
              <select
                value={form.scope}
                onChange={(event) => setForm({ ...form, scope: event.target.value as MemoryScope })}
              >
                {SCOPES.map((scope) => (
                  <option key={scope} value={scope}>
                    {scope}
                  </option>
                ))}
              </select>
            </FormField>
            <FormField label="Type" id="memory-type">
              <select
                value={form.type}
                onChange={(event) => setForm({ ...form, type: event.target.value as MemoryType })}
              >
                {TYPES.map((type) => (
                  <option key={type} value={type}>
                    {type}
                  </option>
                ))}
              </select>
            </FormField>
            <FormField label="Tags" id="memory-tags" helperText="Comma-separated tags.">
              <input value={form.tags} onChange={(event) => setForm({ ...form, tags: event.target.value })} />
            </FormField>
            <FormField label="Content" id="memory-content" errorText={errors.content}>
              <textarea
                rows={8}
                value={form.content}
                onChange={(event) => setForm({ ...form, content: event.target.value })}
              />
            </FormField>
            <Button type="submit">Create Memory</Button>
          </form>
          {creationNotice ? <p className="memory-warning">{creationNotice}</p> : null}
        </Card>
      </section>
    </main>
  );
};

const MemoryDetail: React.FC<{ entry: MemoryEntry | null }> = ({ entry }) => {
  if (!entry) {
    return (
      <Card className="memory-detail-panel">
        <p className="memory-muted">Select a memory to inspect metadata, retention, and markdown content.</p>
      </Card>
    );
  }

  return (
    <Card className="memory-detail-panel">
      <div className="memory-detail-header">
        <div>
          <h2>{entry.title}</h2>
          <p>{entry.locationPath}</p>
        </div>
      </div>
      <div className="memory-metadata-grid">
        <div>
          <span>Scope</span>
          <strong>{entry.scope}</strong>
        </div>
        <div>
          <span>Type</span>
          <strong>{entry.type}</strong>
        </div>
        <div>
          <span>Updated</span>
          <strong>{formatTimestamp(entry.updatedAt)}</strong>
        </div>
        <div>
          <span>Location</span>
          <strong>{entry.locationPath}</strong>
        </div>
      </div>
      <div className="memory-tag-row">
        {entry.tags.map((tag) => (
          <Badge key={tag}>{tag}</Badge>
        ))}
      </div>
      {entry.retention ? <div className="memory-retention">{entry.retention}</div> : null}
      <Prose className="memory-prose">{renderMemoryMarkdown(entry.content)}</Prose>
    </Card>
  );
};

async function loadMemoryViewerData(): Promise<MemoryViewerData> {
  const [sessions, agentDirs] = await Promise.all([goCoreClient.listSessions(), goCoreClient.getAgentDirs()]);
  const terminalGroups = await Promise.all(
    sessions.map(async (session) => {
      const terminals = await goCoreClient.listTerminalsInSession(session.name);
      return terminals.map((terminal) => ({ terminal, sessionName: session.name }));
    })
  );
  const terminalSources = terminalGroups.flat();

  const contexts = await Promise.all(
    terminalSources.map(async ({ terminal, sessionName }) => ({
      terminal,
      sessionName,
      context: await goCoreClient.getTerminalMemoryContext(terminal.id),
    }))
  );

  return {
    entries: contexts
      .map((source) => parseTerminalMemoryContext(source))
      .filter((entry): entry is MemoryEntry => entry !== null),
    agentDirs: agentDirs.dirs,
    terminalCount: terminalSources.length,
  };
}

function splitTags(value: string): string[] {
  return value
    .split(',')
    .map((tag) => tag.trim())
    .filter(Boolean);
}

function slugify(value: string): string {
  return value
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-|-$/g, '');
}

function formatTimestamp(value: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString();
}

export default MemoryViewerPage;

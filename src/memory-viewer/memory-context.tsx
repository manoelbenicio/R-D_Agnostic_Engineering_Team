import React from 'react';
import type { Terminal } from '@/api';
import type { MemoryEntry, MemoryScope, MemoryType } from './types';

interface TerminalContextSource {
  terminal: Terminal;
  sessionName: string;
  context: string;
}

const FRONTMATTER_BOUNDARY = '---';
const SCOPES: MemoryScope[] = ['global', 'project', 'session', 'agent'];
const TYPES: MemoryType[] = ['project', 'user', 'feedback', 'reference'];

export function parseTerminalMemoryContext({
  terminal,
  sessionName,
  context,
}: TerminalContextSource): MemoryEntry | null {
  const trimmed = context.trim();
  if (!trimmed) return null;

  const { metadata, body } = parseFrontmatter(trimmed);
  const scope = parseEnum(metadata.scope, SCOPES, 'session');
  const type = parseEnum(metadata.type, TYPES, 'reference');
  const title =
    metadata.title ??
    `${terminal.display_name ?? terminal.profile ?? terminal.id} memory context`;
  const tags = parseTags(metadata.tags ?? terminal.profile);
  const locationPath =
    metadata.location ??
    `memory/${scope}/wiki/${sessionName}/${terminal.profile || terminal.id}.md`;
  const retention =
    metadata.retention ??
    (scope === 'session' ? `Persists until session ${sessionName} ends` : undefined);

  return {
    id: `${terminal.id}:${title}`,
    title,
    scope,
    type,
    tags,
    content: body || trimmed,
    updatedAt: metadata.updatedAt ?? terminal.updated_at ?? terminal.created_at ?? new Date().toISOString(),
    retention,
    locationPath,
    terminalId: terminal.id,
    sessionName,
    source: 'terminal-context',
  };
}

export function renderMemoryMarkdown(markdown: string): React.ReactNode[] {
  const lines = markdown.replace(/\r\n/g, '\n').split('\n');
  const rendered: React.ReactNode[] = [];
  let listItems: string[] = [];
  let codeLines: string[] = [];
  let inCodeBlock = false;

  const flushList = () => {
    if (listItems.length === 0) return;
    rendered.push(
      <ul key={`list-${rendered.length}`}>
        {listItems.map((item, index) => (
          <li key={`${item}-${index}`}>{item}</li>
        ))}
      </ul>
    );
    listItems = [];
  };

  const flushCode = () => {
    if (codeLines.length === 0) return;
    rendered.push(
      <pre key={`code-${rendered.length}`}>
        <code>{codeLines.join('\n')}</code>
      </pre>
    );
    codeLines = [];
  };

  for (const line of lines) {
    if (line.startsWith('```')) {
      if (inCodeBlock) {
        inCodeBlock = false;
        flushCode();
      } else {
        flushList();
        inCodeBlock = true;
      }
      continue;
    }

    if (inCodeBlock) {
      codeLines.push(line);
      continue;
    }

    const trimmed = line.trim();
    if (!trimmed) {
      flushList();
      continue;
    }

    if (trimmed.startsWith('- ')) {
      listItems.push(trimmed.slice(2));
      continue;
    }

    flushList();
    if (trimmed.startsWith('### ')) {
      rendered.push(<h3 key={`h3-${rendered.length}`}>{trimmed.slice(4)}</h3>);
    } else if (trimmed.startsWith('## ')) {
      rendered.push(<h2 key={`h2-${rendered.length}`}>{trimmed.slice(3)}</h2>);
    } else if (trimmed.startsWith('# ')) {
      rendered.push(<h1 key={`h1-${rendered.length}`}>{trimmed.slice(2)}</h1>);
    } else {
      rendered.push(<p key={`p-${rendered.length}`}>{trimmed}</p>);
    }
  }

  flushList();
  flushCode();
  return rendered;
}

export function memoryMatchesSearch(entry: MemoryEntry, rawSearch: string): boolean {
  const search = rawSearch.trim().toLowerCase();
  if (!search) return true;
  return [entry.title, entry.content, entry.locationPath, ...entry.tags]
    .join(' ')
    .toLowerCase()
    .includes(search);
}

function parseFrontmatter(markdown: string): { metadata: Record<string, string>; body: string } {
  const lines = markdown.replace(/\r\n/g, '\n').split('\n');
  if (lines[0] !== FRONTMATTER_BOUNDARY) return { metadata: {}, body: markdown };

  const endIndex = lines.findIndex((line, index) => index > 0 && line === FRONTMATTER_BOUNDARY);
  if (endIndex < 0) return { metadata: {}, body: markdown };

  const metadata: Record<string, string> = {};
  for (const line of lines.slice(1, endIndex)) {
    const separator = line.indexOf(':');
    if (separator < 0) continue;
    metadata[line.slice(0, separator).trim()] = line.slice(separator + 1).trim();
  }

  return {
    metadata,
    body: lines.slice(endIndex + 1).join('\n').trim(),
  };
}

function parseTags(value: string | undefined): string[] {
  if (!value) return [];
  return value
    .replace(/^\[|\]$/g, '')
    .split(',')
    .map((tag) => tag.trim())
    .filter(Boolean);
}

function parseEnum<T extends string>(value: string | undefined, options: readonly T[], fallback: T): T {
  return value && options.includes(value as T) ? (value as T) : fallback;
}

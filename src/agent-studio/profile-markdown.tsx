import React from 'react';

export interface ProfileFrontmatter {
  name: string;
  role: string;
  provider: string;
  model?: string;
  allowedTools?: string[];
  mcpServers?: string[];
  permissionMode?: string;
}

export interface ParsedProfileMarkdown {
  frontmatter: Partial<ProfileFrontmatter> & Record<string, string | string[] | undefined>;
  body: string;
}

const FRONTMATTER_BOUNDARY = '---';

export function parseProfileMarkdown(markdown: string | undefined): ParsedProfileMarkdown {
  if (!markdown) return { frontmatter: {}, body: '' };
  const lines = markdown.replace(/\r\n/g, '\n').split('\n');
  if (lines[0] !== FRONTMATTER_BOUNDARY) {
    return { frontmatter: {}, body: markdown };
  }

  const endIndex = lines.findIndex((line, index) => index > 0 && line === FRONTMATTER_BOUNDARY);
  if (endIndex === -1) {
    return { frontmatter: {}, body: markdown };
  }

  const frontmatterLines = lines.slice(1, endIndex);
  const body = lines.slice(endIndex + 1).join('\n').trim();
  const frontmatter: ParsedProfileMarkdown['frontmatter'] = {};

  for (const line of frontmatterLines) {
    const separator = line.indexOf(':');
    if (separator === -1) continue;
    const key = line.slice(0, separator).trim();
    const rawValue = line.slice(separator + 1).trim();
    frontmatter[key] = parseYamlishValue(rawValue);
  }

  return { frontmatter, body };
}

export function serializeProfileMarkdown(frontmatter: ProfileFrontmatter, body: string): string {
  const lines = [
    FRONTMATTER_BOUNDARY,
    `name: ${frontmatter.name}`,
    `role: ${frontmatter.role}`,
    `provider: ${frontmatter.provider}`,
  ];

  if (frontmatter.model) lines.push(`model: ${frontmatter.model}`);
  if (frontmatter.allowedTools?.length) {
    lines.push(`allowedTools: [${frontmatter.allowedTools.join(', ')}]`);
  }
  if (frontmatter.mcpServers?.length) {
    lines.push(`mcpServers: [${frontmatter.mcpServers.join(', ')}]`);
  }
  if (frontmatter.permissionMode) {
    lines.push(`permissionMode: ${frontmatter.permissionMode}`);
  }

  return `${lines.join('\n')}\n${FRONTMATTER_BOUNDARY}\n${body.trim()}\n`;
}

export function firstLineDescription(markdown: string | undefined, fallback = ''): string {
  const parsed = parseProfileMarkdown(markdown);
  const firstLine = parsed.body
    .split('\n')
    .map((line) => line.replace(/^#+\s*/, '').trim())
    .find(Boolean);
  return firstLine ?? fallback;
}

export function renderMarkdown(markdown: string): React.ReactNode[] {
  const lines = markdown.replace(/\r\n/g, '\n').split('\n');
  const rendered: React.ReactNode[] = [];
  let index = 0;
  let listItems: string[] = [];
  let codeLines: string[] = [];
  let inCodeBlock = false;

  const flushList = () => {
    if (listItems.length > 0) {
      rendered.push(
        <ul key={`list-${rendered.length}`}>
          {listItems.map((item, itemIndex) => (
            <li key={`${item}-${itemIndex}`}>{item}</li>
          ))}
        </ul>
      );
      listItems = [];
    }
  };

  const flushCode = () => {
    if (codeLines.length > 0) {
      rendered.push(
        <pre key={`code-${rendered.length}`}>
          <code>{codeLines.join('\n')}</code>
        </pre>
      );
      codeLines = [];
    }
  };

  while (index < lines.length) {
    const line = lines[index] ?? '';

    if (line.startsWith('```')) {
      if (inCodeBlock) {
        inCodeBlock = false;
        flushCode();
      } else {
        flushList();
        inCodeBlock = true;
      }
      index += 1;
      continue;
    }

    if (inCodeBlock) {
      codeLines.push(line);
      index += 1;
      continue;
    }

    const trimmed = line.trim();
    if (!trimmed) {
      flushList();
      index += 1;
      continue;
    }

    if (trimmed.startsWith('- ')) {
      listItems.push(trimmed.slice(2));
      index += 1;
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
    index += 1;
  }

  flushList();
  flushCode();
  return rendered;
}

function parseYamlishValue(value: string): string | string[] {
  if (value.startsWith('[') && value.endsWith(']')) {
    return value
      .slice(1, -1)
      .split(',')
      .map((item) => item.trim())
      .filter(Boolean);
  }
  return value.replace(/^["']|["']$/g, '');
}

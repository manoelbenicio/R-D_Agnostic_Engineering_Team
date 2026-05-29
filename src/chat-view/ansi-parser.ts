import type { ParsedSegment, ParsedSegmentKind } from './types';

/* eslint-disable no-control-regex */
const ANSI_PATTERN = new RegExp(
  '[\\u001b\\u009b][[\\]()#;?]*(?:(?:(?:[a-zA-Z\\d]*(?:;[a-zA-Z\\d]*)*)?\\u0007)|(?:(?:\\d{1,4}(?:;\\d{0,4})*)?[\\dA-PR-TZcf-nq-uy=><~]))',
  'g'
);
/* eslint-enable no-control-regex */

const TOOL_CALL_PATTERN =
  /^\s*(?:tool\s*call|calling\s+tool|running\s+tool|bash\(|read\(|edit\(|write\(|grep\(|search\(|run\s*:)/i;
const TOOL_RESULT_PATTERN = /^\s*(?:tool\s*result|result\s*:|output\s*:|exit\s+code\s*:)/i;
const SYSTEM_PATTERN = /^\s*(?:system|notice|warning|error)\s*:/i;
const AGENT_BOUNDARY_PATTERN =
  /^\s*(?:\[[^\]]+\]\s*)?(?:agent|assistant|supervisor|developer|coder|codex|claude|gemini|gpt|user)[\w .-]{0,48}:\s+/i;

export function stripAnsi(input: string): string {
  return input.replace(ANSI_PATTERN, '');
}

export function segmentByAgentBoundary(buffer: string, terminalId = ''): ParsedSegment[] {
  const clean = stripAnsi(buffer);
  if (clean.length === 0) return [];

  const lines = clean.split(/\r?\n/);
  const segments: ParsedSegment[] = [];
  let current: ParsedSegment | null = null;

  for (const line of lines) {
    const kind = detectSegmentKind(line);
    const startsBoundary = current === null || kind !== current.kind || isAgentBoundary(line);

    if (startsBoundary) {
      if (current) segments.push(trimSegment(current));
      current = {
        terminalId,
        content: line,
        kind,
      };
      continue;
    }

    if (current) current.content += `\n${line}`;
  }

  if (current) segments.push(trimSegment(current));
  return segments.filter((segment) => segment.content.length > 0);
}

function detectSegmentKind(line: string): ParsedSegmentKind {
  if (TOOL_CALL_PATTERN.test(line)) return 'tool_call';
  if (TOOL_RESULT_PATTERN.test(line)) return 'tool_result';
  if (SYSTEM_PATTERN.test(line)) return 'system';
  return 'output';
}

function isAgentBoundary(line: string): boolean {
  return AGENT_BOUNDARY_PATTERN.test(line);
}

function trimSegment(segment: ParsedSegment): ParsedSegment {
  return {
    ...segment,
    content: segment.content.trim(),
  };
}

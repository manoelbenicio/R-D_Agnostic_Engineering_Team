import { describe, expect, it } from 'vitest';
import { segmentByAgentBoundary, stripAnsi } from '../ansi-parser';

describe('ansi-parser', () => {
  it('strips a simple SGR color sequence', () => {
    expect(stripAnsi('\x1b[31mERROR\x1b[0m')).toBe('ERROR');
  });

  it('strips multiple ANSI and VT100 sequences', () => {
    expect(stripAnsi('\x1b[1mBold\x1b[0m \x1b[2K\x1b[32mOK\x1b[0m')).toBe('Bold OK');
  });

  it('does not alter benign text', () => {
    expect(stripAnsi('plain [31m text without escape')).toBe('plain [31m text without escape');
  });

  it('returns one output segment for benign output', () => {
    expect(segmentByAgentBoundary('hello world', 'term-1')).toEqual([
      {
        terminalId: 'term-1',
        content: 'hello world',
        kind: 'output',
      },
    ]);
  });
});

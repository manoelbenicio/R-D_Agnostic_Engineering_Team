import { describe, expect, it } from 'vitest';
import { ALLOWED_TOOL_IDS, HIGH_PRIVILEGE_TOOLS, unknownTools } from '../allowed-tools';

describe('allowed-tools registry', () => {
  it('exposes the 9 canonical tool ids', () => {
    expect(ALLOWED_TOOL_IDS).toEqual([
      'handoff',
      'assign',
      'send_message',
      'read_file',
      'grep',
      'shell',
      'apply_patch',
      'test',
      'web_search',
    ]);
  });

  it('flags shell and apply_patch as high privilege', () => {
    expect(HIGH_PRIVILEGE_TOOLS).toEqual(['shell', 'apply_patch']);
  });

  it('detects unknown/typo tools and ignores valid ones', () => {
    expect(unknownTools(['read_file', 'read-file', 'shell', 'bogus'])).toEqual(['read-file', 'bogus']);
    expect(unknownTools(['handoff', 'assign'])).toEqual([]);
  });
});

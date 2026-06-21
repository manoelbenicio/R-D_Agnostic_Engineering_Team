// @ts-expect-error - eslint v8 does not ship bundled TypeScript declarations in this repo.
import { ESLint } from 'eslint';
import { createRequire } from 'node:module';
import { describe, expect, it } from 'vitest';

const require = createRequire(import.meta.url);
const agentversePlugin = require('../../../eslint-rules/index.cjs') as { rules: Record<string, unknown> };

describe('agentverse/no-direct-go-core-fetch', () => {
  it('reports direct fetch calls to GO Core routes outside src/api/go-core-client.ts', async () => {
    const eslint = new ESLint({
      useEslintrc: false,
      plugins: {
        agentverse: agentversePlugin,
      },
      baseConfig: {
        plugins: ['agentverse'],
        parser: '@typescript-eslint/parser',
        parserOptions: { ecmaVersion: 2022, sourceType: 'module' },
        rules: {
          'agentverse/no-direct-go-core-fetch': 'error',
        },
      },
    });

    const [result] = await eslint.lintText("fetch('/health');", {
      filePath: 'C:/repo/src/dashboard/bad.ts',
    });

    expect(result?.messages).toEqual(
      expect.arrayContaining([
        expect.objectContaining({
          ruleId: 'agentverse/no-direct-go-core-fetch',
          message: expect.stringContaining('Direct fetch to a GO Core route is forbidden'),
        }),
      ])
    );
  });
});

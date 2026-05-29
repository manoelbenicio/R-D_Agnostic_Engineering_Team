/* eslint-env node */
module.exports = {
  root: true,
  env: { browser: true, es2022: true, node: true },
  parser: '@typescript-eslint/parser',
  parserOptions: {
    ecmaVersion: 2022,
    sourceType: 'module',
    ecmaFeatures: { jsx: true },
  },
  settings: { react: { version: '18.3' } },
  plugins: ['@typescript-eslint', 'react', 'react-hooks', 'react-refresh', 'agentverse'],
  extends: [
    'eslint:recommended',
    'plugin:@typescript-eslint/recommended',
    'plugin:react/recommended',
    'plugin:react/jsx-runtime',
    'plugin:react-hooks/recommended',
  ],
  rules: {
    'react-refresh/only-export-components': 'warn',
    'react/prop-types': 'off',
    '@typescript-eslint/no-explicit-any': 'warn',
    '@typescript-eslint/no-unused-vars': [
      'error',
      { argsIgnorePattern: '^_', varsIgnorePattern: '^_' },
    ],
    'agentverse/no-sideways-capability-imports': 'error',
    'agentverse/no-direct-cao-fetch': 'error',
  },
  ignorePatterns: [
    'dist/',
    'node_modules/',
    'eslint-rules/',
    'playwright-report/',
    'test-results/',
    'coverage/',
    'data_expert_skills/',
  ],
  overrides: [
    {
      files: ['src/**/*.ts', 'src/**/*.tsx', 'tests/**/*.ts', 'tests/**/*.tsx'],
      parserOptions: { project: ['./tsconfig.app.json'] },
    },
    {
      files: ['**/__tests__/**/*', '**/*.test.{ts,tsx}', '**/*.spec.{ts,tsx}'],
      env: { node: true },
      rules: {
        '@typescript-eslint/no-explicit-any': 'off',
        'agentverse/no-direct-cao-fetch': 'off',
      },
    },
  ],
};

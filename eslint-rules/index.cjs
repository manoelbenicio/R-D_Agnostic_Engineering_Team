/**
 * Local ESLint plugin enforcing AgentVerse architectural rules.
 *
 * Rules:
 *   - no-sideways-capability-imports (task 1.3, design D9)
 *       Inside `src/<capability>/...`, only allow imports from:
 *         - the same capability (relative or alias-resolved into same root)
 *         - `@/shared/...` (the SUP-gated cross-cutting types)
 *         - `@/design-system/...` (the locked design system, consumed everywhere)
 *         - external packages
 *   - no-direct-go-core-fetch (task 4.13)
 *       Outside `src/api/go-core-client.ts`, calling `fetch(...)` with a literal that
 *       points to a GO Core route (anything containing `/health`, `/sessions`,
 *       `/terminals`, `/agents/`, `/flows`, `/skills`, `/agent-dirs`) fails the
 *       build.
 */

const path = require('node:path');

const CAPABILITIES = [
  'agent-studio',
  'canvas-builder',
  'canvas-document',
  'canvas-reconciler',
  'canvas-templates',
  'chat-view',
  'dashboard',
  'design-system',
  'finops',
  'flows',
  'health',
  'memory-viewer',
  'terminal',
  'terminal-grid',
  'voice',
  'api',
  'settings',
  'shell',
  'shared',
];

// Capabilities that anyone can import from (cross-cutting, gated by ownership rules).
//   - 'shared': SUP-gated cross-cutting types and IDB infra (D9).
//   - 'design-system': locked tokens and base components — every UI consumes them.
//   - 'api': the only path to GO Core; every capability needs goCoreClient / useHealthStore /
//     query-keys. Keeping the cross-import allowed here mirrors design D11.
//   - 'shell': exposes cross-cutting utilities (appFetch, useToast, ErrorBoundary).
//     Every capability that does HTTP or surfaces errors needs these.
const CROSS_CAPABILITY_OK = new Set(['shared', 'design-system', 'api', 'shell']);

// GO Core route fragments — appearing in a fetch URL outside the client is a violation.
const GO_CORE_ROUTE_FRAGMENTS = [
  '/health',
  '/sessions',
  '/terminals',
  '/agents/',
  '/flows',
  '/skills',
  '/agent-dirs',
];

function srcRelative(filename) {
  const idx = filename.lastIndexOf(`${path.sep}src${path.sep}`);
  if (idx < 0) return null;
  return filename.slice(idx + 5).replace(/\\/g, '/');
}

function resolveCapability(rel) {
  if (!rel) return null;
  const head = rel.split('/')[0];
  return CAPABILITIES.includes(head) ? head : null;
}

function resolveImportCapability(importPath, fileRel) {
  // Alias `@/...`
  if (importPath.startsWith('@/')) {
    const sub = importPath.slice(2);
    return resolveCapability(sub);
  }
  // Bare module
  if (!importPath.startsWith('.')) return null;
  // Relative — resolve against fileRel
  const dir = path.posix.dirname(fileRel);
  const joined = path.posix.normalize(path.posix.join(dir, importPath));
  return resolveCapability(joined);
}

const noSideways = {
  meta: {
    type: 'problem',
    docs: { description: 'Forbid sideways imports across capabilities (D9).' },
    messages: {
      sideways:
        "Sideways capability import: '{{from}}' imports from '{{to}}'. Cross-capability code MUST go through @/shared/ or @/design-system/ (design D9).",
    },
    schema: [],
  },
  create(context) {
    const filename = context.filename || context.getFilename();
    const fileRel = srcRelative(filename);
    const fromCap = resolveCapability(fileRel);
    if (!fromCap) return {};
    return {
      ImportDeclaration(node) {
        const target = node.source.value;
        if (typeof target !== 'string') return;
        const toCap = resolveImportCapability(target, fileRel);
        if (!toCap) return; // external package or unresolved
        if (toCap === fromCap) return;
        if (CROSS_CAPABILITY_OK.has(toCap)) return;
        context.report({
          node: node.source,
          messageId: 'sideways',
          data: { from: fromCap, to: toCap },
        });
      },
    };
  },
};

const noDirectGoCoreFetch = {
  meta: {
    type: 'problem',
    docs: { description: 'Forbid direct fetch() to GO Core endpoints (4.13).' },
    messages: {
      direct:
        "Direct fetch to a GO Core route is forbidden. Use goCoreClient in @/api/go-core-client (task 4.13).",
    },
    schema: [],
  },
  create(context) {
    const filename = context.filename || context.getFilename();
    if (filename.includes(`${path.sep}api${path.sep}go-core-client`)) return {};
    if (filename.includes(`__tests__`)) return {};

    function literalLooksLikeGoCoreRoute(str) {
      if (typeof str !== 'string') return false;
      return GO_CORE_ROUTE_FRAGMENTS.some((fragment) => str.includes(fragment));
    }

    return {
      CallExpression(node) {
        const callee = node.callee;
        if (callee.type !== 'Identifier' || callee.name !== 'fetch') return;
        const first = node.arguments[0];
        if (!first) return;
        if (first.type === 'Literal' && literalLooksLikeGoCoreRoute(first.value)) {
          context.report({ node, messageId: 'direct' });
          return;
        }
        if (first.type === 'TemplateLiteral') {
          const raw = first.quasis.map((q) => q.value.cooked || '').join('');
          if (literalLooksLikeGoCoreRoute(raw)) {
            context.report({ node, messageId: 'direct' });
          }
        }
      },
    };
  },
};

module.exports = {
  rules: {
    'no-sideways-capability-imports': noSideways,
    'no-direct-go-core-fetch': noDirectGoCoreFetch,
    'no-direct-cao-fetch': noDirectGoCoreFetch, // Backward-compat alias
  },
};

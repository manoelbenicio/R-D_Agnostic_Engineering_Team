#!/usr/bin/env node
/**
 * Verify dist/ total gzip size <= 1.5 MB (task 21.3).
 * Reports per-file breakdown for visibility.
 */
import { readdir, readFile, stat } from 'node:fs/promises';
import { gzipSync } from 'node:zlib';
import { join } from 'node:path';

const BUDGET_BYTES = 1.5 * 1024 * 1024; // 1.5 MB gzipped

async function* walk(dir) {
  const entries = await readdir(dir, { withFileTypes: true });
  for (const entry of entries) {
    const p = join(dir, entry.name);
    if (entry.isDirectory()) {
      yield* walk(p);
    } else {
      yield p;
    }
  }
}

async function main() {
  const dist = 'dist';
  try {
    await stat(dist);
  } catch {
    console.error(`Cannot find ${dist}/. Run \`npm run build\` first.`);
    process.exit(2);
  }

  let total = 0;
  const breakdown = [];
  for await (const file of walk(dist)) {
    if (file.endsWith('.map')) continue;
    const buf = await readFile(file);
    const gz = gzipSync(buf).byteLength;
    total += gz;
    breakdown.push({ file, gz });
  }

  breakdown.sort((a, b) => b.gz - a.gz);
  console.log('Top 15 files by gzipped size:');
  for (const { file, gz } of breakdown.slice(0, 15)) {
    console.log(`  ${(gz / 1024).toFixed(1).padStart(7)} KB  ${file}`);
  }
  const totalKB = (total / 1024).toFixed(1);
  const budgetKB = (BUDGET_BYTES / 1024).toFixed(1);
  console.log(`\nTotal gzipped: ${totalKB} KB (budget: ${budgetKB} KB)`);
  if (total > BUDGET_BYTES) {
    console.error('Bundle exceeds 1.5 MB gzipped budget (task 21.3).');
    process.exit(1);
  }
  console.log('Within budget ✓');
}

main().catch((err) => {
  console.error(err);
  process.exit(2);
});

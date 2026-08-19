// Merge vite-plugin-istanbul JSON coverage dumps into a single lcov.info
// for downstream CI artifacts and coverage analysis.
//
// Usage: node scripts/merge-coverage-to-lcov.mjs
//
// Expects raw coverage JSON files under coverage/.tmp/ (written by the
// Playwright coverage fixture). Moves them into .nyc_output/, then runs
// nyc report to produce coverage/frontend/lcov.info.
import { execSync } from 'node:child_process';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const SCRIPT_DIR = path.dirname(fileURLToPath(import.meta.url));
const ROOT = path.resolve(SCRIPT_DIR, '..');
const TMP = path.join(ROOT, 'coverage', '.tmp');
const OUT = path.join(ROOT, 'coverage', 'frontend');
const NYC_OUTPUT = path.join(ROOT, '.nyc_output');

if (!fs.existsSync(TMP)) {
  console.log('No coverage temp directory. Skipping merge.');
  process.exit(0);
}

const files = fs.readdirSync(TMP).filter((f) => f.endsWith('.json'));
if (files.length === 0) {
  console.log('No coverage JSON files found. Skipping merge.');
  process.exit(0);
}

// Clear previous output
fs.rmSync(NYC_OUTPUT, { recursive: true, force: true });
fs.mkdirSync(NYC_OUTPUT, { recursive: true });
fs.rmSync(OUT, { recursive: true, force: true });
fs.mkdirSync(OUT, { recursive: true });

// Move raw coverage into .nyc_output/
for (const f of files) {
  fs.copyFileSync(path.join(TMP, f), path.join(NYC_OUTPUT, f));
}

// nyc report expects istanbul-format coverage in .nyc_output/
execSync('npx nyc report --reporter=lcovonly --report-dir=coverage/frontend', {
  stdio: 'inherit',
  cwd: ROOT,
});

console.log(`lcov written to ${path.relative(ROOT, OUT)}/lcov.info (${files.length} files merged)`);

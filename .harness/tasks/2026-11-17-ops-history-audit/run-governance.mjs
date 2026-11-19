import fs from 'node:fs';
import path from 'node:path';
import { spawnSync } from 'node:child_process';

const evidenceDir = '.harness/evidence/2026-11-17-ops-history-audit';
const prefix = process.argv[2] ?? 'baseline';
if (!/^[a-z0-9-]+$/.test(prefix)) throw new Error('Invalid evidence prefix');
if (fs.existsSync(`${evidenceDir}/${prefix}-checks.json`)) throw new Error('Evidence already exists; choose a fresh prefix');
fs.mkdirSync(evidenceDir, { recursive: true });
const checks = [
  ['task', 'scripts/harness/check-task-packet.mjs', '--json'],
  ['evidence', 'scripts/harness/check-evidence.mjs', '--json', '--strict'],
  ['review', 'scripts/harness/check-review.mjs', '--json', '--strict'],
  ['links', 'scripts/harness/check-doc-links.mjs', '--json', '--strict'],
  ['frontmatter', 'scripts/frontmatter-check.mjs'],
  ['template', 'scripts/check-task-packet-template.mjs'],
  ['overlay', 'scripts/business-overlay/check-business-overlay.mjs'],
  ['boundaries', 'scripts/check-business-module-boundaries.mjs'],
];
const results = [];
for (const [name, ...args] of checks) {
  const started = new Date().toISOString();
  const run = spawnSync(process.execPath, args, { encoding: 'utf8', windowsHide: true, maxBuffer: 16 * 1024 * 1024 });
  const output = `${run.stdout ?? ''}${run.stderr ?? ''}`;
  const log = `${evidenceDir}/${prefix}-${name}.${args.includes('--json') ? 'json' : 'log'}`;
  fs.writeFileSync(log, output, { flag: 'wx' });
  let summary;
  try { const payload = JSON.parse(run.stdout); summary = payload.summary ?? payload.counts ?? payload.totals; } catch { /* Text check. */ }
  const row = { name, command: `node ${args.join(' ')}`, cwd: path.resolve('.'), status: run.status === 0 ? 'passed' : 'failed',
    exitCode: run.status, started, finishedAt: new Date().toISOString(), evidence: log, summary };
  results.push(row);
  console.log(JSON.stringify(row));
}
fs.writeFileSync(`${evidenceDir}/${prefix}-checks.json`, JSON.stringify(results, null, 2) + '\n');
process.exitCode = results.some((r) => r.status === 'failed') ? 1 : 0;

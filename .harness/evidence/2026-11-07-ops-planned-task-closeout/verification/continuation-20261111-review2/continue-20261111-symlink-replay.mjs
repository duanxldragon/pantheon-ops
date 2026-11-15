// Finite regression replay only: this does not approve or assemble a real candidate.
import assert from 'node:assert/strict';
import crypto from 'node:crypto';
import fs from 'node:fs';
import path from 'node:path';
import { spawnSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';

const root = path.dirname(fileURLToPath(import.meta.url));
const manifest = JSON.parse(fs.readFileSync(path.join(root, 'manifest.json'), 'utf8'));
const hash = bytes => crypto.createHash('sha256').update(bytes).digest('hex');
for (const item of manifest.files) {
  const file = path.join(root, item.path);
  assert.equal(fs.lstatSync(file).isSymbolicLink(), false, item.path);
  assert.equal(hash(fs.readFileSync(file)), item.sha256, item.path);
}
if (process.argv.includes('--verify-only')) {
  console.log(JSON.stringify({ filesVerified: manifest.files.length, runtimeTestExecuted: false }));
} else {
  assert.equal(process.platform, 'linux', 'This replay requires an existing Linux runtime');
  assert.equal(Number(process.versions.node.split('.')[0]), 24, 'Use Node 24, matching the existing quality workflow');
  const git = spawnSync('git', ['--version'], { encoding: 'utf8', timeout: 10000 });
  assert.equal(git.status, 0, 'An installed Git executable is required; do not install or pull dependencies implicitly');
  const test = spawnSync(process.execPath, [
    '--test', '--test-reporter=tap',
    '--test-name-pattern=^selected inputs: selected symlink still refuses before stage creation$',
    'pantheon-ops/scripts/business-overlay/foundation-candidate.test.mjs',
  ], { cwd: root, encoding: 'utf8', timeout: 120000, maxBuffer: 1024 * 1024 });
  const tap = (test.stdout ?? '') + (test.stderr ?? '');
  fs.writeFileSync(path.join(root, 'symlink-test.tap'), tap, { flag: 'wx' });
  const counts = {};
  for (const name of ['tests', 'pass', 'fail', 'cancelled', 'skipped', 'todo']) {
    const match = tap.match(new RegExp(`^# ${name} (\\d+)$`, 'm'));
    counts[name] = match ? Number(match[1]) : null;
  }
  for (const item of manifest.files)
    assert.equal(hash(fs.readFileSync(path.join(root, item.path))), item.sha256, item.path);
  const passed = test.status === 0 && counts.tests === 1 && counts.pass === 1 &&
    ['fail', 'cancelled', 'skipped', 'todo'].every(name => counts[name] === 0);
  const receipt = { sourceStable: true, test: manifest.test, runtime: process.platform,
    node: process.version, git: git.stdout.trim(), exitCode: test.status,
    errorCode: test.error?.code ?? null, counts, passed,
    manifestSHA256: hash(fs.readFileSync(path.join(root, 'manifest.json'))),
    tapSHA256: hash(Buffer.from(tap)), actualCompleteCandidateAssembly: false,
    independentApproval: false };
  fs.writeFileSync(path.join(root, 'result.json'), JSON.stringify(receipt, null, 2) + '\n', { flag: 'wx' });
  console.log(JSON.stringify(receipt));
  process.exitCode = passed ? 0 : 1;
}

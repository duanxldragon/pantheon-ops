import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { spawnSync } from 'node:child_process';
import test from 'node:test';

function runDriftCheck(root, base) {
  return spawnSync(process.execPath, [
    path.resolve('scripts/business-overlay/check-shared-drift.mjs'),
    '--root', root,
    '--base', base,
  ], { cwd: path.resolve('.'), encoding: 'utf8' });
}

function initGitRepo(root, files) {
  fs.mkdirSync(root, { recursive: true });
  for (const [file, content] of Object.entries(files)) {
    const target = path.join(root, file);
    fs.mkdirSync(path.dirname(target), { recursive: true });
    fs.writeFileSync(target, content, 'utf8');
  }
  const run = (args) => spawnSync('git', args, { cwd: root, encoding: 'utf8' });
  run(['init']);
  run(['config', 'user.email', 'test@example.com']);
  run(['config', 'user.name', 'Test']);
  run(['add', '.']);
  run(['commit', '-m', 'fixture']);
}

function fixtureLock(root, sharedPaths) {
  fs.mkdirSync(root, { recursive: true });
  fs.writeFileSync(
    path.join(root, 'foundation-release.lock.json'),
    JSON.stringify({ schemaVersion: 1, sharedPaths }),
    'utf8',
  );
}

test('passes when all shared files match base HEAD', () => {
  const temp = fs.mkdtempSync(path.join(os.tmpdir(), 'pantheon-drift-'));
  const baseRoot = path.join(temp, 'base');
  const opsRoot = path.join(temp, 'ops');
  initGitRepo(baseRoot, { 'backend/pkg/shared.go': 'package shared\n' });
  fixtureLock(opsRoot, { backend: ['backend/pkg'] });
  fs.mkdirSync(path.join(opsRoot, 'backend/pkg'), { recursive: true });
  fs.writeFileSync(path.join(opsRoot, 'backend/pkg/shared.go'), 'package shared\n', 'utf8');

  const result = runDriftCheck(opsRoot, baseRoot);
  assert.equal(result.status, 0, result.stderr);
  assert.match(result.stdout, /OK shared drift check: 1 shared files identical/);
});

test('fails on drifted shared file and passes once allowlisted', () => {
  const temp = fs.mkdtempSync(path.join(os.tmpdir(), 'pantheon-drift-'));
  const baseRoot = path.join(temp, 'base');
  const opsRoot = path.join(temp, 'ops');
  initGitRepo(baseRoot, { 'backend/pkg/shared.go': 'package shared\n' });
  fixtureLock(opsRoot, { backend: ['backend/pkg'] });
  fs.mkdirSync(path.join(opsRoot, 'backend/pkg'), { recursive: true });
  fs.writeFileSync(path.join(opsRoot, 'backend/pkg/shared.go'), 'package shared // ops change\n', 'utf8');

  const failing = runDriftCheck(opsRoot, baseRoot);
  assert.equal(failing.status, 1);
  assert.match(failing.stderr, /DRIFT_MODIFIED backend\/pkg\/shared\.go/);

  fs.mkdirSync(path.join(opsRoot, '.foundation'), { recursive: true });
  fs.writeFileSync(
    path.join(opsRoot, '.foundation', 'allowed-drift.json'),
    JSON.stringify({ 'backend/pkg/shared.go': 'intentional overlay extension' }),
    'utf8',
  );
  const allowed = runDriftCheck(opsRoot, baseRoot);
  assert.equal(allowed.status, 0, allowed.stderr);
  assert.match(allowed.stdout, /ALLOWED_DRIFT backend\/pkg\/shared\.go/);
});

test('excludes business overlay paths from comparison', () => {
  const temp = fs.mkdtempSync(path.join(os.tmpdir(), 'pantheon-drift-'));
  const baseRoot = path.join(temp, 'base');
  const opsRoot = path.join(temp, 'ops');
  initGitRepo(baseRoot, { 'backend/pkg/shared.go': 'package shared\n' });
  fixtureLock(opsRoot, { backend: ['backend/modules', 'backend/pkg'] });
  fs.mkdirSync(path.join(opsRoot, 'backend/pkg'), { recursive: true });
  fs.mkdirSync(path.join(opsRoot, 'backend/modules/business'), { recursive: true });
  fs.writeFileSync(path.join(opsRoot, 'backend/modules/business/ours.go'), 'package business\n', 'utf8');
  fs.writeFileSync(path.join(opsRoot, 'backend/pkg/shared.go'), 'package shared\n', 'utf8');
  fs.writeFileSync(
    path.join(opsRoot, 'business-overlay.json'),
    JSON.stringify({ businessPaths: ['backend/modules/business'] }),
    'utf8',
  );

  const result = runDriftCheck(opsRoot, baseRoot);
  assert.equal(result.status, 0, result.stderr);
  assert.match(result.stdout, /OK shared drift check: 1 shared files identical/);
});

test('fails loudly when the lock declares no sharedPaths', () => {
  const temp = fs.mkdtempSync(path.join(os.tmpdir(), 'pantheon-drift-'));
  const baseRoot = path.join(temp, 'base');
  const opsRoot = path.join(temp, 'ops');
  initGitRepo(baseRoot, { 'backend/pkg/shared.go': 'package shared\n' });
  fs.mkdirSync(opsRoot, { recursive: true });
  fs.writeFileSync(
    path.join(opsRoot, 'foundation-release.lock.json'),
    JSON.stringify({ schemaVersion: 1 }),
    'utf8',
  );

  const result = runDriftCheck(opsRoot, baseRoot);
  assert.equal(result.status, 2);
  assert.match(result.stderr, /declares no sharedPaths/);
});

test('reports stale allowlist entries', () => {
  const temp = fs.mkdtempSync(path.join(os.tmpdir(), 'pantheon-drift-'));
  const baseRoot = path.join(temp, 'base');
  const opsRoot = path.join(temp, 'ops');
  initGitRepo(baseRoot, { 'backend/pkg/shared.go': 'package shared\n' });
  fixtureLock(opsRoot, { backend: ['backend/pkg'] });
  fs.mkdirSync(path.join(opsRoot, 'backend/pkg'), { recursive: true });
  fs.writeFileSync(path.join(opsRoot, 'backend/pkg/shared.go'), 'package shared\n', 'utf8');
  fs.mkdirSync(path.join(opsRoot, '.foundation'), { recursive: true });
  fs.writeFileSync(
    path.join(opsRoot, '.foundation', 'allowed-drift.json'),
    JSON.stringify({ 'backend/pkg/gone.go': 'no longer drifted' }),
    'utf8',
  );

  const result = runDriftCheck(opsRoot, baseRoot);
  assert.equal(result.status, 1);
  assert.match(result.stderr, /STALE_ALLOWLIST backend\/pkg\/gone\.go/);
});

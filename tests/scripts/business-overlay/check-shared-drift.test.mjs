import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { spawnSync } from 'node:child_process';
import test from 'node:test';
import { pathToFileURL } from 'node:url';

function runDriftCheck(root, base) {
  const moduleURL = pathToFileURL(path.resolve('scripts/business-overlay/check-shared-drift.mjs')).href;
  const script = `import { checkSharedDrift } from ${JSON.stringify(moduleURL)}; import fs from 'node:fs'; import path from 'node:path';
    process.exitCode = checkSharedDrift({root:process.argv[1],snapshot:{baseRoot:process.argv[2],manifest:JSON.parse(fs.readFileSync(path.join(process.argv[1],'foundation-release.lock.json'),'utf8'))}});`;
  return spawnSync(process.execPath, ['--input-type=module', '-e', script, root, base], { encoding: 'utf8' });
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

test('passes when all shared files match snapshot bytes', () => {
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
    JSON.stringify({ schemaVersion: 1, businessPaths: ['backend/modules/business'] }),
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


function write(root, file, content) {
  const target = path.join(root, file);
  fs.mkdirSync(path.dirname(target), { recursive: true });
  fs.writeFileSync(target, content);
}

function migrationFixture(t, baseFiles = {}) {
  const temp = fs.mkdtempSync(path.join(os.tmpdir(), 'pantheon-drift-migrations-'));
  t.after(() => fs.rmSync(temp, { recursive: true, force: true }));
  const baseRoot = path.join(temp, 'base');
  const opsRoot = path.join(temp, 'ops');
  const foundation = 'backend/pkg/database/migrations/000031_foundation.up.sql';
  initGitRepo(baseRoot, { [foundation]: 'SELECT 31;\n', ...baseFiles });
  fixtureLock(opsRoot, { backend: ['backend/modules', 'backend/pkg'] });
  write(opsRoot, foundation, 'SELECT 31;\n');
  const migrations = ['up', 'down'].map((direction) => ({
    source: `backend/modules/business/cmdb/host/migrations/000032_backfill.${direction}.sql`,
    target: `backend/pkg/database/migrations/000032_backfill.${direction}.sql`,
  }));
  const manifest = {
    schemaVersion: 1,
    businessPaths: ['backend/modules/business/cmdb'],
    businessMigrations: migrations.map(({ source }) => source),
  };
  write(opsRoot, 'business-overlay.json', JSON.stringify(manifest));
  for (const { source, target } of migrations) {
    write(opsRoot, source, 'SELECT 32;\r\n');
    write(opsRoot, target, fs.readFileSync(path.join(opsRoot, source)));
  }
  return { baseRoot, opsRoot, foundation, manifest, migrations };
}

test('accepts only byte-identical derived business migration targets', (t) => {
  const { baseRoot, opsRoot } = migrationFixture(t);
  const result = runDriftCheck(opsRoot, baseRoot);
  assert.equal(result.status, 0, result.stderr);
  assert.match(result.stdout, /1 shared files identical to base/);
  assert.match(result.stdout, /2 derived business migrations verified/);
});

test('rejects tampered migration bytes even when the target is allowlisted', async (t) => {
  for (const direction of ['up', 'down']) {
    await t.test(direction, (t) => {
      const { baseRoot, opsRoot, migrations } = migrationFixture(t);
      const { target } = migrations.find((entry) => entry.target.endsWith(`.${direction}.sql`));
      write(opsRoot, target, 'SELECT 32;\n');
      write(opsRoot, '.foundation/allowed-drift.json', JSON.stringify({ [target]: 'must not bypass canonical SQL' }));
      const result = runDriftCheck(opsRoot, baseRoot);
      assert.equal(result.status, 1);
      assert.ok(result.stderr.includes(`Business migration target differs from source: ${target}`), result.stderr);
    });
  }
});

test('rejects undeclared targets even when the report calls them generated', (t) => {
  const { baseRoot, opsRoot } = migrationFixture(t);
  const undeclared = [
    'backend/pkg/database/migrations/000032_backfill_extra.up.sql',
    'backend/pkg/database/migrations/000033_undeclared.up.sql',
  ];
  for (const target of undeclared) write(opsRoot, target, 'SELECT 32;\r\n');
  write(opsRoot, '.business-overlay-report.json', JSON.stringify({ generatedFiles: undeclared }));
  const result = runDriftCheck(opsRoot, baseRoot);
  assert.equal(result.status, 1);
  for (const target of undeclared) {
    assert.ok(result.stderr.includes(`DRIFT_MISSING_IN_BASE ${target}`), result.stderr);
  }
});

test('continues to protect Base migrations listed in generatedFiles', (t) => {
  const { baseRoot, opsRoot, foundation } = migrationFixture(t);
  write(opsRoot, foundation, 'SELECT 99;\n');
  write(opsRoot, '.business-overlay-report.json', JSON.stringify({ generatedFiles: [foundation] }));
  const result = runDriftCheck(opsRoot, baseRoot);
  assert.equal(result.status, 1);
  assert.ok(result.stderr.includes(`DRIFT_MODIFIED ${foundation}`), result.stderr);
});

test('requires regular canonical migration source files', async (t) => {
  for (const direction of ['up', 'down']) {
    for (const kind of ['missing', 'directory']) {
      await t.test(`${direction} ${kind}`, (t) => {
        const { baseRoot, opsRoot, migrations } = migrationFixture(t);
        const { source } = migrations.find((entry) => entry.source.endsWith(`.${direction}.sql`));
        fs.unlinkSync(path.join(opsRoot, source));
        if (kind === 'directory') fs.mkdirSync(path.join(opsRoot, source));
        const result = runDriftCheck(opsRoot, baseRoot);
        assert.equal(result.status, 1);
        assert.ok(result.stderr.includes(`Business migration source must be a regular file: ${source}`), result.stderr);
      });
    }
  }
});

test('rejects a missing derived target', (t) => {
  const { baseRoot, opsRoot, migrations } = migrationFixture(t);
  const { target } = migrations[1];
  fs.unlinkSync(path.join(opsRoot, target));
  const result = runDriftCheck(opsRoot, baseRoot);
  assert.equal(result.status, 1);
  assert.ok(result.stderr.includes(`Business migration target must be a regular file: ${target}`), result.stderr);
});

test('rejects Base version collisions in the snapshot', async (t) => {
  for (const name of ['000032_backfill.up.sql', '32_future_base.down.sql']) {
    await t.test(name, (t) => {
      const collision = `backend/pkg/database/migrations/${name}`;
      const { baseRoot, opsRoot } = migrationFixture(t, { [collision]: 'SELECT 32;\r\n' });
      const result = runDriftCheck(opsRoot, baseRoot);
      assert.equal(result.status, 1);
      assert.match(result.stderr, /Business migration version conflicts with Base: 000032/);
    });
  }
});

test('cannot exempt generic product paths through migration declarations', async (t) => {
  for (const field of ['businessPaths', 'repositoryOverlayPaths']) {
    await t.test(field, (t) => {
      const { baseRoot, opsRoot, manifest } = migrationFixture(t);
      manifest[field] = [...(manifest[field] ?? []), 'backend/pkg/database'];
      write(opsRoot, 'business-overlay.json', JSON.stringify(manifest));
      const result = runDriftCheck(opsRoot, baseRoot);
      assert.equal(result.status, 1);
      assert.match(result.stderr, /cannot own (generic )?product path: backend\/pkg\/database/);
    });
  }
});

test('uses snapshot bytes and migration inventory nested inside a differing Ops HEAD', (t) => {
  const temp = fs.mkdtempSync(path.join(os.tmpdir(), 'pantheon-nested-snapshot-'));
  t.after(() => fs.rmSync(temp, { recursive: true, force: true }));
  const opsRoot = path.join(temp, 'ops');
  const shared = 'backend/pkg/shared.go';
  initGitRepo(opsRoot, { [shared]: 'package oldops\n', 'backend/pkg/database/migrations/000032_ops.up.sql': 'SELECT 32;\n' });
  const baseRoot = path.join(opsRoot, '.foundation/releases/pantheon-base-v0.18.4/repo');
  write(baseRoot, shared, 'package releasedbase\n');
  fixtureLock(opsRoot, { backend: ['backend/pkg'] });
  write(opsRoot, shared, 'package releasedbase\n');
  fs.unlinkSync(path.join(opsRoot, 'backend/pkg/database/migrations/000032_ops.up.sql'));
  let result = runDriftCheck(opsRoot, baseRoot);
  assert.equal(result.status, 0, result.stderr);
  write(opsRoot, shared, 'package oldops\n');
  result = runDriftCheck(opsRoot, baseRoot);
  assert.equal(result.status, 1, result.stderr);
  assert.match(result.stderr, /DRIFT_MODIFIED backend\/pkg\/shared.go/);
});

test('detects shared files deleted from the consumer', (t) => {
  const { baseRoot, opsRoot, foundation } = migrationFixture(t);
  fs.unlinkSync(path.join(opsRoot, foundation));
  const result = runDriftCheck(opsRoot, baseRoot);
  assert.equal(result.status, 1, result.stderr);
  assert.ok(result.stderr.includes(`DRIFT_MODIFIED ${foundation}`));
});

test('production drift command refuses local Base arguments', () => {
  const result = spawnSync(process.execPath, ['scripts/business-overlay/check-shared-drift.mjs', '--base', '../pantheon-base'], { encoding: 'utf8' });
  assert.equal(result.status, 1);
  assert.match(result.stderr, /Local Base or alternative lock arguments are not supported/);
});

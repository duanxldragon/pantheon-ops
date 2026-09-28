import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { spawnSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';

const TEST_DIR = path.dirname(fileURLToPath(import.meta.url));
const SCRIPT = path.resolve(TEST_DIR, '..', '..', 'scripts', 'harness', 'check-boundaries.mjs');

function createRepo() {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'boundary-check-'));
  const repoRoot = path.join(root, 'pantheon-base');
  fs.mkdirSync(repoRoot, { recursive: true });
  const write = (relative, content) => {
    const full = path.join(repoRoot, relative);
    fs.mkdirSync(path.dirname(full), { recursive: true });
    fs.writeFileSync(full, content);
  };
  return { root, repoRoot, write };
}

function run(root, extra = []) {
  return spawnSync(
    process.execPath,
    [SCRIPT, '--json', '--strict', '--root', root, '--repo', 'pantheon-base', ...extra],
    { encoding: 'utf8' },
  );
}

function runLoose(root, extra = []) {
  return spawnSync(
    process.execPath,
    [SCRIPT, '--json', '--root', root, '--repo', 'pantheon-base', ...extra],
    { encoding: 'utf8' },
  );
}

function parse(result) {
  return JSON.parse(result.stdout);
}

function writeBaseline(root, entries) {
  const baseline = path.join(root, 'config', 'boundary-baseline.json');
  fs.mkdirSync(path.dirname(baseline), { recursive: true });
  fs.writeFileSync(baseline, JSON.stringify({ entries }));
}

test('flags a platform production import of a system module', () => {
  const { root, write } = createRepo();
  write(
    'backend/modules/platform/routes.go',
    'package platform\n\nimport (\n\t_ "github.com/duanxldragon/pantheon-base/backend/modules/system/iam/user"\n)\n',
  );
  const result = run(root);
  assert.equal(result.status, 1, result.stdout);
  const body = parse(result);
  assert.equal(body.findingCount, 1);
  assert.match(body.results[0].findings[0].reason, /platform must not import system/);
});

test('exempts Go test files from production boundary rules', () => {
  const { root, write } = createRepo();
  write(
    'backend/modules/platform/routes_test.go',
    'package platform\n\nimport (\n\t_ "github.com/duanxldragon/pantheon-base/backend/modules/system/iam/user"\n)\n',
  );
  const result = run(root);
  assert.equal(result.status, 0, result.stdout);
  assert.equal(parse(result).findingCount, 0);
});

test('allows auth to use system api modules but flags internal component imports', () => {
  const { root, write } = createRepo();
  write('frontend/src/modules/auth/security/usesApi.ts', "import { getSettingGroup } from '../../../system/setting/api';\n");
  write('frontend/src/modules/auth/security/usesCss.ts', "import '../../../system/components/shared/list-page.css';\n");
  const result = run(root);
  assert.equal(result.status, 1, result.stdout);
  const body = parse(result);
  assert.equal(body.findingCount, 1);
  assert.match(body.results[0].findings[0].file, /usesCss\.ts$/);
});

test('a baselined finding does not fail strict mode', () => {
  const { root, write } = createRepo();
  write(
    'backend/modules/auth/login/login_service.go',
    'package login\n\nimport (\n\t_ "github.com/duanxldragon/pantheon-base/backend/modules/system/iam/user"\n)\n',
  );
  const baseline = path.join(root, 'config', 'boundary-baseline.json');
  fs.mkdirSync(path.dirname(baseline), { recursive: true });
  fs.writeFileSync(
    baseline,
    JSON.stringify({
      entries: [
        {
          file: 'backend/modules/auth/login/login_service.go',
          importPath: 'github.com/duanxldragon/pantheon-base/backend/modules/system/iam/user',
          reason: 'test fixture',
        },
      ],
    }),
  );
  const result = run(root, ['--baseline', 'config/boundary-baseline.json']);
  assert.equal(result.status, 0, result.stdout);
  const body = parse(result);
  assert.equal(body.findingCount, 0);
  assert.equal(body.baselinedCount, 1);
});

// A stale entry is not harmless bookkeeping: it still whitelists its
// file/import pair, so --strict fails on it (see check-boundaries.mjs help).
test('a stale baseline entry fails strict mode', () => {
  const { root, write } = createRepo();
  write('backend/modules/platform/routes.go', 'package platform\n');
  writeBaseline(root, [
    { file: 'backend/modules/platform/gone.go', importPath: 'github.com/x/backend/modules/system/iam/user', reason: 'no longer present' },
  ]);
  const result = run(root, ['--baseline', 'config/boundary-baseline.json']);
  assert.equal(result.status, 1, result.stdout);
  const body = parse(result);
  assert.equal(body.staleCount, 1);
  assert.match(body.results[0].warnings.join('\n'), /stale/);
});

test('a stale baseline entry only warns outside strict mode', () => {
  const { root, write } = createRepo();
  write('backend/modules/platform/routes.go', 'package platform\n');
  writeBaseline(root, [
    { file: 'backend/modules/platform/gone.go', importPath: 'github.com/x/backend/modules/system/iam/user', reason: 'no longer present' },
  ]);
  const result = runLoose(root, ['--baseline', 'config/boundary-baseline.json']);
  assert.equal(result.status, 0, result.stdout);
  assert.match(parse(result).results[0].warnings.join('\n'), /stale/);
});

test('a new violation alongside a baseline still fails strict mode', () => {
  const { root, write } = createRepo();
  write(
    'backend/modules/platform/routes.go',
    'package platform\n\nimport (\n\t_ "github.com/duanxldragon/pantheon-base/backend/modules/system/org/dept"\n)\n',
  );
  const baseline = path.join(root, 'config', 'boundary-baseline.json');
  fs.mkdirSync(path.dirname(baseline), { recursive: true });
  fs.writeFileSync(
    baseline,
    JSON.stringify({
      entries: [
        { file: 'backend/modules/auth/login/login_service.go', importPath: 'github.com/x/backend/modules/system/iam/user', reason: 'unrelated known debt' },
      ],
    }),
  );
  const result = run(root, ['--baseline', 'config/boundary-baseline.json']);
  assert.equal(result.status, 1, result.stdout);
  assert.equal(parse(result).findingCount, 1);
});

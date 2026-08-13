import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { execFileSync, spawnSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';

const TEST_DIR = path.dirname(fileURLToPath(import.meta.url));
const SCRIPT = path.resolve(TEST_DIR, '..', '..', 'scripts', 'harness', 'check-structure-contract.mjs');

function createFixture() {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'structure-check-'));
  execFileSync('git', ['-C', root, 'init', '-q']);
  const track = (relative, content = '// ok\n') => {
    const full = path.join(root, relative);
    fs.mkdirSync(path.dirname(full), { recursive: true });
    fs.writeFileSync(full, content);
    execFileSync('git', ['-C', root, 'add', relative]);
  };
  track('backend/modules/system/iam/user/user_service.go', 'package user\n');
  track('backend/go.mod', 'module example\n');
  track('frontend/src/modules/system/user/UserList.tsx');
  track('frontend/src/components/patterns/table/TableToolbar.tsx');
  track('frontend/src/hooks/usePagination.ts');
  track('frontend/package.json', '{}\n');
  return { root, track };
}

function run(root, extra = []) {
  return spawnSync(process.execPath, [SCRIPT, '--json', '--strict', '--root', root, ...extra], { encoding: 'utf8' });
}

test('passes on a conforming layout', () => {
  const { root } = createFixture();
  const result = run(root);
  assert.equal(result.status, 0, result.stdout);
  assert.equal(JSON.parse(result.stdout).findingCount, 0);
});

test('flags a Go file outside backend/', () => {
  const { root, track } = createFixture();
  track('scripts/helper.go', 'package main\n');
  const result = run(root);
  assert.equal(result.status, 1);
  assert.match(result.stdout, /go-placement/);
});

test('flags a non-snake_case Go file', () => {
  const { root, track } = createFixture();
  track('backend/modules/system/UserService.go', 'package system\n');
  const result = run(root);
  assert.equal(result.status, 1);
  assert.match(result.stdout, /go-file-naming/);
});

test('flags an unknown backend module domain', () => {
  const { root, track } = createFixture();
  track('backend/modules/reporting/report_service.go', 'package reporting\n');
  const result = run(root);
  assert.equal(result.status, 1);
  assert.match(result.stdout, /backend-module-domain/);
});

test('flags an unknown frontend src top-level directory', () => {
  const { root, track } = createFixture();
  track('frontend/src/utils/random.ts');
  const result = run(root);
  assert.equal(result.status, 1);
  assert.match(result.stdout, /frontend-src-root/);
});

test('flags an unknown frontend module domain', () => {
  const { root, track } = createFixture();
  track('frontend/src/modules/reporting/ReportList.tsx');
  const result = run(root);
  assert.equal(result.status, 1);
  assert.match(result.stdout, /frontend-module-domain/);
});

test('flags test files inside frontend/src', () => {
  const { root, track } = createFixture();
  track('frontend/src/modules/system/user/user.test.ts');
  const result = run(root);
  assert.equal(result.status, 1);
  assert.match(result.stdout, /no-tests-in-src/);
});

test('flags non-PascalCase shared component tsx', () => {
  const { root, track } = createFixture();
  // Distinct basename from every fixture file: case-insensitive filesystems
  // (Windows/macOS) would silently merge a camelCase twin of an existing
  // PascalCase path and the finding would never materialize.
  track('frontend/src/components/patterns/table/dataGrid.tsx');
  const result = run(root);
  assert.equal(result.status, 1);
  assert.match(result.stdout, /component-naming/);
});

test('flags a wrongly named hook', () => {
  const { root, track } = createFixture();
  track('frontend/src/hooks/paginationHelper.ts');
  const result = run(root);
  assert.equal(result.status, 1);
  assert.match(result.stdout, /hook-naming/);
});

test('flags tracked binaries', () => {
  const { root, track } = createFixture();
  track('backend/tests/server.exe', 'MZ');
  const result = run(root);
  assert.equal(result.status, 1);
  assert.match(result.stdout, /no-tracked-binaries/);
});

test('report-only mode exits 0 even with findings', () => {
  const { root, track } = createFixture();
  track('frontend/src/utils/random.ts');
  const result = spawnSync(process.execPath, [SCRIPT, '--root', root], { encoding: 'utf8' });
  assert.equal(result.status, 0);
  assert.match(result.stdout, /finding\(s\)/);
});

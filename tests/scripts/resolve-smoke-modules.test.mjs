import assert from 'node:assert/strict';
import { test } from 'node:test';

import { resolveSmokeModules } from '../../scripts/resolve-smoke-modules.mjs';

test('a single-module backend change selects only that module', () => {
  assert.deepEqual(
    resolveSmokeModules(['backend/modules/system/iam/role/role_service.go']),
    ['test:smoke:module:iam'],
  );
});

test('a single-module frontend change selects only that module', () => {
  assert.deepEqual(
    resolveSmokeModules(['frontend/src/modules/system/i18n/api/i18n.ts']),
    ['test:smoke:module:i18n'],
  );
});

test('a module spec change selects only that module', () => {
  assert.deepEqual(
    resolveSmokeModules(['frontend/tests/smoke/system/system-audit.spec.ts']),
    ['test:smoke:module:audit'],
  );
});

test('role-authorization.spec.ts belongs to iam', () => {
  assert.deepEqual(
    resolveSmokeModules(['frontend/tests/smoke/system/role-authorization.spec.ts']),
    ['test:smoke:module:iam'],
  );
});

test('multiple module changes are returned sorted and de-duplicated', () => {
  assert.deepEqual(
    resolveSmokeModules([
      'frontend/src/modules/system/org/api/dept.ts',
      'backend/modules/system/iam/user/user_service.go',
      'frontend/src/modules/system/org/api/post.ts',
    ]),
    ['test:smoke:module:iam', 'test:smoke:module:org'],
  );
});

test('a shared frontend change escalates to every module', () => {
  assert.deepEqual(
    resolveSmokeModules(['frontend/src/core/router/componentRegistry.ts']),
    ['test:smoke:module:all'],
  );
});

test('shared smoke scaffolding escalates to every module', () => {
  assert.deepEqual(
    resolveSmokeModules(['frontend/tests/smoke/system/system-pages-helpers.ts']),
    ['test:smoke:module:all'],
  );
});

test('a backend change outside the system modules escalates to every module', () => {
  assert.deepEqual(
    resolveSmokeModules(['backend/pkg/contracts/envelope.go']),
    ['test:smoke:module:all'],
  );
});

test('docs-only changes select nothing', () => {
  assert.deepEqual(
    resolveSmokeModules(['docs/designs/REPOSITORY_LAYOUT.md', 'README.md', '.harness/tasks/x/manifest.json']),
    [],
  );
});

test('a module change alongside docs still selects the module', () => {
  assert.deepEqual(
    resolveSmokeModules(['docs/guide.md', 'backend/modules/system/config/setting/setting.go']),
    ['test:smoke:module:config'],
  );
});

test('an unrecognised path escalates to every module (safe default)', () => {
  assert.deepEqual(resolveSmokeModules(['.github/workflows/quality.yml']), ['test:smoke:module:all']);
});

test('windows-style separators and leading ./ are normalised', () => {
  assert.deepEqual(
    resolveSmokeModules(['./backend\\modules\\system\\audit\\loginlog\\login_log.go']),
    ['test:smoke:module:audit'],
  );
});

test('an empty change set selects nothing', () => {
  assert.deepEqual(resolveSmokeModules([]), []);
  assert.deepEqual(resolveSmokeModules(['', '  ']), []);
});

import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import test from 'node:test';

import {
  collectGoConsts,
  extractField,
  goConstTable,
  recordUnresolvedConstRefs,
  takeUnresolvedGoConstRefs,
} from '../../frontend/scripts/check-menu-contract.mjs';

const BLOCK_FORM = [
  'package iam',
  '',
  'const (',
  '\tmenuKeyUser = "system.menu.user"',
  '\tmenuPathUser string = "/system/iam/users"',
  ')',
  '',
].join('\n');

const INLINE_FORM = [
  'package setting',
  '',
  'const settingModuleAuth = "system.auth"',
  'const settingValueDefault string = "[1,7,30]" // gofmt keeps the comment',
  '',
].join('\n');

function fixturePackage() {
  const pkgDir = fs.mkdtempSync(path.join(os.tmpdir(), 'pantheon-menu-contract-'));
  fs.writeFileSync(path.join(pkgDir, 'seed.go'), BLOCK_FORM, 'utf8');
  fs.writeFileSync(path.join(pkgDir, 'setting.go'), INLINE_FORM, 'utf8');
  fs.writeFileSync(
    path.join(pkgDir, 'setting_test.go'),
    'package setting\n\nconst testOnlyConst = "not a seed value"\n',
    'utf8',
  );
  return pkgDir;
}

function localScope(source) {
  return { filePath: 'fixture.go', consts: collectGoConsts(source) };
}

test('collectGoConsts collects const block entries', () => {
  const table = collectGoConsts(BLOCK_FORM);
  assert.equal(table.get('menuKeyUser'), 'system.menu.user');
  assert.equal(table.get('menuPathUser'), '/system/iam/users');
});

test('collectGoConsts collects single-line const declarations, with and without a type', () => {
  const table = collectGoConsts(INLINE_FORM);
  assert.equal(table.get('settingModuleAuth'), 'system.auth');
  assert.equal(table.get('settingValueDefault'), '[1,7,30]');
});

test('collectGoConsts does not treat runtime assignments as constants', () => {
  const table = collectGoConsts(
    ['package p', '', 'func f() {', '\tvar label = "runtime"', '\tk := "short"', '}'].join('\n'),
  );
  assert.equal(table.has('label'), false);
  assert.equal(table.has('k'), false);
});

test('extractField resolves a seed field through an inline-declared constant', () => {
  const scope = localScope(INLINE_FORM);
  assert.equal(extractField('{Module: settingModuleAuth}', 'Module', scope), 'system.auth');
});

test('extractField resolves a seed field through a block-declared constant', () => {
  const scope = localScope(BLOCK_FORM);
  assert.equal(extractField('{Key: menuKeyUser}', 'Key', scope), 'system.menu.user');
});

test('extractField keeps a literal field and an unrelated field name literal-free', () => {
  const scope = localScope(BLOCK_FORM);
  assert.equal(extractField('{Key: "system.menu.user"}', 'Key', scope), 'system.menu.user');
  assert.equal(extractField('{TitleKey: titleKeyUser}', 'Key', scope), '');
});

test('recordUnresolvedConstRefs reports a bare reference the const table cannot resolve', () => {
  takeUnresolvedGoConstRefs();
  const scope = localScope(BLOCK_FORM);
  recordUnresolvedConstRefs('{Key: keyMenuDepartment}', ['Key'], scope);
  assert.deepEqual(takeUnresolvedGoConstRefs(), [
    { filePath: 'fixture.go', fieldName: 'Key', identifier: 'keyMenuDepartment' },
  ]);
});

test('recordUnresolvedConstRefs ignores expressions and resolved references', () => {
  takeUnresolvedGoConstRefs();
  const scope = localScope(BLOCK_FORM);
  recordUnresolvedConstRefs(
    [
      '{Locale: item.Locale, Key: item.Key}',
      '{Key: strings.ToLower(key)}',
      '{Key: menuKeyUser}',
      '{Key: "system.menu.user"}',
    ].join('\n'),
    ['Key', 'Locale'],
    scope,
  );
  assert.deepEqual(takeUnresolvedGoConstRefs(), []);
});

test('takeUnresolvedGoConstRefs drains the accumulator', () => {
  takeUnresolvedGoConstRefs();
  const scope = localScope(BLOCK_FORM);
  recordUnresolvedConstRefs('{Key: keyMenuDepartment}', ['Key'], scope);
  assert.equal(takeUnresolvedGoConstRefs().length, 1);
  assert.deepEqual(takeUnresolvedGoConstRefs(), []);
});

test('goConstTable builds the scope from every non-test file of the package directory', () => {
  const pkgDir = fixturePackage();
  try {
    const scope = goConstTable(path.join(pkgDir, 'seed.go'));
    assert.equal(scope.filePath, path.join(pkgDir, 'seed.go'));
    assert.equal(scope.consts.get('menuKeyUser'), 'system.menu.user');
    assert.equal(scope.consts.get('settingModuleAuth'), 'system.auth');
    assert.equal(scope.consts.has('testOnlyConst'), false);
    assert.equal(extractField('{Module: settingModuleAuth}', 'Module', scope), 'system.auth');
  } finally {
    fs.rmSync(pkgDir, { recursive: true, force: true });
  }
});

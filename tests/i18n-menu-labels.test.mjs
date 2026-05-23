import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import vm from 'node:vm';

const repoRoot = path.resolve(import.meta.dirname, '..');
const frontendResourcesRoot = path.join(repoRoot, 'frontend', 'src', 'i18n', 'resources');
const backendSeedPath = path.join(repoRoot, 'backend', 'modules', 'system', 'i18n', 'seed_data.go');

const expectedEnglishMenuLabels = {
  'system.menu.access': 'Access & Permissions',
  'system.menu.org': 'Organizations',
  'system.menu.config': 'System Configuration',
  'system.menu.lowcode': 'Low-Code',
  'system.menu.security': 'Security & Audit',
  'system.menu.user': 'Users',
  'system.menu.role': 'Roles',
  'system.menu.permission': 'Permissions',
  'system.menu.menu': 'Menus',
  'system.menu.dept': 'Departments',
  'system.menu.post': 'Positions',
  'system.menu.dict': 'Dictionaries',
  'system.menu.setting': 'Settings',
  'system.menu.i18n': 'Internationalization',
  'system.menu.modules': 'Modules',
  'system.menu.generator': 'Code Generator',
  'system.menu.loginLog': 'Login Logs',
  'system.menu.session': 'Sessions',
  'system.menu.securityEvent': 'Security Events',
  'system.menu.operationLog': 'Operation Logs',
};

function loadTypeScriptObject(modulePath) {
  const source = fs.readFileSync(modulePath, 'utf8');
  const sanitized = source.replace(/export default\s+([A-Za-z0-9_$]+);?\s*$/m, 'module.exports = $1;');
  const context = {
    module: { exports: {} },
    exports: {},
  };
  vm.runInNewContext(sanitized, context, { filename: modulePath });
  return context.module.exports;
}

test('frontend English menu labels use concise but clear wording', () => {
  const resource = loadTypeScriptObject(path.join(frontendResourcesRoot, 'en-US.ts'));

  for (const [key, expectedValue] of Object.entries(expectedEnglishMenuLabels)) {
    assert.equal(resource[key], expectedValue, `expected ${key} to be "${expectedValue}"`);
  }
});

test('backend i18n seed keeps English menu labels aligned with frontend', () => {
  const source = fs.readFileSync(backendSeedPath, 'utf8');

  for (const [key, expectedValue] of Object.entries(expectedEnglishMenuLabels)) {
    assert.match(
      source,
      new RegExp(`Locale: "en-US"[\\s\\S]*?Key: "${key}", Value: "${expectedValue.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}"`),
      `expected backend seed to include ${key} -> ${expectedValue}`,
    );
  }
});

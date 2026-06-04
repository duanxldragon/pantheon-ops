import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import test from 'node:test';
import { loadResourceModule } from './lib/load-resource-module.mjs';

function writeModule(filePath, content) {
  fs.mkdirSync(path.dirname(filePath), { recursive: true });
  fs.writeFileSync(filePath, content, 'utf8');
}

test('loadResourceModule resolves imported default objects and spread assignments', () => {
  const tmpRoot = fs.mkdtempSync(path.join(os.tmpdir(), 'pantheon-resource-loader-'));
  const runtimePath = path.join(tmpRoot, 'runtime-fixes.zh-CN.ts');
  const localePath = path.join(tmpRoot, 'zh-CN.ts');

  writeModule(
    runtimePath,
    "const runtimeFixes = {\n  'auth.login.error': '登录失败',\n};\n\nexport default runtimeFixes;\n",
  );
  writeModule(
    localePath,
    "import runtimeFixes from './runtime-fixes.zh-CN';\n\nconst locale = {\n  'app.name': 'Pantheon Base',\n  ...runtimeFixes,\n};\n\nexport default locale;\n",
  );

  assert.deepEqual(loadResourceModule(localePath), {
    'app.name': 'Pantheon Base',
    'auth.login.error': '登录失败',
  });
});

test('loadResourceModule rejects executable expressions', () => {
  const tmpRoot = fs.mkdtempSync(path.join(os.tmpdir(), 'pantheon-resource-loader-'));
  const invalidPath = path.join(tmpRoot, 'invalid.ts');

  writeModule(
    invalidPath,
    "const locale = buildLocale();\n\nexport default locale;\n",
  );

  assert.throws(
    () => loadResourceModule(invalidPath),
    /Unsupported expression CallExpression/,
  );
});

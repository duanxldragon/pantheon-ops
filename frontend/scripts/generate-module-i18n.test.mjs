import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import test from 'node:test';
import { loadModuleLocales } from './generate-module-i18n.mjs';

function writeJson(filePath, payload) {
  fs.mkdirSync(path.dirname(filePath), { recursive: true });
  fs.writeFileSync(filePath, JSON.stringify(payload, null, 2), 'utf8');
}

function writeGenerated(filePath, variableName, payload) {
  fs.mkdirSync(path.dirname(filePath), { recursive: true });
  const body = Object.entries(payload)
    .map(([key, value]) => `  ${JSON.stringify(key)}: ${JSON.stringify(value)},`)
    .join('\n');
  fs.writeFileSync(
    filePath,
    `const ${variableName} = {\n${body}\n};\n\nexport default ${variableName};\n`,
    'utf8',
  );
}

test('loadModuleLocales preserves existing generated translations and falls back per key', () => {
  const tmpRoot = fs.mkdtempSync(path.join(os.tmpdir(), 'pantheon-i18n-'));
  const modulesDir = path.join(tmpRoot, 'modules');
  const generatedDir = path.join(tmpRoot, 'generated');

  writeJson(path.join(modulesDir, 'business', 'cmdb', 'locales', 'zh-CN.json'), {
    'business.cmdb.sample.title': '主机台账',
    'business.cmdb.sample.zhOnly': '仅中文说明',
  });
  writeJson(path.join(modulesDir, 'business', 'cmdb', 'locales', 'en-US.json'), {
    'business.cmdb.sample.title': 'Host Inventory',
  });

  writeGenerated(path.join(generatedDir, 'ja-JP.ts'), 'generatedjaJPFallback', {
    'business.cmdb.sample.title': 'ホスト台帳',
  });
  writeGenerated(path.join(generatedDir, 'ko-KR.ts'), 'generatedkoKRFallback', {});
  writeGenerated(path.join(generatedDir, 'fr-FR.ts'), 'generatedfrFRFallback', {});
  writeGenerated(path.join(generatedDir, 'zh-CN.ts'), 'generatedzhCNFallback', {});
  writeGenerated(path.join(generatedDir, 'en-US.ts'), 'generatedenUSFallback', {});

  const resources = loadModuleLocales({
    modulesRoot: modulesDir,
    generatedResourcesRoot: generatedDir,
    locales: ['zh-CN', 'en-US', 'ja-JP', 'ko-KR', 'fr-FR'],
  });

  assert.equal(resources['ja-JP']['business.cmdb.sample.title'], 'ホスト台帳');
  assert.equal(resources['ko-KR']['business.cmdb.sample.title'], 'Host Inventory');
  assert.equal(resources['fr-FR']['business.cmdb.sample.zhOnly'], '仅中文说明');
  assert.equal(resources['zh-CN']['business.cmdb.sample.title'], '主机台账');
});

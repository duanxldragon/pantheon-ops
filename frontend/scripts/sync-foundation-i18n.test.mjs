import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import test from 'node:test';
import {
  buildFoundationFallbackResources,
  syncFoundationI18n,
} from './sync-foundation-i18n.mjs';

function writeFile(filePath, content) {
  fs.mkdirSync(path.dirname(filePath), { recursive: true });
  fs.writeFileSync(filePath, content, 'utf8');
}

test('foundation i18n sync derives only missing static keys from the release resource', () => {
  const tmpRoot = fs.mkdtempSync(path.join(os.tmpdir(), 'pantheon-foundation-i18n-'));
  const sourceRoot = path.join(tmpRoot, 'src');
  const resourcesRoot = path.join(sourceRoot, 'i18n', 'resources');
  const foundationRoot = path.join(resourcesRoot, 'foundation');
  const builtinResourcePath = path.join(tmpRoot, 'builtin_locale_resources.json');
  const locales = ['zh-CN', 'en-US'];

  writeFile(
    path.join(sourceRoot, 'Example.tsx'),
    "t('common.existing');\ni18n.t(\"system.foundation.only\");\nt(dynamicKey);\n",
  );
  writeFile(
    path.join(resourcesRoot, 'zh-CN.ts'),
    "export default { 'common.existing': '已有文案' };\n",
  );
  writeFile(
    path.join(resourcesRoot, 'en-US.ts'),
    "export default { 'common.existing': 'Existing copy' };\n",
  );
  writeFile(
    builtinResourcePath,
    `${JSON.stringify({
      'zh-CN': {
        'common.existing': '不应覆盖',
        'system.foundation.only': '底座文案',
      },
      'en-US': {
        'common.existing': 'Must not override',
        'system.foundation.only': 'Foundation copy',
      },
    })}\n`,
  );

  const resources = buildFoundationFallbackResources({
    sourceRoot,
    fallbackResourcesRoot: resourcesRoot,
    builtinResourcePath,
    locales,
  });
  assert.deepEqual(resources['zh-CN'], { 'system.foundation.only': '底座文案' });
  assert.deepEqual(resources['en-US'], { 'system.foundation.only': 'Foundation copy' });

  assert.throws(
    () =>
      syncFoundationI18n({
        sourceRoot,
        fallbackResourcesRoot: resourcesRoot,
        foundationResourcesRoot: foundationRoot,
        builtinResourcePath,
        locales,
        checkOnly: true,
      }),
    /Foundation i18n fallback resources are out of date/,
  );

  const applied = syncFoundationI18n({
    sourceRoot,
    fallbackResourcesRoot: resourcesRoot,
    foundationResourcesRoot: foundationRoot,
    builtinResourcePath,
    locales,
  });
  assert.equal(applied.changes.length, 2);
  assert.doesNotThrow(() =>
    syncFoundationI18n({
      sourceRoot,
      fallbackResourcesRoot: resourcesRoot,
      foundationResourcesRoot: foundationRoot,
      builtinResourcePath,
      locales,
      checkOnly: true,
    }),
  );
});

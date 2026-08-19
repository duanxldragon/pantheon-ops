import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import test from 'node:test';

import { loadResourceModule } from '../../frontend/scripts/check-i18n-generated-scope.mjs';

function withFixture(callback) {
  const repoRoot = fs.mkdtempSync(path.join(os.tmpdir(), 'pantheon-i18n-generated-scope-'));
  try {
    const resourcesRoot = path.join(repoRoot, 'frontend', 'src', 'i18n', 'resources');
    fs.mkdirSync(resourcesRoot, { recursive: true });
    callback({ repoRoot, resourcesRoot });
  } finally {
    fs.rmSync(repoRoot, { recursive: true, force: true });
  }
}

test('loadResourceModule resolves default imports used by derived locale modules', () => {
  withFixture(({ resourcesRoot }) => {
    fs.writeFileSync(
      path.join(resourcesRoot, 'locale-utils.ts'),
      [
        'const buildLocaleFromBase = (base, values) => {',
        '  const keys = Object.keys(base);',
        '  return Object.fromEntries(keys.map((key, index) => [key, values[index]]));',
        '};',
        'export default buildLocaleFromBase;',
        '',
      ].join('\n'),
      'utf8',
    );
    fs.writeFileSync(
      path.join(resourcesRoot, 'zh-CN.ts'),
      "const zhCNFallback = { 'app.name': '盘古', 'app.notice.title': '通知中心' };\nexport default zhCNFallback;\n",
      'utf8',
    );
    fs.writeFileSync(
      path.join(resourcesRoot, 'ja-JP.ts'),
      [
        "import zhCNFallback from './zh-CN';",
        "import buildLocaleFromBase from './locale-utils';",
        'const jaJPFallback = buildLocaleFromBase(zhCNFallback, [',
        "  'Pantheon Base',",
        "  '通知センター',",
        ']);',
        'export default jaJPFallback;',
        '',
      ].join('\n'),
      'utf8',
    );

    const resource = loadResourceModule(path.join(resourcesRoot, 'ja-JP.ts'));
    assert.deepEqual({ ...resource }, {
      'app.name': 'Pantheon Base',
      'app.notice.title': '通知センター',
    });
  });
});

test('locale-utils rebuilds a full locale map from base key order', () => {
  const helperPath = path.resolve(
    process.cwd(),
    'frontend',
    'src',
    'i18n',
    'resources',
    'locale-utils.js',
  );
  const buildLocaleFromBase = loadResourceModule(helperPath);
  const resource = buildLocaleFromBase(
    {
      'app.name': '盘古',
      'app.notice.title': '通知中心',
      'app.preference.title': '平台偏好',
    },
    ['Pantheon Base', 'Notification Center', 'Platform Preferences'],
    'en-US',
  );

  assert.deepEqual({ ...resource }, {
    'app.name': 'Pantheon Base',
    'app.notice.title': 'Notification Center',
    'app.preference.title': 'Platform Preferences',
  });
});

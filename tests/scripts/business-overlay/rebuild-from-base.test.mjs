import assert from 'node:assert/strict';
import crypto from 'node:crypto';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { spawnSync } from 'node:child_process';
import test from 'node:test';
import { fileURLToPath, pathToFileURL } from 'node:url';

import { rebuildFromBase } from '../../../scripts/business-overlay/rebuild-from-base.mjs';

const rebuildModulePath = pathToFileURL(
  path.resolve('scripts/business-overlay/rebuild-from-base.mjs'),
).href;
const { validateReleaseBundleChecksum } = await import(rebuildModulePath);

function git(root, args) {
  const result = spawnSync('git', args, { cwd: root, encoding: 'utf8' });
  assert.equal(result.status, 0, result.stderr);
}

function write(root, file, content) {
  const target = path.join(root, file);
  fs.mkdirSync(path.dirname(target), { recursive: true });
  fs.writeFileSync(target, content, 'utf8');
}

function checkOverlay(root) {
  return spawnSync(process.execPath, [
    path.resolve('scripts/business-overlay/check-business-overlay.mjs'),
    '--root',
    root,
  ], { cwd: path.resolve('.'), encoding: 'utf8' });
}

function fixtureRepository(root, files) {
  fs.mkdirSync(root, { recursive: true });
  for (const [file, content] of Object.entries(files)) write(root, file, content);
  git(root, ['init']);
  git(root, ['config', 'user.email', 'test@example.com']);
  git(root, ['config', 'user.name', 'Test']);
  git(root, ['add', '.']);
  git(root, ['commit', '-m', 'fixture']);
}

test('rebuilds a deterministic Base snapshot with generated business registries', () => {
  const temp = fs.mkdtempSync(path.join(os.tmpdir(), 'pantheon-overlay-'));
  const baseRoot = path.join(temp, 'base');
  const opsRoot = path.join(temp, 'ops');
  const targetRoot = path.join(temp, 'target');
  try {
    fixtureRepository(baseRoot, {
      'package.json': '{"scripts":{}}\n',
      'frontend/package.json': '{"scripts":{}}\n',
      'frontend/tests/smoke/README.md': '# Smoke\n',
      'frontend/scripts/cleanup-generated-modules.mjs': [
        "import path from 'node:path';",
        "const repoRoot = '';",
        "const I18N_LOCALES = [];",
        'function removeGeneratedSubdirs(parentDir, repoBase, trackedFiles) {',
        '  for (const entry of []) {',
        "    const targetPath = path.join(parentDir, entry.name);",
        '    if (hasTrackedDescendant(targetPath, repoBase, trackedFiles)) {}',
        '  }',
        '}',
        'function cleanup(registryFiles, repoBase, trackedFiles) {',
        '  for (const [key, filePath] of Object.entries(registryFiles)) {',
        '    const relative = normalizePath(path.relative(repoBase, filePath));',
        '    const baseline = null;',
        '  }',
        '}',
      ].join('\n'),
      'backend/go.mod': 'module pantheon-base\n\ngo 1.24.0\n',
      'backend/go.sum': '',
      'backend/modules/business/business.go': 'package business\nfunc InitBusinessModules(r any, db any) {\n\tInitGeneratedBusinessModules(r, db)\n}\n',
      'backend/modules/business/generated_registry.go': 'package business\nfunc InitGeneratedBusinessModules(r any, db any) {}\n',
      'backend/modules/system/iam/menu/component_registry.go': 'package iam\nvar registeredMenuComponentKeys = mergeMenuComponentKeys(staticRegisteredMenuComponentKeys, generatedMenuComponentKeys)\n',
      'frontend/src/core/router/modules.ts': "import { generatedBusinessModules } from '../../modules/generated/business';\nexport const businessModules: ModuleConfig[] = [...generatedBusinessModules];\n",
      'frontend/src/core/router/componentRegistry.ts': "import { generatedComponentRegistry } from './generatedComponentRegistry';\nconst componentRegistry = {\n  ...staticComponentRegistry,\n  ...generatedComponentRegistry,\n};\n",
      'frontend/src/i18n/resources/generated/zh-CN.ts': 'export default {};\n',
      'frontend/src/i18n/resources/zh-CN.ts': 'const zhCNFallback = {};\nexport default zhCNFallback;\n',
      'frontend/src/i18n/resources/en-US.ts': 'const enUSFallback = {};\nexport default enUSFallback;\n',
      'docs/base-only.md': 'base docs\n',
      'scripts/foundation-release/publish.mjs': 'throw new Error();\n',
    });
    const manifest = {
      schemaVersion: 1,
      base: { source: '../base', module: 'pantheon-base' },
      sourceModule: 'pantheon-ops/backend',
      goDependencies: { 'gorm.io/datatypes': 'v1.2.7' },
      replaceRoots: ['docs'],
      baseExcludedPaths: ['scripts/foundation-release'],
      businessPaths: ['backend/modules/business/cmdb', 'frontend/src/modules/business/cmdb'],
      repositoryOverlayPaths: ['scripts/business-overlay'],
      backendModules: [{ alias: 'cmdb', importPath: 'pantheon-base/modules/business/cmdb', init: 'InitCmdbModule' }],
      frontendModules: [{ export: 'CmdbModule', importPath: './business/cmdb' }],
      components: [{ key: 'business/cmdb/CmdbList', importPath: '../../modules/business/cmdb/CmdbList' }],
      businessSmokeScripts: ['test:smoke:business:cmdb'],
    };
    fixtureRepository(opsRoot, {
      'business-overlay.json': `${JSON.stringify(manifest)}\n`,
      'package.json': '{"scripts":{}}\n',
      'backend/go.sum': 'gorm.io/datatypes v1.2.7 h1:test\ngorm.io/datatypes v1.2.7/go.mod h1:testmod\n',
      'frontend/package.json': '{"scripts":{"test:smoke:business:cmdb":"playwright test tests/smoke/business/cmdb/pages.spec.ts"}}\n',
      'backend/modules/business/cmdb/module.go': 'package cmdb\nimport "pantheon-ops/backend/pkg/contracts"\n',
      'backend/modules/business/generated_registry.go': 'package business\n',
      'backend/modules/system/iam/menu/generated_component_registry.go': 'package menu\n',
      'frontend/src/modules/business/cmdb/index.ts': 'export const CmdbModule = {};\n',
      'frontend/src/modules/generated/business.ts': 'export {};\n',
      'frontend/src/core/router/generatedComponentRegistry.ts': 'export {};\n',
      'frontend/src/modules/business/cmdb/locales/zh-CN.json': '{"business.cmdb.title":"CMDB"}\n',
      'frontend/tests/smoke/business/cmdb/pages.spec.ts': 'export {};\n',
      'scripts/business-overlay/rebuild-from-base.mjs': '// fixture\n',
    });

    const first = rebuildFromBase({ opsRoot, baseRoot, targetRoot });
    const firstReport = fs.readFileSync(path.join(targetRoot, '.business-overlay-report.json'), 'utf8');
    assert.match(fs.readFileSync(path.join(targetRoot, 'backend/modules/business/cmdb/module.go'), 'utf8'), /pantheon-base\/pkg\/contracts/u);
    assert.match(fs.readFileSync(path.join(targetRoot, 'backend/modules/business/business_overlay_registry.go'), 'utf8'), /cmdb\.InitCmdbModule/u);
    const businessOverlay = fs.readFileSync(path.join(targetRoot, 'frontend/src/modules/businessOverlay.ts'), 'utf8');
    assert.match(businessOverlay, /from '\.\.\/core\/router\/types'/u);
    assert.match(businessOverlay, /from '\.\/business\/cmdb'/u);
    assert.match(fs.readFileSync(path.join(targetRoot, 'frontend/src/core/router/businessOverlayComponentRegistry.ts'), 'utf8'), /business\/cmdb\/CmdbList/u);
    assert.match(fs.readFileSync(path.join(targetRoot, 'frontend/src/i18n/resources/business/zh-CN.ts'), 'utf8'), /business\.cmdb\.title/u);
    assert.match(fs.readFileSync(path.join(targetRoot, 'frontend/src/i18n/resources/zh-CN.ts'), 'utf8'), /\.\.\.businessFallback/u);
    assert.match(fs.readFileSync(path.join(targetRoot, 'backend/go.mod'), 'utf8'), /gorm\.io\/datatypes v1\.2\.7/u);
    assert.match(fs.readFileSync(path.join(targetRoot, 'frontend/tests/smoke/README.md'), 'utf8'), /`test:smoke:business:cmdb` -> `business\/cmdb\/pages\.spec\.ts`/u);
    assert.match(fs.readFileSync(path.join(targetRoot, 'frontend/scripts/cleanup-generated-modules.mjs'), 'utf8'), /BUSINESS_OVERLAY_OWNED_FILES/u);
    assert.equal(
      fs.readFileSync(path.join(targetRoot, 'frontend/scripts/cleanup-generated-modules.mjs'), 'utf8')
        .match(/hasBusinessOverlayOwnedDescendant\(targetPath\)/gu)?.length,
      2,
    );
    assert.match(fs.readFileSync(path.join(targetRoot, 'backend/modules/business/business.go'), 'utf8'), /initOverlayBusinessModules/u);
    assert.match(fs.readFileSync(path.join(targetRoot, 'frontend/src/core/router/modules.ts'), 'utf8'), /\.\.\.overlayBusinessModules/u);
    assert.match(fs.readFileSync(path.join(targetRoot, 'frontend/src/core/router/componentRegistry.ts'), 'utf8'), /\.\.\.businessOverlayComponentRegistry/u);
    assert.match(fs.readFileSync(path.join(targetRoot, 'backend/modules/system/iam/menu/component_registry.go'), 'utf8'), /businessOverlayMenuComponentKeys/u);
    const overlayRegistry = fs.readFileSync(path.join(targetRoot, 'backend/modules/business/business_overlay_registry.go'), 'utf8');
    write(targetRoot, 'backend/modules/business/generated_registry.go', 'package business\n// temporary orderqa registry\n');
    assert.equal(fs.readFileSync(path.join(targetRoot, 'backend/modules/business/business_overlay_registry.go'), 'utf8'), overlayRegistry);
    write(targetRoot, 'backend/modules/business/generated_registry.go', 'package business\nfunc InitGeneratedBusinessModules(r any, db any) {}\n');
    assert.equal(fs.readFileSync(path.join(targetRoot, 'backend/modules/business/business_overlay_registry.go'), 'utf8'), overlayRegistry);
    assert.equal(fs.existsSync(path.join(targetRoot, 'docs/base-only.md')), false);
    assert.equal(fs.existsSync(path.join(targetRoot, 'scripts/foundation-release/publish.mjs')), false);
    assert.equal(checkOverlay(opsRoot).status, 0, checkOverlay(opsRoot).stderr);
    assert.equal(checkOverlay(targetRoot).status, 0, checkOverlay(targetRoot).stderr);
    const second = rebuildFromBase({ opsRoot, baseRoot, targetRoot });
    assert.equal(fs.readFileSync(path.join(targetRoot, '.business-overlay-report.json'), 'utf8'), firstReport);
    assert.deepEqual(second.report, first.report);
  } finally {
    fs.rmSync(temp, { recursive: true, force: true });
  }
});

test('rejects a business path that overwrites a Base hook file', () => {
  const temp = fs.mkdtempSync(path.join(os.tmpdir(), 'pantheon-overlay-overwrite-'));
  const baseRoot = path.join(temp, 'base');
  const opsRoot = path.join(temp, 'ops');
  try {
    fixtureRepository(baseRoot, {
      'backend/modules/business/hook.go': 'package business\nvar hookBase = true\n',
    });
    const manifest = {
      schemaVersion: 1,
      base: { source: '../base', module: 'pantheon-base' },
      sourceModule: 'pantheon-ops/backend',
      businessPaths: ['backend/modules/business/hook.go'],
      repositoryOverlayPaths: [],
      backendModules: [], frontendModules: [], components: [], businessSmokeScripts: [],
    };
    fixtureRepository(opsRoot, {
      'business-overlay.json': `${JSON.stringify(manifest)}\n`,
      'backend/modules/business/hook.go': 'package business\nvar hookOps = false\n',
    });
    assert.throws(() => rebuildFromBase({ opsRoot, baseRoot }), /Business overlay overwrites Base files/u);
  } finally {
    fs.rmSync(temp, { recursive: true, force: true });
  }
});

test('rejects a manifest that claims a generic product path', () => {
  const temp = fs.mkdtempSync(path.join(os.tmpdir(), 'pantheon-overlay-reject-'));
  const baseRoot = path.join(temp, 'base');
  const opsRoot = path.join(temp, 'ops');
  try {
    fixtureRepository(baseRoot, { 'package.json': '{"scripts":{}}', 'frontend/package.json': '{"scripts":{}}' });
    const manifest = {
      schemaVersion: 1,
      base: { source: '../base', module: 'pantheon-base' },
      sourceModule: 'pantheon-ops/backend',
      businessPaths: ['frontend/src/shared/local.ts'],
      repositoryOverlayPaths: [],
      backendModules: [], frontendModules: [], components: [], businessSmokeScripts: [],
    };
    fixtureRepository(opsRoot, {
      'business-overlay.json': `${JSON.stringify(manifest)}\n`,
      'package.json': '{"scripts":{}}',
      'frontend/package.json': '{"scripts":{}}',
      'frontend/src/shared/local.ts': 'export {};\n',
    });
    assert.throws(() => rebuildFromBase({ opsRoot, baseRoot }), /cannot own generic product path/u);
  } finally {
    fs.rmSync(temp, { recursive: true, force: true });
  }
});

test('validateReleaseBundleChecksum accepts a matching checksum', () => {
  const temp = fs.mkdtempSync(path.join(os.tmpdir(), 'pantheon-checksum-match-'));
  try {
    const bundlePath = path.join(temp, 'foundation-release-pantheon-base-v0.10.22.tgz');
    fs.writeFileSync(bundlePath, 'release bundle content', 'utf8');
    const expectedChecksum = crypto.createHash('sha256').update(fs.readFileSync(bundlePath)).digest('hex');
    assert.doesNotThrow(() => validateReleaseBundleChecksum(bundlePath, expectedChecksum));
  } finally {
    fs.rmSync(temp, { recursive: true, force: true });
  }
});

test('validateReleaseBundleChecksum rejects a mismatched checksum', () => {
  const temp = fs.mkdtempSync(path.join(os.tmpdir(), 'pantheon-checksum-mismatch-'));
  try {
    const bundlePath = path.join(temp, 'foundation-release-pantheon-base-v0.10.22.tgz');
    fs.writeFileSync(bundlePath, 'release bundle content', 'utf8');
    const wrongChecksum = '0'.repeat(64);
    assert.throws(
      () => validateReleaseBundleChecksum(bundlePath, wrongChecksum),
      /release bundle checksum mismatch/u,
    );
  } finally {
    fs.rmSync(temp, { recursive: true, force: true });
  }
});

test('validateReleaseBundleChecksum skips validation when no checksum is provided', () => {
  const temp = fs.mkdtempSync(path.join(os.tmpdir(), 'pantheon-checksum-skip-'));
  try {
    const bundlePath = path.join(temp, 'foundation-release-pantheon-base-v0.10.22.tgz');
    fs.writeFileSync(bundlePath, 'release bundle content', 'utf8');
    assert.doesNotThrow(() => validateReleaseBundleChecksum(bundlePath, undefined));
    assert.doesNotThrow(() => validateReleaseBundleChecksum(bundlePath, null));
    assert.doesNotThrow(() => validateReleaseBundleChecksum(bundlePath, ''));
  } finally {
    fs.rmSync(temp, { recursive: true, force: true });
  }
});

test('release bundle checksum validation rejects tampered bundles via lock file path', () => {
  const temp = fs.mkdtempSync(path.join(os.tmpdir(), 'pantheon-lock-checksum-'));
  const baseRoot = path.join(temp, 'base');
  const opsRoot = path.join(temp, 'ops');
  try {
    fixtureRepository(baseRoot, {
      'package.json': '{"scripts":{}}\n',
      'frontend/package.json': '{"scripts":{}}\n',
      'frontend/tests/smoke/README.md': '# Smoke\n',
      'frontend/scripts/cleanup-generated-modules.mjs': [
        "import path from 'node:path';",
        "const repoRoot = '';",
        "const I18N_LOCALES = [];",
        'function removeGeneratedSubdirs(parentDir, repoBase, trackedFiles) {}',
        'function cleanup(registryFiles, repoBase, trackedFiles) {}',
      ].join('\n'),
      'backend/go.mod': 'module pantheon-base\n\ngo 1.24.0\n',
      'backend/go.sum': '',
      'backend/modules/business/business.go': 'package business\nfunc InitBusinessModules(r any, db any) {}\n',
      'backend/modules/system/iam/menu/component_registry.go': 'package iam\nvar registeredMenuComponentKeys = mergeMenuComponentKeys(staticRegisteredMenuComponentKeys)\n',
      'frontend/src/core/router/modules.ts': "import { generatedBusinessModules } from '../../modules/generated/business';\nexport const businessModules: ModuleConfig[] = [...generatedBusinessModules];\n",
      'frontend/src/core/router/componentRegistry.ts': "import { generatedComponentRegistry } from './generatedComponentRegistry';\nconst componentRegistry = { ...staticComponentRegistry, ...generatedComponentRegistry };\n",
      'frontend/src/i18n/resources/generated/zh-CN.ts': 'export default {};\n',
      'frontend/src/i18n/resources/zh-CN.ts': 'const zhCNFallback = {};\nexport default zhCNFallback;\n',
      'frontend/src/i18n/resources/en-US.ts': 'const enUSFallback = {};\nexport default enUSFallback;\n',
    });

    // Create a release bundle .tgz in the release directory.
    const releaseVersion = 'pantheon-base-v0.10.22';
    const releaseDir = path.join(opsRoot, '.foundation', 'releases', releaseVersion);
    fs.mkdirSync(releaseDir, { recursive: true });
    const bundleAssetName = `foundation-release-${releaseVersion}.tgz`;
    const bundlePath = path.join(releaseDir, bundleAssetName);
    fs.writeFileSync(bundlePath, 'mock release bundle content', 'utf8');
    const bundleChecksum = crypto.createHash('sha256').update(fs.readFileSync(bundlePath)).digest('hex');

    // Create a repo.tar in the release directory.
    const repoDir = path.join(releaseDir, 'repo');
    fs.mkdirSync(repoDir, { recursive: true });
    // Copy all base files into repo/ to simulate extracted repo.tar.
    const baseFiles = fs.readdirSync(baseRoot, { withFileTypes: true });
    for (const entry of baseFiles) {
      if (entry.name === '.git') continue;
      const src = path.join(baseRoot, entry.name);
      const dst = path.join(repoDir, entry.name);
      fs.cpSync(src, dst, { recursive: true });
    }
    // Also create a repo.tar (needed by resolveBaseFromLock validation).
    const repoTarPath = path.join(releaseDir, 'repo.tar');
    spawnSync('git', ['archive', '--format=tar', '--output', repoTarPath, 'HEAD'], {
      cwd: baseRoot, encoding: 'utf8',
    });
    const repoTarChecksum = crypto.createHash('sha256').update(fs.readFileSync(repoTarPath)).digest('hex');

    const lockData = {
      schemaVersion: 1,
      baseRepo: 'duanxldragon/pantheon-base',
      releaseLine: 'release/0.10',
      releaseVersion,
      baseCommit: 'deadbeefdeadbeefdeadbeefdeadbeefdeadbeef',
      consumerMode: 'foundation-release-consumer',
      releaseArtifact: {
        githubRepo: 'duanxldragon/pantheon-base',
        assetName: bundleAssetName,
        localPath: `.foundation/releases/${releaseVersion}`,
        checksum: bundleChecksum,
      },
      repoSnapshot: {
        assetName: 'repo.tar',
        sha256: repoTarChecksum,
        baseCommit: 'deadbeefdeadbeefdeadbeefdeadbeefdeadbeef',
        generatedFrom: 'git-archive',
      },
    };
    fs.writeFileSync(path.join(opsRoot, 'foundation-release.lock.json'), `${JSON.stringify(lockData, null, 2)}\n`, 'utf8');
    fixtureRepository(opsRoot, {
      'business-overlay.json': JSON.stringify({
        schemaVersion: 1,
        base: { source: '../base', module: 'pantheon-base' },
        sourceModule: 'pantheon-ops/backend',
        businessPaths: [],
        repositoryOverlayPaths: [],
        backendModules: [], frontendModules: [], components: [], businessSmokeScripts: [],
      }),
      'foundation-release.lock.json': `${JSON.stringify(lockData, null, 2)}\n`,
    });

    // Tamper with the bundle after computing its checksum.
    fs.writeFileSync(bundlePath, 'tampered release bundle', 'utf8');
    assert.throws(
      () => rebuildFromBase({ opsRoot, targetRoot: path.join(temp, 'target-fail') }),
      /release bundle checksum mismatch/u,
    );
  } finally {
    fs.rmSync(temp, { recursive: true, force: true });
  }
});

test('refuses to consume a live base tree when no lock snapshot is present', () => {
  const temp = fs.mkdtempSync(path.join(os.tmpdir(), 'pantheon-overlay-nolock-'));
  const opsRoot = path.join(temp, 'ops');
  try {
    fixtureRepository(opsRoot, {
      'business-overlay.json': JSON.stringify({
        schemaVersion: 1,
        base: { module: 'pantheon-base' },
        sourceModule: 'pantheon-ops/backend',
        businessPaths: [],
        repositoryOverlayPaths: [],
        backendModules: [], frontendModules: [], components: [], businessSmokeScripts: [],
      }),
    });
    assert.throws(
      () => rebuildFromBase({ opsRoot }),
      /published GitHub release, not a local/u,
    );
  } finally {
    fs.rmSync(temp, { recursive: true, force: true });
  }
});

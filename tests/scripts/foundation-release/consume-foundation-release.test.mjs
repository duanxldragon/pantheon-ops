import assert from 'node:assert/strict';
import crypto from 'node:crypto';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { spawnSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';
import test from 'node:test';
import {
  computeFileSha256,
  computeReleaseTreeSha256,
} from '../../../scripts/foundation-release/shared-foundation-rules.mjs';

const currentFilePath = fileURLToPath(import.meta.url);
const repoRoot = path.resolve(path.dirname(currentFilePath), '..', '..', '..');
const scriptPath = path.join(repoRoot, 'scripts', 'foundation-release', 'consume-foundation-release.mjs');

function withTempDir(callback) {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'pantheon-foundation-consumer-'));
  try {
    callback(root);
  } finally {
    fs.rmSync(root, { recursive: true, force: true });
  }
}

function writeJson(filePath, value) {
  fs.mkdirSync(path.dirname(filePath), { recursive: true });
  fs.writeFileSync(filePath, `${JSON.stringify(value, null, 2)}\n`, 'utf8');
}

function writeText(filePath, value) {
  fs.mkdirSync(path.dirname(filePath), { recursive: true });
  fs.writeFileSync(filePath, value, 'utf8');
}

function writeTestVerificationMarker(args) {
  const bundleIndex = args.indexOf('--bundle');
  if (bundleIndex < 0) return;
  const releaseRoot = path.resolve(args[bundleIndex + 1]);
  const manifest = JSON.parse(fs.readFileSync(path.join(releaseRoot, 'manifest.json'), 'utf8'));
  writeJson(path.join(releaseRoot, '.foundation-release-verified.json'), {
    schemaVersion: 1,
    releaseVersion: manifest.releaseVersion,
    baseCommit: manifest.baseCommit,
    archiveAssetName: `foundation-release-${manifest.releaseVersion}.tgz`,
    archiveSha256: crypto.createHash('sha256').update(manifest.releaseVersion).digest('hex'),
    manifestSha256: computeFileSha256(path.join(releaseRoot, 'manifest.json')),
    releaseTreeSha256: computeReleaseTreeSha256(releaseRoot),
    verifiedAt: '2026-08-05T00:00:00.000Z',
  });
}

function runRawScript(args, cwd) {
  return spawnSync(process.execPath, [scriptPath, ...args], {
    cwd,
    encoding: 'utf8',
  });
}

function runScript(args, cwd) {
  writeTestVerificationMarker(args);
  return runRawScript(args, cwd);
}

function runNpmApply(args) {
  writeTestVerificationMarker(args);
  return spawnSync('npm', ['run', 'upgrade:foundation:apply', '--', ...args], {
    cwd: repoRoot,
    encoding: 'utf8',
    shell: process.platform === 'win32',
  });
}

function createFixture(root) {
  const bundleRoot = path.join(root, 'release-bundle');
  const manifestPath = path.join(bundleRoot, 'manifest.json');
  const bundlePath = path.join(bundleRoot, 'bundle');
  const opsRoot = path.join(root, 'pantheon-ops');

  writeJson(manifestPath, {
    releaseVersion: 'base-v0.8.0',
    releaseLine: 'release/0.8',
    baseCommit: 'deadbeefdeadbeefdeadbeefdeadbeefdeadbeef',
    sourceRepo: 'pantheon-base',
    consumerMode: 'foundation-release-consumer',
    baseGoModule: 'pantheon-base',
    sharedPaths: {
      backend: ['backend/pkg'],
    },
  });
  writeJson(path.join(opsRoot, 'foundation-release.lock.json'), {
    schemaVersion: 1,
    baseRepo: '../pantheon-base',
    releaseLine: 'release/0.7',
    releaseVersion: 'base-v0.7.9',
    baseCommit: 'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa',
    consumerMode: 'foundation-release-consumer',
    releaseArtifact: {
      githubRepo: 'duanxldragon/pantheon-base',
      assetName: 'foundation-release-base-v0.7.9.tgz',
      localPath: '.foundation/releases/base-v0.7.9',
    },
    sharedPaths: {
      backend: ['backend/pkg'],
    },
  });
  writeJson(path.join(bundlePath, 'manifest.paths.json'), {
    releaseVersion: 'base-v0.8.0',
    backend: [{ source: 'backend/pkg', target: 'backend/pkg' }],
    frontend: [],
    docs: [],
  });
  writeText(path.join(opsRoot, 'go.mod'), 'module pantheon-ops/backend\n\ngo 1.24.0\n');
  writeText(
    path.join(bundlePath, 'shared-backend', 'backend', 'pkg', 'service.go'),
    [
      'package pkg',
      '',
      'import "pantheon-base/backend/internal/middleware"',
      '',
      'func Use() {',
      '\t_ = middleware.WithOperationLog',
      '}',
      '',
    ].join('\n'),
  );

  writeText(
    path.join(opsRoot, 'docs', 'PROJECT_INHERITANCE.md'),
    [
      '# 项目继承说明',
      '',
      '- Base repository：当前继承源是 `../pantheon-base`',
      '- Base branch：当前跟随 `main`',
      '- Base version：当前锁定到 `old`（`old`）',
      '- Inheritance mode：`foundation-only`',
      '',
    ].join('\n'),
  );
  writeText(
    path.join(opsRoot, 'docs', 'PROJECT_INHERITANCE.en.md'),
    [
      '# Project Inheritance',
      '',
      '- Base repository: `../pantheon-base`',
      '- Base branch: `main`',
      '- Base version: `old` (`old`)',
      '- Inheritance mode: `foundation-only`',
      '',
    ].join('\n'),
  );
  writeText(
    path.join(opsRoot, 'scripts', 'check-inheritance-contract.mjs'),
    "console.log('OK inheritance contract');\n",
  );
  writeText(
    path.join(opsRoot, 'scripts', 'check-base-backend-sync.mjs'),
    "console.log('OK shared backend is aligned with pantheon-base');\n",
  );
  writeText(
    path.join(opsRoot, 'frontend', 'scripts', 'sync-base-shared.mjs'),
    [
      "const sourceIndex = process.argv.indexOf('--source-root');",
      "const manifestIndex = process.argv.indexOf('--manifest');",
      "const sourceRoot = sourceIndex >= 0 ? process.argv[sourceIndex + 1] : '';",
      "const manifestPath = manifestIndex >= 0 ? process.argv[manifestIndex + 1] : '';",
      "if (!sourceRoot.includes('release-bundle') || !manifestPath.endsWith('manifest.json')) process.exit(1);",
      "console.log('OK shared frontend is aligned with pantheon-base');",
      '',
    ].join('\n'),
  );
  writeText(
    path.join(opsRoot, 'frontend', 'scripts', 'sync-foundation-i18n.mjs'),
    [
      "import path from 'node:path';",
      "const index = process.argv.indexOf('--builtin-resource');",
      "const resourcePath = index >= 0 ? process.argv[index + 1] : '';",
      "if (!resourcePath || path.basename(resourcePath) !== 'builtin_locale_resources.json' || !resourcePath.includes('release-bundle')) process.exit(1);",
      "console.log('OK foundation i18n fallback is aligned');",
      '',
    ].join('\n'),
  );
  writeText(
    path.join(opsRoot, 'frontend', 'scripts', 'generate-module-i18n.mjs'),
    [
      "import fs from 'node:fs';",
      "import path from 'node:path';",
      "const target = path.join(process.cwd(), 'frontend', 'src', 'i18n', 'resources', 'generated', 'fixture.ts');",
      "if (process.argv.includes('--check')) {",
      "  if (!fs.existsSync(target) || fs.readFileSync(target, 'utf8') !== 'generated\\n') process.exit(1);",
      '} else {',
      "  fs.mkdirSync(path.dirname(target), { recursive: true });",
      "  fs.writeFileSync(target, 'generated\\n', 'utf8');",
      '}',
      '',
    ].join('\n'),
  );
  writeText(
    path.join(opsRoot, 'frontend', 'scripts', 'check-i18n-missing-keys.mjs'),
    "console.log('OK no missing i18n keys');\n",
  );
  writeText(
    path.join(opsRoot, 'frontend', 'scripts', 'check-menu-contract.mjs'),
    "console.log('OK menu contract');\n",
  );
  writeText(
    path.join(opsRoot, 'frontend', 'src', 'i18n', 'resources', 'generated', 'fixture.ts'),
    'generated\n',
  );

  return { bundleRoot, manifestPath, opsRoot };
}

test('dry-run prints the target release and planned checks', () => {
  withTempDir((root) => {
    const { manifestPath, bundleRoot, opsRoot } = createFixture(root);
    const result = runScript(
      ['--ops-root', opsRoot, '--manifest', manifestPath, '--bundle', bundleRoot, '--check'],
      repoRoot,
    );

    assert.equal(result.status, 0, result.stderr || result.stdout || result.error?.message);
    assert.match(result.stdout, /base-v0\.8\.0/);
    assert.match(result.stdout, /check-inheritance-contract/);
    assert.match(result.stdout, /check-base-backend-sync/);
    assert.match(result.stdout, /sync-base-shared/);
    assert.match(result.stdout, /check-menu-contract/);
  });
});

test('apply mode updates inheritance anchors in both Chinese and English docs', () => {
  withTempDir((root) => {
    const { manifestPath, bundleRoot, opsRoot } = createFixture(root);
    const result = runScript(
      [
        '--ops-root',
        opsRoot,
        '--manifest',
        manifestPath,
        '--bundle',
        bundleRoot,
        '--update-inheritance-docs',
        '--rollback-on-error',
      ],
      repoRoot,
    );

    assert.equal(result.status, 0, result.stderr || result.stdout || result.error?.message);

    const zhDoc = fs.readFileSync(path.join(opsRoot, 'docs', 'PROJECT_INHERITANCE.md'), 'utf8');
    const enDoc = fs.readFileSync(path.join(opsRoot, 'docs', 'PROJECT_INHERITANCE.en.md'), 'utf8');

    assert.match(zhDoc, /Base release line：当前跟随 `release\/0\.8`/);
    assert.match(zhDoc, /Base version：当前锁定到 `base-v0\.8\.0`/);
    assert.match(zhDoc, /Inheritance mode：`foundation-release-consumer`/);
    assert.match(enDoc, /Base release line: `release\/0\.8`/);
    assert.match(enDoc, /Base version: `base-v0\.8\.0`/);
    assert.match(enDoc, /Inheritance mode: `foundation-release-consumer`/);
    assert.equal(
      fs.existsSync(path.join(opsRoot, '.foundation', 'releases', 'base-v0.8.0', 'manifest.json')),
      true,
    );
  });
});

test('apply mode copies shared backend files from the bundle into ops', () => {
  withTempDir((root) => {
    const { manifestPath, bundleRoot, opsRoot } = createFixture(root);
    const result = runScript(
      [
        '--ops-root',
        opsRoot,
        '--manifest',
        manifestPath,
        '--bundle',
        bundleRoot,
        '--apply-shared-backend',
        '--skip-go-validation',
        '--rollback-on-error',
      ],
      repoRoot,
    );

    assert.equal(result.status, 0, result.stderr || result.stdout || result.error?.message);
    assert.equal(fs.existsSync(path.join(opsRoot, 'backend', 'pkg', 'service.go')), true);
    const serviceSource = fs.readFileSync(path.join(opsRoot, 'backend', 'pkg', 'service.go'), 'utf8');
    assert.match(serviceSource, /pantheon-ops\/backend\/internal\/middleware/);
    assert.doesNotMatch(serviceSource, /pantheon-platform\/backend\/internal\/middleware/);
  });
});

test('apply mode updates shared lowcode generator tests', () => {
  withTempDir((root) => {
    const { manifestPath, bundleRoot, opsRoot } = createFixture(root);
    const targetPath = path.join(
      opsRoot,
      'backend',
      'modules',
      'lowcode',
      'generator',
      'generator_service_test.go',
    );
    writeText(
      path.join(
        bundleRoot,
        'bundle',
        'shared-backend',
        'backend',
        'modules',
        'lowcode',
        'generator',
        'generator_service_test.go',
      ),
      'package generator\n\nconst datasourcePolicy = "base"\n',
    );
    writeText(targetPath, 'package generator\n\nconst datasourcePolicy = "stale ops"\n');

    const result = runScript(
      [
        '--ops-root',
        opsRoot,
        '--manifest',
        manifestPath,
        '--bundle',
        bundleRoot,
        '--apply-shared-backend',
        '--skip-go-validation',
        '--rollback-on-error',
      ],
      repoRoot,
    );

    assert.equal(result.status, 0, result.stderr || result.stdout || result.error?.message);
    assert.equal(
      fs.readFileSync(targetPath, 'utf8'),
      'package generator\n\nconst datasourcePolicy = "base"\n',
    );
  });
});

test('apply mode rewrites both legacy and current Base Go import prefixes', () => {
  withTempDir((root) => {
    const { manifestPath, bundleRoot, opsRoot } = createFixture(root);
    writeText(
      path.join(bundleRoot, 'bundle', 'shared-backend', 'backend', 'pkg', 'current_layout.go'),
      [
        'package pkg',
        '',
        'import "pantheon-base/internal/middleware"',
        '',
        'func CurrentLayout() {',
        '\t_ = middleware.WithOperationLog',
        '}',
        '',
      ].join('\n'),
    );

    const result = runScript(
      [
        '--ops-root',
        opsRoot,
        '--manifest',
        manifestPath,
        '--bundle',
        bundleRoot,
        '--apply-shared-backend',
        '--skip-go-validation',
        '--rollback-on-error',
      ],
      repoRoot,
    );

    assert.equal(result.status, 0, result.stderr || result.stdout || result.error?.message);
    const source = fs.readFileSync(path.join(opsRoot, 'backend', 'pkg', 'current_layout.go'), 'utf8');
    assert.match(source, /pantheon-ops\/backend\/internal\/middleware/);
    assert.doesNotMatch(source, /pantheon-base\/internal\/middleware/);
  });
});

test('apply mode relocates Base menu component keys for the Ops frontend structure', () => {
  withTempDir((root) => {
    const { manifestPath, bundleRoot, opsRoot } = createFixture(root);
    writeText(
      path.join(bundleRoot, 'bundle', 'shared-backend', 'backend', 'modules', 'system', 'seed.go'),
      [
        'package system',
        '',
        'var menuComponent = "system/user/UserList"',
        '',
      ].join('\n'),
    );

    const result = runScript(
      [
        '--ops-root',
        opsRoot,
        '--manifest',
        manifestPath,
        '--bundle',
        bundleRoot,
        '--apply-shared-backend',
        '--skip-go-validation',
        '--rollback-on-error',
      ],
      repoRoot,
    );

    assert.equal(result.status, 0, result.stderr || result.stdout || result.error?.message);
    const seedSource = fs.readFileSync(path.join(opsRoot, 'backend', 'modules', 'system', 'seed.go'), 'utf8');
    assert.match(seedSource, /system\/iam\/user\/UserList/);
    assert.doesNotMatch(seedSource, /"system\/user\/UserList"/);
  });
});

test('apply mode preserves backend and frontend overlay files while updating shared files', () => {
  withTempDir((root) => {
    const { manifestPath, bundleRoot, opsRoot } = createFixture(root);

    writeText(
      path.join(bundleRoot, 'bundle', 'shared-backend', 'backend', 'modules', 'system', 'iam', 'menu', 'component_registry.go'),
      'base component registry\n',
    );
    writeText(
      path.join(opsRoot, 'backend', 'modules', 'system', 'iam', 'menu', 'component_registry.go'),
      'ops component registry\n',
    );
    writeText(
      path.join(bundleRoot, 'bundle', 'shared-backend', 'backend', 'modules', 'business', 'cmdb', 'host.go'),
      'base business module\n',
    );
    writeText(
      path.join(opsRoot, 'backend', 'modules', 'business', 'cmdb', 'host.go'),
      'ops business module\n',
    );
    writeText(
      path.join(bundleRoot, 'bundle', 'shared-frontend', 'frontend', 'src', 'core', 'router', 'generatedComponentRegistry.ts'),
      'export const generatedComponentRegistry = { base: true };\n',
    );
    writeText(
      path.join(opsRoot, 'frontend', 'src', 'core', 'router', 'generatedComponentRegistry.ts'),
      'export const generatedComponentRegistry = { ops: true };\n',
    );
    writeText(
      path.join(bundleRoot, 'bundle', 'shared-frontend', 'frontend', 'src', 'core', 'shell.ts'),
      'export const shell = "base";\n',
    );

    const result = runScript(
      [
        '--ops-root',
        opsRoot,
        '--manifest',
        manifestPath,
        '--bundle',
        bundleRoot,
        '--apply-shared-backend',
        '--apply-shared-frontend',
        '--skip-go-validation',
        '--rollback-on-error',
      ],
      repoRoot,
    );

    assert.equal(result.status, 0, result.stderr || result.stdout || result.error?.message);
    assert.equal(
      fs.readFileSync(path.join(opsRoot, 'backend', 'modules', 'system', 'iam', 'menu', 'component_registry.go'), 'utf8'),
      'ops component registry\n',
    );
    assert.equal(
      fs.readFileSync(path.join(opsRoot, 'frontend', 'src', 'core', 'router', 'generatedComponentRegistry.ts'), 'utf8'),
      'export const generatedComponentRegistry = { ops: true };\n',
    );
    assert.equal(
      fs.readFileSync(path.join(opsRoot, 'frontend', 'src', 'core', 'shell.ts'), 'utf8'),
      'export const shell = "base";\n',
    );
    assert.equal(
      fs.readFileSync(path.join(opsRoot, 'backend', 'modules', 'business', 'cmdb', 'host.go'), 'utf8'),
      'ops business module\n',
    );
  });
});

test('apply mode removes obsolete files inside shared frontend paths before checking drift', () => {
  withTempDir((root) => {
    const { manifestPath, bundleRoot, opsRoot } = createFixture(root);
    const staleFilePath = path.join(
      opsRoot,
      'frontend',
      'src',
      'modules',
      'lowcode',
      'generator',
      'backend-generator.ts',
    );
    writeText(staleFilePath, 'export const staleGenerator = true;\n');
    writeText(
      path.join(opsRoot, 'frontend', 'scripts', 'sync-base-shared.mjs'),
      [
        "import fs from 'node:fs';",
        "if (process.argv.includes('--check')) process.exit(1);",
        `fs.rmSync(${JSON.stringify(staleFilePath)}, { force: true });`,
        "console.log('Removed obsolete shared frontend files');",
        '',
      ].join('\n'),
    );

    const result = runScript(
      [
        '--ops-root',
        opsRoot,
        '--manifest',
        manifestPath,
        '--bundle',
        bundleRoot,
        '--apply-shared-frontend',
        '--rollback-on-error',
      ],
      repoRoot,
    );

    assert.equal(result.status, 0, result.stderr || result.stdout || result.error?.message);
    assert.equal(fs.existsSync(staleFilePath), false);
  });
});

test('apply mode regenerates frontend i18n output from the supplied bundle', () => {
  withTempDir((root) => {
    const { manifestPath, bundleRoot, opsRoot } = createFixture(root);
    const generatedPath = path.join(
      opsRoot,
      'frontend',
      'src',
      'i18n',
      'resources',
      'generated',
      'fixture.ts',
    );
    fs.rmSync(generatedPath);
    writeText(
      path.join(bundleRoot, 'bundle', 'shared-frontend', 'frontend', 'src', 'core', 'shell.ts'),
      'export const shell = "base";\n',
    );

    const result = runScript(
      [
        '--ops-root',
        opsRoot,
        '--manifest',
        manifestPath,
        '--bundle',
        bundleRoot,
        '--apply-shared-frontend',
        '--check',
        '--rollback-on-error',
      ],
      repoRoot,
    );

    assert.equal(result.status, 0, result.stderr || result.stdout || result.error?.message);
    assert.equal(fs.readFileSync(generatedPath, 'utf8'), 'generated\n');
  });
});

test('apply mode updates platform health source with its shared Base tests', () => {
  withTempDir((root) => {
    const { manifestPath, bundleRoot, opsRoot } = createFixture(root);
    writeText(
      path.join(bundleRoot, 'bundle', 'shared-backend', 'backend', 'modules', 'platform', 'health.go'),
      'package platform\n\nconst healthSource = "base"\n',
    );
    writeText(
      path.join(bundleRoot, 'bundle', 'shared-backend', 'backend', 'modules', 'platform', 'health_test.go'),
      'package platform\n\nfunc TestHealthSource() { _ = healthSource }\n',
    );
    writeText(
      path.join(opsRoot, 'backend', 'modules', 'platform', 'health.go'),
      'package platform\n\nconst healthSource = "stale ops copy"\n',
    );

    const result = runScript(
      [
        '--ops-root',
        opsRoot,
        '--manifest',
        manifestPath,
        '--bundle',
        bundleRoot,
        '--apply-shared-backend',
        '--skip-go-validation',
        '--rollback-on-error',
      ],
      repoRoot,
    );

    assert.equal(result.status, 0, result.stderr || result.stdout || result.error?.message);
    assert.equal(
      fs.readFileSync(path.join(opsRoot, 'backend', 'modules', 'platform', 'health.go'), 'utf8'),
      'package platform\n\nconst healthSource = "base"\n',
    );
    assert.equal(
      fs.existsSync(path.join(opsRoot, 'backend', 'modules', 'platform', 'health_test.go')),
      true,
    );
  });
});

test('apply mode relocates shared frontend system module paths into the ops structure', () => {
  withTempDir((root) => {
    const { manifestPath, bundleRoot, opsRoot } = createFixture(root);

    writeText(
      path.join(bundleRoot, 'bundle', 'shared-frontend', 'frontend', 'src', 'modules', 'system', 'user', 'index.ts'),
      [
        "import { loadUser } from './api';",
        '',
        "export const userRoute = { componentKey: 'system/user/list' };",
        'export { loadUser };',
        '',
      ].join('\n'),
    );
    writeText(
      path.join(bundleRoot, 'bundle', 'shared-frontend', 'frontend', 'src', 'modules', 'system', 'user', 'api.ts'),
      'export const loadUser = () => null;\n',
    );

    const result = runScript(
      [
        '--ops-root',
        opsRoot,
        '--manifest',
        manifestPath,
        '--bundle',
        bundleRoot,
        '--apply-shared-frontend',
        '--rollback-on-error',
      ],
      repoRoot,
    );

    assert.equal(result.status, 0, result.stderr || result.stdout || result.error?.message);
    assert.equal(
      fs.existsSync(path.join(opsRoot, 'frontend', 'src', 'modules', 'system', 'user', 'index.ts')),
      false,
    );
    const relocatedSource = fs.readFileSync(
      path.join(opsRoot, 'frontend', 'src', 'modules', 'system', 'iam', 'user', 'index.ts'),
      'utf8',
    );
    assert.match(relocatedSource, /componentKey: 'system\/iam\/user\/list'/);
    assert.match(relocatedSource, /from '\.\/api'/);
  });
});

test('apply mode merges shared i18n updates without dropping ops business locale keys', () => {
  withTempDir((root) => {
    const { manifestPath, bundleRoot, opsRoot } = createFixture(root);

    writeJson(
      path.join(bundleRoot, 'bundle', 'shared-backend', 'backend', 'modules', 'system', 'i18n', 'builtin_locale_resources.json'),
      {
        'zh-CN': {
          'app.name': 'Pantheon Base',
          'business.shared': 'base should not win',
        },
        'en-US': {
          'app.name': 'Pantheon Base',
        },
      },
    );
    writeJson(
      path.join(opsRoot, 'backend', 'modules', 'system', 'i18n', 'builtin_locale_resources.json'),
      {
        'zh-CN': {
          'app.name': 'Old Ops Name',
          'business.cmdb.host.title': '主机台账',
        },
        'en-US': {
          'app.name': 'Old Ops Name',
          'business.cmdb.host.title': 'Host Inventory',
        },
      },
    );

    const result = runScript(
      [
        '--ops-root',
        opsRoot,
        '--manifest',
        manifestPath,
        '--bundle',
        bundleRoot,
        '--apply-shared-backend',
        '--skip-go-validation',
        '--rollback-on-error',
      ],
      repoRoot,
    );

    assert.equal(result.status, 0, result.stderr || result.stdout || result.error?.message);

    const locales = JSON.parse(
      fs.readFileSync(path.join(opsRoot, 'backend', 'modules', 'system', 'i18n', 'builtin_locale_resources.json'), 'utf8'),
    );
    assert.equal(locales['zh-CN']['app.name'], 'Pantheon Base');
    assert.equal(locales['zh-CN']['business.cmdb.host.title'], '主机台账');
    assert.equal(locales['en-US']['app.name'], 'Pantheon Base');
    assert.equal(locales['en-US']['business.cmdb.host.title'], 'Host Inventory');
    assert.equal(locales['zh-CN']['business.shared'], 'base should not win');
  });
});

test('--dry-run mode prints "DRY RUN" and shows no changes when files are already aligned', () => {
  withTempDir((root) => {
    const { manifestPath, bundleRoot, opsRoot } = createFixture(root);
    // Write an identical file to ops so dry-run finds no diff
    writeText(
      path.join(opsRoot, 'backend', 'pkg', 'service.go'),
      [
        'package pkg',
        '',
        'import "pantheon-ops/backend/internal/middleware"',
        '',
        'func Use() {',
        '\t_ = middleware.WithOperationLog',
        '}',
        '',
      ].join('\n'),
    );

    const result = runScript(
      ['--ops-root', opsRoot, '--manifest', manifestPath, '--bundle', bundleRoot, '--dry-run'],
      repoRoot,
    );

    assert.equal(result.status, 0, result.stderr || result.stdout || result.error?.message);
    assert.match(result.stdout, /DRY RUN/);
    assert.match(result.stdout, /No changes needed/);
  });
});

test('--dry-run mode lists files that would change with CREATE or REWRITE actions', () => {
  withTempDir((root) => {
    const { manifestPath, bundleRoot, opsRoot } = createFixture(root);
    // ops has a stale file — dry-run should report it as REWRITE
    writeText(
      path.join(opsRoot, 'backend', 'pkg', 'service.go'),
      '// stale content\n',
    );

    const result = runScript(
      ['--ops-root', opsRoot, '--manifest', manifestPath, '--bundle', bundleRoot, '--dry-run'],
      repoRoot,
    );

    assert.equal(result.status, 0, result.stderr || result.stdout || result.error?.message);
    assert.match(result.stdout, /DRY RUN/);
    assert.match(result.stdout, /REWRITE/);
    assert.match(result.stdout, /pkg\/service\.go/);
  });
});

test('shared frontend tooling is reported by dry-run and applied from the release bundle', () => {
  withTempDir((root) => {
    const { manifestPath, bundleRoot, opsRoot } = createFixture(root);
    const manifest = JSON.parse(fs.readFileSync(manifestPath, 'utf8'));
    manifest.sharedPaths.frontend = ['frontend/scripts/lib/css-declarations.mjs'];
    writeJson(manifestPath, manifest);
    const toolingPath = path.join('frontend', 'scripts', 'lib', 'css-declarations.mjs');
    writeText(
      path.join(bundleRoot, 'bundle', 'shared-frontend', toolingPath),
      'export const matcher = "release";\n',
    );
    writeText(path.join(opsRoot, toolingPath), 'export const matcher = "stale";\n');

    const dryRun = runScript(
      ['--ops-root', opsRoot, '--manifest', manifestPath, '--bundle', bundleRoot, '--dry-run'],
      repoRoot,
    );
    assert.equal(dryRun.status, 0, dryRun.stderr || dryRun.stdout || dryRun.error?.message);
    assert.match(dryRun.stdout, /REWRITE frontend[\\/]scripts[\\/]lib[\\/]css-declarations\.mjs/);
    assert.equal(fs.readFileSync(path.join(opsRoot, toolingPath), 'utf8'), 'export const matcher = "stale";\n');

    const applyResult = runScript(
      [
        '--ops-root',
        opsRoot,
        '--manifest',
        manifestPath,
        '--bundle',
        bundleRoot,
        '--apply-shared-frontend',
        '--rollback-on-error',
      ],
      repoRoot,
    );
    assert.equal(applyResult.status, 0, applyResult.stderr || applyResult.stdout || applyResult.error?.message);
    assert.equal(fs.readFileSync(path.join(opsRoot, toolingPath), 'utf8'), 'export const matcher = "release";\n');
  });
});

test('apply mode writes lockedAt and lockedBy to foundation-release.lock.json', () => {
  withTempDir((root) => {
    const { manifestPath, bundleRoot, opsRoot } = createFixture(root);

    const result = runScript(
      [
        '--ops-root',
        opsRoot,
        '--manifest',
        manifestPath,
        '--bundle',
        bundleRoot,
        '--update-inheritance-docs',
        '--rollback-on-error',
      ],
      repoRoot,
    );

    assert.equal(result.status, 0, result.stderr || result.stdout || result.error?.message);

    const lock = JSON.parse(
      fs.readFileSync(path.join(opsRoot, 'foundation-release.lock.json'), 'utf8'),
    );
    assert.ok(lock.lockedAt, 'lockedAt should be set');
    assert.ok(lock.lockedBy, 'lockedBy should be set');
    assert.match(lock.lockedAt, /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}/);
  });
});

test('--dry-run does not modify any files on disk', () => {
  withTempDir((root) => {
    const { manifestPath, bundleRoot, opsRoot } = createFixture(root);
    // Touch the lock file timestamp before dry-run
    const lockPath = path.join(opsRoot, 'foundation-release.lock.json');
    const mtimeBefore = fs.statSync(lockPath).mtimeMs;

    runScript(
      ['--ops-root', opsRoot, '--manifest', manifestPath, '--bundle', bundleRoot, '--dry-run'],
      repoRoot,
    );

    const mtimeAfter = fs.statSync(lockPath).mtimeMs;
    assert.equal(mtimeAfter, mtimeBefore, 'foundation-release.lock.json should not be modified by dry-run');
  });
});

test('write operations require rollback-on-error', () => {
  withTempDir((root) => {
    const { manifestPath, bundleRoot, opsRoot } = createFixture(root);
    const result = runScript(
      [
        '--ops-root',
        opsRoot,
        '--manifest',
        manifestPath,
        '--bundle',
        bundleRoot,
        '--apply-shared-backend',
        '--skip-go-validation',
      ],
      repoRoot,
    );

    assert.notEqual(result.status, 0);
    assert.match(result.stderr, /--rollback-on-error is required/);
  });
});

test('consumer rejects an unverified or modified release root', () => {
  withTempDir((root) => {
    const { manifestPath, bundleRoot, opsRoot } = createFixture(root);
    const args = ['--ops-root', opsRoot, '--manifest', manifestPath, '--bundle', bundleRoot, '--dry-run'];

    const unverified = runRawScript(args, repoRoot);
    assert.notEqual(unverified.status, 0);
    assert.match(unverified.stderr, /verification marker not found/);

    writeTestVerificationMarker(args);
    writeText(
      path.join(bundleRoot, 'bundle', 'shared-backend', 'backend', 'pkg', 'service.go'),
      'package pkg\n// modified after verification\n',
    );
    const modified = runRawScript(args, repoRoot);
    assert.notEqual(modified.status, 0);
    assert.match(modified.stderr, /contents changed after verification/);
  });
});

test('the formal npm apply entry executes with rollback protection', () => {
  withTempDir((root) => {
    const { manifestPath, bundleRoot, opsRoot } = createFixture(root);
    const result = runNpmApply([
      '--ops-root',
      opsRoot,
      '--manifest',
      manifestPath,
      '--bundle',
      bundleRoot,
      '--skip-go-validation',
    ]);

    assert.equal(result.status, 0, result.stderr || result.stdout || result.error?.message);
    assert.match(result.stdout, /Updated foundation-release\.lock\.json/);
  });
});

test('consumer blocks incompatible release lines unless the forward jump is explicit', () => {
  withTempDir((root) => {
    const { manifestPath, bundleRoot, opsRoot } = createFixture(root);
    const manifest = JSON.parse(fs.readFileSync(manifestPath, 'utf8'));
    manifest.consumerCompatibility = {
      'pantheon-ops': {
        minimumCurrentRelease: 'release/0.8',
      },
    };
    writeJson(manifestPath, manifest);

    const blocked = runScript(
      ['--ops-root', opsRoot, '--manifest', manifestPath, '--bundle', bundleRoot, '--dry-run'],
      repoRoot,
    );
    assert.notEqual(blocked.status, 0);
    assert.match(blocked.stderr, /requires current release release\/0\.8 or newer/);

    const allowed = runScript(
      [
        '--ops-root',
        opsRoot,
        '--manifest',
        manifestPath,
        '--bundle',
        bundleRoot,
        '--dry-run',
        '--allow-release-line-jump',
      ],
      repoRoot,
    );
    assert.equal(allowed.status, 0, allowed.stderr || allowed.stdout || allowed.error?.message);
    assert.match(allowed.stdout, /Explicit release-line jump accepted/);
  });
});

test('rollback restores inheritance anchors and release artifacts when a required check fails', () => {
  withTempDir((root) => {
    const { manifestPath, bundleRoot, opsRoot } = createFixture(root);
    const lockPath = path.join(opsRoot, 'foundation-release.lock.json');
    const zhDocPath = path.join(opsRoot, 'docs', 'PROJECT_INHERITANCE.md');
    const enDocPath = path.join(opsRoot, 'docs', 'PROJECT_INHERITANCE.en.md');
    const generatedPath = path.join(
      opsRoot,
      'frontend',
      'src',
      'i18n',
      'resources',
      'generated',
      'fixture.ts',
    );
    const originalLock = fs.readFileSync(lockPath, 'utf8');
    const originalZhDoc = fs.readFileSync(zhDocPath, 'utf8');
    const originalEnDoc = fs.readFileSync(enDocPath, 'utf8');
    const manifest = JSON.parse(fs.readFileSync(manifestPath, 'utf8'));
    manifest.sharedPaths.frontend = ['frontend/scripts/lib/css-declarations.mjs'];
    writeJson(manifestPath, manifest);
    const toolingPath = path.join(opsRoot, 'frontend', 'scripts', 'lib', 'css-declarations.mjs');
    writeText(toolingPath, 'export const matcher = "original";\n');
    writeText(
      path.join(
        bundleRoot,
        'bundle',
        'shared-frontend',
        'frontend',
        'scripts',
        'lib',
        'css-declarations.mjs',
      ),
      'export const matcher = "release";\n',
    );
    writeText(generatedPath, 'original generated output\n');
    writeText(
      path.join(bundleRoot, 'bundle', 'shared-frontend', 'frontend', 'src', 'core', 'shell.ts'),
      'export const shell = "base";\n',
    );
    writeText(
      path.join(opsRoot, 'scripts', 'check-base-backend-sync.mjs'),
      "console.error('expected check failure'); process.exit(1);\n",
    );

    const result = runScript(
      [
        '--ops-root',
        opsRoot,
        '--manifest',
        manifestPath,
        '--bundle',
        bundleRoot,
        '--update-inheritance-docs',
        '--apply-shared-frontend',
        '--check',
        '--rollback-on-error',
      ],
      repoRoot,
    );

    assert.notEqual(result.status, 0);
    assert.match(result.stderr, /Rolled back files changed by this foundation apply/);
    assert.equal(fs.readFileSync(lockPath, 'utf8'), originalLock);
    assert.equal(fs.readFileSync(zhDocPath, 'utf8'), originalZhDoc);
    assert.equal(fs.readFileSync(enDocPath, 'utf8'), originalEnDoc);
    assert.equal(fs.readFileSync(generatedPath, 'utf8'), 'original generated output\n');
    assert.equal(fs.readFileSync(toolingPath, 'utf8'), 'export const matcher = "original";\n');
    assert.equal(
      fs.existsSync(path.join(opsRoot, '.foundation', 'releases', 'base-v0.8.0')),
      false,
    );
  });
});

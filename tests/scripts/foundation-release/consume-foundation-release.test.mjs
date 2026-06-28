import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { spawnSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';
import test from 'node:test';

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

function runScript(args, cwd) {
  return spawnSync(process.execPath, [scriptPath, ...args], {
    cwd,
    encoding: 'utf8',
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
  writeText(path.join(opsRoot, 'go.mod'), 'module pantheon-ops\n\ngo 1.24.0\n');
  writeText(
    path.join(bundlePath, 'shared-backend', 'backend', 'pkg', 'service.go'),
    [
      'package pkg',
      '',
      'import "pantheon-platform/backend/internal/middleware"',
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
    "console.log('OK shared frontend is aligned with pantheon-base');\n",
  );
  writeText(
    path.join(opsRoot, 'frontend', 'scripts', 'check-menu-contract.mjs'),
    "console.log('OK menu contract');\n",
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
      ],
      repoRoot,
    );

    assert.equal(result.status, 0, result.stderr || result.stdout || result.error?.message);

    const zhDoc = fs.readFileSync(path.join(opsRoot, 'docs', 'PROJECT_INHERITANCE.md'), 'utf8');
    const enDoc = fs.readFileSync(path.join(opsRoot, 'docs', 'PROJECT_INHERITANCE.en.md'), 'utf8');
    const releaseLock = JSON.parse(fs.readFileSync(path.join(opsRoot, 'foundation-release.lock.json'), 'utf8'));

    assert.match(zhDoc, /Base release line：当前跟随 `release\/0\.8`/);
    assert.match(zhDoc, /Base version：当前锁定到 `base-v0\.8\.0`/);
    assert.match(zhDoc, /Inheritance mode：`foundation-release-consumer`/);
    assert.match(enDoc, /Base release line: `release\/0\.8`/);
    assert.match(enDoc, /Base version: `base-v0\.8\.0`/);
    assert.match(enDoc, /Inheritance mode: `foundation-release-consumer`/);
    assert.equal(releaseLock.releaseVersion, 'base-v0.8.0');
    assert.equal(releaseLock.releaseDisplayName, 'v0.8.0');
    assert.equal(releaseLock.releaseArtifact.tagName, 'base-v0.8.0');
    assert.equal(releaseLock.releaseArtifact.releaseName, 'v0.8.0');
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

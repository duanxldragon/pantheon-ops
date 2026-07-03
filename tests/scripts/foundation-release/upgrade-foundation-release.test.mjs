import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { spawnSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';
import test from 'node:test';

const currentFilePath = fileURLToPath(import.meta.url);
const repoRoot = path.resolve(path.dirname(currentFilePath), '..', '..', '..');
const scriptPath = path.join(repoRoot, 'scripts', 'foundation-release', 'upgrade-foundation-release.mjs');

function withTempDir(callback) {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'pantheon-foundation-upgrade-'));
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
  const opsRoot = path.join(root, 'pantheon-ops');
  const releaseVersion = 'base-v0.8.0';
  const releaseRoot = path.join(opsRoot, '.foundation', 'releases', releaseVersion);

  writeJson(path.join(releaseRoot, 'manifest.json'), {
    releaseVersion,
    releaseLine: 'release/0.8',
    baseCommit: 'deadbeefdeadbeefdeadbeefdeadbeefdeadbeef',
    sourceRepo: 'pantheon-base',
    consumerMode: 'foundation-release-consumer',
    baseGoModule: 'pantheon-platform/backend',
    releaseArtifact: {
      assetName: 'foundation-release-base-v0.8.0.tgz',
    },
    sharedPaths: {
      backend: ['backend/pkg'],
    },
  });
  writeText(path.join(releaseRoot, 'go.mod'), 'module pantheon-platform\n\ngo 1.24.0\n');
  writeText(path.join(releaseRoot, 'bundle', 'shared-backend', 'backend', 'go.mod'), 'module pantheon-platform/backend\n\ngo 1.24.0\n');
  writeText(path.join(releaseRoot, 'bundle', 'shared-backend', 'backend', 'pkg', 'service.go'), 'package pkg\n');
  writeText(path.join(releaseRoot, 'bundle', 'shared-frontend', 'frontend', 'src', 'components', 'index.ts'), 'export const baseComponent = true;\n');
  writeText(path.join(releaseRoot, 'bundle', 'shared-frontend', 'frontend', 'src', 'core', 'layout.ts'), 'export const baseCore = true;\n');
  writeText(path.join(releaseRoot, 'bundle', 'shared-frontend', 'frontend', 'src', 'store', 'useStore.ts'), 'export const baseStore = true;\n');
  writeText(path.join(releaseRoot, 'bundle', 'shared-frontend', 'frontend', 'src', 'modules', 'auth', 'index.ts'), 'export const baseAuth = true;\n');
  writeText(path.join(releaseRoot, 'bundle', 'shared-frontend', 'frontend', 'src', 'modules', 'lowcode', 'index.ts'), 'export const baseLowcode = true;\n');
  writeText(path.join(releaseRoot, 'bundle', 'shared-frontend', 'frontend', 'src', 'modules', 'platform', 'index.ts'), 'export const basePlatform = true;\n');
  writeText(path.join(releaseRoot, 'bundle', 'shared-frontend', 'frontend', 'src', 'modules', 'system', 'index.ts'), 'export const baseSystem = true;\n');
  writeText(path.join(releaseRoot, 'bundle', 'shared-frontend', 'frontend', 'src', 'index.css'), 'body { color: black; }\n');
  writeJson(path.join(releaseRoot, 'bundle', 'manifest.paths.json'), {
    releaseVersion,
    backend: [{ source: 'backend/pkg', target: 'backend/pkg' }],
    frontend: [],
    docs: [],
  });

  writeText(path.join(opsRoot, 'go.mod'), 'module pantheon-ops\n\ngo 1.24.0\n');
  writeText(path.join(opsRoot, 'backend', 'go.mod'), 'module pantheon-ops/backend\n\ngo 1.24.0\n');
  writeText(path.join(opsRoot, 'backend', 'pkg', 'service.go'), 'package pkg\n');
  writeText(path.join(opsRoot, 'frontend', 'src', 'components', 'index.ts'), 'export const baseComponent = true;\n');
  writeText(path.join(opsRoot, 'frontend', 'src', 'core', 'layout.ts'), 'export const baseCore = true;\n');
  writeText(path.join(opsRoot, 'frontend', 'src', 'store', 'useStore.ts'), 'export const baseStore = true;\n');
  writeText(path.join(opsRoot, 'frontend', 'src', 'modules', 'auth', 'index.ts'), 'export const baseAuth = true;\n');
  writeText(path.join(opsRoot, 'frontend', 'src', 'modules', 'lowcode', 'index.ts'), 'export const baseLowcode = true;\n');
  writeText(path.join(opsRoot, 'frontend', 'src', 'modules', 'platform', 'index.ts'), 'export const basePlatform = true;\n');
  writeText(path.join(opsRoot, 'frontend', 'src', 'modules', 'system', 'index.ts'), 'export const baseSystem = true;\n');
  writeText(path.join(opsRoot, 'frontend', 'src', 'index.css'), 'body { color: black; }\n');

  writeText(
    path.join(opsRoot, 'docs', 'PROJECT_INHERITANCE.md'),
    [
      '# 项目继承说明',
      '',
      '- Base repository：当前继承源是 `pantheon-base`',
      '- Base release line：当前跟随 `release/0.8`',
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
      '- Base repository: `pantheon-base`',
      '- Base release line: `release/0.8`',
      '- Base version: `old` (`old`)',
      '- Inheritance mode: `foundation-only`',
      '',
    ].join('\n'),
  );
  writeJson(path.join(opsRoot, 'foundation-release.lock.json'), {
    schemaVersion: 1,
    baseRepo: 'pantheon-base',
    sourceRepo: 'pantheon-base',
    consumerMode: 'foundation-release-consumer',
    releaseLine: 'release/0.8',
    releaseVersion: 'base-v0.8.9',
    releaseDisplayName: 'v0.8.9',
    baseCommit: 'deadbeefdeadbeefdeadbeefdeadbeefdeadbeef',
    releaseArtifact: {
      githubRepo: 'duanxldragon/pantheon-base',
      tagName: 'base-v0.8.9',
      releaseName: 'v0.8.9',
      assetName: 'foundation-release-base-v0.8.9.tgz',
      localPath: '.foundation/releases/base-v0.8.9',
    },
  });
  writeText(path.join(opsRoot, 'scripts', 'check-inheritance-contract.mjs'), "console.log('OK inheritance contract');\n");
  writeText(path.join(opsRoot, 'scripts', 'check-base-backend-sync.mjs'), "console.log('OK shared backend is aligned with pantheon-base');\n");
  writeText(path.join(opsRoot, 'frontend', 'scripts', 'sync-base-shared.mjs'), "console.log('OK shared frontend is aligned with pantheon-base');\n");
  writeText(path.join(opsRoot, 'frontend', 'scripts', 'check-menu-contract.mjs'), "console.log('OK menu contract');\n");

  return { opsRoot, releaseVersion };
}

test('wrapper consumes the requested cached release version', () => {
  withTempDir((root) => {
    const { opsRoot, releaseVersion } = createFixture(root);
    const result = runScript(['--ops-root', opsRoot, '--release-version', releaseVersion], repoRoot);

    assert.equal(result.status, 0, result.stderr || result.stdout || result.error?.message);
    assert.match(result.stdout, /Target foundation release: base-v0\.8\.0/);
    assert.match(result.stdout, /Applied shared-backend bundle/);
    assert.match(result.stdout, /Updated inheritance docs and release lock/);
    const lock = JSON.parse(fs.readFileSync(path.join(opsRoot, 'foundation-release.lock.json'), 'utf8'));
    assert.equal(lock.releaseVersion, 'base-v0.8.0');
    assert.equal(lock.releaseArtifact.localPath, '.foundation/releases/base-v0.8.0');
  });
});

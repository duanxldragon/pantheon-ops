import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { spawnSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';
import test from 'node:test';

const currentFilePath = fileURLToPath(import.meta.url);
const repoRoot = path.resolve(path.dirname(currentFilePath), '..', '..');
const sourceSyncScript = path.join(repoRoot, 'frontend', 'scripts', 'sync-base-shared.mjs');
const sourceRulesScript = path.join(repoRoot, 'scripts', 'foundation-release', 'shared-foundation-rules.mjs');

function withTempDir(callback) {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'pantheon-sync-base-shared-'));
  try {
    callback(root);
  } finally {
    fs.rmSync(root, { recursive: true, force: true });
  }
}

function writeText(filePath, value) {
  fs.mkdirSync(path.dirname(filePath), { recursive: true });
  fs.writeFileSync(filePath, value, 'utf8');
}

function copyFixtureScripts(opsRoot) {
  const syncScriptPath = path.join(opsRoot, 'frontend', 'scripts', 'sync-base-shared.mjs');
  const rulesScriptPath = path.join(opsRoot, 'scripts', 'foundation-release', 'shared-foundation-rules.mjs');
  const lockPath = path.join(opsRoot, 'foundation-release.lock.json');
  fs.mkdirSync(path.dirname(syncScriptPath), { recursive: true });
  fs.mkdirSync(path.dirname(rulesScriptPath), { recursive: true });
  fs.copyFileSync(sourceSyncScript, syncScriptPath);
  fs.copyFileSync(sourceRulesScript, rulesScriptPath);
  writeText(
    lockPath,
    `${JSON.stringify({
      schemaVersion: 1,
      baseRepo: '../pantheon-base-fixture',
      releaseLine: 'release/test',
      releaseVersion: 'base-vtest',
      baseCommit: 'HEAD',
      consumerMode: 'foundation-release-consumer',
      sharedPaths: {
        frontend: [
          'frontend/src/components',
          'frontend/src/core',
          'frontend/src/store',
          'frontend/src/modules/auth',
          'frontend/src/modules/dashboard',
          'frontend/src/modules/system',
          'frontend/src/index.css',
        ],
      },
    }, null, 2)}\n`,
  );
  return syncScriptPath;
}

function createSharedFrontendTree(rootPath, contents) {
  writeText(path.join(rootPath, 'frontend', 'src', 'components', 'index.ts'), contents.components);
  writeText(path.join(rootPath, 'frontend', 'src', 'core', 'layout.ts'), contents.core);
  writeText(path.join(rootPath, 'frontend', 'src', 'modules', 'auth', 'index.ts'), contents.auth);
  writeText(path.join(rootPath, 'frontend', 'src', 'modules', 'dashboard', 'index.ts'), contents.dashboard);
  writeText(path.join(rootPath, 'frontend', 'src', 'modules', 'system', 'index.ts'), contents.system);
  writeText(path.join(rootPath, 'frontend', 'src', 'index.css'), contents.indexCss);
}

function runSync(scriptPath, cwd, envOverrides = {}, args = ['--workspace-head']) {
  return spawnSync(process.execPath, [scriptPath, ...args], {
    cwd,
    encoding: 'utf8',
    env: {
      ...process.env,
      ...envOverrides,
    },
  });
}

test('sync-base-shared respects PANTHEON_BASE_REPO_ROOT and the current ops worktree path', () => {
  withTempDir((root) => {
    const baseRoot = path.join(root, 'pantheon-base-fixture');
    const opsRoot = path.join(root, 'ops-worktree-fixture');
    const syncScriptPath = copyFixtureScripts(opsRoot);

    createSharedFrontendTree(baseRoot, {
      components: 'export const baseComponent = true;\n',
      core: 'export const baseCore = true;\n',
      auth: 'export const baseAuth = true;\n',
      dashboard: 'export const baseDashboard = true;\n',
      system: 'export const baseSystem = true;\n',
      indexCss: 'body { color: black; }\n',
    });
    createSharedFrontendTree(opsRoot, {
      components: 'export const oldComponent = true;\n',
      core: 'export const oldCore = true;\n',
      auth: 'export const oldAuth = true;\n',
      dashboard: 'export const oldDashboard = true;\n',
      system: 'export const oldSystem = true;\n',
      indexCss: 'body { color: red; }\n',
    });

    const applyResult = runSync(syncScriptPath, opsRoot, { PANTHEON_BASE_REPO_ROOT: baseRoot });
    assert.equal(applyResult.status, 0, applyResult.stderr || applyResult.stdout || applyResult.error?.message);

    assert.equal(
      fs.readFileSync(path.join(opsRoot, 'frontend', 'src', 'components', 'index.ts'), 'utf8'),
      'export const baseComponent = true;\n',
    );
    assert.equal(
      fs.readFileSync(path.join(opsRoot, 'frontend', 'src', 'core', 'layout.ts'), 'utf8'),
      'export const baseCore = true;\n',
    );
    assert.equal(
      fs.readFileSync(path.join(opsRoot, 'frontend', 'src', 'index.css'), 'utf8'),
      'body { color: black; }\n',
    );

    const checkResult = runSync(
      syncScriptPath,
      opsRoot,
      { PANTHEON_BASE_REPO_ROOT: baseRoot },
      ['--workspace-head', '--check'],
    );
    assert.equal(checkResult.status, 0, checkResult.stderr || checkResult.stdout || checkResult.error?.message);
    assert.match(checkResult.stdout, /OK shared frontend is aligned with pantheon-base workspace HEAD/);
  });
});

test('sync-base-shared uses the installed release artifact by default', () => {
  withTempDir((root) => {
    const releaseRoot = path.join(root, 'release-root');
    const opsRoot = path.join(root, 'ops-worktree-fixture');
    const syncScriptPath = copyFixtureScripts(opsRoot);

    writeText(
      path.join(releaseRoot, 'manifest.json'),
      `${JSON.stringify({
        schemaVersion: 1,
        releaseVersion: 'base-vtest',
        releaseLine: 'release/test',
        baseCommit: 'HEAD',
        sourceRepo: 'pantheon-base',
        consumerMode: 'foundation-release-consumer',
        sharedPaths: {
          frontend: ['frontend/src/components', 'frontend/src/index.css'],
        },
      }, null, 2)}\n`,
    );
    createSharedFrontendTree(path.join(releaseRoot, 'bundle', 'shared-frontend'), {
      components: 'export const releaseComponent = true;\n',
      core: 'export const releaseCore = true;\n',
      auth: 'export const releaseAuth = true;\n',
      dashboard: 'export const releaseDashboard = true;\n',
      system: 'export const releaseSystem = true;\n',
      indexCss: 'body { color: green; }\n',
    });
    createSharedFrontendTree(opsRoot, {
      components: 'export const oldComponent = true;\n',
      core: 'export const oldCore = true;\n',
      auth: 'export const oldAuth = true;\n',
      dashboard: 'export const oldDashboard = true;\n',
      system: 'export const oldSystem = true;\n',
      indexCss: 'body { color: red; }\n',
    });

    const applyResult = runSync(
      syncScriptPath,
      opsRoot,
      { PANTHEON_FOUNDATION_RELEASE_ROOT: releaseRoot },
      [],
    );
    assert.equal(applyResult.status, 0, applyResult.stderr || applyResult.stdout || applyResult.error?.message);

    assert.equal(
      fs.readFileSync(path.join(opsRoot, 'frontend', 'src', 'components', 'index.ts'), 'utf8'),
      'export const releaseComponent = true;\n',
    );
    assert.equal(
      fs.readFileSync(path.join(opsRoot, 'frontend', 'src', 'index.css'), 'utf8'),
      'body { color: green; }\n',
    );
  });
});

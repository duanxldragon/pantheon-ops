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
  fs.mkdirSync(path.dirname(syncScriptPath), { recursive: true });
  fs.mkdirSync(path.dirname(rulesScriptPath), { recursive: true });
  fs.copyFileSync(sourceSyncScript, syncScriptPath);
  fs.copyFileSync(sourceRulesScript, rulesScriptPath);
  return syncScriptPath;
}

function createSharedFrontendTree(rootPath, contents) {
  writeText(path.join(rootPath, 'frontend', 'src', 'components', 'index.ts'), contents.components);
  writeText(path.join(rootPath, 'frontend', 'src', 'core', 'layout.ts'), contents.core);
  writeText(path.join(rootPath, 'frontend', 'src', 'store', 'useStore.ts'), contents.store);
  writeText(path.join(rootPath, 'frontend', 'src', 'modules', 'auth', 'index.ts'), contents.auth);
  writeText(path.join(rootPath, 'frontend', 'src', 'modules', 'lowcode', 'index.ts'), contents.lowcode);
  writeText(path.join(rootPath, 'frontend', 'src', 'modules', 'platform', 'index.ts'), contents.platform);
  writeText(path.join(rootPath, 'frontend', 'src', 'modules', 'system', 'index.ts'), contents.system);
  writeText(path.join(rootPath, 'frontend', 'src', 'index.css'), contents.indexCss);
}

function createReleaseCache(rootPath, contents, releaseVersion = 'base-v0.8.0') {
  const releaseRoot = path.join(rootPath, '.foundation', 'releases', releaseVersion);
  writeText(path.join(rootPath, 'go.mod'), 'module pantheon-ops\n\ngo 1.24.0\n');
  writeText(path.join(releaseRoot, 'go.mod'), 'module pantheon-platform\n\ngo 1.24.0\n');
  createSharedFrontendTree(path.join(releaseRoot, 'bundle', 'shared-frontend'), contents);
  writeText(path.join(releaseRoot, 'manifest.json'), JSON.stringify({
    releaseVersion,
    releaseLine: 'release/0.8',
    baseCommit: 'deadbeefdeadbeefdeadbeefdeadbeefdeadbeef',
    sourceRepo: 'pantheon-base',
    consumerMode: 'foundation-release-consumer',
    releaseArtifact: {
      assetName: `foundation-release-${releaseVersion}.tgz`,
    },
  }, null, 2));
  writeText(path.join(rootPath, 'foundation-release.lock.json'), JSON.stringify({
    schemaVersion: 1,
    baseRepo: 'pantheon-base',
    sourceRepo: 'pantheon-base',
    consumerMode: 'foundation-release-consumer',
    releaseLine: 'release/0.8',
    releaseVersion,
    releaseDisplayName: 'v0.8.0',
    baseCommit: 'deadbeefdeadbeefdeadbeefdeadbeefdeadbeef',
    releaseArtifact: {
      githubRepo: 'duanxldragon/pantheon-base',
      tagName: releaseVersion,
      releaseName: 'v0.8.0',
      assetName: `foundation-release-${releaseVersion}.tgz`,
      localPath: `.foundation/releases/${releaseVersion}`,
    },
    sharedPaths: {
      frontend: ['frontend/src/components'],
    },
    lockedAt: '2026-07-03T00:00:00.000Z',
    lockedBy: 'consume-foundation-release',
  }, null, 2));
}

function runSync(scriptPath, cwd, args = []) {
  return spawnSync(process.execPath, [scriptPath, ...args], {
    cwd,
    encoding: 'utf8',
  });
}

test('sync-base-shared uses the local release cache and the current ops worktree path', () => {
  withTempDir((root) => {
    const opsRoot = path.join(root, 'ops-worktree-fixture');
    const syncScriptPath = copyFixtureScripts(opsRoot);

    createReleaseCache(opsRoot, {
      components: 'export const baseComponent = true;\n',
      core: 'export const baseCore = true;\n',
      store: 'export const baseStore = true;\n',
      auth: 'export const baseAuth = true;\n',
      lowcode: 'export const baseLowcode = true;\n',
      platform: 'export const basePlatform = true;\n',
      system: 'export const baseSystem = true;\n',
      indexCss: 'body { color: black; }\n',
    });
    createSharedFrontendTree(opsRoot, {
      components: 'export const oldComponent = true;\n',
      core: 'export const oldCore = true;\n',
      store: 'export const oldStore = true;\n',
      auth: 'export const oldAuth = true;\n',
      lowcode: 'export const oldLowcode = true;\n',
      platform: 'export const oldPlatform = true;\n',
      system: 'export const oldSystem = true;\n',
      indexCss: 'body { color: red; }\n',
    });

    const applyResult = runSync(syncScriptPath, opsRoot);
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
      fs.readFileSync(path.join(opsRoot, 'frontend', 'src', 'store', 'useStore.ts'), 'utf8'),
      'export const baseStore = true;\n',
    );
    assert.equal(
      fs.readFileSync(path.join(opsRoot, 'frontend', 'src', 'index.css'), 'utf8'),
      'body { color: black; }\n',
    );

    const checkResult = runSync(syncScriptPath, opsRoot, ['--check']);
    assert.equal(checkResult.status, 0, checkResult.stderr || checkResult.stdout || checkResult.error?.message);
    assert.match(checkResult.stdout, /OK shared frontend is aligned with pantheon-base/);
  });
});

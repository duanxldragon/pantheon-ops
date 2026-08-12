import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { spawnSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';
import test from 'node:test';

import {
  computeFileSha256,
  computeReleaseTreeSha256,
} from '../../scripts/foundation-release/shared-foundation-rules.mjs';

const currentFilePath = fileURLToPath(import.meta.url);
const repoRoot = path.resolve(path.dirname(currentFilePath), '..', '..');
const sourceSyncScript = path.join(repoRoot, 'frontend', 'scripts', 'sync-base-shared.mjs');
const sourceRulesScript = path.join(repoRoot, 'scripts', 'foundation-release', 'shared-foundation-rules.mjs');
const lockedArchiveChecksum = 'a'.repeat(64);

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

function writeVerificationMarker(releaseRoot, archiveSha256 = lockedArchiveChecksum) {
  const manifestPath = path.join(releaseRoot, 'manifest.json');
  const manifest = JSON.parse(fs.readFileSync(manifestPath, 'utf8'));
  writeText(
    path.join(releaseRoot, '.foundation-release-verified.json'),
    `${JSON.stringify({
      schemaVersion: 1,
      releaseVersion: manifest.releaseVersion,
      baseCommit: manifest.baseCommit,
      archiveAssetName: `foundation-release-${manifest.releaseVersion}.tgz`,
      archiveSha256,
      manifestSha256: computeFileSha256(manifestPath),
      releaseTreeSha256: computeReleaseTreeSha256(releaseRoot),
      verifiedAt: '2026-08-11T00:00:00.000Z',
    }, null, 2)}\n`,
  );
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
      releaseArtifact: {
        localPath: '.foundation/releases/base-vtest',
        checksum: lockedArchiveChecksum,
      },
      sharedPaths: {
        frontend: [
          'frontend/src/App.tsx',
          'frontend/src/main.tsx',
          'frontend/src/vite-env.d.ts',
          'frontend/src/api',
          'frontend/src/hooks',
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
  writeText(path.join(rootPath, 'frontend', 'src', 'App.tsx'), contents.app ?? 'export const app = true;\n');
  writeText(path.join(rootPath, 'frontend', 'src', 'main.tsx'), contents.main ?? 'export const main = true;\n');
  writeText(path.join(rootPath, 'frontend', 'src', 'vite-env.d.ts'), contents.viteEnv ?? '/// <reference types="vite/client" />\n');
  writeText(path.join(rootPath, 'frontend', 'src', 'api', 'file.ts'), contents.fileApi ?? 'export const fileApi = true;\n');
  writeText(path.join(rootPath, 'frontend', 'src', 'hooks', 'usePermission.ts'), contents.permissionHook ?? 'export const usePermission = true;\n');
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

test('sync-base-shared reports generic frontend files omitted from the lock', () => {
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
      components: 'export const opsComponent = true;\n',
      core: 'export const opsCore = true;\n',
      auth: 'export const opsAuth = true;\n',
      dashboard: 'export const opsDashboard = true;\n',
      system: 'export const opsSystem = true;\n',
      indexCss: 'body { color: black; }\n',
    });
    const lockPath = path.join(opsRoot, 'foundation-release.lock.json');
    const lock = JSON.parse(fs.readFileSync(lockPath, 'utf8'));
    lock.sharedPaths.frontend = lock.sharedPaths.frontend.filter((entry) => entry !== 'frontend/src/App.tsx');
    writeText(lockPath, `${JSON.stringify(lock, null, 2)}\n`);

    const result = runSync(
      syncScriptPath,
      opsRoot,
      { PANTHEON_BASE_REPO_ROOT: baseRoot },
      ['--workspace-head', '--check'],
    );
    assert.notEqual(result.status, 0);
    assert.match(result.stderr, /UNOWNED App\.tsx/);
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
          frontend: [
            'frontend/src/App.tsx',
            'frontend/src/main.tsx',
            'frontend/src/vite-env.d.ts',
            'frontend/src/api',
            'frontend/src/hooks',
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
    createSharedFrontendTree(path.join(releaseRoot, 'bundle', 'shared-frontend'), {
      components: 'export const releaseComponent = true;\n',
      core: 'export const releaseCore = true;\n',
      auth: 'export const releaseAuth = true;\n',
      dashboard: 'export const releaseDashboard = true;\n',
      system: 'export const releaseSystem = true;\n',
      indexCss: 'body { color: green; }\n',
    });
    writeVerificationMarker(releaseRoot);
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

test('sync-base-shared rejects unverified, checksum-mismatched, and modified locked releases', () => {
  withTempDir((root) => {
    const releaseRoot = path.join(root, 'release-root');
    const opsRoot = path.join(root, 'ops-worktree-fixture');
    const syncScriptPath = copyFixtureScripts(opsRoot);
    const manifestPath = path.join(releaseRoot, 'manifest.json');

    writeText(
      manifestPath,
      `${JSON.stringify({
        schemaVersion: 1,
        releaseVersion: 'base-vtest',
        releaseLine: 'release/test',
        baseCommit: 'HEAD',
        sourceRepo: 'pantheon-base',
        consumerMode: 'foundation-release-consumer',
        sharedPaths: {
          frontend: [
            'frontend/src/App.tsx',
            'frontend/src/main.tsx',
            'frontend/src/vite-env.d.ts',
            'frontend/src/api',
            'frontend/src/hooks',
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
    writeText(
      path.join(releaseRoot, 'bundle', 'shared-frontend', 'frontend', 'src', 'components', 'index.ts'),
      'export const releaseComponent = true;\n',
    );
    writeText(
      path.join(opsRoot, 'frontend', 'src', 'components', 'index.ts'),
      'export const releaseComponent = true;\n',
    );

    const env = { PANTHEON_FOUNDATION_RELEASE_ROOT: releaseRoot };
    const unverified = runSync(syncScriptPath, opsRoot, env, ['--check']);
    assert.notEqual(unverified.status, 0);
    assert.match(unverified.stderr, /verification marker not found/);

    writeVerificationMarker(releaseRoot, 'b'.repeat(64));
    const checksumMismatch = runSync(syncScriptPath, opsRoot, env, ['--check']);
    assert.notEqual(checksumMismatch.status, 0);
    assert.match(checksumMismatch.stderr, /checksum mismatch/);

    writeVerificationMarker(releaseRoot);
    writeText(
      path.join(releaseRoot, 'bundle', 'shared-frontend', 'frontend', 'src', 'components', 'index.ts'),
      'export const modifiedReleaseComponent = true;\n',
    );
    const modified = runSync(syncScriptPath, opsRoot, env, ['--check']);
    assert.notEqual(modified.status, 0);
    assert.match(modified.stderr, /contents changed after verification/);
  });
});

test('sync-base-shared checks and applies all allowlisted frontend tooling from the release', () => {
  withTempDir((root) => {
    const releaseRoot = path.join(root, 'release-root');
    const opsRoot = path.join(root, 'ops-worktree-fixture');
    const syncScriptPath = copyFixtureScripts(opsRoot);
    const toolingEntries = [
      'frontend/scripts/check-smoke-web-base.mjs',
      'frontend/scripts/export-generated-module.mjs',
      'frontend/scripts/lib/auth-cookie-session.mjs',
      'frontend/scripts/lib/css-declarations.mjs',
      'frontend/scripts/run-smoke-suite.mjs',
      'frontend/scripts/run-smoke-suite.test.mjs',
      'frontend/scripts/test-fixtures/bind-ready-server.mjs',
      'frontend/scripts/test-fixtures/fake-playwright-cli.mjs',
      'frontend/scripts/test-fixtures/record-cleanup.mjs',
      'frontend/scripts/transpile-typescript-files.mjs',
      'frontend/tests/fixtures/coverage.ts',
      'frontend/tests/smoke/helpers/auth.ts',
      'frontend/tests/smoke/helpers/fixture-policy.ts',
      'frontend/tests/smoke/helpers/shared-read-cache.ts',
      'frontend/tests/smoke/helpers/url-pattern.ts',
      'frontend/tests/smoke/platform/shell-visual-contract.spec.ts',
      'frontend/tests/smoke/system/system-pages.spec.ts',
      'frontend/tests/smoke/system/system-workspace-task-depth.ts',
    ];
    const toolingDirectory = 'frontend/tests/smoke/system';
    const lockPath = path.join(opsRoot, 'foundation-release.lock.json');
    const lock = JSON.parse(fs.readFileSync(lockPath, 'utf8'));
    lock.sharedPaths.frontend = [
      'frontend/src/App.tsx',
      'frontend/src/main.tsx',
      'frontend/src/vite-env.d.ts',
      'frontend/src/api',
      'frontend/src/hooks',
      'frontend/src/components',
      'frontend/src/core',
      'frontend/src/store',
      'frontend/src/modules/auth',
      'frontend/src/modules/dashboard',
      'frontend/src/modules/system',
      'frontend/src/index.css',
      ...toolingEntries,
      toolingDirectory,
    ];
    writeText(lockPath, `${JSON.stringify(lock, null, 2)}\n`);

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
          frontend: [
            'frontend/src/App.tsx',
            'frontend/src/main.tsx',
            'frontend/src/vite-env.d.ts',
            'frontend/src/api',
            'frontend/src/hooks',
            'frontend/src/components',
            'frontend/src/core',
            'frontend/src/store',
            'frontend/src/modules/auth',
            'frontend/src/modules/dashboard',
            'frontend/src/modules/system',
            'frontend/src/index.css',
            ...toolingEntries,
            toolingDirectory,
          ],
        },
      }, null, 2)}\n`,
    );
    createSharedFrontendTree(path.join(releaseRoot, 'bundle', 'shared-frontend'), {
      components: 'export const component = true;\n',
      core: 'export const core = true;\n',
      auth: 'export const auth = true;\n',
      dashboard: 'export const dashboard = true;\n',
      system: 'export const system = true;\n',
      indexCss: 'body { color: black; }\n',
    });
    for (const toolingEntry of toolingEntries) {
      writeText(
        path.join(releaseRoot, 'bundle', 'shared-frontend', toolingEntry),
        `release:${toolingEntry}\n`,
      );
    }
    const currentSharedSpec = `${toolingDirectory}/governance/current.spec.ts`;
    const obsoleteSharedSpec = `${toolingDirectory}/governance/legacy.spec.ts`;
    writeText(
      path.join(releaseRoot, 'bundle', 'shared-frontend', currentSharedSpec),
      'release:current shared spec\n',
    );
    writeVerificationMarker(releaseRoot);
    writeText(
      path.join(opsRoot, 'frontend', 'src', 'components', 'index.ts'),
      'export const component = true;\n',
    );

    const env = { PANTHEON_FOUNDATION_RELEASE_ROOT: releaseRoot };
    const missingResult = runSync(syncScriptPath, opsRoot, env, ['--check']);
    assert.notEqual(missingResult.status, 0);
    for (const toolingEntry of toolingEntries) {
      assert.match(missingResult.stderr, new RegExp(`MISSING ${toolingEntry.replaceAll('.', '\\.')}`));
    }
    assert.match(missingResult.stderr, /MISSING frontend\/tests\/smoke\/system\/governance\/current\.spec\.ts/);

    for (const toolingEntry of toolingEntries) {
      writeText(path.join(opsRoot, toolingEntry), `stale:${toolingEntry}\n`);
    }
    writeText(path.join(opsRoot, currentSharedSpec), 'stale:current shared spec\n');
    writeText(path.join(opsRoot, obsoleteSharedSpec), 'stale:obsolete shared spec\n');
    const driftResult = runSync(syncScriptPath, opsRoot, env, ['--check']);
    assert.notEqual(driftResult.status, 0);
    for (const toolingEntry of toolingEntries) {
      assert.match(driftResult.stderr, new RegExp(`DIFF ${toolingEntry.replaceAll('.', '\\.')}`));
    }
    assert.match(driftResult.stderr, /DIFF frontend\/tests\/smoke\/system\/governance\/current\.spec\.ts/);
    assert.match(driftResult.stderr, /OPS_ONLY frontend\/tests\/smoke\/system\/governance\/legacy\.spec\.ts/);

    const applyResult = runSync(syncScriptPath, opsRoot, env, []);
    assert.equal(applyResult.status, 0, applyResult.stderr || applyResult.stdout || applyResult.error?.message);
    for (const toolingEntry of toolingEntries) {
      assert.equal(
        fs.readFileSync(path.join(opsRoot, toolingEntry), 'utf8'),
        `release:${toolingEntry}\n`,
      );
    }
    assert.equal(fs.readFileSync(path.join(opsRoot, currentSharedSpec), 'utf8'), 'release:current shared spec\n');
    assert.equal(fs.existsSync(path.join(opsRoot, obsoleteSharedSpec)), false);

    const checkResult = runSync(syncScriptPath, opsRoot, env, ['--check']);
    assert.equal(checkResult.status, 0, checkResult.stderr || checkResult.stdout || checkResult.error?.message);
    assert.match(checkResult.stdout, /OK shared frontend is aligned/);
  });
});

test('sync-base-shared projects Base smoke contracts without erasing Ops business suites', () => {
  withTempDir((root) => {
    const releaseRoot = path.join(root, 'release-root');
    const opsRoot = path.join(root, 'ops-worktree-fixture');
    const syncScriptPath = copyFixtureScripts(opsRoot);
    const lockPath = path.join(opsRoot, 'foundation-release.lock.json');
    const lock = JSON.parse(fs.readFileSync(lockPath, 'utf8'));
    lock.sharedPaths.frontend.push(
      'frontend/package.json',
      'frontend/tests/smoke/README.md',
    );
    writeText(lockPath, `${JSON.stringify(lock, null, 2)}\n`);

    writeText(
      path.join(releaseRoot, 'manifest.json'),
      `${JSON.stringify({
        schemaVersion: 1,
        releaseVersion: 'base-vtest',
        releaseLine: 'release/test',
        baseCommit: 'HEAD',
        sourceRepo: 'pantheon-base',
        consumerMode: 'foundation-release-consumer',
        sharedPaths: lock.sharedPaths,
      }, null, 2)}\n`,
    );
    createSharedFrontendTree(path.join(releaseRoot, 'bundle', 'shared-frontend'), {
      components: 'export const component = true;\n',
      core: 'export const core = true;\n',
      auth: 'export const auth = true;\n',
      dashboard: 'export const dashboard = true;\n',
      system: 'export const system = true;\n',
      indexCss: 'body { color: black; }\n',
    });
    createSharedFrontendTree(opsRoot, {
      components: 'export const component = true;\n',
      core: 'export const core = true;\n',
      auth: 'export const auth = true;\n',
      dashboard: 'export const dashboard = true;\n',
      system: 'export const system = true;\n',
      indexCss: 'body { color: black; }\n',
    });
    writeText(
      path.join(releaseRoot, 'bundle', 'shared-frontend', 'frontend', 'package.json'),
      `${JSON.stringify({ scripts: {
        'test:smoke:business': 'npm run test:smoke:business:generated && npm run test:smoke:business:database-import',
        'test:smoke:business:generated': 'playwright test tests/smoke/business/generated/module.spec.ts',
        'test:smoke:business:database-import': 'playwright test tests/smoke/business/generated/import.spec.ts',
      } }, null, 2)}\n`,
    );
    writeText(
      path.join(releaseRoot, 'bundle', 'shared-frontend', 'frontend', 'tests', 'smoke', 'README.md'),
      '# Base Smoke Matrix\n',
    );
    writeText(
      path.join(opsRoot, 'frontend', 'package.json'),
      `${JSON.stringify({ scripts: {
        build: 'vite build',
        'test:smoke:business': 'stale aggregate',
        'test:smoke:business:cmdb': 'playwright test tests/smoke/business/cmdb/cmdb.spec.ts',
        'test:smoke:business:deploy:api': 'playwright test tests/smoke/business/deploy/deploy-api.spec.ts',
        'test:smoke:business:deploy': 'playwright test tests/smoke/business/deploy/deploy.spec.ts',
      } }, null, 2)}\n`,
    );
    writeText(
      path.join(opsRoot, 'docs', 'designs', 'BUSINESS_SMOKE_OVERLAY.md'),
      '`business/cmdb/cmdb.spec.ts`\n',
    );
    writeVerificationMarker(releaseRoot);

    const env = { PANTHEON_FOUNDATION_RELEASE_ROOT: releaseRoot };
    const applyResult = runSync(syncScriptPath, opsRoot, env, []);
    assert.equal(applyResult.status, 0, applyResult.stderr || applyResult.stdout || applyResult.error?.message);

    const nextPackage = JSON.parse(fs.readFileSync(path.join(opsRoot, 'frontend', 'package.json'), 'utf8'));
    assert.equal(nextPackage.scripts.build, 'vite build');
    assert.equal(
      nextPackage.scripts['test:smoke:business'],
      'npm run test:smoke:business:cmdb && npm run test:smoke:business:deploy:api && npm run test:smoke:business:deploy && npm run test:smoke:business:generated && npm run test:smoke:business:database-import',
    );
    assert.match(nextPackage.scripts['test:smoke:business:cmdb'], /business\/cmdb/);
    assert.match(nextPackage.scripts['test:smoke:business:deploy:api'], /business\/deploy/);
    assert.match(nextPackage.scripts['test:smoke:business:deploy'], /business\/deploy/);
    const readme = fs.readFileSync(path.join(opsRoot, 'frontend', 'tests', 'smoke', 'README.md'), 'utf8');
    assert.match(readme, /Base Smoke Matrix/);
    assert.match(readme, /Ops Business Smoke Overlay/);

    const checkResult = runSync(syncScriptPath, opsRoot, env, ['--check']);
    assert.equal(checkResult.status, 0, checkResult.stderr || checkResult.stdout || checkResult.error?.message);
  });
});

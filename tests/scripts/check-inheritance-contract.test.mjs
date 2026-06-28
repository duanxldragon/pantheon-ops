import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { spawnSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';
import test from 'node:test';

const currentFilePath = fileURLToPath(import.meta.url);
const repoRoot = path.resolve(path.dirname(currentFilePath), '..', '..');
const scriptPath = path.join(repoRoot, 'scripts', 'check-inheritance-contract.mjs');

function withTempDir(callback) {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'pantheon-inheritance-contract-'));
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

function writeJson(filePath, value) {
  writeText(filePath, `${JSON.stringify(value, null, 2)}\n`);
}

function createFixture(root) {
  writeText(
    path.join(root, 'AGENTS.md'),
    [
      'pantheon-base',
      'business/*',
      '先判断是否应在 `pantheon-base` 修复，再同步到 ops',
      '',
    ].join('\n'),
  );
  writeText(
    path.join(root, 'docs', 'README.md'),
    [
      'PROJECT_INHERITANCE.md',
      'TASK_PACKET_OPS_TEMPLATE.md',
      '',
    ].join('\n'),
  );
  writeText(
    path.join(root, 'docs', 'README.en.md'),
    [
      'PROJECT_INHERITANCE.md',
      'TASK_PACKET_OPS_TEMPLATE.md',
      '',
    ].join('\n'),
  );
  writeText(
    path.join(root, 'docs', 'PROJECT_INHERITANCE.md'),
    [
      '- Base repository：当前继承源是 `../pantheon-base`',
      '- Base release line：当前跟随 `release/0.8`',
      '- Base version：当前锁定到 `base-v0.8.6`（`dec10461ecc8d9ed1422ea1538dd6872b2a13283`）',
      'business/cmdb',
      'business/deploy',
      '如果 foundation 规则必须变更，先改 `pantheon-base`，再升级 `pantheon-ops`',
      '这次共享改动对应的 base commit 是什么',
      '共享路径哪些已同步，哪些故意未同步',
      '是否分别验证了 base 和 ops 的最小启动、build 或 smoke',
      '',
    ].join('\n'),
  );
  writeText(
    path.join(root, 'docs', 'PROJECT_INHERITANCE.en.md'),
    [
      '- Base repository: `../pantheon-base`',
      '- Base release line: `release/0.8`',
      '- Base version: `base-v0.8.6` (`dec10461ecc8d9ed1422ea1538dd6872b2a13283`)',
      'business/cmdb',
      'business/deploy',
      'update `pantheon-base` first and then upgrade `pantheon-ops`',
      'which base commit introduced the shared change',
      'which shared paths were synced and which were intentionally left out',
      'whether base and ops each received their minimum validation pass',
      '',
    ].join('\n'),
  );
  writeJson(path.join(root, 'foundation-release.lock.json'), {
    schemaVersion: 1,
    baseRepo: '../pantheon-base',
    sourceRepo: 'pantheon-base',
    consumerMode: 'foundation-release-consumer',
    releaseLine: 'release/0.8',
    releaseVersion: 'base-v0.8.6',
    releaseDisplayName: 'v0.8.6',
    baseCommit: 'dec10461ecc8d9ed1422ea1538dd6872b2a13283',
    releaseArtifact: {
      githubRepo: 'duanxldragon/pantheon-base',
      tagName: 'base-v0.8.6',
      releaseName: 'v0.8.6',
      assetName: 'foundation-release-base-v0.8.6.tgz',
      localPath: '.foundation/releases/base-v0.8.6',
    },
  });
}

function runScript(cwd) {
  return spawnSync(process.execPath, [scriptPath], {
    cwd,
    encoding: 'utf8',
  });
}

test('check-inheritance-contract passes when lock and docs are aligned', () => {
  withTempDir((root) => {
    createFixture(root);
    const result = runScript(root);
    assert.equal(result.status, 0, result.stderr || result.stdout || result.error?.message);
    assert.match(result.stdout, /OK pantheon-ops inheritance contract markers are present/);
  });
});

test('check-inheritance-contract fails when docs drift from the release lock', () => {
  withTempDir((root) => {
    createFixture(root);
    writeText(
      path.join(root, 'docs', 'PROJECT_INHERITANCE.en.md'),
      [
        '- Base repository: `../pantheon-base`',
        '- Base release line: `release/0.8`',
        '- Base version: `base-v0.8.5` (`old`)',
        'business/cmdb',
        'business/deploy',
        'update `pantheon-base` first and then upgrade `pantheon-ops`',
        'which base commit introduced the shared change',
        'which shared paths were synced and which were intentionally left out',
        'whether base and ops each received their minimum validation pass',
        '',
      ].join('\n'),
    );
    const result = runScript(root);
    assert.notEqual(result.status, 0);
    assert.match(result.stderr, /docs\/PROJECT_INHERITANCE\.en\.md: release version\/base commit must match foundation-release\.lock\.json/);
  });
});

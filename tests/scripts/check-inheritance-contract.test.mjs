import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { spawnSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';
import test from 'node:test';

const currentFilePath = fileURLToPath(import.meta.url);
const repoRoot = path.resolve(path.dirname(currentFilePath), '..', '..');
const sourceScript = path.join(repoRoot, 'scripts', 'check-inheritance-contract.mjs');

const requiredFrontendPaths = [
  'frontend/src/App.tsx',
  'frontend/src/main.tsx',
  'frontend/src/vite-env.d.ts',
  'frontend/src/api',
  'frontend/src/hooks',
];

function withTempDir(callback) {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'pantheon-inheritance-contract-'));
  try {
    callback(root);
  } finally {
    fs.rmSync(root, { recursive: true, force: true });
  }
}

function writeText(root, relativePath, content) {
  const filePath = path.join(root, relativePath);
  fs.mkdirSync(path.dirname(filePath), { recursive: true });
  fs.writeFileSync(filePath, content, 'utf8');
}

function writeFixture(root, frontendPaths = requiredFrontendPaths) {
  writeText(root, 'scripts/check-inheritance-contract.mjs', fs.readFileSync(sourceScript, 'utf8'));
  writeText(root, 'AGENTS.md', 'pantheon-base\nbusiness/*\n先判断是否应在 `pantheon-base` 修复，再同步到 ops\n');
  writeText(root, 'docs/README.md', 'PROJECT_INHERITANCE.md\nTASK_PACKET_OPS_TEMPLATE.md\n');
  writeText(root, 'docs/README.en.md', 'PROJECT_INHERITANCE.md\nTASK_PACKET_OPS_TEMPLATE.md\n');
  writeText(root, 'docs/PROJECT_INHERITANCE.md', [
    'Base repository：当前继承源是 `../pantheon-base`',
    'Base release line：当前跟随 `release/0.10`',
    'Base version：当前锁定到 `pantheon-base-v0.10.13`（`aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa`）',
    'business/cmdb', 'business/deploy', '如果 foundation 规则必须变更，先改 `pantheon-base`，再升级 `pantheon-ops`',
    '这次共享改动对应的 base commit 是什么', '共享路径哪些已同步，哪些故意未同步',
    '是否分别验证了 base 和 ops 的最小启动、build 或 smoke', '`foundation-release.lock.json`', '`npm run check:base-sync:workspace`',
  ].join('\n'));
  writeText(root, 'docs/PROJECT_INHERITANCE.en.md', [
    'Base repository: `../pantheon-base`', 'Base release line: `release/0.10`',
    'Base version: `pantheon-base-v0.10.13` (`aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa`)',
    'business/cmdb', 'business/deploy', 'update `pantheon-base` first and then upgrade `pantheon-ops`',
    'which base commit introduced the shared change', 'which shared paths were synced and which were intentionally left out',
    'whether base and ops each received their minimum validation pass', '`foundation-release.lock.json`', '`npm run check:base-sync:workspace`',
  ].join('\n'));
  writeText(root, 'foundation-release.lock.json', `${JSON.stringify({
    releaseLine: 'release/0.10', releaseVersion: 'pantheon-base-v0.10.13',
    baseCommit: 'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa', consumerMode: 'foundation-release-consumer',
    releaseArtifact: { localPath: '.foundation/releases/pantheon-base-v0.10.13' },
    sharedPaths: { frontend: frontendPaths },
  }, null, 2)}\n`);
}

function runCheck(root) {
  return spawnSync(process.execPath, ['scripts/check-inheritance-contract.mjs'], { cwd: root, encoding: 'utf8' });
}

test('check-inheritance-contract accepts complete generic frontend ownership', () => {
  withTempDir((root) => {
    writeFixture(root);
    const result = runCheck(root);
    assert.equal(result.status, 0, result.stderr || result.stdout);
  });
});

test('check-inheritance-contract rejects a lock missing shared API ownership', () => {
  withTempDir((root) => {
    writeFixture(root, requiredFrontendPaths.filter((entry) => entry !== 'frontend/src/api'));
    const result = runCheck(root);
    assert.notEqual(result.status, 0);
    assert.match(result.stderr, /sharedPaths\.frontend must include frontend\/src\/api/);
  });
});

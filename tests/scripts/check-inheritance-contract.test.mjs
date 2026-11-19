import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { spawnSync } from 'node:child_process';
import test from 'node:test';
import { resolveBaseFromLock } from '../../scripts/business-overlay/foundation-release.mjs';

const sourceScript = path.resolve('scripts/harness/check-inheritance-contract.mjs');

function withTempDir(callback) {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'pantheon-inheritance-contract-'));
  try {
    callback(root);
  } finally {
    assert.equal(path.dirname(path.resolve(root)), path.resolve(os.tmpdir()));
    fs.rmSync(root, { recursive: true, force: true });
  }
}

function writeText(root, relativePath, content) {
  const filePath = path.join(root, relativePath);
  fs.mkdirSync(path.dirname(filePath), { recursive: true });
  fs.writeFileSync(filePath, content, 'utf8');
}

function writeFixture(root) {
  writeText(root, 'docs/WORKSPACE_INHERITANCE.md', [
    '`pantheon-base`: the only authority', '`pantheon-ops`: a derived business repository',
    'Change base rules in `pantheon-base`, then let business repositories upgrade',
  ].join('\n'));
  writeText(root, 'pantheon-ops/CLAUDE.md', [
    'pantheon-ops inherits pantheon-base as its foundation', 'Business work reading order',
    '../docs/WORKSPACE_INHERITANCE.md', 'docs/PROJECT_INHERITANCE.md', '../pantheon-base/DESIGN.md',
    '../pantheon-base/AGENTS.md', 'Do not fix platform or system-domain drift locally in pantheon-ops',
  ].join('\n'));
  writeText(root, 'pantheon-ops/docs/PROJECT_INHERITANCE.md', [
    'Base repository: `../pantheon-base`', 'Base version: `pantheon-base-v0.18.9` (`' + 'a'.repeat(40) + '`)',
    'business/cmdb', 'business/deploy', 'If a foundation rule must change, update `pantheon-base` first',
  ].join('\n'));
  writeText(root, '.github/pull_request_template.md', 'Base/ops inheritance\nBase version checked\ngeneric drift\npseudo-drift\nbusiness-only\n');
}

function runCheck(root) {
  const result = spawnSync(process.execPath, [sourceScript, '--root', root, '--json', '--strict'], { encoding: 'utf8', windowsHide: true, shell: false });
  assert.ifError(result.error);
  return { ...result, report: JSON.parse(result.stdout) };
}

test('workspace inheritance accepts pinned Base-first business boundaries', () => {
  withTempDir((root) => {
    writeFixture(root);
    const result = runCheck(root);
    assert.equal(result.status, 0, result.stderr || result.stdout);
    assert.equal(result.report.findingCount, 0);
  });
});

test('workspace inheritance rejects a missing Base-first boundary and floating version', () => {
  withTempDir((root) => {
    writeFixture(root);
    writeText(root, 'pantheon-ops/docs/PROJECT_INHERITANCE.md', 'Base repository: `../pantheon-base`\nBase version: `latest`\nbusiness/cmdb\nbusiness/deploy\n');
    const result = runCheck(root);
    assert.notEqual(result.status, 0);
    assert.ok(result.report.findings.some(r => r.reason.includes('update `pantheon-base` first')));
    assert.ok(result.report.findings.some(r => r.reason.includes('explicit tag or commit hash')));
  });
});

test('official snapshot consumption rejects legacy partial-path locks without accessing the network', () => {
  withTempDir((root) => {
    const legacy = JSON.stringify({ releaseVersion: 'pantheon-base-v0.10.13', releaseLine: 'release/0.10',
      baseCommit: 'a'.repeat(40), consumerMode: 'foundation-release-consumer',
      releaseArtifact: { localPath: '.foundation/releases/pantheon-base-v0.10.13' },
      sharedPaths: { frontend: ['frontend/src/api'] } });
    writeText(root, 'foundation-release.lock.json', legacy);
    let externalCalls = 0;
    assert.throws(() => resolveBaseFromLock(root, { run: () => { externalCalls++; throw new Error('unexpected external access'); } }), /official GitHub Base release/);
    assert.equal(externalCalls, 0);
    assert.equal(fs.readFileSync(path.join(root, 'foundation-release.lock.json'), 'utf8'), legacy);
  });
});

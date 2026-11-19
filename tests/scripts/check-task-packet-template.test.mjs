import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { spawnSync } from 'node:child_process';
import test from 'node:test';
import { fileURLToPath } from 'node:url';

// Black-box test: drive scripts/check-task-packet-template.mjs (which exits
// with a status and has no named export) as a subprocess, like CI does.
const checkerPath = path.resolve(
  path.dirname(fileURLToPath(import.meta.url)),
  '../../scripts/check-task-packet-template.mjs',
);
const node = process.execPath;

// The Ops template's real required markers (mirrors the checker's
// requiredFiles/requiredMarkers for docs/README* and
// docs/TASK_PACKET_OPS_TEMPLATE*; keep in sync if the checker changes).
const OPS_FILES = [
  'docs/README.md',
  'docs/README.en.md',
  'docs/TASK_PACKET_OPS_TEMPLATE.md',
  'docs/TASK_PACKET_OPS_TEMPLATE.en.md',
];
const OPS_MARKERS = {
  'docs/README.md': ['TASK_PACKET_OPS_TEMPLATE.md', '`base -> ops`'],
  'docs/README.en.md': ['TASK_PACKET_OPS_TEMPLATE.md', '`base -> ops` sync'],
  'docs/TASK_PACKET_OPS_TEMPLATE.md': [
    '目标仓库：pantheon-ops',
    '或 `base -> ops` 同步',
    '写清 base commit',
  ],
  'docs/TASK_PACKET_OPS_TEMPLATE.en.md': [
    'Target repo: pantheon-ops',
    'or `base -> ops` sync',
    'record the base commit',
  ],
};

function writeOpsRepo(repoRoot, { withInheritanceMarkers = true } = {}) {
  const contents = {
    'docs/README.md': withInheritanceMarkers
      ? 'index [TASK_PACKET_OPS_TEMPLATE.md](./TASK_PACKET_OPS_TEMPLATE.md) for `base -> ops`\n'
      : 'index\n',
    'docs/README.en.md': withInheritanceMarkers
      ? 'index [TASK_PACKET_OPS_TEMPLATE.md](./TASK_PACKET_OPS_TEMPLATE.md) for `base -> ops` sync\n'
      : 'index\n',
    'docs/TASK_PACKET_OPS_TEMPLATE.md': withInheritanceMarkers
      ? '目标仓库：pantheon-ops\n或 `base -> ops` 同步需写清 base commit\n'
      : '目标仓库：pantheon-ops\n',
    'docs/TASK_PACKET_OPS_TEMPLATE.en.md': withInheritanceMarkers
      ? 'Target repo: pantheon-ops\nor `base -> ops` sync must record the base commit\n'
      : 'Target repo: pantheon-ops\n',
  };
  for (const [rel, content] of Object.entries(contents)) {
    const abs = path.join(repoRoot, rel);
    fs.mkdirSync(path.dirname(abs), { recursive: true });
    fs.writeFileSync(abs, content, 'utf8');
  }
  return contents;
}

function withOpsRepo(callback) {
  const repoRoot = fs.mkdtempSync(path.join(os.tmpdir(), 'pantheon-task-template-'));
  try {
    callback(repoRoot);
  } finally {
    assert.equal(
      path.dirname(path.resolve(repoRoot)),
      path.resolve(os.tmpdir()),
      'fixture must live directly under the OS temp directory',
    );
    fs.rmSync(repoRoot, { recursive: true, force: true });
  }
}

test('checker passes when all Ops template files and required markers are present', () => {
  withOpsRepo((repoRoot) => {
    const contents = writeOpsRepo(repoRoot);
    for (const rel of OPS_FILES) {
      const body = contents[rel];
      for (const marker of OPS_MARKERS[rel])
        assert.ok(body.includes(marker), `${rel} fixture must carry marker ${marker}`);
    }
    const result = spawnSync(node, [checkerPath], { cwd: repoRoot, encoding: 'utf8', windowsHide: true });
    assert.equal(result.status, 0, result.stderr);
    assert.match(result.stdout, /OK/);
  });
});

test('checker fails when a required Ops template file is missing', () => {
  withOpsRepo((repoRoot) => {
    writeOpsRepo(repoRoot);
    const missing = 'docs/TASK_PACKET_OPS_TEMPLATE.en.md';
    fs.rmSync(path.join(repoRoot, missing));
    const result = spawnSync(node, [checkerPath], { cwd: repoRoot, encoding: 'utf8', windowsHide: true });
    assert.equal(result.status, 1, 'missing required file must fail');
    assert.match(result.stderr, /missing/);
    assert.match(result.stderr, /TASK_PACKET_OPS_TEMPLATE\.en\.md/);
  });
});

test('checker fails when an inheritance marker is missing', () => {
  withOpsRepo((repoRoot) => {
    writeOpsRepo(repoRoot, { withInheritanceMarkers: false });
    const result = spawnSync(node, [checkerPath], { cwd: repoRoot, encoding: 'utf8', windowsHide: true });
    assert.equal(result.status, 1, 'missing inheritance marker must fail');
    assert.match(result.stderr, /missing required marker/);
    assert.match(result.stderr, /`base -> ops`/);
  });
});

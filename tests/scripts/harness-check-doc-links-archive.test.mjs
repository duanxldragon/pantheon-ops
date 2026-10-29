import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { spawnSync } from 'node:child_process';
import test from 'node:test';
import { fileURLToPath } from 'node:url';

const testDir = path.dirname(fileURLToPath(import.meta.url));
const checker = path.resolve(testDir, '../../scripts/harness/check-doc-links.mjs');

function runChecker(root) {
  return spawnSync(process.execPath, [checker, '--root', root, '--strict'], {
    encoding: 'utf8',
  });
}

function withTempRoot(seed) {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'pantheon-base-doc-links-'));
  try {
    fs.mkdirSync(path.join(root, 'docs'), { recursive: true });
    seed(root);
    return runChecker(root);
  } finally {
    fs.rmSync(root, { recursive: true, force: true });
  }
}

const DOC_LINKING_EVIDENCE = [
  '# Review',
  '',
  '- [Summary](../../.harness/evidence/archived-task/summary.md)',
  '',
].join('\n');

test('a canonical evidence link is satisfied once the task moved into the archive', () => {
  const result = withTempRoot((root) => {
    fs.mkdirSync(path.join(root, 'docs', 'reviews'), { recursive: true });
    fs.writeFileSync(path.join(root, 'docs', 'reviews', 'review.md'), DOC_LINKING_EVIDENCE, 'utf8');
    const evidenceDir = path.join(root, '.harness', 'archive', '2026-10', 'evidence', 'archived-task');
    fs.mkdirSync(evidenceDir, { recursive: true });
    fs.writeFileSync(path.join(evidenceDir, 'summary.md'), '# Summary\n', 'utf8');
  });

  assert.equal(result.status, 0, result.stdout + result.stderr);
  assert.match(result.stdout, /0 finding/);
});

test('a canonical evidence link that is neither live nor archived is still a finding', () => {
  const result = withTempRoot((root) => {
    fs.mkdirSync(path.join(root, 'docs', 'reviews'), { recursive: true });
    fs.writeFileSync(path.join(root, 'docs', 'reviews', 'review.md'), DOC_LINKING_EVIDENCE, 'utf8');
  });

  assert.equal(result.status, 1, result.stdout + result.stderr);
  assert.match(result.stdout, /missing internal link target/);
});

import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import test from 'node:test';
import { fileURLToPath, pathToFileURL } from 'node:url';

const testDir = path.dirname(fileURLToPath(import.meta.url));
const moduleUrl = pathToFileURL(path.resolve(testDir, '../../scripts/create-pr.mjs')).href;
const { buildGhArguments, resolveBodyFile, validateBodyFile } = await import(moduleUrl);

test('buildGhArguments creates a PR with the validated body file', () => {
  assert.deepEqual(
    buildGhArguments({
      title: 'fix(governance): enforce PR body validation',
      bodyFile: 'D:/repo/pr-body.md',
      base: 'main',
      draft: true,
    }),
    [
      'pr',
      'create',
      '--title',
      'fix(governance): enforce PR body validation',
      '--body-file',
      'D:/repo/pr-body.md',
      '--base',
      'main',
      '--draft',
    ],
  );
});

test('buildGhArguments supports editing an existing PR', () => {
  assert.deepEqual(
    buildGhArguments({
      prNumber: '222',
      bodyFile: 'D:/repo/pr-body.md',
      title: 'ignored',
      base: 'main',
      draft: false,
    }),
    ['pr', 'edit', '222', '--body-file', 'D:/repo/pr-body.md'],
  );
});

test('resolveBodyFile rejects paths outside the repository', () => {
  const repoRoot = fs.mkdtempSync(path.join(os.tmpdir(), 'pantheon-create-pr-'));
  try {
    assert.throws(
      () => resolveBodyFile(repoRoot, path.join(repoRoot, '..', 'pr-body.md')),
      /must stay inside the repository/,
    );
  } finally {
    fs.rmSync(repoRoot, { recursive: true, force: true });
  }
});

test('the checked-in SonarCloud PR body passes the CI validator', () => {
  const repoRoot = path.resolve(testDir, '../..');
  const bodyPath = '.harness/evidence/2026-08-02-sonarcloud-open-issues/pr-body.md';
  assert.equal(
    path.relative(repoRoot, validateBodyFile(repoRoot, bodyPath)).replaceAll('\\', '/'),
    bodyPath,
  );
});

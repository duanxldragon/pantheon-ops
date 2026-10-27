import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

import {
  TASK_MANIFEST_STATUSES,
  listTaskManifestPaths,
  readTaskManifest,
} from '../../scripts/task-manifest.mjs';

const TEST_DIR = path.dirname(fileURLToPath(import.meta.url));
const REPO_ROOT = path.resolve(TEST_DIR, '..', '..');

function baseManifest(overrides = {}) {
  return {
    taskId: 'sample-task',
    goal: 'sample goal',
    primaryLayer: 'platform',
    scope: { in: [], out: [] },
    linkage: {
      evidenceDir: '.harness/evidence/sample-task/',
      reviewFile: '.harness/evidence/sample-task/review.md',
      changeRef: 'chore/sample-task',
      planRefs: [],
    },
    ...overrides,
  };
}

test('accepts every canonical status value', () => {
  for (const status of TASK_MANIFEST_STATUSES) {
    assert.doesNotThrow(() =>
      readTaskManifestFromObject(baseManifest({ status })),
    );
  }
});

test('rejects status synonyms that used to drift in', () => {
  for (const status of ['done', 'complete', 'complete-with-explicit-gates']) {
    assert.throws(
      () => readTaskManifestFromObject(baseManifest({ status })),
      /taskManifest\.status must be one of/,
      `expected "${status}" to be rejected`,
    );
  }
});

test('treats status as optional so inherited repos without it stay legal', () => {
  assert.doesNotThrow(() => readTaskManifestFromObject(baseManifest()));
});

test('every committed manifest uses the canonical vocabulary', () => {
  const manifests = listTaskManifestPaths(REPO_ROOT);
  assert.ok(manifests.length > 0, 'expected at least one task manifest');
  for (const manifestPath of manifests) {
    const { payload } = readTaskManifest(REPO_ROOT, manifestPath);
    if (!('status' in payload)) {
      continue;
    }
    assert.ok(
      TASK_MANIFEST_STATUSES.has(payload.status.trim()),
      `${manifestPath}: status "${payload.status}" is not canonical`,
    );
  }
});

// validateTaskManifest is only reachable through readTaskManifest, which reads
// from disk; this helper exercises the same schema without a fixture tree.
function readTaskManifestFromObject(payload) {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'task-manifest-'));
  const manifestDir = path.join(dir, '.harness', 'tasks', 'sample-task');
  fs.mkdirSync(manifestDir, { recursive: true });
  const manifestFile = path.join(manifestDir, 'manifest.json');
  fs.writeFileSync(manifestFile, JSON.stringify(payload));
  try {
    return readTaskManifest(dir, 'sample-task');
  } finally {
    fs.rmSync(dir, { recursive: true, force: true });
  }
}

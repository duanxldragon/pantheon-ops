import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

import {
  extractTaskIdFromManifestPath,
  findArchivedTaskManifestPath,
  readTaskManifest,
} from '../../scripts/task-manifest.mjs';

const TEST_DIR = path.dirname(fileURLToPath(import.meta.url));
const REPO_ROOT = path.resolve(TEST_DIR, '..', '..');

function baseManifest(taskId) {
  return {
    taskId,
    goal: 'sample goal',
    primaryLayer: 'platform',
    scope: { in: [], out: [] },
    linkage: {
      evidenceDir: `.harness/evidence/${taskId}/`,
      reviewFile: `.harness/evidence/${taskId}/review.md`,
      changeRef: 'none',
      planRefs: [],
    },
  };
}

function writeManifest(rootDir, physicalDir, taskId) {
  const manifestDir = path.join(rootDir, physicalDir);
  fs.mkdirSync(manifestDir, { recursive: true });
  fs.writeFileSync(path.join(manifestDir, 'manifest.json'), JSON.stringify(baseManifest(taskId)));
}

function withTempRoot(run) {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'task-manifest-archive-'));
  try {
    return run(dir);
  } finally {
    fs.rmSync(dir, { recursive: true, force: true });
  }
}

test('extracts task id from archive-layout manifest paths', () => {
  assert.equal(
    extractTaskIdFromManifestPath('.harness/archive/2026-09/tasks/2026-08-11-delivery-certification/manifest.json'),
    '2026-08-11-delivery-certification',
  );
  // Canonical layout is unchanged.
  assert.equal(
    extractTaskIdFromManifestPath('.harness/tasks/sample-task/manifest.json'),
    'sample-task',
  );
  assert.equal(extractTaskIdFromManifestPath('docs/harness/tasks/x.task.md'), null);
});

test('findArchivedTaskManifestPath resolves the newest matching archive month', () => {
  withTempRoot((root) => {
    writeManifest(root, '.harness/archive/2026-08/tasks/old-task', 'old-task');
    writeManifest(root, '.harness/archive/2026-09/tasks/old-task', 'old-task');
    assert.equal(
      findArchivedTaskManifestPath(root, 'old-task'),
      '.harness/archive/2026-09/tasks/old-task/manifest.json',
    );
  });
});

test('findArchivedTaskManifestPath ignores non-month directories and missing tasks', () => {
  withTempRoot((root) => {
    fs.mkdirSync(path.join(root, '.harness', 'archive', 'not-a-month', 'tasks', 'old-task'), {
      recursive: true,
    });
    writeManifest(root, '.harness/archive/2026-09/tasks/other-task', 'other-task');
    assert.equal(findArchivedTaskManifestPath(root, 'old-task'), null);
    assert.equal(findArchivedTaskManifestPath(root, ''), null);
    assert.equal(findArchivedTaskManifestPath(root, null), null);
  });
});

test('readTaskManifest falls back to archived manifests for canonical task ids', () => {
  withTempRoot((root) => {
    writeManifest(root, '.harness/archive/2026-09/tasks/2026-08-11-delivery-certification', '2026-08-11-delivery-certification');
    const { path: resolvedPath, payload } = readTaskManifest(root, '2026-08-11-delivery-certification');
    assert.equal(resolvedPath, '.harness/archive/2026-09/tasks/2026-08-11-delivery-certification/manifest.json');
    assert.equal(payload.taskId, '2026-08-11-delivery-certification');
  });
});

test('readTaskManifest falls back to archived manifests for canonical manifest references', () => {
  withTempRoot((root) => {
    writeManifest(root, '.harness/archive/2026-07/tasks/old-task', 'old-task');
    const { path: resolvedPath } = readTaskManifest(
      root,
      '.harness/tasks/old-task/manifest.json',
    );
    assert.equal(resolvedPath, '.harness/archive/2026-07/tasks/old-task/manifest.json');
  });
});

test('readTaskManifest prefers the live manifest over an archived copy', () => {
  withTempRoot((root) => {
    writeManifest(root, '.harness/tasks/old-task', 'old-task');
    writeManifest(root, '.harness/archive/2026-09/tasks/old-task', 'old-task');
    const { path: resolvedPath } = readTaskManifest(root, 'old-task');
    assert.equal(resolvedPath, '.harness/tasks/old-task/manifest.json');
  });
});

test('readTaskManifest still fails loudly when neither live nor archived manifest exists', () => {
  withTempRoot((root) => {
    assert.throws(() => readTaskManifest(root, 'missing-task'), /task manifest does not exist/);
  });
});

test('archived manifests still validate against the manifest path task id', () => {
  withTempRoot((root) => {
    writeManifest(root, '.harness/archive/2026-09/tasks/some-task', 'different-task');
    assert.throws(
      () => readTaskManifest(root, 'some-task'),
      /taskManifest\.taskId must match manifest path task id/,
    );
  });
});

test('every committed archived manifest uses the canonical task id vocabulary', () => {
  const archiveRoot = path.join(REPO_ROOT, '.harness', 'archive');
  assert.ok(fs.existsSync(archiveRoot), 'expected an archive root to exist');
  const archived = [];
  const monthDirs = fs
    .readdirSync(archiveRoot, { withFileTypes: true })
    .filter((entry) => entry.isDirectory() && /^\d{4}-\d{2}$/.test(entry.name));
  for (const month of monthDirs) {
    const tasksDir = path.join(archiveRoot, month.name, 'tasks');
    if (!fs.existsSync(tasksDir)) {
      continue;
    }
    for (const entry of fs.readdirSync(tasksDir, { withFileTypes: true })) {
      if (entry.isDirectory()) {
        archived.push(path.join(tasksDir, entry.name, 'manifest.json'));
      }
    }
  }
  assert.ok(archived.length > 0, 'expected at least one archived manifest');
  for (const manifestFile of archived) {
    const payload = JSON.parse(fs.readFileSync(manifestFile, 'utf8'));
    assert.equal(
      payload.taskId,
      path.basename(path.dirname(manifestFile)),
      `${manifestFile}: taskId must match its directory name`,
    );
  }
});

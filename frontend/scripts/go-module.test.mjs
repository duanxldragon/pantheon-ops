import assert from 'node:assert/strict';
import fs from 'node:fs/promises';
import os from 'node:os';
import path from 'node:path';
import test from 'node:test';

import { readGoBackendImportPrefix } from '../tests/smoke/helpers/go-module.ts';

async function withRepoLayout(
  goModRelativePath,
  modulePath,
  verify,
) {
  const repoRoot = await fs.mkdtemp(path.join(os.tmpdir(), 'pantheon-go-module-'));
  try {
    const goModPath = path.join(repoRoot, goModRelativePath);
    await fs.mkdir(path.dirname(goModPath), { recursive: true });
    await fs.writeFile(goModPath, `module ${modulePath}\n\ngo 1.26.0\n`, 'utf8');
    await verify(repoRoot);
  } finally {
    await fs.rm(repoRoot, { recursive: true, force: true });
  }
}

test('resolves a backend-local Go module import prefix', async () => {
  await withRepoLayout(path.join('backend', 'go.mod'), 'pantheon-base', async (repoRoot) => {
    assert.equal(await readGoBackendImportPrefix(repoRoot), 'pantheon-base');
  });
});

test('resolves a repository-root Go module import prefix', async () => {
  await withRepoLayout('go.mod', 'pantheon-ops', async (repoRoot) => {
    assert.equal(await readGoBackendImportPrefix(repoRoot), 'pantheon-ops/backend');
  });
});

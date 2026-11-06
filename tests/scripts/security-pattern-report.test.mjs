import assert from 'node:assert/strict';
import { spawnSync } from 'node:child_process';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import test from 'node:test';
import { fileURLToPath } from 'node:url';

const repoRoot = fileURLToPath(new URL('../../', import.meta.url));

test('a token-storage match produces its warning report without crashing', (t) => {
  const tempParent = fs.realpathSync(os.tmpdir());
  const root = fs.mkdtempSync(path.join(tempParent, 'security-pattern-report-'));
  t.after(() => {
    assert.equal(path.dirname(fs.realpathSync(root)), tempParent);
    fs.rmSync(root, { recursive: true, force: true });
  });
  for (const name of ['scripts/check-security-patterns.mjs', 'scripts/harness/scan-tree.mjs', 'scripts/harness/pattern-baseline.mjs']) {
    fs.mkdirSync(path.dirname(path.join(root, name)), { recursive: true });
    fs.copyFileSync(path.join(repoRoot, name), path.join(root, name));
  }
  fs.mkdirSync(path.join(root, 'backend'));
  fs.mkdirSync(path.join(root, 'frontend/src'), { recursive: true });
  fs.writeFileSync(path.join(root, 'frontend/src/fixture.ts'), "localStorage.setItem('access_token', 'fixture-value');\n");
  const result = spawnSync(process.execPath, [path.join(root, 'scripts/check-security-patterns.mjs')], { encoding: 'utf8' });
  assert.equal(result.status, 0, result.stdout + result.stderr);
  assert.match(result.stdout, /localStorage token storage/);
  assert.match(result.stdout, /Warnings: 1/);
  assert.doesNotMatch(result.stderr, /TypeError/);
});

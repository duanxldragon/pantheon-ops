import assert from 'node:assert/strict';
import { execFileSync } from 'node:child_process';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import test from 'node:test';
import { fileURLToPath } from 'node:url';

const checkerPath = fileURLToPath(new URL('../../scripts/harness/check-checker-wiring.mjs', import.meta.url));

function makeRoot() {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'checker-wiring-'));
  fs.mkdirSync(path.join(root, 'scripts', 'harness'), { recursive: true });
  fs.mkdirSync(path.join(root, '.github', 'workflows'), { recursive: true });

  for (const name of [
    'check-wired.mjs',
    'check-in-workflow.mjs',
    'check-unwired.mjs',
    // Uses the documented mirror-only name so the exception path is exercised.
    'check-doc-frontmatter.mjs',
  ]) {
    fs.writeFileSync(path.join(root, 'scripts', 'harness', name), '');
  }

  writePackageJson(root, 'node scripts/harness/check-wired.mjs');
  fs.writeFileSync(
    path.join(root, '.github', 'workflows', 'quality.yml'),
    'jobs:\n  docs:\n    steps:\n      - run: node scripts/harness/check-in-workflow.mjs\n',
  );

  return root;
}

function writePackageJson(root, wiredScript) {
  fs.writeFileSync(
    path.join(root, 'package.json'),
    `${JSON.stringify({ scripts: { 'check:wired': wiredScript } }, null, 2)}\n`,
  );
}

function run(root) {
  try {
    const stdout = execFileSync(process.execPath, [checkerPath, '--root', root], {
      encoding: 'utf8',
      stdio: ['ignore', 'pipe', 'pipe'],
    });
    return { code: 0, output: stdout };
  } catch (error) {
    return { code: error.status, output: `${error.stdout ?? ''}${error.stderr ?? ''}` };
  }
}

test('fails when a checker is referenced by neither package.json nor a workflow', () => {
  const root = makeRoot();

  const result = run(root);

  assert.equal(result.code, 1);
  assert.match(result.output, /NOT WIRED\s+scripts\/harness\/check-unwired\.mjs/);
  assert.doesNotMatch(result.output, /NOT WIRED\s+scripts\/harness\/check-wired\.mjs/);
  assert.doesNotMatch(result.output, /NOT WIRED\s+scripts\/harness\/check-in-workflow\.mjs/);
});

test('accepts a documented exception and ignores it in the failure count', () => {
  const root = makeRoot();

  const result = run(root);

  assert.match(result.output, /exception\s+scripts\/harness\/check-doc-frontmatter\.mjs/);
  assert.match(result.output, /4 checkers \/ 2 wired \/ 1 exception \/ 1 unwired/);
});

test('passes once the unwired checker gains a package.json script', () => {
  const root = makeRoot();
  writePackageJson(root, 'node scripts/harness/check-wired.mjs && node scripts/harness/check-unwired.mjs');

  const result = run(root);

  assert.equal(result.code, 0);
  assert.match(result.output, /4 checkers \/ 3 wired \/ 1 exception \/ 0 unwired/);
});

test('reports machine-readable JSON for evidence capture', () => {
  const root = makeRoot();
  // Fully wire the tree so the JSON report is written on a passing run.
  writePackageJson(root, 'node scripts/harness/check-wired.mjs && node scripts/harness/check-unwired.mjs');

  const stdout = execFileSync(process.execPath, [checkerPath, '--root', root, '--json'], {
    encoding: 'utf8',
  });
  const report = JSON.parse(stdout);

  assert.equal(report.total, 4);
  assert.equal(report.wired, 3);
  assert.deepEqual(report.unwired, []);
});

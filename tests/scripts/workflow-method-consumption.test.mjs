import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { spawnSync } from 'node:child_process';
import test from 'node:test';

const source = fs.readFileSync(path.resolve('.github/workflows/quality.yml'), 'utf8');
const section = source.slice(source.indexOf('  docs-governance:'), source.indexOf('  frontend-contract:'));
const resolver = section.match(/node --input-type=module <<'NODE'\r?\n([\s\S]*?)\r?\n\s*NODE/)[1];

test('CI method pin rejects unreviewed branch names and output injection', (t) => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'ops-method-pin-'));
  t.after(() => fs.rmSync(root, { recursive: true, force: true }));
  fs.mkdirSync(path.join(root, '.harness'));
  const output = path.join(root, 'github-output');
  for (const pin of ['b'.repeat(40), undefined, 'main', 'b'.repeat(40) + '\ncommit=other']) {
    fs.writeFileSync(output, '');
    fs.writeFileSync(path.join(root, '.harness/method-consumption.json'), JSON.stringify({ sourceCommit: pin }));
    const result = spawnSync(process.execPath, ['--input-type=module'], {
      cwd: root, input: resolver, encoding: 'utf8', windowsHide: true,
      env: { ...process.env, GITHUB_OUTPUT: output },
    });
    if (pin === 'b'.repeat(40)) {
      assert.equal(result.status, 0, result.stderr);
      assert.equal(fs.readFileSync(output, 'utf8'), `commit=${pin}\n`);
    } else {
      assert.notEqual(result.status, 0);
      assert.equal(fs.readFileSync(output, 'utf8'), '');
    }
  }
});

test('sibling layout preserves frontend cwd, candidate checkout and blocking method/boundary checks', () => {
  assert.match(section, /working-directory:\s*pantheon-ops\s/);
  assert.match(section, /working-directory:\s*pantheon-ops\/frontend/);
  assert.match(section, /path:\s*pantheon-ops\s/);
  assert.match(section, /ref:\s*\$\{\{ needs\.change-scope\.outputs\.candidate_sha \}\}/);
  assert.match(section, /ref:\s*\$\{\{ steps\.method-source\.outputs\.commit \}\}/);
  assert.match(section, /Verify method consumption\s*\n\s*run: node \.\.\/pantheon-harness\/scripts\/harness\/check-method-consumption\.mjs --root \. --source \.\.\/pantheon-harness/);
  assert.match(section, /Check cross-layer boundaries\s*\n\s*run: node scripts\/harness\/check-boundaries\.mjs --strict --repo pantheon-ops --baseline config\/boundary-baseline\.json/);
  assert.match(section, /Check checker wiring\s*\n\s*run: npm run check:checker-wiring/);
  assert.doesNotMatch(section, /continue-on-error:\s*true|check-method-consumption\.mjs[^\n]*--update/);
  const scripts = JSON.parse(fs.readFileSync('package.json', 'utf8')).scripts;
  assert.equal(scripts['check:coverage'], 'node scripts/harness/check-coverage.mjs');
});

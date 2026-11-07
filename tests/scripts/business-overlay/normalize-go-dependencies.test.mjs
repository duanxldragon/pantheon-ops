import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { spawnSync } from 'node:child_process';
import test from 'node:test';
import { normalizeGoDependencies } from '../../../scripts/business-overlay/rebuild-from-base.mjs';

function write(root, file, text) {
  const target = path.join(root, file);
  fs.mkdirSync(path.dirname(target), { recursive: true });
  fs.writeFileSync(target, text);
}

function fixture(t) {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'pantheon-go-graph-'));
  t.after(() => fs.rmSync(root, { recursive: true, force: true }));
  write(root, 'backend/go.mod', 'module example.test/consumer\n\ngo 1.24.0\n\nrequire example.test/direct v1.0.0\nreplace example.test/direct => ../direct\nreplace example.test/indirect => ../indirect\nreplace example.test/testdep => ../testdep\n');
  write(root, 'backend/go.sum', '');
  write(root, 'backend/main.go', 'package consumer\nimport _ "example.test/direct"\n');
  write(root, 'backend/main_test.go', 'package consumer\nimport _ "example.test/testdep"\n');
  write(root, 'direct/go.mod', 'module example.test/direct\n\ngo 1.24.0\n\nrequire example.test/indirect v1.0.0\n');
  write(root, 'direct/direct.go', 'package direct\nimport _ "example.test/indirect"\n');
  write(root, 'indirect/go.mod', 'module example.test/indirect\n\ngo 1.24.0\n');
  write(root, 'indirect/indirect.go', 'package indirect\n');
  write(root, 'testdep/go.mod', 'module example.test/testdep\n\ngo 1.24.0\n');
  write(root, 'testdep/testdep.go', 'package testdep\n');
  return root;
}

function readonly(root) {
  return spawnSync('go', ['list', '-buildvcs=false', '-mod=readonly', '-deps', '-test', './...'], {
    cwd: path.join(root, 'backend'), encoding: 'utf8', env: { ...process.env, GOPROXY: 'off' },
  });
}

test('normalizes transitive and test-only imports before readonly loading without further module changes', (t) => {
  const root = fixture(t);
  const before = readonly(root);
  assert.notEqual(before.status, 0);
  assert.match(before.stderr, /updates to go.mod needed/);
  assert.deepEqual(normalizeGoDependencies(root), ['backend/go.mod', 'backend/go.sum']);
  const files = ['go.mod', 'go.sum'].map(file => fs.readFileSync(path.join(root, 'backend', file)));
  assert.match(files[0].toString(), /example\.test\/indirect v1\.0\.0/);
  assert.match(files[0].toString(), /example\.test\/testdep/);
  const after = readonly(root);
  assert.equal(after.status, 0, after.stderr);
  files.forEach((bytes, i) => assert.deepEqual(fs.readFileSync(path.join(root, 'backend', ['go.mod', 'go.sum'][i])), bytes));
});

test('fails on an unresolved dependency instead of reporting a complete graph', (t) => {
  const root = fixture(t);
  write(root, 'backend/broken.go', 'package consumer\nimport _ "example.test/direct/missing"\n');
  assert.throws(() => normalizeGoDependencies(root), /Cannot normalize consumer Go dependencies/);
});

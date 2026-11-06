import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import test from 'node:test';
import { fingerprintOf, loadFingerprints, saveFingerprints } from '../../scripts/harness/pattern-baseline.mjs';
import { scanTree } from '../../scripts/harness/scan-tree.mjs';

function makeTree(files) {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'scan-tree-'));
  for (const [relative, content] of Object.entries(files)) {
    const full = path.join(root, relative);
    fs.mkdirSync(path.dirname(full), { recursive: true });
    fs.writeFileSync(full, content);
  }
  return root;
}

test('reports repo-relative path, 1-based line and trimmed text', () => {
  const root = makeTree({ 'backend/a.go': 'package a\n\nfunc f() { db.Exec("x") }\n' });

  const { matches, missing } = scanTree({ root, dirs: ['backend/'], pattern: /Exec\(/ });

  assert.deepEqual(missing, []);
  assert.equal(matches.length, 1);
  assert.equal(matches[0].file, 'backend/a.go');
  assert.equal(matches[0].line, 3);
  assert.equal(matches[0].text, 'func f() { db.Exec("x") }');
});

test('filters by extension and by excluded suffix', () => {
  const root = makeTree({
    'backend/a.go': 'Exec(\n',
    'backend/a_test.go': 'Exec(\n',
    'backend/notes.md': 'Exec(\n',
  });

  const onlyGo = scanTree({ root, dirs: ['backend/'], pattern: /Exec\(/, extensions: new Set(['.go']) });
  assert.equal(onlyGo.matches.length, 2);

  const prod = scanTree({
    root,
    dirs: ['backend/'],
    pattern: /Exec\(/,
    extensions: new Set(['.go']),
    excludeSuffixes: ['_test.go'],
  });
  assert.deepEqual(prod.matches.map((match) => match.file), ['backend/a.go']);
});

test('never descends into build or dependency directories', () => {
  const root = makeTree({
    'frontend/src/ok.tsx': 'radial-gradient\n',
    'frontend/src/node_modules/pkg/bad.tsx': 'radial-gradient\n',
    'frontend/src/dist/bad.tsx': 'radial-gradient\n',
  });

  const { matches } = scanTree({ root, dirs: ['frontend/src/'], pattern: /radial-gradient/ });

  assert.deepEqual(matches.map((match) => match.file), ['frontend/src/ok.tsx']);
});

test('reports a missing scan root instead of returning a clean result', () => {
  const root = makeTree({ 'backend/a.go': 'package a\n' });

  const { matches, missing } = scanTree({ root, dirs: ['backend/', 'ghost/'], pattern: /x/ });

  assert.equal(matches.length, 0);
  assert.deepEqual(missing, ['ghost/']);
});

test('supports look-ahead, which the ripgrep default engine rejected', () => {
  const root = makeTree({
    'backend/a.go': 'SetCookie("session", v)\nSetCookie("session", v, Secure, HttpOnly)\n',
  });

  const { matches } = scanTree({
    root,
    dirs: ['backend/'],
    pattern: /SetCookie.*session.*(?!Secure|HttpOnly)/,
  });

  assert.equal(matches.length, 2);
});

test('baseline round-trips, de-duplicates and sorts fingerprints', () => {
  const root = makeTree({});
  const file = path.join(root, 'config', 'baseline.json');
  const match = { file: 'backend/a.go', text: 'db.Exec("x")' };
  const fingerprint = fingerprintOf('Exec', match);

  assert.equal(loadFingerprints(file), null);

  saveFingerprints(file, [fingerprint, fingerprint, 'zzz'], '2027-01-01');
  const loaded = loadFingerprints(file);

  assert.equal(loaded.size, 2);
  const payload = JSON.parse(fs.readFileSync(file, 'utf8'));
  assert.equal(payload.reviewBy, '2027-01-01');
  assert.deepEqual(payload.fingerprints, [...payload.fingerprints].sort((a, b) => a.localeCompare(b)));
});

test('fingerprint survives a line move and changes when the line changes', () => {
  const before = fingerprintOf('Exec', { file: 'backend/a.go', text: 'db.Exec("x")' });
  const moved = fingerprintOf('Exec', { file: 'backend/a.go', text: 'db.Exec("x")' });
  const edited = fingerprintOf('Exec', { file: 'backend/a.go', text: 'db.Exec("y")' });

  assert.equal(before, moved);
  assert.notEqual(before, edited);
});

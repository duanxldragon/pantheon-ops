import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { spawnSync } from 'node:child_process';
import test from 'node:test';
import { buildCandidateArtifact } from '../../../pantheon-base/scripts/foundation-release/build-candidate-artifact.mjs';
import * as consumer from './foundation-candidate.mjs';
import { rebuildFromCandidate } from './rebuild-from-candidate.mjs';

const source = fs.readFileSync(new URL('./foundation-candidate.test.mjs', import.meta.url), 'utf8');
const start = source.indexOf('function fixture(t) {');
const end = source.indexOf('\nfunction repin(', start);
assert.ok(start >= 0 && end > start);
/** @param {string} root @param {string} relative @param {string | NodeJS.ArrayBufferView} bytes */
const write = (root, relative, bytes) => {
  const file = path.join(root, relative);
  fs.mkdirSync(path.dirname(file), { recursive: true });
  fs.writeFileSync(file, bytes);
};
const fixture = new Function('fs', 'path', 'os', 'spawnSync', 'assert', 'buildCandidateArtifact', 'consumer', 'rebuildFromCandidate', 'write',
  source.slice(start, end) + '\nreturn fixture;')(fs, path, os, spawnSync, assert, buildCandidateArtifact, consumer, rebuildFromCandidate, write);
const evidence = '.harness/evidence/2026-11-07-ops-planned-task-closeout/verification/p0-integration-20261109/o2-current-live-v12/local-verification-1/exact-source-helper-inputs.tar.gz';

test('selected retained source evidence above32MiB assembles with authenticated copy and stable receipt', () => {
  const f = fixture({ after() {} });
  f.manifest.repositoryOverlayPaths.push('.harness');
  write(f.ops, 'business-overlay.json', JSON.stringify(f.manifest));
  write(f.ops, '.gitignore', '.tmp/\n');
  write(f.ops, evidence, '');
  fs.truncateSync(path.join(f.ops, evidence), consumer.LIMITS.fileBytes + 1);
  const result = f.rebuild();
  const copied = path.join(result.targetRoot, evidence);
  assert.equal(fs.statSync(copied).size, consumer.LIMITS.fileBytes + 1);
  assert.deepEqual(fs.readFileSync(copied), fs.readFileSync(path.join(f.ops, evidence)));
  assert.equal(result.receipt.sourceStable, true);
  assert.equal(result.receipt.checks.find((/** @type {{operation: string, exitCode: number}} */ check) => check.operation === 'checkSharedDrift').exitCode, 0);
  assert.equal(consumer.LIMITS.fileBytes, 32 * 1024 * 1024);
  assert.throws(() => consumer.assertInventory([{path: 'docs/ordinary.tar.gz', bytes: consumer.LIMITS.fileBytes + 1, sha256: consumer.sha256('')}]), /file metadata/);
});

test('retained evidence archive allowance refuses near-miss paths and bytes beyond512MiB', async t => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'retained-evidence-limit-'));
  const file = path.join(root, evidence);
  write(root, evidence, '');
  fs.truncateSync(file, consumer.LIMITS.fileBytes + 1);
  const captured = consumer.captureSelectedInputs(root, {declarations: [], files: [evidence]});
  const read = consumer.frozenInputReader(root, captured);
  assert.equal(read(file).length, consumer.LIMITS.fileBytes + 1);
  for (const relative of [
    evidence.replace('/exact-source-helper-inputs.tar.gz', '/other.tar.gz'),
    evidence.replace('/verification/', '/private/'),
    evidence.replace('/verification/p0-integration-20261109/o2-current-live-v12/local-verification-1/', '/verification/'),
    evidence.replace('.harness/evidence/', 'backend/evidence/'),
    evidence.replace('/2026-11-07-ops-planned-task-closeout/', '/task+name/'),
    evidence.replace('/local-verification-1/', '/run+name/'),
    'ordinary/exact-source-helper-inputs.tar.gz',
  ]) await t.test(relative, () => {
    const target = path.join(root, relative);
    fs.mkdirSync(path.dirname(target), {recursive: true});
    fs.renameSync(file, target);
    try {
      assert.throws(() => consumer.captureInputs(root, [relative]), /bounded/);
      const fake = {files: [{...captured.files[0], path: relative}]};
      assert.throws(() => consumer.frozenInputReader(root, fake)(target), /bounded/);
    } finally {fs.renameSync(target, file);}
  });
  fs.truncateSync(file, consumer.LIMITS.archiveBytes + 1);
  assert.throws(() => consumer.captureSelectedInputs(root, {declarations: [], files: [evidence]}), /bounded/);
  assert.throws(() => read(file), /Frozen input stamp changed/);
  const fake = {files: [{...captured.files[0], bytes: consumer.LIMITS.archiveBytes + 1}]};
  assert.throws(() => consumer.frozenInputReader(root, fake)(file), /bounded/);
});

test('retained evidence oversized selected reads preserve frozen byte and file-identity denials before write', t => {
  const f = fixture({ after() {} });
  write(f.ops, evidence, '');
  const file = path.join(f.ops, evidence);
  fs.truncateSync(file, consumer.LIMITS.fileBytes + 1);
  const captured = consumer.captureSelectedInputs(f.ops, {declarations: [], files: [evidence]});
  const snapshot = f.resolve();
  const io = consumer.candidateAssemblyFileSystem(f.cacheRoot, [{root: f.ops, captured}]);
  const target = path.join(f.cacheRoot, 'assembled', evidence);
  io.mkdirSync(path.dirname(target));
  const opened = new Map();
  const open = fs.openSync, close = fs.closeSync, read = fs.readFileSync;
  let hits = 0;
  t.mock.method(fs, 'openSync', function (/** @type {fs.PathLike} */ name, /** @type {string | number} */ flags, /** @type {fs.Mode} */ mode) {
    const fd = open(name, flags, mode); opened.set(fd, name); return fd;
  });
  t.mock.method(fs, 'closeSync', function (/** @type {number} */ fd) {opened.delete(fd); return close(fd);});
  t.mock.method(fs, 'readFileSync', function (/** @type {fs.PathOrFileDescriptor} */ name, /** @type {Parameters<typeof fs.readFileSync>[1]} */ options) {
    const bytes = read(name, options);
    if (typeof name === 'number' && opened.get(name) === file) {hits++; const forged = Buffer.from(bytes); forged[0] ^= 1; return forged;}
    return bytes;
  });
  assert.throws(() => io.copyFileSync(file, target), /Frozen source bytes\/hash mismatch/);
  assert.equal(hits, 1);
  assert.equal(fs.existsSync(target), false);
  assert.equal(fs.existsSync(path.join(f.cacheRoot, 'assembled/.foundation-candidate-receipt.json')), false);
  t.mock.restoreAll();
  fs.truncateSync(file, consumer.LIMITS.fileBytes + 2);
  assert.throws(() => io.copyFileSync(file, target), /Frozen input stamp changed/);
  assert.equal(fs.existsSync(target), false);
  fs.linkSync(file, path.join(f.ops, 'source-hardlink'));
  assert.throws(() => consumer.captureInputs(f.ops, [evidence]), /hard links/);
  assert.throws(() => io.copyFileSync(file, target), /Frozen input stamp changed/);
  assert.equal(fs.existsSync(target), false);
  assert.equal(snapshot.verify(), true);
});

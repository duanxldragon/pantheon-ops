import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { spawnSync } from 'node:child_process';
import test from 'node:test';
import { buildCandidateArtifact } from '../../../pantheon-base/scripts/foundation-release/build-candidate-artifact.mjs';
import * as consumer from './foundation-candidate.mjs';
import * as assembler from './rebuild-from-base.mjs';
import { rebuildFromCandidate } from './rebuild-from-candidate.mjs';

// Reuse the retained fixture without changing its existing regression contracts.
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
const fresh = () => {
  const f = fixture({ after() {} });
  f.manifest.repositoryOverlayPaths.push('.harness');
  write(f.ops, 'business-overlay.json', JSON.stringify(f.manifest));
  write(f.ops, '.gitignore', '.tmp/\n');
  write(f.ops, '.harness/report.md', 'ORIGINAL evidence\n');
  return f;
};
/** @param {import('node:test').TestContext} t @param {string} target @param {Buffer} forged */
function forgedFdRead(t, target, forged) {
  /** @type {Map<number, fs.PathLike>} */
  const opened = new Map();
  const open = fs.openSync, close = fs.closeSync, read = fs.readFileSync;
  let hits = 0;
  t.mock.method(fs, 'openSync', function (
    /** @type {fs.PathLike} */ file,
    /** @type {string | number} */ flags,
    /** @type {fs.Mode} */ mode,
  ) {
    const fd = open(file, flags, mode); opened.set(fd, file); return fd;
  });
  t.mock.method(fs, 'closeSync', function (/** @type {number} */ fd) {
    opened.delete(fd); return close(fd);
  });
  t.mock.method(fs, 'readFileSync', function (
    /** @type {fs.PathOrFileDescriptor} */ file,
    /** @type {Parameters<typeof fs.readFileSync>[1]} */ options,
  ) {
    if (typeof file === 'number' && opened.get(file) === target) { hits++; return Buffer.from(forged); }
    return read(file, options);
  });
  return { read, hits: () => hits };
}

for (const [label, relative, original, altered] of [
  ['Base', 'docs/base.md', 'Reviewed dirty candidate fixture.\n', 'Injected dirty candidate fixture.\n'],
  ['Ops', 'backend/modules/business/cmdb/module.go', 'package cmdb', 'package fake'],
  ['metadata', '.harness/report.md', 'ORIGINAL', 'INJECTED'],
]) test(`copy inlet authenticates same-length returned ${label} bytes before writing`, t => {
  const f = fresh();
  const root = label === 'Base' ? path.join(f.cacheRoot, 'snapshot') : f.ops;
  const file = path.join(root, relative);
  const before = label === 'Base' ? Buffer.from(original) : fs.readFileSync(file);
  const forged = Buffer.from(before.toString().replace(original, altered));
  assert.equal(forged.length, before.length);
  assert.notDeepEqual(forged, before);
  const hook = forgedFdRead(t, file, forged);
  assert.throws(f.rebuild, /Frozen source bytes\/hash mismatch/);
  assert.ok(hook.hits() > 0);
  assert.deepEqual(hook.read(file), before);
  assert.equal(fs.existsSync(path.join(f.cacheRoot, 'assembled', relative)), false);
  assert.equal(fs.existsSync(path.join(f.cacheRoot, 'assembled/.foundation-candidate-receipt.json')), false);
});

test('assembler source reads authenticate Base, Ops and metadata and reject unknown inputs', async t => {
  const f = fresh(), snapshot = f.resolve();
  const captured = consumer.captureSelectedInputs(f.ops, assembler.selectOverlayInputs(f.ops, f.manifest));
  const io = consumer.candidateAssemblyFileSystem(f.cacheRoot, [
    { root: snapshot.baseRoot, captured: { files: snapshot.descriptor.files } },
    { root: f.ops, captured },
  ]);
  for (const [root, relative] of [[snapshot.baseRoot, 'package.json'], [f.ops, 'package.json'], [f.ops, '.harness/report.md']]) {
    await t.test(path.relative(f.temp, path.join(root, relative)), t => {
      const file = path.join(root, relative), original = fs.readFileSync(file), forged = Buffer.from(original);
      forged[0] ^= 1;
      const hook = forgedFdRead(t, file, forged);
      /** @type {Array<BufferEncoding | undefined>} */
      const encodings = [undefined, 'utf8'];
      for (const encoding of encodings) assert.throws(() => io.readFileSync(file, encoding), /Frozen source bytes\/hash mismatch/);
      assert.equal(hook.hits(), 2);
      assert.deepEqual(hook.read(file), original);
    });
  }
  for (const file of [path.join(f.ops, 'unselected.json'), path.join(snapshot.baseRoot, 'docs/unknown.md'), path.join(f.temp, 'outside.txt')]) {
    write(path.dirname(file), path.basename(file), 'unselected');
    assert.throws(() => io.readFileSync(file), /Unselected frozen source input/);
    const target = path.join(f.cacheRoot, 'assembled/unknown.txt');
    io.mkdirSync(path.dirname(target));
    assert.throws(() => io.copyFileSync(file, target), /Unselected frozen source input/);
    assert.equal(fs.existsSync(target), false);
  }
  const assembled = path.join(f.cacheRoot, 'assembled/known.txt');
  io.writeFileSync(assembled, 'derived output');
  assert.equal(io.readFileSync(assembled, 'utf8'), 'derived output');
});

test('frozen archive reads keep exact retained-release bounds and verify returned size', t => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'copy-bound-'));
  const relative = '.foundation/releases/pantheon-base-v0.18.6/repo.tar';
  const file = path.join(root, relative);
  write(root, relative, '');
  fs.truncateSync(file, consumer.LIMITS.fileBytes + 1);
  const captured = consumer.captureSelectedInputs(root, { declarations: [], files: [relative] });
  const read = consumer.frozenInputReader(root, captured);
  assert.equal(read(file).length, consumer.LIMITS.fileBytes + 1);
  const hook = forgedFdRead(t, file, Buffer.alloc(1));
  assert.throws(() => read(file), /Frozen source bytes\/hash mismatch/);
  assert.equal(hook.hits(), 1);
  t.mock.restoreAll();
  const ordinary = path.join(root, 'ordinary.tar');
  fs.renameSync(file, ordinary);
  const ordinaryRead = consumer.frozenInputReader(root, { files: [{ ...captured.files[0], path: 'ordinary.tar' }] });
  assert.throws(() => ordinaryRead(ordinary), /bounded/);
  fs.renameSync(ordinary, file);
  fs.truncateSync(file, consumer.LIMITS.archiveBytes + 1);
  assert.throws(() => read(file), /Frozen input stamp changed|bounded/);
});

test('authenticated copy preserves import rewrite, metadata and all five locale fallbacks', () => {
  const f = fresh();
  for (const locale of ['en-US', 'zh-CN', 'ja-JP', 'ko-KR', 'fr-FR'])
    write(f.ops, `frontend/src/modules/business/cmdb/locales/${locale}.json`, '{"business.copy":"accepted"}');
  const result = f.rebuild();
  assert.equal(fs.readFileSync(path.join(result.targetRoot, 'backend/modules/business/cmdb/module.go'), 'utf8'),
    'package cmdb\nimport "pantheon-base/pkg/contracts"\n');
  assert.equal(fs.readFileSync(path.join(result.targetRoot, '.harness/report.md'), 'utf8'), 'ORIGINAL evidence\n');
  assert.equal(fs.readFileSync(path.join(result.targetRoot, 'docs/base.md'), 'utf8'), 'Reviewed dirty candidate fixture.\n');
  for (const locale of ['en-US', 'zh-CN', 'ja-JP', 'ko-KR', 'fr-FR'])
    assert.match(fs.readFileSync(path.join(result.targetRoot, `frontend/src/i18n/resources/business/${locale}.ts`), 'utf8'), /"business.copy": "accepted"/);
  assert.equal(result.receipt.sourceStable, true);
  assert.equal(result.receipt.checks.find((/** @type {{operation: string, exitCode: number}} */ check) => check.operation === 'checkSharedDrift').exitCode, 0);
});

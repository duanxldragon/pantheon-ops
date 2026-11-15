import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { spawnSync } from 'node:child_process';
import childProcess from 'node:child_process';
import { syncBuiltinESMExports } from 'node:module';
import test from 'node:test';
import { buildCandidateArtifact } from '../../../pantheon-base/scripts/foundation-release/build-candidate-artifact.mjs';
import * as consumer from './foundation-candidate.mjs';
import { rebuildFromCandidate } from './rebuild-from-candidate.mjs';
import { normalizeGoDependencies } from './rebuild-from-base.mjs';

// The exact fixture extraction used by independent CODE's adversarial-checks-v2
// and replacement-assembly-check. Attack assertions do not reuse implementation.
const source = fs.readFileSync(new URL('./foundation-candidate.test.mjs', import.meta.url), 'utf8');
const start = source.indexOf('function fixture(t) {');
const end = source.indexOf('\nfunction repin(', start);
const write = (root, file, bytes) => {
  const target = path.join(root, file);
  fs.mkdirSync(path.dirname(target), { recursive: true });
  fs.writeFileSync(target, bytes);
};
const fixture = new Function('fs', 'path', 'os', 'spawnSync', 'assert', 'buildCandidateArtifact', 'consumer', 'rebuildFromCandidate', 'write',
  source.slice(start, end) + '\nreturn fixture;')(fs, path, os, spawnSync, assert, buildCandidateArtifact, consumer, rebuildFromCandidate, write);
const retainedFixture = () => fixture({ after() {} });
function forbidDeletion(t) {
  let attempts = 0;
  for (const method of ['unlinkSync', 'rmdirSync']) t.mock.method(fs, method, () => {
    attempts++; throw new Error('Test preserved attempted deletion');
  });
  return () => attempts;
}
function replace(f, directory) {
  const retained = path.join(f.temp, 'retained-' + path.basename(directory));
  fs.renameSync(directory, retained);
  fs.mkdirSync(directory);
  return retained;
}

test('independent HIGH1: observed directory replacement refuses copy and preserves foreign bytes', () => {
  const f = retainedFixture(), snapshot = f.resolve();
  const io = consumer.candidateAssemblyFileSystem(f.cacheRoot);
  const docs = path.join(f.cacheRoot, 'assembled/docs'); io.mkdirSync(docs);
  const retained = replace(f, docs), target = path.join(docs, 'base.md');
  fs.writeFileSync(target, 'foreign original must remain');
  assert.throws(() => io.copyFileSync(path.join(snapshot.baseRoot, 'docs/base.md'), target), /Replaced directory identity/);
  assert.equal(fs.readFileSync(target, 'utf8'), 'foreign original must remain');
  assert.equal(fs.statSync(retained).isDirectory(), true);
});

test('independent HIGH1: replacement at native copy/open cannot clobber or emit fixture success receipt', t => {
  const f = retainedFixture(), docs = path.join(f.cacheRoot, 'assembled/docs'), target = path.join(docs, 'base.md');
  let fired = false, retained;
  const attack = file => { if (file === target && !fired) {
    fired = true; retained = replace(f, docs); fs.writeFileSync(target, 'foreign original must remain');
  }};
  const copy = fs.copyFileSync, open = fs.openSync;
  t.mock.method(fs, 'copyFileSync', function(src, dst, ...args) { attack(dst); return copy.call(this, src, dst, ...args); });
  t.mock.method(fs, 'openSync', function(file, flags, ...args) {
    if (typeof flags === 'number' && (flags & fs.constants.O_CREAT)) attack(file);
    return open.call(this, file, flags, ...args);
  });
  const deletions = forbidDeletion(t);
  assert.throws(f.rebuild, /Replaced directory identity|EEXIST/);
  assert.equal(fired, true);
  assert.equal(fs.readFileSync(target, 'utf8'), 'foreign original must remain');
  assert.equal(fs.statSync(retained).isDirectory(), true);
  assert.equal(fs.existsSync(path.join(f.cacheRoot, 'assembled/.foundation-candidate-receipt.json')), false);
  assert.equal(deletions(), 0);
});

test('independent HIGH2: mkdir interception never authorizes foreign directory deletion', t => {
  const f = retainedFixture(); f.resolve();
  const io = consumer.candidateAssemblyFileSystem(f.cacheRoot), docs = path.join(f.cacheRoot, 'assembled/docs');
  const mkdir = fs.mkdirSync; let fired = false, retained;
  t.mock.method(fs, 'mkdirSync', function(file, ...args) {
    const result = mkdir.call(this, file, ...args);
    if (file === docs && !fired) { fired = true; retained = replace(f, docs); fs.writeFileSync(path.join(docs, 'foreign.txt'), 'preserve'); }
    return result;
  });
  try { io.mkdirSync(docs); } catch (error) { assert.match(error.message, /Unproven|Unknown/); }
  const deletions = forbidDeletion(t), primary = new Error('independent primary failure');
  assert.throws(() => consumer.removeCandidateStage(f.cacheRoot, { cause: primary }), error => {
    assert.equal(error.cause, primary); assert.match(error.message, /Unknown|Unproven|deferred/); return true;
  });
  assert.equal(fired, true); assert.equal(deletions(), 0);
  assert.equal(fs.readFileSync(path.join(docs, 'foreign.txt'), 'utf8'), 'preserve');
  assert.equal(fs.statSync(retained).isDirectory(), true);
});

test('independent HIGH3: unknown ordinary sibling rejects whole cleanup preflight before any deletion', t => {
  const f = retainedFixture(), snapshot = f.resolve(), io = consumer.candidateAssemblyFileSystem(f.cacheRoot);
  const assembled = path.join(f.cacheRoot, 'assembled'); io.mkdirSync(assembled);
  const foreign = path.join(assembled, 'foreign-sibling.txt'); fs.writeFileSync(foreign, 'preserve');
  const deletions = forbidDeletion(t);
  assert.throws(() => consumer.removeCandidateStage(f.cacheRoot), /Unknown.*file/);
  assert.equal(deletions(), 0); assert.equal(fs.readFileSync(foreign, 'utf8'), 'preserve');
  assert.equal(fs.existsSync(path.join(snapshot.baseRoot, 'backend/go.mod')), true);
});

for (const excluded of ['docs/base.md', 'docs']) test(`independent MEDIUM4: ${excluded} exclusion preserves drift validation and produces fixture receipt`, () => {
  const f = retainedFixture(); f.manifest.baseExcludedPaths = [excluded];
  write(f.ops, 'business-overlay.json', JSON.stringify(f.manifest));
  // This fixture intentionally covers docs in sharedPaths. A missing shared
  // file requires its existing explicit drift declaration, even when excluded.
  write(f.ops, '.foundation/allowed-drift.json', JSON.stringify({ 'docs/base.md': 'fixture deliberate exclusion' }));
  const result = f.rebuild();
  assert.equal(fs.existsSync(path.join(result.targetRoot, excluded)), false);
  assert.equal(result.receipt.sourceStable, true);
});

test('file exclusions require recorded exclusive-created identity; replacement and unknown file remain', () => {
  const f = retainedFixture(); f.resolve(); const io = consumer.candidateAssemblyFileSystem(f.cacheRoot);
  const directory = path.join(f.cacheRoot, 'assembled'); io.mkdirSync(directory);
  const known = path.join(directory, 'known.txt'); io.writeFileSync(known, 'owned');
  assert.throws(() => io.rmSync(known), /cleanup deferred/);
  assert.equal(fs.readFileSync(known, 'utf8'), 'owned');
  fs.renameSync(known, path.join(f.temp, 'retained-known.txt')); fs.writeFileSync(known, 'foreign');
  assert.throws(() => io.rmSync(known), /Replaced.*file/);
  assert.throws(() => io.writeFileSync(known, 'clobber'), /Replaced.*file/);
  const unknown = path.join(directory, 'unknown.txt'); fs.writeFileSync(unknown, 'foreign');
  assert.throws(() => io.rmSync(unknown), /Unknown.*file|Replaced.*file/);
  assert.throws(() => io.writeFileSync(unknown, 'clobber'), /Unknown.*file/);
  assert.equal(fs.readFileSync(known, 'utf8'), 'foreign');
  assert.equal(fs.readFileSync(unknown, 'utf8'), 'foreign');
});

test('root and assembler directory cleanup is explicitly deferred with no mutation or success', t => {
  const f = retainedFixture(); f.resolve(); const io = consumer.candidateAssemblyFileSystem(f.cacheRoot);
  const assembled = path.join(f.cacheRoot, 'assembled'); io.mkdirSync(assembled); io.writeFileSync(path.join(assembled, 'known.txt'), 'keep');
  const deletions = forbidDeletion(t), primary = new Error('same primary object', { cause: new Error('same nested cause') });
  assert.throws(() => consumer.removeCandidateStage(f.cacheRoot, { cause: primary }), error => {
    assert.equal(error.cause, primary); assert.equal(error.cleanupStatus, 'deferred-unproven');
    assert.ok(error.retainedDirectories.includes(f.cacheRoot)); assert.ok(error.retainedDirectories.includes(assembled)); return true;
  });
  assert.equal(deletions(), 0); assert.equal(fs.readFileSync(path.join(assembled, 'known.txt'), 'utf8'), 'keep');
});

test('receipt open rechecks every ancestor and never writes into a replacement target', t => {
  const f = retainedFixture(), receipt = path.join(f.cacheRoot, 'assembled/.foundation-candidate-receipt.json');
  const open = fs.openSync; let fired = false;
  t.mock.method(fs, 'openSync', function(file, flags, ...args) {
    if (file === receipt && !fired) {
      fired = true; replace(f, path.dirname(receipt)); fs.writeFileSync(receipt, 'foreign receipt');
    }
    return open.call(this, file, flags, ...args);
  });
  const deletions = forbidDeletion(t);
  assert.throws(f.rebuild, /Replaced directory identity|EEXIST/);
  assert.equal(fired, true); assert.equal(fs.readFileSync(receipt, 'utf8'), 'foreign receipt'); assert.equal(deletions(), 0);
});

test('existing-file open never truncates a replaced regular file', t => {
  const f = retainedFixture(); f.resolve(); const io = consumer.candidateAssemblyFileSystem(f.cacheRoot);
  const directory = path.join(f.cacheRoot, 'assembled'); io.mkdirSync(directory);
  const target = path.join(directory, 'known.txt'); io.writeFileSync(target, 'owned');
  const open = fs.openSync; let fired = false;
  t.mock.method(fs, 'openSync', function(file, flags, ...args) {
    if (file === target && !fired) {
      fired = true; assert.equal(flags & fs.constants.O_TRUNC, 0);
      fs.renameSync(target, path.join(f.temp, 'retained-original-file')); fs.writeFileSync(target, 'foreign');
    }
    return open.call(this, file, flags, ...args);
  });
  assert.throws(() => io.writeFileSync(target, 'clobber'), /Replaced.*file/);
  assert.equal(fired, true); assert.equal(fs.readFileSync(target, 'utf8'), 'foreign');
});

test('replacement during descriptor write preserves foreign file and rejects publication', t => {
  const f = retainedFixture(); f.resolve(); const io = consumer.candidateAssemblyFileSystem(f.cacheRoot);
  const directory = path.join(f.cacheRoot, 'assembled'); io.mkdirSync(directory);
  const target = path.join(directory, 'known.txt'); io.writeFileSync(target, 'owned');
  const nativeWrite = fs.writeFileSync; let fired = false, descriptorBlockedReplacement = false;
  t.mock.method(fs, 'writeFileSync', function(file, ...args) {
    if (typeof file === 'number' && !fired) {
      fired = true;
      try { replace(f, directory); }
      catch (error) {
        if (process.platform === 'win32' && error.code === 'EPERM') descriptorBlockedReplacement = true;
        throw error;
      }
      nativeWrite(target, 'foreign');
    }
    return nativeWrite.call(this, file, ...args);
  });
  assert.throws(() => io.writeFileSync(target, 'new owned bytes'), /Replaced directory identity|EPERM/);
  assert.equal(fired, true);
  if (descriptorBlockedReplacement) {
    assert.equal(fs.readFileSync(target).subarray(0, 5).toString(), 'owned');
    io.assertTargetStable();
  } else {
    assert.equal(fs.readFileSync(target, 'utf8'), 'foreign');
    assert.throws(() => io.assertTargetStable(), /Replaced directory identity/);
  }
});

test('unknown sibling blocks known file exclusion before first unlink', t => {
  const f = retainedFixture(); f.resolve(); const io = consumer.candidateAssemblyFileSystem(f.cacheRoot);
  const directory = path.join(f.cacheRoot, 'assembled'); io.mkdirSync(directory);
  const known = path.join(directory, 'known.txt'); io.writeFileSync(known, 'keep known');
  const foreign = path.join(directory, 'z-foreign.txt'); fs.writeFileSync(foreign, 'keep foreign');
  const deletions = forbidDeletion(t);
  assert.throws(() => io.rmSync(known), /Unknown.*file/);
  assert.equal(deletions(), 0); assert.equal(fs.readFileSync(known, 'utf8'), 'keep known');
});

test('final target validation refuses a recorded file or directory removed from the observed tree', () => {
  const f = retainedFixture(); f.resolve(); const io = consumer.candidateAssemblyFileSystem(f.cacheRoot);
  const assembled = path.join(f.cacheRoot, 'assembled'); io.mkdirSync(assembled);
  const file = path.join(assembled, 'known.txt'); io.writeFileSync(file, 'retain');
  fs.renameSync(file, path.join(f.temp, 'retained-missing-file'));
  assert.throws(() => io.assertTargetStable(), /Missing owned regular file identity/);
  fs.renameSync(path.join(f.temp, 'retained-missing-file'), file);
  const nested = path.join(assembled, 'nested'); io.mkdirSync(nested);
  fs.renameSync(nested, path.join(f.temp, 'retained-missing-directory'));
  assert.throws(() => io.assertTargetStable(), /Missing observed directory identity/);
});

test('Go normalization uses alternate modfiles and imports through owned descriptors', () => {
  const f = retainedFixture(); f.resolve(); const io = consumer.candidateAssemblyFileSystem(f.cacheRoot);
  const assembled = path.join(f.cacheRoot, 'assembled'); io.mkdirSync(path.join(assembled, 'backend'));
  io.writeFileSync(path.join(assembled, 'backend/go.mod'), 'module example.test/owned\n\ngo 1.24.0\n');
  io.writeFileSync(path.join(assembled, 'backend/go.sum'), '');
  io.writeFileSync(path.join(assembled, 'backend/main.go'), 'package owned\n');
  assert.deepEqual(normalizeGoDependencies(assembled, io), ['backend/go.mod', 'backend/go.sum']);
  io.assertTargetStable();
  const work = fs.readdirSync(f.cacheRoot).find(name => name.startsWith('go-work-'));
  assert.ok(work); assert.equal(fs.existsSync(path.join(f.cacheRoot, work, 'go.mod')), true);
  assert.throws(() => consumer.removeCandidateStage(f.cacheRoot), /Unknown.*file/);
});

test('Go boundary rejects observed ancestor replacement before import and preserves primary failure', () => {
  const f = retainedFixture(); f.resolve(); const io = consumer.candidateAssemblyFileSystem(f.cacheRoot);
  const assembled = path.join(f.cacheRoot, 'assembled'); io.mkdirSync(path.join(assembled, 'backend'));
  const mod = path.join(assembled, 'backend/go.mod'); io.writeFileSync(mod, 'module example.test/owned\n');
  io.writeFileSync(path.join(assembled, 'backend/go.sum'), '');
  const original = childProcess.spawnSync;
  let fired = false;
  childProcess.spawnSync = (command, args) => {
    fired = true; assert.equal(command, 'go'); assert.ok(args.some(arg => arg.startsWith('-modfile=')));
    replace(f, assembled); write(assembled, 'backend/go.mod', 'foreign mod must remain');
    return { status: 1, stderr: 'independent Go primary failure' };
  };
  syncBuiltinESMExports();
  try {
    assert.throws(() => io.runGo(['mod', 'edit', '-require=example.test/a@v1.0.0'], { cwd: path.dirname(mod), encoding: 'utf8' }), /independent Go primary failure/);
  } finally { childProcess.spawnSync = original; syncBuiltinESMExports(); }
  assert.equal(fired, true); assert.equal(fs.readFileSync(mod, 'utf8'), 'foreign mod must remain');
});

test('throwing Go primary object and nested cause survive identity validation', () => {
  const f = retainedFixture(); f.resolve(); const io = consumer.candidateAssemblyFileSystem(f.cacheRoot);
  const assembled = path.join(f.cacheRoot, 'assembled'); io.mkdirSync(path.join(assembled, 'backend'));
  const mod = path.join(assembled, 'backend/go.mod'); io.writeFileSync(mod, 'module example.test/owned\n');
  io.writeFileSync(path.join(assembled, 'backend/go.sum'), '');
  const primary = new Error('same thrown Go primary', { cause: new Error('same inner Go cause') });
  const original = childProcess.spawnSync;
  childProcess.spawnSync = () => { replace(f, assembled); throw primary; };
  syncBuiltinESMExports();
  try {
    assert.throws(() => io.runGo(['mod', 'edit', '-require=example.test/a@v1.0.0'], { cwd: path.dirname(mod), encoding: 'utf8' }), error => error === primary && error.cause === primary.cause);
    assert.throws(() => consumer.removeCandidateStage(f.cacheRoot, {cause:primary}), error => error.cause === primary);
  } finally { childProcess.spawnSync = original; syncBuiltinESMExports(); }
});

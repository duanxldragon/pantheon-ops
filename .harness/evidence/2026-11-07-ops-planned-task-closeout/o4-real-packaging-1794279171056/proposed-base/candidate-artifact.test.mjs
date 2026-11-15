import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { spawnSync } from 'node:child_process';
import test from 'node:test';
import { buildCandidateArtifact } from './build-candidate-artifact.mjs';
import { aggregateSha256, assertCandidatePath, assertInventory, CAPABILITY_EXPORTS, readCandidateDescriptor, sha256, validateCandidateDescriptor, verifyCandidateArchive, verifyCandidateExecutable } from './candidate-descriptor.mjs';

function fixture(t) {
  const temporary = fs.mkdtempSync(path.join(os.tmpdir(), 'pantheon-candidate-test-'));
  t.after(() => fs.rmSync(temporary, { recursive: true, force: true }));
  const root = path.join(temporary, 'source'); fs.mkdirSync(root);
  const write = (relative, bytes) => {
    const file = path.join(root, relative); fs.mkdirSync(path.dirname(file), { recursive: true }); fs.writeFileSync(file, bytes);
  };
  const api = 'backend/pkg/permissionroutes/registry.go';
  write(api, 'package permissionroutes\nfunc Register() {}\n');
  write('frontend/src/shared.ts', 'export const fixture = true;\n');
  write('docs/fixture.md', 'Explicit fixture, not a product snapshot.\n');
  const git = (...args) => {
    const result = spawnSync('git', args, { cwd: root, encoding: 'utf8' });
    assert.equal(result.status, 0, result.stderr); return result.stdout.trim();
  };
  git('init'); git('-c', 'core.autocrlf=false', 'add', '.');
  git('-c', 'user.name=Candidate fixture', '-c', 'user.email=fixture@example.invalid', 'commit', '-m', 'fixture provenance');
  write(api, 'package permissionroutes\nfunc Register() {}\n// reviewed uncommitted fixture\n');
  write('backend/pkg/permissionroutes/new_test.go', 'package permissionroutes\n// explicit untracked fixture\n');
  const options = { root, baseCommit: git('rev-parse', 'HEAD'), sourcePaths: ['backend', 'frontend', 'docs'], sharedPaths: { backend: ['backend'], frontend: ['frontend/src'], docs: ['docs'] }, evidenceIds: ['fixture-source-verification'], reviewIds: ['fixture-review-reference'], capabilities: ['permission-route-registration-v1'], outputRoot: path.join(temporary, 'sealed') };
  return { temporary, root, write, api, options, build: (extra = {}) => buildCandidateArtifact({ ...options, ...extra }) };
}

test('seals exact dirty/untracked fixture bytes with detached descriptor and genuine provenance', (t) => {
  const f = fixture(t); const result = f.build();
  const descriptor = readCandidateDescriptor(result.descriptorPath, result.descriptorSha256);
  assert.equal(result.sourceStable, true);
  assert.equal(descriptor.baseCommit, f.options.baseCommit);
  assert.equal(descriptor.aggregateSha256, aggregateSha256(descriptor.files));
  assert.equal(descriptor.candidateIdentity, `candidate-${descriptor.aggregateSha256}`);
  assert.equal(Object.hasOwn(descriptor, 'releaseVersion'), false);
  const archive = fs.readFileSync(result.artifactPath);
  const entries = verifyCandidateArchive(archive, descriptor);
  assert.equal(entries.length, 4);
  for (const entry of entries) assert.deepEqual(entry.bytes, fs.readFileSync(path.join(f.root, entry.path)));
  assert.ok(descriptor.files.some((file) => file.path.endsWith('/new_test.go')));
  assert.equal(sha256(fs.readFileSync(result.descriptorPath)), result.descriptorSha256);
  assert.throws(() => f.build(), /must not already exist/);
  assert.deepEqual(fs.readdirSync(result.outputRoot).sort(), ['candidate-descriptor.json', descriptor.artifact.assetName].sort());
});

test('identical source inventory has deterministic archives; review revisions have distinct descriptor digests', (t) => {
  const f = fixture(t); const first = f.build();
  const second = f.build({ outputRoot: path.join(f.temporary, 'second'), reviewIds: ['fixture-another-review'] });
  assert.equal(first.descriptor.candidateIdentity, second.descriptor.candidateIdentity);
  assert.equal(first.descriptor.artifact.sha256, second.descriptor.artifact.sha256);
  assert.notEqual(first.descriptorSha256, second.descriptorSha256);
});

test('descriptor tampering, archive tampering and missing pins refuse', (t) => {
  const f = fixture(t); const result = f.build();
  assert.throws(() => readCandidateDescriptor(result.descriptorPath), /Pinned/);
  const archive = fs.readFileSync(result.artifactPath); archive[512] ^= 1;
  assert.throws(() => verifyCandidateArchive(archive, result.descriptor), /digest/);
  fs.appendFileSync(result.descriptorPath, ' ');
  assert.throws(() => readCandidateDescriptor(result.descriptorPath, result.descriptorSha256), /digest mismatch/);
  for (const delta of [{ kind: 'foundation-release' }, { baseCommit: 'not-a-commit' }, { schemaVersion: 2 }, { releaseVersion: 'pantheon-base-v0.18.6' }, { aggregateSha256: '0'.repeat(64) }, { candidateIdentity: 'candidate-false' }, { requiredExports: [] }, { capabilities: ['invented-capability'] }, { evidenceIds: [] }, { sharedPaths: { backend: ['missing'] } }]) {
    assert.throws(() => validateCandidateDescriptor({ ...result.descriptor, ...delta }));
  }
});

test('rejects unsafe/secret/output paths, platform aliases and file/directory collisions', (t) => {
  const f = fixture(t);
  for (const unsafe of ['../escape', '/absolute', 'C:/outside', '//host/share', 'dir/../escape', 'dir\\file', 'x:y', 'x\0y', 'file.', 'file ', 'aux.txt', 'COM¹.txt', 'a|b', 'a?b', 'a*b', 'a<b', 'a"b', 'a//b', '.git/config', '.env.local', '.aws/credentials', 'backend/uploads/file', 'frontend/node_modules/a', '.harness/evidence/file']) assert.throws(() => assertCandidatePath(unsafe));
  const row = (file) => ({ path: file, bytes: 0, sha256: sha256('') });
  for (const files of [['a', 'a'], ['A', 'a'], ['Foo/a', 'foo/b'], ['a', 'a/b'], ['a/b', 'b', 'a']]) assert.throws(() => assertInventory(files.map(row)));
  for (const outputRoot of [f.root, path.dirname(f.root), path.join(f.root, 'backend', 'output'), path.join(f.root, 'releases', 'candidate'), path.join(f.root, 'dist', 'candidate')]) assert.throws(() => f.build({ outputRoot }));
  f.write('.env', 'synthetic fixture secret');
  assert.throws(() => f.build({ sourcePaths: ['.env'] }));
  f.write('config/synthetic.json', '{}');
  assert.throws(() => f.build({ sourcePaths: ['config'] }), /explicitly declared files/);
  if (process.platform === 'win32') assert.throws(() => f.build({ outputRoot: path.join(f.root.toUpperCase(), 'backend', 'new') }), /overlaps/);
  assert.equal(fs.existsSync(f.options.outputRoot), false);
});

test('protected target and fixture lock remain byte-identical; write failure leaves no partial candidate', (t) => {
  const f = fixture(t);
  const protectedTarget = path.join(f.temporary, 'protected'); fs.mkdirSync(protectedTarget);
  const lock = path.join(protectedTarget, 'foundation-release.lock.json');
  const originalLock = Buffer.from('{"fixture":"official-lock-unchanged"}'); fs.writeFileSync(lock, originalLock);
  assert.throws(() => f.build({ outputRoot: protectedTarget }), /must not already exist/);
  assert.deepEqual(fs.readFileSync(lock), originalLock);
  const originalWrite = fs.writeFileSync;
  t.mock.method(fs, 'writeFileSync', function (file, ...args) {
    if (typeof file === 'string' && file.endsWith('candidate-descriptor.json')) throw new Error('fixture disk write failure');
    return originalWrite.call(this, file, ...args);
  });
  assert.throws(() => f.build(), /disk write failure/);
  assert.equal(fs.existsSync(f.options.outputRoot), false);
  assert.equal(fs.readdirSync(f.temporary).some((entry) => entry.startsWith('.candidate-seal-')), false);
  assert.deepEqual(fs.readFileSync(lock), originalLock);
});

test('source junctions, hard links and redirected output directories refuse', (t) => {
  const f = fixture(t);
  fs.symlinkSync(path.join(f.root, 'frontend'), path.join(f.root, 'linked'), 'junction');
  assert.throws(() => f.build({ sourcePaths: ['linked'] }), /real directories/);
  fs.linkSync(path.join(f.root, f.api), path.join(f.root, 'hardlink.go'));
  assert.throws(() => f.build(), /hard links/);
  fs.symlinkSync(path.join(f.root, 'docs'), path.join(f.temporary, 'redirect'), 'junction');
  assert.throws(() => f.build({ outputRoot: path.join(f.temporary, 'redirect', 'new') }), /real directories/);
});

test('exported producer preserves directory replacements and original write/verification failures', async (t) => {
  for (const phase of ['write', 'verification']) for (const directory of ['stage', 'ancestor', 'nested']) await t.test(`${phase}/${directory}`, (t) => {
    const f = fixture(t);
    const originalWrite = fs.writeFileSync;
    const originalRead = fs.readFileSync;
    const retained = path.join(f.temporary, 'retained-original');
    const parent = path.join(f.temporary, 'output-parent');
    const outputRoot = path.join(parent, 'sealed');
    const primary = new Error(`injected producer ${phase} failure`);
    let stage; let attacked; let originalID; let replacementID;
    const replace = (file) => {
      stage = path.dirname(file);
      attacked = directory === 'ancestor' ? parent : directory === 'nested' ? path.join(stage, 'candidate-descriptor.json') : stage;
      if (directory === 'nested') {
        // The producer creates no nested directories. Even a known file name
        // must not authorize recursion into a foreign directory at that path.
        if (fs.existsSync(attacked)) fs.unlinkSync(attacked);
        fs.mkdirSync(attacked);
        originalWrite(path.join(attacked, 'original.txt'), 'retain original nested directory');
      }
      originalID = fs.lstatSync(attacked, { bigint: true }).ino;
      fs.renameSync(attacked, retained);
      fs.mkdirSync(attacked);
      replacementID = fs.lstatSync(attacked, { bigint: true }).ino;
      originalWrite(path.join(attacked, 'foreign.txt'), 'preserve foreign replacement');
      throw primary;
    };
    const isArchive = (file) => typeof file === 'string' && file.endsWith('.tar') && path.basename(path.dirname(file)).startsWith('.candidate-seal-');
    if (phase === 'write') t.mock.method(fs, 'writeFileSync', function (file, ...args) {
      const result = originalWrite.call(this, file, ...args);
      if (isArchive(file)) replace(file);
      return result;
    });
    else t.mock.method(fs, 'readFileSync', function (file, ...args) {
      if (isArchive(file)) replace(file);
      return originalRead.call(this, file, ...args);
    });
    try {
      assert.throws(() => f.build({ outputRoot }), (error) => {
        assert.match(error.message, /Candidate cleanup refused/);
        assert.equal(error.cause, primary);
        return true;
      });
    } finally { t.mock.restoreAll(); }
    assert.ok(attacked, 'actual producer failure hook ran');
    assert.notEqual(originalID, replacementID, 'actual directory instances differ');
    assert.equal(fs.readFileSync(path.join(attacked, 'foreign.txt'), 'utf8'), 'preserve foreign replacement');
    const retainedArchive = directory === 'ancestor' ? path.join(retained, path.basename(stage)) : retained;
    if (directory === 'nested') assert.equal(fs.readFileSync(path.join(retained, 'original.txt'), 'utf8'), 'retain original nested directory');
    else assert.ok(fs.readdirSync(retainedArchive).some((name) => name.endsWith('.tar')), 'renamed original artifact survives');
    if (directory === 'nested') assert.ok(fs.readdirSync(stage).some((name) => name.endsWith('.tar')), 'preflight refuses before any unlink');
    assert.equal(fs.existsSync(outputRoot), false);
  });
});

test('exported producer refuses stage/ancestor replacement before the successful seal rename', async (t) => {
  for (const directory of ['stage', 'ancestor']) await t.test(directory, (t) => {
    const f = fixture(t);
    const parent = path.join(f.temporary, 'output-parent');
    const outputRoot = path.join(parent, 'sealed');
    const retained = path.join(f.temporary, 'retained-original');
    const originalWrite = fs.writeFileSync;
    const originalExists = fs.existsSync;
    let stage; let attacked; let fired = false;
    t.mock.method(fs, 'writeFileSync', function (file, ...args) {
      const result = originalWrite.call(this, file, ...args);
      if (typeof file === 'string' && file.endsWith('candidate-descriptor.json')) stage = path.dirname(file);
      return result;
    });
    t.mock.method(fs, 'existsSync', function (file) {
      if (!fired && stage && file === outputRoot) {
        fired = true;
        attacked = directory === 'ancestor' ? parent : stage;
        const originalID = fs.lstatSync(attacked, { bigint: true }).ino;
        fs.renameSync(attacked, retained);
        fs.mkdirSync(attacked);
        assert.notEqual(fs.lstatSync(attacked, { bigint: true }).ino, originalID);
        fs.mkdirSync(stage, { recursive: true });
        const retainedStage = directory === 'ancestor' ? path.join(retained, path.basename(stage)) : retained;
        // Byte-identical valid files in the replacement cannot authorize seal.
        for (const name of fs.readdirSync(retainedStage)) fs.copyFileSync(path.join(retainedStage, name), path.join(stage, name));
      }
      return originalExists.call(this, file);
    });
    try {
      assert.throws(() => f.build({ outputRoot }), (error) => {
        assert.match(error.message, /Candidate cleanup refused/);
        assert.match(error.cause?.message ?? '', /Replaced candidate directory identity/);
        return true;
      });
    } finally { t.mock.restoreAll(); }
    assert.equal(fired, true);
    assert.equal(fs.lstatSync(attacked).isDirectory(), true);
    const retainedStage = directory === 'ancestor' ? path.join(retained, path.basename(stage)) : retained;
    for (const name of fs.readdirSync(retainedStage)) assert.deepEqual(fs.readFileSync(path.join(stage, name)), fs.readFileSync(path.join(retainedStage, name)));
    assert.equal(fs.existsSync(outputRoot), false);
  });
});

test('exported producer refuses unknown stage entries before cleaning known files', async (t) => {
  for (const directory of [false, true]) await t.test(directory ? 'unknown nested directory' : 'unknown file', (t) => {
    const f = fixture(t); const originalWrite = fs.writeFileSync;
    const primary = new Error('injected producer write failure');
    let stage;
    t.mock.method(fs, 'writeFileSync', function (file, ...args) {
      const result = originalWrite.call(this, file, ...args);
      if (typeof file === 'string' && file.endsWith('.tar')) {
        stage = path.dirname(file);
        const unknown = path.join(stage, 'foreign');
        if (directory) fs.mkdirSync(unknown);
        originalWrite(directory ? path.join(unknown, 'foreign.txt') : unknown, 'preserve');
        throw primary;
      }
      return result;
    });
    try {
      assert.throws(() => f.build(), (error) => {
        assert.match(error.message, /Candidate cleanup refused/);
        assert.equal(error.cause, primary);
        return true;
      });
    } finally { t.mock.restoreAll(); }
    assert.equal(fs.readFileSync(path.join(stage, 'foreign', ...(directory ? ['foreign.txt'] : [])), 'utf8'), 'preserve');
    assert.ok(fs.readdirSync(stage).some((name) => name.endsWith('.tar')));
    assert.equal(fs.existsSync(f.options.outputRoot), false);
  });
});

test('exported producer retains the stage when its directory identity is unavailable', (t) => {
  const f = fixture(t); const originalStat = fs.lstatSync;
  let stage;
  t.mock.method(fs, 'lstatSync', function (file, options) {
    const stat = originalStat.call(this, file, options);
    if (options?.bigint && path.basename(String(file)).startsWith('.candidate-seal-')) {
      stage = file;
      stat.ino = 0n;
    }
    return stat;
  });
  try {
    assert.throws(() => f.build(), (error) => {
      assert.match(error.message, /Candidate cleanup refused/);
      assert.match(error.cause?.message ?? '', /Unavailable candidate directory identity/);
      return true;
    });
  } finally { t.mock.restoreAll(); }
  assert.equal(fs.lstatSync(stage).isDirectory(), true);
  assert.deepEqual(fs.readdirSync(stage), []);
  assert.equal(fs.existsSync(f.options.outputRoot), false);
});

test('exported producer rechecks stage identity after an owned unlink', (t) => {
  const f = fixture(t); const originalWrite = fs.writeFileSync; const originalUnlink = fs.unlinkSync;
  const retained = path.join(f.temporary, 'retained-original');
  const primary = new Error('injected producer final write failure');
  let stage; let fired = false;
  t.mock.method(fs, 'writeFileSync', function (file, ...args) {
    const result = originalWrite.call(this, file, ...args);
    if (typeof file === 'string' && file.endsWith('candidate-descriptor.json')) {
      stage = path.dirname(file);
      throw primary;
    }
    return result;
  });
  t.mock.method(fs, 'unlinkSync', function (file) {
    const result = originalUnlink.call(this, file);
    if (!fired && path.dirname(file) === stage) {
      fired = true;
      fs.renameSync(stage, retained);
      fs.mkdirSync(stage);
      originalWrite(path.join(stage, 'foreign.txt'), 'preserve replacement');
    }
    return result;
  });
  try {
    assert.throws(() => f.build(), (error) => {
      assert.match(error.message, /Candidate cleanup refused.*Replaced candidate directory identity/);
      assert.equal(error.cause, primary);
      return true;
    });
  } finally { t.mock.restoreAll(); }
  assert.equal(fired, true);
  assert.equal(fs.readFileSync(path.join(stage, 'foreign.txt'), 'utf8'), 'preserve replacement');
  // One owned unlink already happened. Preservation does not imply rollback.
  assert.equal(fs.readFileSync(path.join(retained, 'candidate-descriptor.json')).length > 0, true);
  assert.equal(fs.existsSync(f.options.outputRoot), false);
});

test('capabilities require declared actual exports; comments/strings cannot fabricate an API', (t) => {
  const f = fixture(t);
  assert.throws(() => f.build({ capabilities: ['business-tenant-v2'] }), /Missing required API/);
  assert.throws(() => f.build({ capabilities: ['unknown'] }), /Unknown candidate capability/);
  f.write(f.api, 'package permissionroutes\n// func Register() {}\nvar Fake = `\nfunc Register() {}\n`\n/*\nfunc Register() {}\n*/\n');
  assert.throws(() => f.build(), /Missing required API export/);
  assert.equal(fs.existsSync(f.options.outputRoot), false);
  for (const exports of Object.values(CAPABILITY_EXPORTS)) for (const [file, name] of exports) assert.ok(file.endsWith('.go') && /^[A-Z]/u.test(name));
});

test('source edits, additions, deletion and edit/revert during sealing leave no output or staging debris', async (t) => {
  for (const mode of ['edit', 'add', 'delete', 'revert']) await t.test(mode, (t) => {
    const f = fixture(t); const originalWrite = fs.writeFileSync;
    const originalBytes = fs.readFileSync(path.join(f.root, f.api));
    t.mock.method(fs, 'writeFileSync', function (file, ...args) {
      const result = originalWrite.call(this, file, ...args);
      if (typeof file === 'string' && file.endsWith('candidate-descriptor.json')) {
        if (mode === 'add') originalWrite(path.join(f.root, 'backend/new.go'), 'package fixture\n');
        else if (mode === 'delete') fs.unlinkSync(path.join(f.root, f.api));
        else {
          originalWrite(path.join(f.root, f.api), 'package changed\n');
          if (mode === 'revert') originalWrite(path.join(f.root, f.api), originalBytes);
        }
      }
      return result;
    });
    assert.throws(() => f.build(), /changed|ENOENT/u);
    assert.equal(fs.existsSync(f.options.outputRoot), false);
    assert.equal(fs.readdirSync(f.temporary).some((entry) => entry.startsWith('.candidate-seal-')), false);
  });
});

function hostileArchive(result, mutate) {
  const archive = Buffer.from(fs.readFileSync(result.artifactPath));
  mutate(archive);
  return { archive, descriptor: { ...result.descriptor, artifact: { ...result.descriptor.artifact, sha256: sha256(archive) } } };
}
function checksum(archive, offset = 0) {
  archive.fill(32, offset + 148, offset + 156);
  const value = [...archive.subarray(offset, offset + 512)].reduce((a, b) => a + b, 0);
  archive.write(`${value.toString(8).padStart(6, '0')}\0 `, offset + 148, 8, 'ascii');
}

test('independently re-pinned hostile tar headers still refuse links/extensions/unsafe entries/content corruption', (t) => {
  const result = fixture(t).build();
  for (const type of ['1', '2', '3', '4', '5', '6', '7', 'x', 'g', 'L', 'K']) {
    const h = hostileArchive(result, (archive) => { archive[156] = type.charCodeAt(0); checksum(archive); });
    assert.throws(() => verifyCandidateArchive(h.archive, h.descriptor), /Unsupported/);
  }
  for (const unsafe of ['../escape', '/absolute', 'C:/outside', '\\server\\share', 'dir/../../x', '.git/config']) {
    const h = hostileArchive(result, (archive) => { archive.fill(0, 0, 100); archive.write(unsafe, 0); checksum(archive); });
    assert.throws(() => verifyCandidateArchive(h.archive, h.descriptor), /Unsafe|Excluded/u);
  }
  const content = hostileArchive(result, (archive) => { archive[512] ^= 1; });
  assert.throws(() => verifyCandidateArchive(content.archive, content.descriptor), /inventory mismatch/);
  const link = hostileArchive(result, (archive) => { archive.write('destination', 157); checksum(archive); });
  assert.throws(() => verifyCandidateArchive(link.archive, link.descriptor), /Unsupported/);
  const trailer = hostileArchive(result, (archive) => { archive[archive.length - 1] = 1; });
  assert.throws(() => verifyCandidateArchive(trailer.archive, trailer.descriptor), /trailer/);
  const badChecksum = hostileArchive(result, (archive) => { archive[100] = 49; });
  assert.throws(() => verifyCandidateArchive(badChecksum.archive, badChecksum.descriptor), /Unsupported/);
  const firstPath = result.descriptor.files[0].path;
  const firstSize = result.descriptor.files[0].bytes;
  const secondOffset = 512 + firstSize + (512 - firstSize % 512) % 512;
  for (const name of [firstPath, firstPath.toUpperCase(), `${firstPath}/child`]) {
    const h = hostileArchive(result, (archive) => {
      archive.fill(0, secondOffset, secondOffset + 100); archive.write(name, secondOffset); checksum(archive, secondOffset);
    });
    assert.throws(() => verifyCandidateArchive(h.archive, h.descriptor), /unique|sorted|collision/u);
  }
});

test('separate pinned executable build receipt binds descriptor identity, capabilities and actual executable bytes', (t) => {
  const f = fixture(t); const result = f.build();
  const executablePath = path.join(f.temporary, 'fixture-executable');
  fs.writeFileSync(executablePath, 'synthetic executable bytes, never launched');
  const executable = fs.readFileSync(executablePath);
  const buildReceiptPath = path.join(f.temporary, 'build-receipt.json');
  const receipt = { schemaVersion: 1, kind: 'pantheon-candidate-build', candidateIdentity: result.descriptor.candidateIdentity, descriptorSha256: result.descriptorSha256, aggregateSha256: result.descriptor.aggregateSha256, capabilities: result.descriptor.capabilities, executable: { bytes: executable.length, sha256: sha256(executable) }, evidenceIds: ['fixture-build'], reviewIds: ['fixture-build-review'] };
  fs.writeFileSync(buildReceiptPath, JSON.stringify(receipt));
  const options = { descriptorPath: result.descriptorPath, expectedDescriptorSha256: result.descriptorSha256, buildReceiptPath, expectedBuildReceiptSha256: sha256(fs.readFileSync(buildReceiptPath)), executablePath, expectedExecutableSha256: sha256(executable) };
  assert.deepEqual(verifyCandidateExecutable(options), receipt);
  assert.throws(() => verifyCandidateExecutable({ ...options, expectedDescriptorSha256: '0'.repeat(64) }), /digest mismatch/);
  assert.throws(() => verifyCandidateExecutable({ ...options, expectedExecutableSha256: '0'.repeat(64) }), /executable hash/);
  fs.appendFileSync(executablePath, 'mutation');
  assert.throws(() => verifyCandidateExecutable(options), /executable hash/);
  fs.writeFileSync(buildReceiptPath, JSON.stringify({ ...receipt, capabilities: ['observability-business-tenant-v2'] }));
  assert.throws(() => verifyCandidateExecutable(options), /receipt digest/);
  assert.throws(() => verifyCandidateExecutable({ ...options, expectedBuildReceiptSha256: sha256(fs.readFileSync(buildReceiptPath)) }), /capability/);
});

function sealSourceState(root) {
  return ['', ...fs.readdirSync(root, { recursive: true }).sort()].map((relative) => {
    const absolute = path.join(root, relative);
    const stat = fs.lstatSync(absolute, { bigint: true });
    return { relative, stamp: [stat.dev, stat.ino, stat.size, stat.mtimeNs, stat.ctimeNs, stat.mode, stat.nlink], digest: stat.isFile() ? sha256(fs.readFileSync(absolute)) : null };
  });
}

test('exported producer refuses changed saved bytes in the final seal window with stable sources/directories', async (t) => {
  for (const [kind, mode] of [['descriptor', 'newline after write'], ['archive', 'append after early validation'], ['descriptor', 'same-size replacement'], ['archive', 'same-size replacement']]) await t.test(`${kind}/${mode}`, (t) => {
    const f = fixture(t); const sourceState = sealSourceState(f.root);
    const originalWrite = fs.writeFileSync; const originalRead = fs.readFileSync; const originalExists = fs.existsSync;
    const saved = new Map();
    let stage; let earlyArchiveRead = false; let finalWindow = false; let fired = false; let directoryIDs;
    const ids = () => [stage, path.dirname(stage)].map((directory) => {
      const stat = fs.lstatSync(directory, { bigint: true }); return [stat.dev, stat.ino];
    });
    const mutate = () => {
      directoryIDs = ids();
      const [file, bytes] = [...saved].find(([file]) => kind === 'descriptor' ? file.endsWith('.json') : file.endsWith('.tar'));
      const changed = mode === 'same-size replacement' ? Buffer.from(bytes) : Buffer.concat([bytes, Buffer.from('\n')]);
      if (mode === 'same-size replacement') changed[0] ^= 1;
      assert.notEqual(sha256(changed), sha256(bytes));
      if (mode === 'same-size replacement') assert.equal(changed.length, bytes.length);
      originalWrite(file, changed);
      fired = true;
      assert.deepEqual(ids(), directoryIDs, 'original stage and parent instances remain stable');
    };
    t.mock.method(fs, 'writeFileSync', function (file, ...args) {
      const result = originalWrite.call(this, file, ...args);
      if (typeof file === 'string' && path.basename(path.dirname(file)).startsWith('.candidate-seal-')) {
        stage = path.dirname(file); saved.set(file, Buffer.from(args[0]));
        if (file.endsWith('candidate-descriptor.json') && mode === 'newline after write') mutate();
      }
      return result;
    });
    t.mock.method(fs, 'readFileSync', function (file, ...args) {
      const bytes = originalRead.call(this, file, ...args);
      if (saved.has(file) && file.endsWith('.tar')) {
        assert.equal(sha256(bytes), sha256(saved.get(file)), 'early archive read is still valid');
        earlyArchiveRead = true;
      }
      return bytes;
    });
    t.mock.method(fs, 'existsSync', function (file) {
      if (!finalWindow && stage && file === f.options.outputRoot) {
        finalWindow = true;
        assert.equal(earlyArchiveRead, true, 'mutation window follows real archive validation and source scan');
        if (!fired) mutate();
      }
      return originalExists.call(this, file);
    });
    let failure;
    try { f.build(); } catch (error) { failure = error; } finally { t.mock.restoreAll(); }
    assert.equal(fired, true, 'real saved file mutation ran');
    assert.equal(finalWindow, true, 'actual producer reached final preflight');
    assert.deepEqual(sealSourceState(f.root), sourceState, 'source bytes and file/directory stamps remain stable');
    const parentStat = fs.lstatSync(path.dirname(stage), { bigint: true });
    assert.deepEqual([parentStat.dev, parentStat.ino], directoryIDs[1]);
    assert.ok(failure, 'producer must refuse altered saved bytes instead of returning success');
    assert.match(failure.message, /Candidate saved (?:bytes\/digest|file shape\/size) mismatch/);
    assert.equal(fs.existsSync(f.options.outputRoot), false);
    assert.equal(fs.existsSync(stage), false, 'bounded owned-file cleanup completes');
  });
});

test('exported producer refuses byte-identical file rename/recreate during its final saved-file read', async (t) => {
  for (const kind of ['descriptor', 'archive']) await t.test(kind, (t) => {
    const f = fixture(t); const sourceState = sealSourceState(f.root);
    const originalWrite = fs.writeFileSync; const originalRead = fs.readFileSync; const originalExists = fs.existsSync;
    const originalOpen = fs.openSync; const originalReadSync = fs.readSync;
    const retained = path.join(f.temporary, 'retained-original-file');
    let stage; let target; let targetFD; let expected; let earlyArchiveRead = false; let finalWindow = false; let fired = false; let directoryIDs;
    t.mock.method(fs, 'writeFileSync', function (file, ...args) {
      const result = originalWrite.call(this, file, ...args);
      if (typeof file === 'string' && path.basename(path.dirname(file)).startsWith('.candidate-seal-')) {
        stage = path.dirname(file);
        if (kind === 'descriptor' ? file.endsWith('.json') : file.endsWith('.tar')) { target = file; expected = Buffer.from(args[0]); }
      }
      return result;
    });
    t.mock.method(fs, 'readFileSync', function (file, ...args) {
      const bytes = originalRead.call(this, file, ...args);
      if (typeof file === 'string' && file.endsWith('.tar') && path.dirname(file) === stage) earlyArchiveRead = true;
      return bytes;
    });
    t.mock.method(fs, 'existsSync', function (file) {
      if (!finalWindow && stage && file === f.options.outputRoot) {
        finalWindow = true; assert.equal(earlyArchiveRead, true);
      }
      return originalExists.call(this, file);
    });
    t.mock.method(fs, 'openSync', function (file, ...args) {
      const fd = originalOpen.call(this, file, ...args);
      if (finalWindow && file === target) targetFD = fd;
      return fd;
    });
    t.mock.method(fs, 'readSync', function (fd, ...args) {
      const count = originalReadSync.call(this, fd, ...args);
      if (!fired && finalWindow && fd === targetFD && count > 0) {
        directoryIDs = [stage, path.dirname(stage)].map((directory) => {
          const stat = fs.lstatSync(directory, { bigint: true }); return [stat.dev, stat.ino];
        });
        const originalID = fs.lstatSync(target, { bigint: true }).ino;
        fs.renameSync(target, retained); originalWrite(target, expected);
        fired = true;
        assert.notEqual(fs.lstatSync(target, { bigint: true }).ino, originalID, 'known path now names a different regular file');
        assert.deepEqual(fs.readFileSync(target), expected, 'replacement preserves exact bytes and size');
        assert.deepEqual([stage, path.dirname(stage)].map((directory) => {
          const stat = fs.lstatSync(directory, { bigint: true }); return [stat.dev, stat.ino];
        }), directoryIDs, 'original stage and parent instances remain stable');
      }
      return count;
    });
    let failure;
    try { f.build(); } catch (error) { failure = error; } finally { t.mock.restoreAll(); }
    assert.equal(finalWindow, true);
    assert.equal(fired, true, 'actual final fd read was exercised');
    assert.deepEqual(sealSourceState(f.root), sourceState);
    const parentStat = fs.lstatSync(path.dirname(stage), { bigint: true });
    assert.deepEqual([parentStat.dev, parentStat.ino], directoryIDs[1]);
    assert.ok(failure, 'producer must refuse a saved pathname replacement');
    assert.match(failure.message, /Candidate saved file\/path changed while reading/);
    assert.deepEqual(fs.readFileSync(retained), expected, 'renamed original survives non-recursive cleanup');
    assert.equal(fs.existsSync(f.options.outputRoot), false);
    assert.equal(fs.existsSync(stage), false);
  });
});


test('curated public archive leaves survive sealed packaging; registry mismatches refuse', (t) => {
  const f = fixture(t);
  const refs = [
    '.harness/archive/2026-08/evidence/2026-08-31-base-operational-workbench-design/review.md',
    '.harness/archive/2026-09/evidence/2026-09-04-readme-v0.11.0-update/review.md',
    '.harness/archive/2026-09/evidence/2026-09-10-tenant-canary-slice/review.md',
    '.harness/archive/2026-09/evidence/2026-09-10-tenant-contract-design/review.md',
    '.harness/archive/2026-09/evidence/2026-09-10-tenant-core-auth-iam/review.md',
    '.harness/archive/2026-09/evidence/2026-09-10-tenant-migration-runbook/review.md',
    '.harness/archive/2026-09/evidence/2026-09-10-tenant-ready-guardrails/review.md',
    '.harness/archive/2026-09/evidence/2026-09-24-fix-report-residual-closeout/review.md',
  ];
  const config = 'config/foundation-consumer-artifacts.json';
  const registry = 'config/review-loop-legacy-artifacts.json';
  f.write(config, JSON.stringify({ schemaVersion: 1, sourceOwner: 'pantheon-base',
    reviewLegacyRegistry: { path: registry, artifactRoot: '.harness/archive' } }));
  const original = JSON.stringify({ schemaVersion: 1, entries: refs });
  f.write(registry, original);
  for (const ref of refs) f.write(ref, '# Historical fixture review\n');
  const sourcePaths = [...f.options.sourcePaths, config, registry, ...refs];
  const result = f.build({ sourcePaths });
  const entries = verifyCandidateArchive(fs.readFileSync(result.artifactPath), result.descriptor);
  for (const ref of [config, registry, ...refs])
    assert.deepEqual(entries.find(entry => entry.path === ref).bytes, fs.readFileSync(path.join(f.root, ref)));
  const rejected = paths => f.build({ sourcePaths: paths, outputRoot: path.join(f.temporary, 'refused') });
  for (const ref of ['.harness', '.harness/archive', path.dirname(refs[0]),
    refs[0].toUpperCase(), refs[0].replace('/review.md', '/../review.md'),
    '.harness/archive/private/review.md', '.harness/archive/.aws/credentials'])
    assert.throws(() => rejected([...f.options.sourcePaths, ref]));
  for (const missing of [config, registry, refs[0]])
    assert.throws(() => rejected(sourcePaths.filter(ref => ref !== missing)), /packaging|payload/);
  f.write(registry, JSON.stringify({ schemaVersion: 1, entries: refs.slice(1) }));
  assert.throws(() => rejected(sourcePaths), /payload/);
  f.write(registry, JSON.stringify({ schemaVersion: 1, entries: [...refs, refs[0]] }));
  assert.throws(() => rejected(sourcePaths), /curated/);
  f.write(registry, original);
  fs.unlinkSync(path.join(f.root, refs[0]));
  assert.throws(() => rejected(sourcePaths), /ENOENT/);
  fs.mkdirSync(path.join(f.root, refs[0]));
  assert.throws(() => rejected(sourcePaths), /regular leaves/);
  assert.equal(fs.existsSync(path.join(f.temporary, 'refused')), false);
});

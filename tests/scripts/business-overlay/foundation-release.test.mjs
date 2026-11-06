import assert from 'node:assert/strict';
import crypto from 'node:crypto';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { spawnSync } from 'node:child_process';
import test from 'node:test';
import { BASE_REPO, fetchFoundationRelease, fetchLatestFoundationRelease, resolveBaseFromLock, validateArchiveEntries } from '../../../scripts/business-overlay/foundation-release.mjs';

const sha = (file) => crypto.createHash('sha256').update(fs.readFileSync(file)).digest('hex');
function fixture(t) {
  const temporary = fs.mkdtempSync(path.join(os.tmpdir(), 'pantheon-github-release-'));
  t.after(() => fs.rmSync(temporary, { recursive: true, force: true }));
  const opsRoot = path.join(temporary, 'ops');
  const source = path.join(temporary, 'upstream');
  const assets = path.join(temporary, 'github-assets');
  fs.mkdirSync(opsRoot); fs.mkdirSync(source); fs.mkdirSync(assets);
  const native = (command, args, options = {}) => {
    const result = spawnSync(command, args, { encoding: 'utf8', ...options });
    assert.equal(result.status, 0, result.stderr);
    return result.stdout.trim();
  };
  native('git', ['init'], { cwd: source });
  native('git', ['config', 'core.autocrlf', 'false'], { cwd: source });
  native('git', ['config', 'user.email', 'fixture@example.com'], { cwd: source });
  native('git', ['config', 'user.name', 'Release fixture'], { cwd: source });
  fs.writeFileSync(path.join(source, 'shared.txt'), 'published base bytes\n');
  native('git', ['add', '.'], { cwd: source });
  native('git', ['commit', '-m', 'published snapshot'], { cwd: source });
  const commit = native('git', ['rev-parse', 'HEAD'], { cwd: source });
  native('git', ['archive', '--format=tar', '--output', path.join(assets, 'repo.tar'), 'HEAD'], { cwd: source });
  const version = 'pantheon-base-v0.18.4';
  const bundle = `foundation-release-${version}.tgz`;
  const manifest = { schemaVersion: 1, consumerMode: 'foundation-release-consumer', sourceRepo: 'pantheon-base', releaseVersion: version, releaseLine: 'release/0.18', baseCommit: commit,
    releaseArtifact: { assetName: bundle }, repoSnapshot: { assetName: 'repo.tar', generatedFrom: 'git-archive' }, sharedPaths: { frontend: ['shared.txt'] } };
  const bundleSource = path.join(temporary, 'bundle'); fs.mkdirSync(bundleSource);
  fs.writeFileSync(path.join(bundleSource, 'manifest.json'), JSON.stringify(manifest));
  native('tar', ['-czf', `github-assets/${bundle}`, '-C', 'bundle', 'manifest.json'], { cwd: temporary });
  for (const name of [bundle, 'repo.tar']) fs.writeFileSync(path.join(assets, `${name}.sha256`), `${sha(path.join(assets, name))}  ${name}\n`);
  const release = { id: 184, tag_name: version, draft: false, prerelease: false,
    assets: [bundle, `${bundle}.sha256`, 'repo.tar', 'repo.tar.sha256'].map((name) => ({ name, size: fs.statSync(path.join(assets, name)).size, state: 'uploaded', digest: `sha256:${sha(path.join(assets, name))}` })) };
  const calls = [];
  const run = (command, args, options) => {
    if (command !== 'gh') return native(command, args, options);
    calls.push(args);
    if (args[0] === 'api') {
      assert.ok(args[1].startsWith(`repos/${BASE_REPO}/`));
      if (args[1].endsWith('/releases/latest') || args[1].endsWith(`/releases/tags/${version}`)) return JSON.stringify(release);
      if (args[1].endsWith(`/git/ref/tags/${version}`)) return JSON.stringify({ object: { type: 'tag', sha: 'a'.repeat(40) } });
      if (args[1].endsWith(`/git/tags/${'a'.repeat(40)}`)) return JSON.stringify({ object: { type: 'commit', sha: commit } });
      throw new Error(`Unexpected GitHub endpoint: ${args[1]}`);
    }
    assert.equal(args[0], 'release'); assert.equal(args[1], 'download'); assert.equal(args[2], version);
    assert.equal(args[args.indexOf('--repo') + 1], BASE_REPO);
    const directory = args[args.indexOf('--dir') + 1];
    args.forEach((arg, i) => { if (arg === '--pattern') fs.copyFileSync(path.join(assets, args[i + 1]), path.join(directory, args[i + 1])); });
    return '';
  };
  return { opsRoot, source, assets, commit, release, version, bundle, calls, run, native };
}

test('selects latest stable GitHub release, validates four assets and atomically pins it', (t) => {
  const f = fixture(t);
  fs.writeFileSync(path.join(f.opsRoot, 'foundation-release.lock.json'), '{"old":"lock"}');
  const result = fetchLatestFoundationRelease(f);
  assert.equal(result.lock.releaseVersion, f.version);
  assert.equal(result.lock.baseCommit, f.commit);
  assert.equal(result.lock.releaseArtifact.githubRepo, BASE_REPO);
  assert.equal(result.lock.repoSnapshot.sha256, sha(path.join(f.assets, 'repo.tar')));
  assert.equal(f.calls[0][1], `repos/${BASE_REPO}/releases/latest`);
  assert.equal(f.calls.find((args) => args[1] === 'download').filter((arg) => arg === '--pattern').length, 4);
  assert.equal(fs.readFileSync(path.join(result.baseRoot, 'shared.txt'), 'utf8'), 'published base bytes\n');
  assert.deepEqual(JSON.parse(fs.readFileSync(path.join(f.opsRoot, 'foundation-release.lock.json'), 'utf8')), result.lock);
});

test('rebuild source is re-extracted from verified archive after cache tampering', (t) => {
  const f = fixture(t); const initial = fetchLatestFoundationRelease(f);
  fs.writeFileSync(path.join(initial.baseRoot, 'shared.txt'), 'tampered cache');
  fs.writeFileSync(path.join(initial.baseRoot, 'unexpected.txt'), 'stale cache');
  const snapshot = resolveBaseFromLock(f.opsRoot, { run: f.run });
  assert.equal(fs.readFileSync(path.join(snapshot.baseRoot, 'shared.txt'), 'utf8'), 'published base bytes\n');
  assert.equal(fs.existsSync(path.join(snapshot.baseRoot, 'unexpected.txt')), false);
  assert.ok(f.calls.some((args) => args[1] === `repos/${BASE_REPO}/releases/tags/${f.version}`));
});

test('missing asset, prerelease, wrong tag and altered GitHub digest preserve prior lock', async (t) => {
  for (const kind of ['missing', 'prerelease', 'wrong-tag', 'digest']) await t.test(kind, (t) => {
    const f = fixture(t); const prior = '{"preserve":"prior lock"}';
    const lockFile = path.join(f.opsRoot, 'foundation-release.lock.json'); fs.writeFileSync(lockFile, prior);
    if (kind === 'missing') f.release.assets.pop();
    if (kind === 'prerelease') f.release.prerelease = true;
    if (kind === 'wrong-tag') f.release.tag_name = '../base';
    if (kind === 'digest') f.release.assets[0].digest = `sha256:${'0'.repeat(64)}`;
    assert.throws(() => fetchLatestFoundationRelease(f));
    assert.equal(fs.readFileSync(lockFile, 'utf8'), prior);
  });
});

test('pinned archive corruption is rejected before extraction or network access', (t) => {
  const f = fixture(t); const result = fetchLatestFoundationRelease(f);
  const directory = path.dirname(result.baseRoot);
  fs.writeFileSync(path.join(directory, 'repo.tar'), 'corrupt archive');
  const lockBefore = fs.readFileSync(path.join(f.opsRoot, 'foundation-release.lock.json'), 'utf8');
  const callsBefore = f.calls.length;
  assert.throws(() => resolveBaseFromLock(f.opsRoot, { run: f.run }), /repo.tar checksum mismatch/);
  assert.equal(f.calls.length, callsBefore);
  assert.equal(fs.readFileSync(path.join(f.opsRoot, 'foundation-release.lock.json'), 'utf8'), lockBefore);
  assert.equal(fs.readFileSync(path.join(result.baseRoot, 'shared.txt'), 'utf8'), 'published base bytes\n');
});

test('repo.tar commit and pinned GitHub tag must agree', (t) => {
  const f = fixture(t); const originalRun = f.run;
  const run = (command, args, options) => command === 'gh' && args[1]?.includes('/git/tags/')
    ? JSON.stringify({ object: { type: 'commit', sha: 'b'.repeat(40) } }) : originalRun(command, args, options);
  assert.throws(() => fetchLatestFoundationRelease({ ...f, run }), /manifest does not match/);
  const result = fetchLatestFoundationRelease(f);
  assert.throws(() => resolveBaseFromLock(f.opsRoot, { run }), /tag differs from pinned lock commit/);
  assert.equal(result.lock.baseCommit, f.commit);
});

test('network failure and unsafe lock cache path cannot change the lock', (t) => {
  const f = fixture(t); const result = fetchLatestFoundationRelease(f);
  const lockFile = path.join(f.opsRoot, 'foundation-release.lock.json'); const prior = fs.readFileSync(lockFile, 'utf8');
  assert.throws(() => fetchLatestFoundationRelease({ ...f, run: () => { throw new Error('network unavailable'); } }), /network unavailable/);
  assert.equal(fs.readFileSync(lockFile, 'utf8'), prior);
  const unsafe = { ...result.lock, releaseArtifact: { ...result.lock.releaseArtifact, localPath: '../pantheon-base' } };
  fs.writeFileSync(lockFile, JSON.stringify(unsafe));
  assert.throws(() => resolveBaseFromLock(f.opsRoot, { run: f.run }), /official GitHub Base release/);
});

test('rejects archive traversal, Git metadata and Windows drive paths', () => {
  for (const entry of ['../escape', '/absolute', 'C:/outside', 'dir/../../escape', 'dir\\file', '.git/config']) assert.throws(() => validateArchiveEntries(entry), /Unsafe release archive entry/);
  assert.doesNotThrow(() => validateArchiveEntries('./\nbackend/\nbackend/shared.go\n'));
});

test('git-archive commit is checked even when checksum and manifest assets look valid', (t) => {
  const f = fixture(t);
  fs.writeFileSync(path.join(f.source, 'extra.txt'), 'another upstream commit');
  f.native('git', ['add', '.'], { cwd: f.source });
  f.native('git', ['commit', '-m', 'different source commit'], { cwd: f.source });
  f.native('git', ['archive', '--format=tar', '--output', path.join(f.assets, 'repo.tar'), 'HEAD'], { cwd: f.source });
  fs.writeFileSync(path.join(f.assets, 'repo.tar.sha256'), `${sha(path.join(f.assets, 'repo.tar'))}  repo.tar\n`);
  for (const asset of f.release.assets) { asset.size = fs.statSync(path.join(f.assets, asset.name)).size; asset.digest = `sha256:${sha(path.join(f.assets, asset.name))}`; }
  assert.throws(() => fetchLatestFoundationRelease(f), /repo.tar git-archive commit differs/);
  assert.equal(fs.existsSync(path.join(f.opsRoot, 'foundation-release.lock.json')), false);
});

test('cache junction cannot redirect official assets outside Ops', (t) => {
  const f = fixture(t);
  fs.mkdirSync(path.join(f.opsRoot, '.foundation'));
  fs.symlinkSync(f.assets, path.join(f.opsRoot, '.foundation/releases'), 'junction');
  assert.throws(() => fetchLatestFoundationRelease(f), /Release cache must use real Ops directories/);
});

test('explicit v0.18.4 consumption fetches the requested GitHub tag instead of latest', (t) => {
  const f = fixture(t);
  const result = fetchFoundationRelease({ ...f, releaseVersion: 'pantheon-base-v0.18.4' });
  assert.equal(result.lock.releaseVersion, 'pantheon-base-v0.18.4');
  assert.equal(f.calls[0][1], `repos/${BASE_REPO}/releases/tags/pantheon-base-v0.18.4`);
  assert.equal(f.calls.some((args) => args[1].endsWith('/releases/latest')), false);
});

test('wrong requested release leaves existing lock untouched', (t) => {
  const f = fixture(t);
  const lockFile = path.join(f.opsRoot, 'foundation-release.lock.json');
  fs.writeFileSync(lockFile, '{"preserve":"existing version"}');
  f.release.tag_name = 'pantheon-base-v0.18.5';
  assert.throws(() => fetchFoundationRelease({ ...f, releaseVersion: f.version }), /different requested release/);
  assert.equal(fs.readFileSync(lockFile, 'utf8'), '{"preserve":"existing version"}');
});

test('invalid explicit release cannot trigger GitHub access or local consumption', (t) => {
  const f = fixture(t);
  assert.throws(() => fetchFoundationRelease({ ...f, releaseVersion: '../pantheon-base' }), /Expected a stable/);
  assert.equal(f.calls.length, 0);
  const result = spawnSync(process.execPath, ['scripts/business-overlay/foundation-release.mjs', '--base', '../pantheon-base'], { encoding: 'utf8' });
  assert.equal(result.status, 1);
  assert.match(result.stderr, /local source arguments are not supported/);
});

import crypto from 'node:crypto';
import fs from 'node:fs';
import path from 'node:path';
import { spawnSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';

export const BASE_REPO = 'duanxldragon/pantheon-base';
const defaultRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..');
const hash = (file) => crypto.createHash('sha256').update(fs.readFileSync(file)).digest('hex');

function runCommand(command, args, options = {}) {
  const result = spawnSync(command, args, { encoding: 'utf8', maxBuffer: 64 * 1024 * 1024, timeout: 300000, ...options });
  if (result.status !== 0) throw new Error(result.stderr?.trim() || result.error?.message || `${command} failed`);
  return result.stdout.trim();
}

export function releaseIdentity(version) {
  const match = /^pantheon-base-v(\d+)\.(\d+)\.(\d+)$/u.exec(version ?? '');
  if (!match) throw new Error('Expected a stable pantheon-base-vX.Y.Z release');
  return { version, line: `release/${match[1]}.${match[2]}`, bundle: `foundation-release-${version}.tgz` };
}

function api(run, endpoint) {
  return JSON.parse(run('gh', ['api', `repos/${BASE_REPO}/${endpoint}`]));
}

export function validateGitHubRelease(release) {
  const identity = releaseIdentity(release.tag_name);
  if (release.draft || release.prerelease || !Number.isInteger(release.id)) throw new Error('Expected a published stable GitHub release');
  const names = [identity.bundle, `${identity.bundle}.sha256`, 'repo.tar', 'repo.tar.sha256'];
  for (const name of names) {
    const matching = (release.assets ?? []).filter((asset) => asset.name === name && asset.state === 'uploaded' && asset.size > 0);
    if (matching.length !== 1) throw new Error(`GitHub release must contain one uploaded asset: ${name}`);
  }
  return identity;
}

function tagCommit(run, version) {
  let object = api(run, `git/ref/tags/${version}`).object;
  for (let depth = 0; object?.type === 'tag' && depth < 4; depth += 1) object = api(run, `git/tags/${object.sha}`).object;
  if (object?.type !== 'commit' || !/^[a-f0-9]{40}$/u.test(object.sha)) throw new Error('Release tag must resolve to an immutable commit');
  return object.sha;
}

export function validateChecksum(file, expected, label = 'release bundle') {
  if (!/^[a-f0-9]{64}$/u.test(expected ?? '')) throw new Error(`Missing or invalid ${label} checksum`);
  const actual = hash(file);
  if (actual !== expected) throw new Error(`${label} checksum mismatch: expected ${expected}, got ${actual}`);
  return actual;
}

function sidecarChecksum(directory, asset) {
  const match = /^([a-f0-9]{64})\s+\*?([^\r\n]+)\s*$/u.exec(fs.readFileSync(path.join(directory, `${asset}.sha256`), 'utf8'));
  if (!match || match[2].trim() !== asset) throw new Error(`Invalid SHA256 sidecar for ${asset}`);
  return validateChecksum(path.join(directory, asset), match[1], asset === 'repo.tar' ? 'repo.tar' : 'release bundle');
}

function releaseDirectory(opsRoot, version) {
  releaseIdentity(version);
  opsRoot = path.resolve(opsRoot);
  for (const relative of ['.foundation', '.foundation/releases', `.foundation/releases/${version}`]) {
    const candidate = path.join(opsRoot, relative);
    if (fs.existsSync(candidate) && (fs.lstatSync(candidate).isSymbolicLink() || !fs.lstatSync(candidate).isDirectory())) throw new Error('Release cache must use real Ops directories: ' + relative);
  }
  const directory = path.resolve(opsRoot, '.foundation/releases', version);
  const parent = path.resolve(opsRoot, '.foundation/releases');
  if (path.dirname(directory) !== parent) throw new Error('Release cache escapes Ops');
  fs.mkdirSync(directory, { recursive: true });
  return directory;
}

function acquireAssets(run, release, directory) {
  const { bundle } = validateGitHubRelease(release);
  const assets = [bundle, `${bundle}.sha256`, 'repo.tar', 'repo.tar.sha256'];
  const missing = assets.filter((name) => !fs.existsSync(path.join(directory, name)));
  if (missing.length) run('gh', ['release', 'download', release.tag_name, '--repo', BASE_REPO, '--dir', directory, ...missing.flatMap((name) => ['--pattern', name])]);
  for (const name of assets) {
    if (!fs.lstatSync(path.join(directory, name)).isFile()) throw new Error('Release asset must be a regular file: ' + name);
    const digest = release.assets.find((asset) => asset.name === name).digest;
    if (digest) {
      if (!/^sha256:[a-f0-9]{64}$/u.test(digest)) throw new Error(`Invalid GitHub asset digest: ${name}`);
      validateChecksum(path.join(directory, name), digest.slice(7), name);
    }
  }
  return { bundle: sidecarChecksum(directory, bundle), repo: sidecarChecksum(directory, 'repo.tar') };
}

export function validateArchiveEntries(entries) {
  for (const raw of entries.split(/\r?\n/u).filter(Boolean)) {
    const entry = raw.replace(/^\.\//u, '').replace(/\/$/u, '');
    if (!entry) continue;
    if (entry.startsWith('/') || entry.includes('\\') || entry.includes(':') || entry.split('/').some((part) => part === '..' || part === '.git')) {
      throw new Error(`Unsafe release archive entry: ${raw}`);
    }
  }
}

function checkArchive(run, directory, asset) {
  validateArchiveEntries(run('tar', ['-tf', asset], { cwd: directory }));
  const listing = run('tar', ['-tvf', asset], { cwd: directory });
  if (listing.split(/\r?\n/u).filter(Boolean).some((line) => !/^[d-]/u.test(line))) throw new Error(`Release archive contains links or special files: ${asset}`);
}

function validateManifest(manifest, identity, commit) {
  if (manifest.schemaVersion !== 1 || manifest.consumerMode !== 'foundation-release-consumer' || manifest.sourceRepo !== 'pantheon-base' ||
      manifest.releaseVersion !== identity.version || manifest.releaseLine !== identity.line || manifest.baseCommit !== commit ||
      manifest.releaseArtifact?.assetName !== identity.bundle || manifest.repoSnapshot?.assetName !== 'repo.tar' || manifest.repoSnapshot?.generatedFrom !== 'git-archive' ||
      !Object.keys(manifest.sharedPaths ?? {}).length) throw new Error('Release manifest does not match official tag and consumer contract');
}

function verifySnapshot(run, directory, identity, commit) {
  checkArchive(run, directory, identity.bundle);
  checkArchive(run, directory, 'repo.tar');
  const manifest = JSON.parse(run('tar', ['-xOf', identity.bundle, 'manifest.json'], { cwd: directory }));
  validateManifest(manifest, identity, commit);
  const archivedCommit = run('git', ['get-tar-commit-id'], { input: fs.readFileSync(path.join(directory, 'repo.tar')) });
  if (archivedCommit !== commit) throw new Error('repo.tar git-archive commit differs from official tag');
  const repo = path.join(directory, 'repo');
  // Cache contents are derived: re-extract checked bytes so stale/tampered files
  // cannot become authority. This fixed target must remain inside the cache.
  if (path.dirname(repo) !== directory) throw new Error('Invalid snapshot extraction target');
  fs.rmSync(repo, { recursive: true, force: true });
  fs.mkdirSync(repo);
  run('tar', ['-xf', 'repo.tar', '-C', 'repo'], { cwd: directory });
  return { baseRoot: repo, baseCommit: commit, manifest };
}

export function resolveBaseFromLock(opsRoot, { run = runCommand } = {}) {
  const lockFile = path.join(opsRoot, 'foundation-release.lock.json');
  if (!fs.existsSync(lockFile)) throw new Error('No locked Base snapshot found (foundation-release.lock.json); fetch the official GitHub release first');
  const lock = JSON.parse(fs.readFileSync(lockFile, 'utf8'));
  const identity = releaseIdentity(lock.releaseVersion);
  if (lock.schemaVersion !== 1 || lock.consumerMode !== 'foundation-release-consumer' || lock.baseRepo !== BASE_REPO || lock.releaseArtifact?.githubRepo !== BASE_REPO || lock.releaseLine !== identity.line ||
      lock.releaseArtifact.assetName !== identity.bundle || lock.releaseArtifact.localPath !== `.foundation/releases/${identity.version}` ||
      lock.repoSnapshot?.assetName !== 'repo.tar' || lock.repoSnapshot.baseCommit !== lock.baseCommit || lock.repoSnapshot.generatedFrom !== 'git-archive') throw new Error('Lock must identify the official GitHub Base release');
  const directory = releaseDirectory(opsRoot, identity.version);
  // Reject locally corrupted pinned assets before any network request/extraction.
  for (const [asset, checksum, label] of [[identity.bundle, lock.releaseArtifact.checksum, 'release bundle'], ['repo.tar', lock.repoSnapshot.sha256, 'repo.tar']]) {
    if (fs.existsSync(path.join(directory, asset))) validateChecksum(path.join(directory, asset), checksum, label);
  }
  const release = api(run, `releases/tags/${identity.version}`);
  if (release.tag_name !== identity.version) throw new Error('GitHub returned a different release');
  const checksums = acquireAssets(run, release, directory);
  if (checksums.bundle !== lock.releaseArtifact.checksum || checksums.repo !== lock.repoSnapshot.sha256) throw new Error('GitHub release assets differ from pinned lock');
  const commit = tagCommit(run, identity.version);
  if (commit !== lock.baseCommit) throw new Error('Official release tag differs from pinned lock commit');
  const snapshot = verifySnapshot(run, directory, identity, commit);
  if (JSON.stringify(snapshot.manifest.sharedPaths) !== JSON.stringify(lock.sharedPaths)) throw new Error('Pinned shared paths differ from release manifest');
  return snapshot;
}

export function fetchFoundationRelease({ opsRoot = defaultRoot, run = runCommand, releaseVersion } = {}) {
  if (releaseVersion !== undefined) releaseIdentity(releaseVersion);
  const release = api(run, releaseVersion ? `releases/tags/${releaseVersion}` : 'releases/latest');
  if (releaseVersion && release.tag_name !== releaseVersion) throw new Error('GitHub returned a different requested release');
  const identity = validateGitHubRelease(release);
  const directory = releaseDirectory(opsRoot, identity.version);
  const checksums = acquireAssets(run, release, directory);
  const commit = tagCommit(run, identity.version);
  const snapshot = verifySnapshot(run, directory, identity, commit);
  const lock = {
    schemaVersion: 1, baseRepo: BASE_REPO, releaseLine: identity.line, releaseVersion: identity.version, baseCommit: commit,
    consumerMode: 'foundation-release-consumer',
    releaseArtifact: { githubRepo: BASE_REPO, assetName: identity.bundle, localPath: `.foundation/releases/${identity.version}`, checksum: checksums.bundle },
    repoSnapshot: { assetName: 'repo.tar', sha256: checksums.repo, baseCommit: commit, generatedFrom: 'git-archive' },
    sharedPaths: snapshot.manifest.sharedPaths, lockedAt: new Date().toISOString(), lockedBy: 'github-foundation-release-consumer',
  };
  const temporary = path.join(opsRoot, 'foundation-release.lock.json.tmp');
  fs.writeFileSync(temporary, `${JSON.stringify(lock, null, 2)}\n`);
  fs.renameSync(temporary, path.join(opsRoot, 'foundation-release.lock.json'));
  return { ...snapshot, lock };
}

export function fetchLatestFoundationRelease(options = {}) {
  return fetchFoundationRelease(options);
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  if (process.argv.length !== 2 && !(process.argv.length === 4 && process.argv[2] === '--release')) throw new Error('Use --release pantheon-base-vX.Y.Z or no arguments for latest; local source arguments are not supported');
  const result = fetchFoundationRelease({ releaseVersion: process.argv[3] });
  console.log(`OK locked GitHub ${BASE_REPO}@${result.lock.releaseVersion} (${result.lock.baseCommit})`);
}

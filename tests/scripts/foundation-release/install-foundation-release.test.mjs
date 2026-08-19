import assert from 'node:assert/strict';
import crypto from 'node:crypto';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { spawnSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';
import test from 'node:test';

const currentFilePath = fileURLToPath(import.meta.url);
const repoRoot = path.resolve(path.dirname(currentFilePath), '..', '..', '..');
const scriptPath = path.join(repoRoot, 'scripts', 'foundation-release', 'install-foundation-release.mjs');

function withTempDir(callback) {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'pantheon-foundation-install-'));
  try {
    callback(root);
  } finally {
    fs.rmSync(root, { recursive: true, force: true });
  }
}

function writeJson(filePath, value) {
  fs.mkdirSync(path.dirname(filePath), { recursive: true });
  fs.writeFileSync(filePath, `${JSON.stringify(value, null, 2)}\n`, 'utf8');
}

function writeText(filePath, value) {
  fs.mkdirSync(path.dirname(filePath), { recursive: true });
  fs.writeFileSync(filePath, value, 'utf8');
}

function runScript(args, cwd) {
  return spawnSync(process.execPath, [scriptPath, ...args], {
    cwd,
    encoding: 'utf8',
  });
}

function createArchive(root, releaseVersion) {
  const releaseRoot = path.join(root, 'release-root');
  writeJson(path.join(releaseRoot, 'manifest.json'), {
    releaseVersion,
    releaseLine: 'release/0.8',
    baseCommit: 'deadbeefdeadbeefdeadbeefdeadbeefdeadbeef',
    sourceRepo: 'pantheon-base',
    consumerMode: 'foundation-release-consumer',
    sharedPaths: {
      backend: ['backend/pkg'],
    },
  });
  writeText(
    path.join(releaseRoot, 'bundle', 'shared-backend', 'backend', 'pkg', 'service.go'),
    'package pkg\n',
  );

  const archivePath = path.join(root, `foundation-release-${releaseVersion}.tgz`);
  const result = spawnSync(
    'tar',
    ['-czf', archivePath, '-C', releaseRoot, 'manifest.json', 'bundle'],
    { encoding: 'utf8' },
  );
  assert.equal(result.status, 0, result.stderr || result.stdout || result.error?.message);
  return archivePath;
}

function fileSha256(filePath) {
  return crypto.createHash('sha256').update(fs.readFileSync(filePath)).digest('hex');
}

function writeChecksum(archivePath, checksum = fileSha256(archivePath)) {
  writeText(`${archivePath}.sha256`, `${checksum}  ${path.basename(archivePath)}\n`);
  return checksum;
}

function writeLock(opsRoot, releaseVersion, checksum) {
  writeJson(path.join(opsRoot, 'foundation-release.lock.json'), {
    schemaVersion: 1,
    releaseLine: 'release/0.8',
    releaseVersion,
    baseCommit: 'deadbeefdeadbeefdeadbeefdeadbeefdeadbeef',
    consumerMode: 'foundation-release-consumer',
    releaseArtifact: {
      githubRepo: 'duanxldragon/pantheon-base',
      assetName: `foundation-release-${releaseVersion}.tgz`,
      localPath: `.foundation/releases/${releaseVersion}`,
      checksum,
    },
    sharedPaths: {
      backend: ['backend/pkg'],
    },
  });
}

test('install-foundation-release installs a local archive under .foundation/releases', () => {
  withTempDir((root) => {
    const opsRoot = path.join(root, 'pantheon-ops');
    const releaseVersion = 'base-v0.8.0';
    const archivePath = createArchive(root, releaseVersion);
    const checksum = writeChecksum(archivePath);
    writeLock(opsRoot, releaseVersion, null);

    const result = runScript(['--ops-root', opsRoot, '--archive', archivePath], repoRoot);

    assert.equal(result.status, 0, result.stderr || result.stdout || result.error?.message);
    assert.equal(
      fs.existsSync(path.join(opsRoot, '.foundation', 'releases', releaseVersion, 'manifest.json')),
      true,
    );
    assert.equal(
      fs.existsSync(
        path.join(opsRoot, '.foundation', 'releases', releaseVersion, 'bundle', 'shared-backend', 'backend', 'pkg', 'service.go'),
      ),
      true,
    );
    const marker = JSON.parse(
      fs.readFileSync(
        path.join(opsRoot, '.foundation', 'releases', releaseVersion, '.foundation-release-verified.json'),
        'utf8',
      ),
    );
    assert.equal(marker.archiveSha256, checksum);
    assert.equal(fs.existsSync(archivePath), true, 'local archive must not be deleted');
  });
});

test('install-foundation-release falls back to the checksum pinned in the lock', () => {
  withTempDir((root) => {
    const opsRoot = path.join(root, 'pantheon-ops');
    const releaseVersion = 'base-v0.8.0';
    const archivePath = createArchive(root, releaseVersion);
    writeLock(opsRoot, releaseVersion, fileSha256(archivePath));

    const result = runScript(['--ops-root', opsRoot, '--archive', archivePath], repoRoot);

    assert.equal(result.status, 0, result.stderr || result.stdout || result.error?.message);
  });
});

test('install-foundation-release rejects archives without any trusted checksum', () => {
  withTempDir((root) => {
    const opsRoot = path.join(root, 'pantheon-ops');
    const releaseVersion = 'base-v0.8.0';
    const archivePath = createArchive(root, releaseVersion);
    writeLock(opsRoot, releaseVersion, null);

    const result = runScript(['--ops-root', opsRoot, '--archive', archivePath], repoRoot);

    assert.notEqual(result.status, 0);
    assert.match(result.stderr, /SHA-256 is required/);
    assert.equal(fs.existsSync(path.join(opsRoot, '.foundation', 'releases', releaseVersion)), false);
  });
});

test('install-foundation-release verifies checksum before replacing an existing release', () => {
  withTempDir((root) => {
    const opsRoot = path.join(root, 'pantheon-ops');
    const releaseVersion = 'base-v0.8.0';
    const archivePath = createArchive(root, releaseVersion);
    writeChecksum(archivePath, '0'.repeat(64));
    writeLock(opsRoot, releaseVersion, null);
    const sentinelPath = path.join(opsRoot, '.foundation', 'releases', releaseVersion, 'sentinel.txt');
    writeText(sentinelPath, 'keep me\n');

    const result = runScript(['--ops-root', opsRoot, '--archive', archivePath], repoRoot);

    assert.notEqual(result.status, 0);
    assert.match(result.stderr, /checksum mismatch/);
    assert.equal(fs.readFileSync(sentinelPath, 'utf8'), 'keep me\n');
  });
});

test('install-foundation-release preserves the previous release when manifest validation fails', () => {
  withTempDir((root) => {
    const opsRoot = path.join(root, 'pantheon-ops');
    const releaseVersion = 'base-v0.8.0';
    const archivePath = createArchive(root, releaseVersion);
    writeChecksum(archivePath);
    writeLock(opsRoot, releaseVersion, fileSha256(archivePath));
    const sentinelPath = path.join(opsRoot, '.foundation', 'releases', releaseVersion, 'sentinel.txt');
    writeText(sentinelPath, 'keep me\n');
    const lock = JSON.parse(fs.readFileSync(path.join(opsRoot, 'foundation-release.lock.json'), 'utf8'));
    lock.baseCommit = 'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa';
    writeJson(path.join(opsRoot, 'foundation-release.lock.json'), lock);

    const result = runScript(['--ops-root', opsRoot, '--archive', archivePath], repoRoot);

    assert.notEqual(result.status, 0);
    assert.match(result.stderr, /commit mismatch/);
    assert.equal(fs.readFileSync(sentinelPath, 'utf8'), 'keep me\n');
  });
});

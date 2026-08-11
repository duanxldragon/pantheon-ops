import assert from 'node:assert/strict';
import crypto from 'node:crypto';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { spawnSync } from 'node:child_process';
import test from 'node:test';

import { verifyAndMarkLocalRelease } from '../../../scripts/foundation-release/upgrade-from-local-base-release.mjs';

function runGit(root, ...args) {
  const result = spawnSync('git', args, { cwd: root, encoding: 'utf8' });
  assert.equal(result.status, 0, result.stderr || result.stdout);
  return result.stdout.trim();
}

function createFixture() {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'pantheon-local-release-'));
  const baseRoot = path.join(root, 'base');
  fs.mkdirSync(baseRoot, { recursive: true });
  runGit(baseRoot, 'init');
  runGit(baseRoot, 'config', 'user.email', 'test@example.com');
  runGit(baseRoot, 'config', 'user.name', 'Test User');
  fs.writeFileSync(path.join(baseRoot, '.gitignore'), 'dist/\n', 'utf8');
  fs.writeFileSync(path.join(baseRoot, 'README.md'), '# Fixture\n', 'utf8');
  runGit(baseRoot, 'add', '.gitignore', 'README.md');
  runGit(baseRoot, 'commit', '-m', 'fixture');
  const baseCommit = runGit(baseRoot, 'rev-parse', 'HEAD');

  const releaseVersion = 'pantheon-base-v0.10.4';
  const bundleRoot = path.join(baseRoot, 'dist', 'foundation-releases', releaseVersion);
  const manifestPath = path.join(bundleRoot, 'manifest.json');
  fs.mkdirSync(path.join(bundleRoot, 'bundle'), { recursive: true });
  fs.writeFileSync(
    manifestPath,
    `${JSON.stringify({ releaseVersion, baseCommit }, null, 2)}\n`,
    'utf8',
  );
  fs.writeFileSync(path.join(bundleRoot, 'bundle', 'fixture.txt'), 'fixture\n', 'utf8');
  const archivePath = path.join(bundleRoot, `foundation-release-${releaseVersion}.tgz`);
  fs.writeFileSync(archivePath, 'archive fixture\n', 'utf8');
  const checksum = crypto.createHash('sha256').update(fs.readFileSync(archivePath)).digest('hex');
  fs.writeFileSync(`${archivePath}.sha256`, `${checksum}  ${path.basename(archivePath)}\n`, 'utf8');

  return { root, baseRoot, bundleRoot, manifestPath, releaseVersion, archivePath };
}

test('local foundation verification writes a commit- and checksum-bound marker', () => {
  const fixture = createFixture();
  try {
    const marker = verifyAndMarkLocalRelease(fixture);
    assert.equal(marker.releaseVersion, fixture.releaseVersion);
    assert.equal(marker.baseCommit, runGit(fixture.baseRoot, 'rev-parse', 'HEAD'));
    assert.match(marker.archiveSha256, /^[a-f0-9]{64}$/u);
    assert.equal(
      fs.existsSync(path.join(fixture.bundleRoot, '.foundation-release-verified.json')),
      true,
    );
  } finally {
    fs.rmSync(fixture.root, { recursive: true, force: true });
  }
});

test('local foundation verification allows metadata-only commits after baseCommit', () => {
  const fixture = createFixture();
  try {
    const manifest = JSON.parse(fs.readFileSync(fixture.manifestPath, 'utf8'));
    fs.writeFileSync(path.join(fixture.baseRoot, 'RELEASE.md'), '# Release metadata\n', 'utf8');
    runGit(fixture.baseRoot, 'add', 'RELEASE.md');
    runGit(fixture.baseRoot, 'commit', '-m', 'release metadata');

    const marker = verifyAndMarkLocalRelease(fixture);
    assert.equal(marker.baseCommit, manifest.baseCommit);
    assert.notEqual(marker.baseCommit, runGit(fixture.baseRoot, 'rev-parse', 'HEAD'));
  } finally {
    fs.rmSync(fixture.root, { recursive: true, force: true });
  }
});

test('local foundation verification rejects dirty, mismatched, and corrupt sources', () => {
  const dirty = createFixture();
  try {
    fs.writeFileSync(path.join(dirty.baseRoot, 'README.md'), '# Dirty\n', 'utf8');
    assert.throws(() => verifyAndMarkLocalRelease(dirty), /clean pantheon-base worktree/u);
  } finally {
    fs.rmSync(dirty.root, { recursive: true, force: true });
  }

  const mismatched = createFixture();
  try {
    const manifest = JSON.parse(fs.readFileSync(mismatched.manifestPath, 'utf8'));
    manifest.baseCommit = 'deadbeefdeadbeefdeadbeefdeadbeefdeadbeef';
    fs.writeFileSync(mismatched.manifestPath, `${JSON.stringify(manifest, null, 2)}\n`, 'utf8');
    assert.throws(() => verifyAndMarkLocalRelease(mismatched), /identity mismatch/u);
  } finally {
    fs.rmSync(mismatched.root, { recursive: true, force: true });
  }

  const corrupt = createFixture();
  try {
    fs.appendFileSync(corrupt.archivePath, 'corrupt\n', 'utf8');
    assert.throws(() => verifyAndMarkLocalRelease(corrupt), /checksum mismatch/u);
  } finally {
    fs.rmSync(corrupt.root, { recursive: true, force: true });
  }
});

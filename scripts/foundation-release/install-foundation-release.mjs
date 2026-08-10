import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import process from 'node:process';
import { spawnSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';

import {
  computeFileSha256,
  computeReleaseTreeSha256,
  readFoundationLock,
  readFoundationReleaseManifest,
  verifiedReleaseMarkerName,
} from './shared-foundation-rules.mjs';

const currentFilePath = fileURLToPath(import.meta.url);
const scriptsDir = path.dirname(currentFilePath);
const defaultOpsRoot = path.resolve(scriptsDir, '..', '..');

function parseArgs(argv) {
  const options = {
    opsRoot: defaultOpsRoot,
  };

  for (let index = 0; index < argv.length; index += 1) {
    const arg = argv[index];
    const value = argv[index + 1];

    if (arg === '--ops-root') {
      if (!value) throw new Error('--ops-root requires a path');
      options.opsRoot = path.resolve(value);
      index += 1;
    } else if (arg === '--archive') {
      if (!value) throw new Error('--archive requires a path');
      options.archivePath = path.resolve(value);
      index += 1;
    } else if (arg === '--repo') {
      if (!value) throw new Error('--repo requires owner/repo');
      options.repo = value;
      index += 1;
    } else if (arg === '--if-exists') {
      options.ifExists = true;
    } else if (arg === '--help' || arg === '-h') {
      options.help = true;
    } else {
      throw new Error(`Unknown argument: ${arg}`);
    }
  }

  return options;
}

function runCommand(command, args, description, options = {}) {
  const result = spawnSync(command, args, {
    cwd: options.cwd,
    encoding: 'utf8',
  });
  if (result.status !== 0) {
    throw new Error(result.stderr?.trim() || result.stdout?.trim() || `${description} failed`);
  }
  return result.stdout.trim();
}

function resolveArtifactMetadata(lock, options) {
  const artifact = lock.releaseArtifact ?? {};
  const assetName = artifact.assetName ?? `foundation-release-${lock.releaseVersion}.tgz`;
  const repo = options.repo ?? artifact.githubRepo;
  return { assetName, repo };
}

function parseChecksumFile(checksumPath, archivePath) {
  const checksumContents = fs.readFileSync(checksumPath, 'utf8').trim();
  const checksumMatch = checksumContents.match(/^([a-fA-F0-9]{64})\s+\*?(.+)$/u);
  if (!checksumMatch) {
    throw new Error(`invalid SHA-256 checksum file: ${checksumPath}`);
  }
  const [, checksum, recordedFileName] = checksumMatch;
  const archiveName = path.basename(archivePath);
  if (path.basename(recordedFileName) !== archiveName) {
    throw new Error(`checksum file ${checksumPath} names ${recordedFileName}, expected ${archiveName}`);
  }
  return checksum.toLowerCase();
}

function checksumFromAdjacentFile(archivePath) {
  const checksumPath = `${archivePath}.sha256`;
  return fs.existsSync(checksumPath) ? parseChecksumFile(checksumPath, archivePath) : null;
}

function requireExpectedChecksum(downloadedChecksum, lockChecksum) {
  const checksum = downloadedChecksum ?? lockChecksum ?? null;
  if (!/^[a-f0-9]{64}$/iu.test(checksum ?? '')) {
    throw new Error('foundation release SHA-256 is required from an adjacent .sha256 file or releaseArtifact.checksum');
  }
  return checksum.toLowerCase();
}

function downloadArchive(lock, options) {
  const { assetName, repo } = resolveArtifactMetadata(lock, options);
  if (!repo) {
    throw new Error('releaseArtifact.githubRepo is missing; pass --repo <owner/repo> or --archive <path>');
  }

  const downloadDir = fs.mkdtempSync(path.join(os.tmpdir(), 'pantheon-foundation-release-'));
  try {
    runCommand(
      'gh',
      ['release', 'download', lock.releaseVersion, '--repo', repo, '--pattern', assetName, '--dir', downloadDir],
      `gh release download ${lock.releaseVersion}`,
      { cwd: options.opsRoot },
    );
  } catch (err) {
    if (options.ifExists && /release not found|no assets to download/i.test(err.message)) {
      fs.rmSync(downloadDir, { recursive: true, force: true });
      console.log(`Foundation release ${lock.releaseVersion} not found on ${repo} — skipping install`);
      process.exit(0);
    }
    fs.rmSync(downloadDir, { recursive: true, force: true });
    throw err;
  }

  const archivePath = path.join(downloadDir, assetName);
  if (!fs.existsSync(archivePath)) {
    throw new Error(`downloaded release artifact not found: ${archivePath}`);
  }

  const checksumAssetName = `${assetName}.sha256`;
  try {
    runCommand(
      'gh',
      ['release', 'download', lock.releaseVersion, '--repo', repo, '--pattern', checksumAssetName, '--dir', downloadDir],
      `gh release download checksum ${lock.releaseVersion}`,
      { cwd: options.opsRoot },
    );
  } catch (error) {
    if (!lock.releaseArtifact?.checksum) {
      fs.rmSync(downloadDir, { recursive: true, force: true });
      throw new Error(`checksum asset is required for ${lock.releaseVersion}: ${error.message}`);
    }
  }

  const checksumPath = path.join(downloadDir, checksumAssetName);
  const checksum = fs.existsSync(checksumPath)
    ? parseChecksumFile(checksumPath, archivePath)
    : null;

  return { archivePath, checksum, downloadDir };
}

function verifyChecksum(archivePath, expectedSha256) {
  const actualSha256 = computeFileSha256(archivePath);
  if (actualSha256.toLowerCase() !== expectedSha256.toLowerCase()) {
    throw new Error(
      `checksum mismatch for ${archivePath}\n` +
      `  expected: ${expectedSha256}\n` +
      `  actual:   ${actualSha256}`,
    );
  }
  return actualSha256.toLowerCase();
}

function writeVerificationMarker(releaseRoot, manifest, archivePath, archiveSha256) {
  const marker = {
    schemaVersion: 1,
    releaseVersion: manifest.releaseVersion,
    baseCommit: manifest.baseCommit,
    archiveAssetName: path.basename(archivePath),
    archiveSha256,
    manifestSha256: computeFileSha256(path.join(releaseRoot, 'manifest.json')),
    releaseTreeSha256: computeReleaseTreeSha256(releaseRoot),
    verifiedAt: new Date().toISOString(),
  };
  fs.writeFileSync(
    path.join(releaseRoot, verifiedReleaseMarkerName),
    `${JSON.stringify(marker, null, 2)}\n`,
    'utf8',
  );
}

function installArchive(archivePath, releaseRoot, expectedSha256, lock) {
  const archiveSha256 = verifyChecksum(archivePath, expectedSha256);
  const releasesRoot = path.dirname(releaseRoot);
  fs.mkdirSync(releasesRoot, { recursive: true });
  const temporaryRoot = fs.mkdtempSync(path.join(releasesRoot, '.install-'));
  const backupRoot = path.join(releasesRoot, `.backup-${path.basename(releaseRoot)}-${process.pid}-${Date.now()}`);
  let previousMoved = false;
  let installedMoved = false;

  try {
    runCommand('tar', ['-xzf', archivePath, '-C', temporaryRoot], `extract ${archivePath}`);
    const manifest = readFoundationReleaseManifest(temporaryRoot, lock);
    writeVerificationMarker(temporaryRoot, manifest, archivePath, archiveSha256);

    if (fs.existsSync(releaseRoot)) {
      fs.renameSync(releaseRoot, backupRoot);
      previousMoved = true;
    }
    fs.renameSync(temporaryRoot, releaseRoot);
    installedMoved = true;
    if (previousMoved) {
      fs.rmSync(backupRoot, { recursive: true, force: true });
      previousMoved = false;
    }
  } catch (error) {
    fs.rmSync(temporaryRoot, { recursive: true, force: true });
    if (installedMoved) {
      fs.rmSync(releaseRoot, { recursive: true, force: true });
    }
    if (previousMoved && fs.existsSync(backupRoot)) {
      fs.renameSync(backupRoot, releaseRoot);
    }
    throw error;
  }
}

function printHelp() {
  console.log(`Usage:
  node scripts/foundation-release/install-foundation-release.mjs [--archive <path>] [--repo <owner/repo>] [--ops-root <path>]

Behavior:
  Installs the locked pantheon-base foundation release under .foundation/releases/<releaseVersion>.
  Without --archive, downloads foundation-release-<releaseVersion>.tgz from the configured GitHub release.
  A SHA-256 from the adjacent .sha256 file or lock is required and verified before extraction.
  The manifest is validated in a temporary directory before the installed release is replaced.
  Temp download directories are cleaned up after successful installation.`);
}

function main() {
  let downloadResult = null;
  try {
    const options = parseArgs(process.argv.slice(2));
    if (options.help) {
      printHelp();
      return 0;
    }

    const lock = readFoundationLock(options.opsRoot);
    const releaseRoot = path.join(options.opsRoot, '.foundation', 'releases', lock.releaseVersion);
    if (options.archivePath) {
      downloadResult = {
        archivePath: path.resolve(options.archivePath),
        checksum: checksumFromAdjacentFile(path.resolve(options.archivePath)),
        downloadDir: null,
      };
    } else {
      downloadResult = downloadArchive(lock, options);
    }

    if (!fs.existsSync(downloadResult.archivePath)) {
      if (options.ifExists) {
        console.log(`Foundation release ${lock.releaseVersion} artifact not found on ${lock.releaseArtifact?.githubRepo ?? 'GitHub'} — skipping install`);
        return 0;
      }
      throw new Error(`downloaded release artifact not found: ${downloadResult.archivePath}`);
    }

    const expectedChecksum = requireExpectedChecksum(
      downloadResult.checksum,
      lock.releaseArtifact?.checksum,
    );
    installArchive(downloadResult.archivePath, releaseRoot, expectedChecksum, lock);

    console.log(`Installed foundation release ${lock.releaseVersion} to ${releaseRoot}`);
    return 0;
  } catch (error) {
    console.error(error.message);
    return 1;
  } finally {
    if (downloadResult?.downloadDir) {
      fs.rmSync(downloadResult.downloadDir, { recursive: true, force: true });
    }
  }
}

if (process.argv[1] && fileURLToPath(import.meta.url) === path.resolve(process.argv[1])) {
  process.exitCode = main();
}

import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import process from 'node:process';
import { spawnSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';

import {
  readFoundationLock,
  readFoundationReleaseManifest,
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

  let checksum = null;
  const checksumAssetName = `${assetName}.sha256`;
  try {
    runCommand(
      'gh',
      ['release', 'download', lock.releaseVersion, '--repo', repo, '--pattern', checksumAssetName, '--dir', downloadDir],
      `gh release download checksum ${lock.releaseVersion}`,
      { cwd: options.opsRoot },
    );
    const checksumPath = path.join(downloadDir, checksumAssetName);
    if (fs.existsSync(checksumPath)) {
      const checksumContent = fs.readFileSync(checksumPath, 'utf8').trim();
      checksum = checksumContent.split(/\s+/)[0];
    }
  } catch {
    // checksum file not present — skip verification
  }

  return { archivePath, checksum, downloadDir };
}

function installArchive(archivePath, releaseRoot, expectedSha256) {
  fs.rmSync(releaseRoot, { recursive: true, force: true });
  fs.mkdirSync(releaseRoot, { recursive: true });
  runCommand('tar', ['-xzf', archivePath, '-C', releaseRoot], `extract ${archivePath}`);
  if (expectedSha256) {
    verifyChecksum(archivePath, expectedSha256);
  }
  fs.rmSync(archivePath, { force: true });
}

function verifyChecksum(archivePath, expectedSha256) {
  const result = spawnSync('certutil', ['-hashfile', archivePath, 'SHA256'], { encoding: 'utf8' });
  if (result.status !== 0) {
    throw new Error(`checksum computation failed: ${result.stderr || result.stdout}`);
  }
  const actualSha256 = result.stdout.split('\n').find((l) => l.trim().length === 64)?.trim();
  if (!actualSha256) {
    throw new Error('could not parse SHA256 from certutil output');
  }
  if (actualSha256.toLowerCase() !== expectedSha256.toLowerCase()) {
    throw new Error(
      `checksum mismatch for ${archivePath}\n` +
      `  expected: ${expectedSha256}\n` +
      `  actual:   ${actualSha256}`,
    );
  }
}

function printHelp() {
  console.log(`Usage:
  node scripts/foundation-release/install-foundation-release.mjs [--archive <path>] [--repo <owner/repo>] [--ops-root <path>]

Behavior:
  Installs the locked pantheon-base foundation release under .foundation/releases/<releaseVersion>.
  Without --archive, downloads foundation-release-<releaseVersion>.tgz from the configured GitHub release.
  If a .sha256 checksum file is present on the release, it is verified after download.
  The lock file may also carry a "checksum" field as fallback verification.
  Temp download directories are cleaned up after successful installation.`);
}

function main() {
  try {
    const options = parseArgs(process.argv.slice(2));
    if (options.help) {
      printHelp();
      return 0;
    }

    const lock = readFoundationLock(options.opsRoot);
    const releaseRoot = path.join(options.opsRoot, '.foundation', 'releases', lock.releaseVersion);
    let downloadResult;

    if (options.archivePath) {
      downloadResult = {
        archivePath: path.resolve(options.archivePath),
        checksum: null,
        downloadDir: null,
      };
    } else {
      downloadResult = downloadArchive(lock, options);
    }

    if (!fs.existsSync(downloadResult.archivePath)) {
      if (options.ifExists) {
        console.log(`Foundation release ${lock.releaseVersion} artifact not found on ${lock.releaseArtifact?.githubRepo ?? 'GitHub'} — skipping install`);
        if (downloadResult.downloadDir) {
          fs.rmSync(downloadResult.downloadDir, { recursive: true, force: true });
        }
        return 0;
      }
      throw new Error(`downloaded release artifact not found: ${downloadResult.archivePath}`);
    }

    installArchive(
      downloadResult.archivePath,
      releaseRoot,
      downloadResult.checksum ?? lock.releaseArtifact?.checksum ?? null,
    );
    readFoundationReleaseManifest(releaseRoot, lock);

    // Clean up the temp download directory
    if (downloadResult.downloadDir) {
      fs.rmSync(downloadResult.downloadDir, { recursive: true, force: true });
    }

    console.log(`Installed foundation release ${lock.releaseVersion} to ${releaseRoot}`);
    return 0;
  } catch (error) {
    console.error(error.message);
    return 1;
  }
}

if (process.argv[1] && fileURLToPath(import.meta.url) === path.resolve(process.argv[1])) {
  process.exitCode = main();
}

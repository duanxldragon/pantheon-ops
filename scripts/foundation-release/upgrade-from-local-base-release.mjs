import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';
import { spawnSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';

import {
  computeFileSha256,
  computeReleaseTreeSha256,
  readFoundationLock,
  verifiedReleaseMarkerName,
} from './shared-foundation-rules.mjs';

const currentFilePath = fileURLToPath(import.meta.url);
const scriptsDir = path.dirname(currentFilePath);
const opsRoot = path.resolve(scriptsDir, '..', '..');

function parseArgs(argv) {
  const options = {
    opsRoot,
  };

  for (let index = 0; index < argv.length; index += 1) {
    const arg = argv[index];
    const value = argv[index + 1];

    if (arg === '--release-version') {
      if (!value) throw new Error('--release-version requires a value');
      options.releaseVersion = value;
      index += 1;
    } else if (arg === '--base-root') {
      if (!value) throw new Error('--base-root requires a path');
      options.baseRoot = path.resolve(value);
      index += 1;
    } else if (arg === '--plan-only') {
      options.planOnly = true;
    } else if (arg === '--help' || arg === '-h') {
      options.help = true;
    } else {
      throw new Error(`Unknown argument: ${arg}`);
    }
  }

  return options;
}

function resolveOptions(options) {
  const lock = readFoundationLock(options.opsRoot);
  const baseRoot = options.baseRoot
    ?? path.resolve(options.opsRoot, lock.baseRepo ?? '../pantheon-base');
  const releaseVersion = options.releaseVersion;
  if (!releaseVersion) {
    throw new Error('--release-version is required');
  }

  const bundleRoot = path.join(baseRoot, 'dist', 'foundation-releases', releaseVersion);
  const manifestPath = path.join(bundleRoot, 'manifest.json');

  return {
    ...options,
    lock,
    baseRoot,
    releaseVersion,
    bundleRoot,
    manifestPath,
  };
}

function runNodeScript(cwd, scriptPath, args) {
  const result = spawnSync(process.execPath, [scriptPath, ...args], {
    cwd,
    encoding: 'utf8',
  });

  if (result.status !== 0) {
    throw new Error(result.stderr || result.stdout || `${scriptPath} failed`);
  }

  return result.stdout.trim();
}

function runGit(baseRoot, args) {
  const result = spawnSync('git', args, { cwd: baseRoot, encoding: 'utf8' });
  if (result.status !== 0) {
    throw new Error(result.stderr || result.stdout || `git ${args.join(' ')} failed`);
  }
  return result.stdout.trim();
}

function readArchiveChecksum(checksumPath, archivePath) {
  const contents = fs.readFileSync(checksumPath, 'utf8').trim();
  const match = contents.match(/^([a-fA-F0-9]{64})\s+\*?(.+)$/u);
  if (!match || path.basename(match[2]) !== path.basename(archivePath)) {
    throw new Error(`invalid foundation release checksum file: ${checksumPath}`);
  }
  return match[1].toLowerCase();
}

export function verifyAndMarkLocalRelease({ baseRoot, bundleRoot, manifestPath, releaseVersion }) {
  const worktreeStatus = runGit(baseRoot, ['status', '--short']);
  if (worktreeStatus) {
    throw new Error('local foundation upgrade requires a clean pantheon-base worktree');
  }

  const manifest = JSON.parse(fs.readFileSync(manifestPath, 'utf8'));
  const headCommit = runGit(baseRoot, ['rev-parse', 'HEAD']);
  if (manifest.releaseVersion !== releaseVersion || manifest.baseCommit !== headCommit) {
    throw new Error(
      `local foundation identity mismatch: release=${manifest.releaseVersion} baseCommit=${manifest.baseCommit} HEAD=${headCommit}`,
    );
  }

  const archivePath = path.join(bundleRoot, `foundation-release-${releaseVersion}.tgz`);
  const checksumPath = `${archivePath}.sha256`;
  const expectedChecksum = readArchiveChecksum(checksumPath, archivePath);
  const actualChecksum = computeFileSha256(archivePath);
  if (actualChecksum !== expectedChecksum) {
    throw new Error(`local foundation archive checksum mismatch: ${archivePath}`);
  }

  const marker = {
    schemaVersion: 1,
    releaseVersion,
    baseCommit: manifest.baseCommit,
    archiveAssetName: path.basename(archivePath),
    archiveSha256: actualChecksum,
    manifestSha256: computeFileSha256(manifestPath),
    releaseTreeSha256: computeReleaseTreeSha256(bundleRoot),
    verifiedAt: new Date().toISOString(),
  };
  fs.writeFileSync(
    path.join(bundleRoot, verifiedReleaseMarkerName),
    `${JSON.stringify(marker, null, 2)}\n`,
    'utf8',
  );
  return marker;
}

function printHelp() {
  console.log(`Usage:
  node scripts/foundation-release/upgrade-from-local-base-release.mjs --release-version <version> [--base-root <path>] [--plan-only]

Behavior:
  1. build pantheon-base dist/foundation-releases/<version>/ bundle locally
  2. run pantheon-ops consume-foundation-release in plan or apply mode`);
}

function main() {
  try {
    const parsed = parseArgs(process.argv.slice(2));
    if (parsed.help) {
      printHelp();
      return 0;
    }

    const options = resolveOptions(parsed);
    const buildScriptPath = path.join(options.baseRoot, 'scripts', 'foundation-release', 'build-release-bundle.mjs');
    const consumeScriptPath = path.join(options.opsRoot, 'scripts', 'foundation-release', 'consume-foundation-release.mjs');

    const summary = [];
    summary.push(`Current locked foundation: ${options.lock.releaseVersion} (${options.lock.baseCommit})`);
    summary.push(`Target local foundation release: ${options.releaseVersion}`);

    runNodeScript(options.baseRoot, buildScriptPath, ['--release-version', options.releaseVersion]);
    verifyAndMarkLocalRelease(options);
    summary.push(`Built local base bundle: ${options.bundleRoot}`);

    const consumeArgs = [
      '--ops-root',
      options.opsRoot,
      '--manifest',
      options.manifestPath,
      '--bundle',
      options.bundleRoot,
    ];

    if (options.planOnly) {
      consumeArgs.push('--dry-run');
    } else {
      consumeArgs.push('--apply-shared-backend', '--apply-shared-frontend', '--update-inheritance-docs', '--rollback-on-error', '--check');
    }

    runNodeScript(options.opsRoot, consumeScriptPath, consumeArgs);
    summary.push(options.planOnly ? 'Planned local foundation upgrade' : 'Applied local foundation upgrade');

    console.log(summary.join('\n'));
    return 0;
  } catch (error) {
    console.error(error.message);
    return 1;
  }
}

if (process.argv[1] && fileURLToPath(import.meta.url) === path.resolve(process.argv[1])) {
  process.exitCode = main();
}

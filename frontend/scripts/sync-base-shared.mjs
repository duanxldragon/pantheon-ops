import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';
import { spawnSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';
import {
  allowedFrontendOpsOnlyPaths,
  collectFiles,
  ensureDir,
  frontendOverlayPaths,
  sharedFrontendEntries,
  toRepoPath,
} from '../../scripts/foundation-release/shared-foundation-rules.mjs';

const currentFilePath = fileURLToPath(import.meta.url);
const scriptsDir = path.dirname(currentFilePath);
const opsFrontendRoot = path.resolve(scriptsDir, '..');
const opsRoot = path.resolve(opsFrontendRoot, '..');
const releaseLockPath = path.join(opsRoot, 'foundation-release.lock.json');

const opsSrcRoot = path.join(opsFrontendRoot, 'src');
const frontendWorktreeRoots = [
  'frontend/src/components',
  'frontend/src/core',
  'frontend/src/store',
  'frontend/src/modules/auth',
  'frontend/src/modules/lowcode',
  'frontend/src/modules/platform',
  'frontend/src/modules/system',
  'frontend/src/index.css',
];

const checkMode = process.argv.includes('--check');

function readJson(filePath, description) {
  try {
    return JSON.parse(fs.readFileSync(filePath, 'utf8'));
  } catch (error) {
    if (error?.code === 'ENOENT') {
      throw new Error(`${description} not found: ${filePath}`);
    }
    throw new Error(`${description} is invalid JSON: ${error.message}`);
  }
}

function resolveCachedReleaseFrontendRoot() {
  const releaseLock = readJson(releaseLockPath, 'foundation release lock');
  const localPath = releaseLock.releaseArtifact?.localPath;
  if (typeof localPath !== 'string' || localPath.length === 0) {
    throw new Error('foundation-release.lock.json: releaseArtifact.localPath must be set');
  }

  const releaseRoot = path.resolve(opsRoot, localPath);
  const baseSrcRoot = path.join(releaseRoot, 'bundle', 'shared-frontend', 'frontend', 'src');
  if (!fs.existsSync(baseSrcRoot)) {
    throw new Error(`cached foundation release frontend root not found: ${baseSrcRoot}`);
  }

  return {
    releaseLock,
    baseSrcRoot,
  };
}

function readFile(filePath) {
  return fs.readFileSync(filePath, 'utf8');
}

function listGitUntrackedPaths(roots) {
  const result = spawnSync('git', ['ls-files', '--others', '--exclude-standard', '--', ...roots], {
    cwd: opsRoot,
    encoding: 'utf8',
  });
  if (result.status !== 0) {
    throw new Error(result.stderr || result.stdout || 'failed to list untracked frontend files');
  }
  return result.stdout
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter(Boolean);
}

function collectSharedBaseFiles(baseSrcRoot) {
  const files = [];
  for (const entry of sharedFrontendEntries) {
    const absolutePath = path.join(baseSrcRoot, entry);
    const stats = fs.statSync(absolutePath);
    if (stats.isDirectory()) {
      files.push(...collectFiles(baseSrcRoot, absolutePath));
      continue;
    }
    files.push(toRepoPath(entry));
  }
  return files.sort((a, b) => a.localeCompare(b));
}

function collectSharedOpsOnlyFiles(baseSrcRoot) {
  const extraFiles = [];
  for (const entry of sharedFrontendEntries) {
    const absolutePath = path.join(opsSrcRoot, entry);
    if (!fs.existsSync(absolutePath)) {
      continue;
    }
    const stats = fs.statSync(absolutePath);
    if (!stats.isDirectory()) {
      continue;
    }
    for (const relativePath of collectFiles(opsSrcRoot, absolutePath)) {
      const basePath = path.join(baseSrcRoot, relativePath);
      if (fs.existsSync(basePath)) {
        continue;
      }
      if (frontendOverlayPaths.has(relativePath) || allowedFrontendOpsOnlyPaths.has(relativePath)) {
        continue;
      }
      extraFiles.push(relativePath);
    }
  }
  return extraFiles.sort((a, b) => a.localeCompare(b));
}

function main() {
  const { baseSrcRoot } = resolveCachedReleaseFrontendRoot();
  const sharedFiles = collectSharedBaseFiles(baseSrcRoot);
  const changedFiles = [];
  const driftFiles = [];
  const missingFiles = [];
  const untrackedFiles = [];

  for (const relativePath of sharedFiles) {
    if (frontendOverlayPaths.has(relativePath)) {
      continue;
    }
    const baseFilePath = path.join(baseSrcRoot, relativePath);
    const opsFilePath = path.join(opsSrcRoot, relativePath);
    if (!fs.existsSync(opsFilePath)) {
      if (checkMode) {
        missingFiles.push(relativePath);
        continue;
      }
      ensureDir(opsFilePath);
      fs.writeFileSync(opsFilePath, readFile(baseFilePath), 'utf8');
      changedFiles.push(relativePath);
      continue;
    }

    const baseSource = readFile(baseFilePath);
    const opsSource = readFile(opsFilePath);
    if (baseSource === opsSource) {
      continue;
    }

    if (checkMode) {
      driftFiles.push(relativePath);
      continue;
    }

    ensureDir(opsFilePath);
    fs.writeFileSync(opsFilePath, baseSource, 'utf8');
    changedFiles.push(relativePath);
  }

  const opsOnlyFiles = collectSharedOpsOnlyFiles(baseSrcRoot);
  for (const relativePath of listGitUntrackedPaths(frontendWorktreeRoots)) {
    if (frontendOverlayPaths.has(relativePath) || allowedFrontendOpsOnlyPaths.has(relativePath)) {
      continue;
    }
    if (!sharedFiles.includes(relativePath)) {
      untrackedFiles.push(relativePath);
      continue;
    }
    untrackedFiles.push(relativePath);
  }

  if (checkMode) {
    if (missingFiles.length === 0 && driftFiles.length === 0 && opsOnlyFiles.length === 0 && untrackedFiles.length === 0) {
      console.log('OK shared frontend is aligned with pantheon-base');
      return;
    }

    console.error('pantheon-ops shared frontend drift detected');
    for (const relativePath of missingFiles) {
      console.error(`MISSING ${relativePath}`);
    }
    for (const relativePath of driftFiles) {
      console.error(`DIFF ${relativePath}`);
    }
    for (const relativePath of opsOnlyFiles) {
      console.error(`OPS_ONLY ${relativePath}`);
    }
    for (const relativePath of untrackedFiles) {
      console.error(`UNTRACKED ${relativePath}`);
    }
    process.exit(1);
  }

  for (const relativePath of opsOnlyFiles) {
    fs.rmSync(path.join(opsSrcRoot, relativePath), { force: true });
    changedFiles.push(relativePath);
  }

  console.log(
    changedFiles.length === 0
      ? 'No shared frontend files needed syncing'
      : `Synced ${changedFiles.length} shared frontend files from pantheon-base`,
  );
}

main();

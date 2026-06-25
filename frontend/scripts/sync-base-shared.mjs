import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';
import { fileURLToPath } from 'node:url';
import {
  allowedFrontendOpsOnlyPaths,
  collectFiles,
  ensureDir,
  frontendOverlayPaths,
  listFilesFromGitCommit,
  readFileFromGitCommit,
  readFoundationLock,
  resolveFoundationReleasePaths,
  resolveBaseRepoRoot,
  rewriteFrontendBaseSource,
  sharedFrontendEntriesFromLock,
  stripTreePrefix,
  normalizeLineEndings,
  toOriginalFrontendPath,
  toRelocatedFrontendPath,
  toRepoPath,
} from '../../scripts/foundation-release/shared-foundation-rules.mjs';

const currentFilePath = fileURLToPath(import.meta.url);
const scriptsDir = path.dirname(currentFilePath);
const opsFrontendRoot = path.resolve(scriptsDir, '..');
const opsRoot = path.resolve(opsFrontendRoot, '..');
const foundationLock = readFoundationLock(opsRoot);
const compareWorkspaceHead = process.argv.includes('--workspace-head');

const opsSrcRoot = path.join(opsFrontendRoot, 'src');

const checkMode = process.argv.includes('--check');

function readFile(filePath) {
  return fs.readFileSync(filePath, 'utf8');
}

function statIfPresent(filePath) {
  try {
    return fs.statSync(filePath);
  } catch (error) {
    if (error?.code === 'ENOENT') {
      return null;
    }
    throw error;
  }
}

function resolveSharedSource() {
  if (compareWorkspaceHead) {
    const baseRepoRoot = resolveBaseRepoRoot(opsRoot, foundationLock);
    if (!fs.existsSync(baseRepoRoot)) {
      throw new Error(`pantheon-base repo root not found: ${baseRepoRoot}`);
    }
    return {
      baseRepoRoot,
      targetCommit: 'HEAD',
      sourceLabel: 'pantheon-base workspace HEAD',
    };
  }

  const releasePaths = resolveFoundationReleasePaths(opsRoot, foundationLock);
  return {
    releaseRoot: releasePaths.releaseRoot,
    sourceRoot: releasePaths.sharedFrontendRoot,
    targetCommit: foundationLock.baseCommit,
    sourceLabel: `foundation release ${foundationLock.releaseVersion} (${foundationLock.baseCommit})`,
  };
}

const sharedSource = resolveSharedSource();

function readRewrittenBaseSource(relativePath) {
  const originalRelativePath = toOriginalFrontendPath(relativePath);
  const baseSource = sharedSource.sourceRoot
    ? readFile(path.join(sharedSource.sourceRoot, originalRelativePath))
    : readFileFromGitCommit(sharedSource.baseRepoRoot, sharedSource.targetCommit, `frontend/src/${originalRelativePath}`);
  return rewriteFrontendBaseSource(baseSource, originalRelativePath, relativePath);
}

function collectSharedBaseFiles() {
  const files = new Set();
  for (const entry of sharedFrontendEntriesFromLock(foundationLock)) {
    if (sharedSource.sourceRoot) {
      const entryPath = path.join(sharedSource.sourceRoot, entry);
      const stats = statIfPresent(entryPath);
      if (!stats) {
        if (entry.includes('.')) {
          files.add(toRepoPath(entry));
        }
        continue;
      }
      if (!stats.isDirectory()) {
        files.add(toRelocatedFrontendPath(toRepoPath(entry)));
        continue;
      }
      for (const filePath of collectFiles(sharedSource.sourceRoot, entryPath)) {
        files.add(toRelocatedFrontendPath(filePath));
      }
    } else {
      const treePrefix = `frontend/src/${entry}`;
      const gitFiles = listFilesFromGitCommit(sharedSource.baseRepoRoot, sharedSource.targetCommit, treePrefix);
      if (gitFiles.length === 0 && !entry.includes('.')) {
        continue;
      }
      if (gitFiles.length === 0) {
        files.add(toRepoPath(entry));
        continue;
      }
      for (const filePath of gitFiles) {
        files.add(toRelocatedFrontendPath(stripTreePrefix(filePath, 'frontend/src')));
      }
    }
  }
  return [...files].sort((a, b) => a.localeCompare(b));
}

function collectSharedOpsOnlyFiles() {
  const extraFiles = [];
  for (const entry of sharedFrontendEntriesFromLock(foundationLock)) {
    const absolutePath = path.join(opsSrcRoot, entry);
    const stats = statIfPresent(absolutePath);
    if (!stats?.isDirectory()) {
      continue;
    }
    for (const relativePath of collectFiles(opsSrcRoot, absolutePath)) {
      const originalRelativePath = toOriginalFrontendPath(relativePath);
      const canonicalRelativePath = toRelocatedFrontendPath(originalRelativePath);
      if (canonicalRelativePath !== relativePath) {
        extraFiles.push(relativePath);
        continue;
      }
      const basePath = `frontend/src/${originalRelativePath}`;
      try {
        if (sharedSource.sourceRoot) {
          readFile(path.join(sharedSource.sourceRoot, originalRelativePath));
        } else {
          readFileFromGitCommit(sharedSource.baseRepoRoot, sharedSource.targetCommit, basePath);
        }
        continue;
      } catch {}
      if (frontendOverlayPaths.has(relativePath) || allowedFrontendOpsOnlyPaths.has(relativePath)) {
        continue;
      }
      extraFiles.push(relativePath);
    }
  }
  return extraFiles.sort((a, b) => a.localeCompare(b));
}

function main() {
  const sharedFiles = collectSharedBaseFiles();
  const changedFiles = [];
  const driftFiles = [];
  const missingFiles = [];

  for (const relativePath of sharedFiles) {
    if (frontendOverlayPaths.has(relativePath)) {
      continue;
    }
    const opsFilePath = path.join(opsSrcRoot, relativePath);
    let opsSource = '';
    try {
      opsSource = readFile(opsFilePath);
    } catch (error) {
      if (error?.code !== 'ENOENT') {
        throw error;
      }
      if (checkMode) {
        missingFiles.push(relativePath);
        continue;
      }
      ensureDir(opsFilePath);
      fs.writeFileSync(opsFilePath, readRewrittenBaseSource(relativePath), 'utf8');
      changedFiles.push(relativePath);
      continue;
    }

    const baseSource = readRewrittenBaseSource(relativePath);
    if (normalizeLineEndings(baseSource) === normalizeLineEndings(opsSource)) {
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

  const opsOnlyFiles = collectSharedOpsOnlyFiles();

  if (checkMode) {
    if (missingFiles.length === 0 && driftFiles.length === 0 && opsOnlyFiles.length === 0) {
      console.log(`OK shared frontend is aligned with ${sharedSource.sourceLabel}`);
      return;
    }

    console.error(`pantheon-ops shared frontend drift detected against ${sharedSource.sourceLabel}`);
    for (const relativePath of missingFiles) {
      console.error(`MISSING ${relativePath}`);
    }
    for (const relativePath of driftFiles) {
      console.error(`DIFF ${relativePath}`);
    }
    for (const relativePath of opsOnlyFiles) {
      console.error(`OPS_ONLY ${relativePath}`);
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
      : `Synced ${changedFiles.length} shared frontend files from ${
        sharedSource.sourceLabel
      }`,
  );
}

main();

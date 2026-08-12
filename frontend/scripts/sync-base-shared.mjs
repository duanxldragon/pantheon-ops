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
  mergeFrontendPackageJson,
  mergeSmokeReadme,
  readFileFromGitCommit,
  readFoundationLock,
  requiredSharedFrontendEntries,
  resolveFoundationReleasePaths,
  resolveBaseRepoRoot,
  rewriteFrontendBaseSource,
  sharedFrontendEntriesFromLock,
  sharedFrontendToolingEntriesFromLock,
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

function argumentValue(name) {
  const index = process.argv.indexOf(name);
  if (index < 0) {
    return undefined;
  }
  const value = process.argv[index + 1];
  if (!value) {
    throw new Error(`${name} requires a path`);
  }
  return path.resolve(value);
}

const explicitSourceRoot = argumentValue('--source-root');
const explicitManifestPath = argumentValue('--manifest');
if (Boolean(explicitSourceRoot) !== Boolean(explicitManifestPath)) {
  throw new Error('--source-root and --manifest must be provided together');
}
const lockedFoundation = readFoundationLock(opsRoot);
const explicitManifest = explicitManifestPath
  ? JSON.parse(fs.readFileSync(explicitManifestPath, 'utf8'))
  : null;
const foundationLock = explicitManifest
  ? {
      ...lockedFoundation,
      releaseVersion: explicitManifest.releaseVersion,
      baseCommit: explicitManifest.baseCommit,
      sharedPaths: {
        ...lockedFoundation.sharedPaths,
        ...explicitManifest.sharedPaths,
      },
    }
  : lockedFoundation;
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
  if (explicitSourceRoot) {
    return {
      sourceRoot: explicitSourceRoot,
      frontendTreeRoot: path.resolve(explicitSourceRoot, '..'),
      targetCommit: foundationLock.baseCommit,
      sourceLabel: `foundation release ${foundationLock.releaseVersion} (${foundationLock.baseCommit})`,
    };
  }
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
    manifest: releasePaths.manifest,
    sourceRoot: releasePaths.sharedFrontendRoot,
    frontendTreeRoot: releasePaths.sharedFrontendTreeRoot,
    targetCommit: foundationLock.baseCommit,
    sourceLabel: `foundation release ${foundationLock.releaseVersion} (${foundationLock.baseCommit})`,
  };
}

function readSharedToolingSource(repoRelativePath) {
  let baseSource;
  if (sharedSource.frontendTreeRoot) {
    baseSource = readFile(
      path.join(sharedSource.frontendTreeRoot, stripTreePrefix(repoRelativePath, 'frontend')),
    );
  } else {
    baseSource = readFileFromGitCommit(
      sharedSource.baseRepoRoot,
      sharedSource.targetCommit,
      repoRelativePath,
    );
  }

  const opsFilePath = path.join(opsRoot, repoRelativePath);
  if (repoRelativePath === 'frontend/package.json' && fs.existsSync(opsFilePath)) {
    return mergeFrontendPackageJson(baseSource, readFile(opsFilePath));
  }
  if (repoRelativePath === 'frontend/tests/smoke/README.md') {
    return mergeSmokeReadme(baseSource, opsRoot);
  }
  return baseSource;
}

function collectSharedToolingFiles() {
  const files = new Set();
  for (const entry of sharedFrontendToolingEntriesFromLock(foundationLock)) {
    if (sharedSource.frontendTreeRoot) {
      const entryPath = path.join(
        sharedSource.frontendTreeRoot,
        stripTreePrefix(entry, 'frontend'),
      );
      const stats = statIfPresent(entryPath);
      if (!stats) {
        files.add(toRepoPath(entry));
        continue;
      }
      if (!stats.isDirectory()) {
        files.add(toRepoPath(entry));
        continue;
      }
      for (const filePath of collectFiles(sharedSource.frontendTreeRoot, entryPath)) {
        files.add(`frontend/${filePath}`);
      }
      continue;
    }

    const gitFiles = listFilesFromGitCommit(
      sharedSource.baseRepoRoot,
      sharedSource.targetCommit,
      entry,
    );
    if (gitFiles.length === 0) {
      files.add(toRepoPath(entry));
      continue;
    }
    for (const filePath of gitFiles) {
      files.add(toRepoPath(filePath));
    }
  }
  return [...files].sort((left, right) => left.localeCompare(right));
}

function collectSharedToolingOpsOnlyFiles(sharedToolingFiles) {
  const expectedFiles = new Set(sharedToolingFiles);
  const extraFiles = [];
  for (const entry of sharedFrontendToolingEntriesFromLock(foundationLock)) {
    const absolutePath = path.join(opsRoot, entry);
    const stats = statIfPresent(absolutePath);
    if (!stats?.isDirectory()) {
      continue;
    }
    for (const relativePath of collectFiles(opsRoot, absolutePath)) {
      const repoRelativePath = toRepoPath(relativePath);
      if (!expectedFiles.has(repoRelativePath)) {
        extraFiles.push(repoRelativePath);
      }
    }
  }
  return extraFiles.sort((left, right) => left.localeCompare(right));
}

const sharedSource = resolveSharedSource();

function normalizeSharedPath(entry) {
  return toRepoPath(entry).replace(/\/+$/u, '');
}

function assertVerifiedManifestMatchesLock() {
  if (!sharedSource.manifest) {
    return;
  }
  const lockEntries = (foundationLock.sharedPaths?.frontend || [])
    .map(normalizeSharedPath)
    .sort((left, right) => (left < right ? -1 : left > right ? 1 : 0));
  const manifestEntries = (sharedSource.manifest.sharedPaths?.frontend || [])
    .map(normalizeSharedPath)
    .sort((left, right) => (left < right ? -1 : left > right ? 1 : 0));
  if (lockEntries.join('\n') !== manifestEntries.join('\n')) {
    throw new Error('foundation-release.lock.json frontend paths do not match the verified release manifest');
  }
}

function collectUnownedGenericFrontendFiles() {
  const sharedEntries = new Set(sharedFrontendEntriesFromLock(foundationLock));
  return requiredSharedFrontendEntries.filter((entry) => {
    const absolutePath = path.join(opsSrcRoot, entry);
    return fs.existsSync(absolutePath) && !sharedEntries.has(entry);
  });
}

assertVerifiedManifestMatchesLock();

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
  const sharedToolingFiles = collectSharedToolingFiles();
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

  for (const repoRelativePath of sharedToolingFiles) {
    const opsFilePath = path.join(opsRoot, repoRelativePath);
    const baseSource = readSharedToolingSource(repoRelativePath);
    let opsSource = null;
    try {
      opsSource = readFile(opsFilePath);
    } catch (error) {
      if (error?.code !== 'ENOENT') {
        throw error;
      }
    }

    if (opsSource === null) {
      if (checkMode) {
        missingFiles.push(repoRelativePath);
        continue;
      }
      ensureDir(opsFilePath);
      fs.writeFileSync(opsFilePath, baseSource, 'utf8');
      changedFiles.push(repoRelativePath);
      continue;
    }

    if (normalizeLineEndings(baseSource) === normalizeLineEndings(opsSource)) {
      continue;
    }
    if (checkMode) {
      driftFiles.push(repoRelativePath);
      continue;
    }

    fs.writeFileSync(opsFilePath, baseSource, 'utf8');
    changedFiles.push(repoRelativePath);
  }

  const opsOnlyFiles = collectSharedOpsOnlyFiles();
  const opsOnlyToolingFiles = collectSharedToolingOpsOnlyFiles(sharedToolingFiles);
  const unownedFiles = collectUnownedGenericFrontendFiles();

  if (checkMode) {
    if (
      missingFiles.length === 0
      && driftFiles.length === 0
      && opsOnlyFiles.length === 0
      && opsOnlyToolingFiles.length === 0
      && unownedFiles.length === 0
    ) {
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
    for (const relativePath of opsOnlyToolingFiles) {
      console.error(`OPS_ONLY ${relativePath}`);
    }
    for (const relativePath of unownedFiles) {
      console.error(`UNOWNED ${relativePath}`);
    }
    process.exit(1);
  }

  for (const relativePath of opsOnlyFiles) {
    fs.rmSync(path.join(opsSrcRoot, relativePath), { force: true });
    changedFiles.push(relativePath);
  }
  for (const repoRelativePath of opsOnlyToolingFiles) {
    fs.rmSync(path.join(opsRoot, repoRelativePath), { force: true });
    changedFiles.push(repoRelativePath);
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

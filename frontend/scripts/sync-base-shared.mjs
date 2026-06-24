import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';
import { fileURLToPath } from 'node:url';
import {
  allowedFrontendOpsOnlyPaths,
  collectFiles,
  ensureDir,
  frontendRelocatedComponentPrefixes,
  frontendOverlayPaths,
  listFilesFromGitCommit,
  readFileFromGitCommit,
  readFoundationLock,
  resolveBaseRepoRoot,
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
const baseRepoRoot = resolveBaseRepoRoot(opsRoot, foundationLock);
const compareWorkspaceHead = process.argv.includes('--workspace-head');
const targetCommit = compareWorkspaceHead ? 'HEAD' : foundationLock.baseCommit;

// In CI, pantheon-base may not be checked out as a sibling directory.
if (!fs.existsSync(baseRepoRoot)) {
  console.warn('[sync-base-shared] pantheon-base not found at', baseRepoRoot);
  console.warn('[sync-base-shared] skipping base sync check (running in CI without base repo)');
  process.exit(0);
}

const opsSrcRoot = path.join(opsFrontendRoot, 'src');

const checkMode = process.argv.includes('--check');

function readFile(filePath) {
  return fs.readFileSync(filePath, 'utf8');
}

function toRelativeImportSpecifier(fromRelativePath, toRelativePath) {
  const fromDir = path.posix.dirname(toRepoPath(fromRelativePath));
  const nextSpecifier = path.posix.relative(fromDir, toRepoPath(toRelativePath));
  if (nextSpecifier.startsWith('../')) {
    return nextSpecifier;
  }
  return `./${nextSpecifier}`;
}

function rewriteRelativeImportSpecifier(specifier, originalRelativePath, relocatedRelativePath) {
  if (!specifier.startsWith('./') && !specifier.startsWith('../')) {
    return specifier;
  }

  const originalDir = path.posix.dirname(toRepoPath(originalRelativePath));
  const originalTarget = path.posix.normalize(path.posix.join(originalDir, specifier));
  const relocatedTarget = toRelocatedFrontendPath(originalTarget);
  const nextSpecifier = toRelativeImportSpecifier(relocatedRelativePath, relocatedTarget);
  return nextSpecifier === '.' ? './' : nextSpecifier;
}

function rewriteFrontendBaseSource(baseSource, originalRelativePath, relocatedRelativePath) {
  let nextSource = baseSource
    .replace(
      /\b(from\s*|import\s*\(\s*|import\s+)(['"])(\.{1,2}\/[^'"]+)\2/gu,
      (match, prefix, quote, specifier) =>
        `${prefix}${quote}${rewriteRelativeImportSpecifier(
          specifier,
          originalRelativePath,
          relocatedRelativePath,
        )}${quote}`,
    );
  nextSource = nextSource.replace(
    /(\bcomponent(?:Key)?\s*:\s*['"])([^'"]+)(['"])/gu,
    (match, prefix, componentKey, suffix) => {
      for (const [fromPrefix, toPrefix] of frontendRelocatedComponentPrefixes.entries()) {
        if (componentKey.startsWith(fromPrefix)) {
          return `${prefix}${toPrefix}${componentKey.slice(fromPrefix.length)}${suffix}`;
        }
      }
      return match;
    },
  );
  return nextSource;
}

function readRewrittenBaseSource(relativePath) {
  const originalRelativePath = toOriginalFrontendPath(relativePath);
  const baseSource = readFileFromGitCommit(baseRepoRoot, targetCommit, `frontend/src/${originalRelativePath}`);
  return rewriteFrontendBaseSource(baseSource, originalRelativePath, relativePath);
}

function collectSharedBaseFiles() {
  const files = new Set();
  for (const entry of sharedFrontendEntriesFromLock(foundationLock)) {
    const treePrefix = `frontend/src/${entry}`;
    const gitFiles = listFilesFromGitCommit(baseRepoRoot, targetCommit, treePrefix);
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
  return [...files].sort((a, b) => a.localeCompare(b));
}

function collectSharedOpsOnlyFiles() {
  const extraFiles = [];
  for (const entry of sharedFrontendEntriesFromLock(foundationLock)) {
    const absolutePath = path.join(opsSrcRoot, entry);
    if (!fs.existsSync(absolutePath)) {
      continue;
    }
    const stats = fs.statSync(absolutePath);
    if (!stats.isDirectory()) {
      continue;
    }
    for (const relativePath of collectFiles(opsSrcRoot, absolutePath)) {
      const originalRelativePath = toOriginalFrontendPath(relativePath);
      const basePath = `frontend/src/${originalRelativePath}`;
      try {
        readFileFromGitCommit(baseRepoRoot, targetCommit, basePath);
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
    if (!fs.existsSync(opsFilePath)) {
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
    const opsSource = readFile(opsFilePath);
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
      const sourceLabel = compareWorkspaceHead
        ? 'pantheon-base workspace HEAD'
        : `pantheon-base ${foundationLock.releaseVersion} (${foundationLock.baseCommit})`;
      console.log(`OK shared frontend is aligned with ${sourceLabel}`);
      return;
    }

    const sourceLabel = compareWorkspaceHead
      ? 'pantheon-base workspace HEAD'
      : `pantheon-base ${foundationLock.releaseVersion} (${foundationLock.baseCommit})`;
    console.error(`pantheon-ops shared frontend drift detected against ${sourceLabel}`);
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
        compareWorkspaceHead ? 'pantheon-base workspace HEAD' : `${foundationLock.releaseVersion}`
      }`,
  );
}

main();

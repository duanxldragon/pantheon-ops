import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';
import { fileURLToPath } from 'node:url';
import {
  allowedBackendOpsOnlyPaths,
  backendMergedJsonPaths,
  collectFiles,
  detectBackendModuleNameFromTree,
  isBackendOverlayPath,
  listFilesFromGitCommit,
  mergeBuiltinLocaleResources,
  normalizeLineEndings,
  readFileFromGitCommit,
  readFoundationLock,
  resolveFoundationReleasePaths,
  readGoModuleName,
  resolveBaseRepoRoot,
  rewriteBackendModuleReferences,
  sharedBackendEntriesFromLock,
  stripTreePrefix,
  toRepoPath,
} from './foundation-release/shared-foundation-rules.mjs';

const currentFilePath = fileURLToPath(import.meta.url);
const opsRoot = path.resolve(path.dirname(currentFilePath), '..');
const opsBackendRoot = path.join(opsRoot, 'backend');
const compareWorkspaceHead = process.argv.includes('--workspace-head');

function readFile(filePath) {
  return fs.readFileSync(filePath, 'utf8');
}

function buildExpectedOpsSource(relativePath, baseSource, opsSource, baseModuleName, opsModuleName) {
  const rewrittenBaseSource = rewriteBackendModuleReferences(baseSource, baseModuleName, opsModuleName);
  if (!backendMergedJsonPaths.has(relativePath) || !opsSource) {
    return rewrittenBaseSource;
  }

  return mergeBuiltinLocaleResources(rewrittenBaseSource, opsSource);
}

function collectSharedBaseFilesFromWorkspace() {
  const lock = readFoundationLock(opsRoot);
  const baseRepoRoot = resolveBaseRepoRoot(opsRoot, lock);
  const targetCommit = 'HEAD';
  if (!fs.existsSync(baseRepoRoot)) {
    return {
      baseRepoRoot,
      lock,
      targetCommit,
      files: [],
      missingBaseRepo: true,
    };
  }
  const files = new Set();

  for (const entry of sharedBackendEntriesFromLock(lock)) {
    const treePrefix = `backend/${entry}`;
    for (const filePath of listFilesFromGitCommit(baseRepoRoot, targetCommit, treePrefix)) {
      files.add(stripTreePrefix(filePath, 'backend'));
    }
  }
  return {
    baseRepoRoot,
    lock,
    targetCommit,
    files: [...files].sort((a, b) => a.localeCompare(b)),
  };
}

function collectSharedBaseFilesFromRelease() {
  const lock = readFoundationLock(opsRoot);
  const releasePaths = resolveFoundationReleasePaths(opsRoot, lock);
  const files = new Set();

  for (const entry of sharedBackendEntriesFromLock(lock)) {
    const entryDir = path.join(releasePaths.sharedBackendRoot, entry);
    if (!fs.existsSync(entryDir)) {
      continue;
    }
    const stats = fs.statSync(entryDir);
    if (!stats.isDirectory()) {
      files.add(toRepoPath(entry));
      continue;
    }
    for (const filePath of collectFiles(releasePaths.sharedBackendRoot, entryDir)) {
      files.add(filePath);
    }
  }

  return {
    lock,
    targetCommit: lock.baseCommit,
    sourceRoot: releasePaths.sharedBackendRoot,
    sourceLabel: `foundation release ${lock.releaseVersion} (${lock.baseCommit})`,
    files: [...files].sort((a, b) => a.localeCompare(b)),
  };
}

function collectSharedBaseFiles() {
  if (compareWorkspaceHead) {
    return collectSharedBaseFilesFromWorkspace();
  }
  return collectSharedBaseFilesFromRelease();
}

function readSharedBaseSource(source, relativePath) {
  if (source.sourceRoot) {
    return readFile(path.join(source.sourceRoot, relativePath));
  }
  return readFileFromGitCommit(source.baseRepoRoot, source.targetCommit, `backend/${relativePath}`);
}

function main() {
  const source = collectSharedBaseFiles();
  const { baseRepoRoot, lock, targetCommit, files: baseFiles, missingBaseRepo } = source;
  if (missingBaseRepo) {
    console.error(`pantheon-base repo root not found: ${baseRepoRoot}`);
    process.exit(1);
  }
  const baseFileSet = new Set(baseFiles);

  const baseModuleName = source.sourceRoot
    ? detectBackendModuleNameFromTree(source.sourceRoot)
    : readGoModuleName(baseRepoRoot);
  const opsModuleName = readGoModuleName(opsRoot);

  const missingFiles = [];
  const diffFiles = [];
  const staleFiles = [];

  // 1) Files in base that are missing or differ in ops
  for (const relativePath of baseFiles) {
    if (isBackendOverlayPath(relativePath)) {
      continue;
    }

    const opsFilePath = path.join(opsBackendRoot, relativePath);

    if (!fs.existsSync(opsFilePath)) {
      missingFiles.push(relativePath);
      continue;
    }

    const baseSource = readSharedBaseSource(source, relativePath);
    const opsSource = readFile(opsFilePath);

    const expectedOpsSource = buildExpectedOpsSource(relativePath, baseSource, opsSource, baseModuleName, opsModuleName);
    if (normalizeLineEndings(expectedOpsSource) !== normalizeLineEndings(opsSource)) {
      diffFiles.push(relativePath);
    }
  }

  // 2) Stale files — exist in ops under shared paths but NOT in base lock
  //    These are files that base removed but ops still holds (e.g. auth sub-services
  //    merged into auth_service.go). Business-only and overlay paths are excluded.
  for (const entry of sharedBackendEntriesFromLock(lock)) {
    const entryDir = path.join(opsBackendRoot, entry);
    if (!fs.existsSync(entryDir) || !fs.statSync(entryDir).isDirectory()) {
      continue;
    }
    for (const relativePath of collectFiles(opsBackendRoot, entryDir)) {
      if (isBackendOverlayPath(relativePath)) {
        continue;
      }
      if (allowedBackendOpsOnlyPaths.has(relativePath)) {
        continue;
      }
      // Skip business/ — business modules are ops-owned, not base-shared
      if (relativePath.startsWith('modules/business/') || relativePath.startsWith('modules\\business\\')) {
        continue;
      }
      if (baseFileSet.has(relativePath)) {
        continue;
      }
      // Try reading from base commit — if it exists, it's just a different file name
      try {
        readSharedBaseSource(source, relativePath);
        continue;
      } catch {}
      staleFiles.push(relativePath);
    }
  }

  if (missingFiles.length === 0 && diffFiles.length === 0 && staleFiles.length === 0) {
    const sourceLabel = source.sourceLabel ?? 'pantheon-base workspace HEAD';
    console.log(`OK shared backend is aligned with ${sourceLabel}`);
    return;
  }

  const sourceLabel = source.sourceLabel ?? 'pantheon-base workspace HEAD';
  console.error(`pantheon-ops shared backend drift detected against ${sourceLabel}`);
  for (const relativePath of missingFiles) {
    console.error(`MISSING ${relativePath}`);
  }
  for (const relativePath of diffFiles) {
    console.error(`DIFF ${relativePath}`);
  }
  for (const relativePath of staleFiles) {
    console.error(`STALE ${relativePath}`);
  }
  process.exit(1);
}

main();

import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';
import { fileURLToPath } from 'node:url';
import {
  allowedBackendOpsOnlyPaths,
  backendMergedJsonPaths,
  collectFiles,
  isBackendOverlayPath,
  listFilesFromGitCommit,
  mergeBuiltinLocaleResources,
  normalizeLineEndings,
  readFileFromGitCommit,
  readFoundationLock,
  readGoModuleName,
  resolveBaseRepoRoot,
  rewriteBackendModuleReferences,
  sharedBackendEntriesFromLock,
  stripTreePrefix,
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

function collectSharedBaseFiles() {
  const lock = readFoundationLock(opsRoot);
  const baseRepoRoot = resolveBaseRepoRoot(opsRoot, lock);
  const targetCommit = compareWorkspaceHead ? 'HEAD' : lock.baseCommit;
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

function main() {
  const { baseRepoRoot, lock, targetCommit, files: baseFiles, missingBaseRepo } = collectSharedBaseFiles();
  if (missingBaseRepo) {
    if (process.env.PANTHEON_CI || process.env.CI) {
      console.warn('[check-base-backend-sync] pantheon-base not found at', baseRepoRoot);
      console.warn('[check-base-backend-sync] skipping backend sync check (running in CI without base repo)');
      process.exit(0);
    }
    console.error(`pantheon-base repo root not found: ${baseRepoRoot}`);
    process.exit(1);
  }
  const baseFileSet = new Set(baseFiles);

  const baseModuleName = readGoModuleName(baseRepoRoot);
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

    const baseSource = readFileFromGitCommit(baseRepoRoot, targetCommit, `backend/${relativePath}`);
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
        readFileFromGitCommit(baseRepoRoot, targetCommit, `backend/${relativePath}`);
        continue;
      } catch {}
      staleFiles.push(relativePath);
    }
  }

  if (missingFiles.length === 0 && diffFiles.length === 0 && staleFiles.length === 0) {
    const sourceLabel = compareWorkspaceHead
      ? 'pantheon-base workspace HEAD'
      : `pantheon-base ${lock.releaseVersion} (${lock.baseCommit})`;
    console.log(`OK shared backend is aligned with ${sourceLabel}`);
    return;
  }

  const sourceLabel = compareWorkspaceHead
    ? 'pantheon-base workspace HEAD'
    : `pantheon-base ${lock.releaseVersion} (${lock.baseCommit})`;
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

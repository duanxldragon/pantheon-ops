import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';
import { spawnSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';
import {
  backendMergedJsonPaths,
  collectFiles,
  detectBackendModuleNameFromTree,
  isBackendOverlayPath,
  mergeBuiltinLocaleResources,
  normalizeLineEndings,
  readGoModuleName,
  rewriteBackendModuleReferences,
  sharedBackendEntries,
} from './foundation-release/shared-foundation-rules.mjs';

const currentFilePath = fileURLToPath(import.meta.url);
const opsRoot = path.resolve(path.dirname(currentFilePath), '..');
const releaseLockPath = path.join(opsRoot, 'foundation-release.lock.json');
const backendWorktreeRoots = [
  'backend/cmd',
  'backend/internal',
  'backend/pkg',
  'backend/modules/auth',
  'backend/modules/lowcode',
  'backend/modules/platform',
  'backend/modules/system',
];

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

function resolveCachedReleaseBackendRoot() {
  const releaseLock = readJson(releaseLockPath, 'foundation release lock');
  const localPath = releaseLock.releaseArtifact?.localPath;
  if (typeof localPath !== 'string' || localPath.length === 0) {
    throw new Error('foundation-release.lock.json: releaseArtifact.localPath must be set');
  }

  const releaseRoot = path.resolve(opsRoot, localPath);
  const baseBackendRoot = path.join(releaseRoot, 'bundle', 'shared-backend', 'backend');
  if (!fs.existsSync(baseBackendRoot)) {
    throw new Error(`cached foundation release backend root not found: ${baseBackendRoot}`);
  }

  return {
    releaseLock,
    baseBackendRoot,
  };
}

const opsBackendRoot = path.join(opsRoot, 'backend');

function readFile(filePath) {
  return fs.readFileSync(filePath, 'utf8');
}

function listGitUntrackedPaths(roots) {
  const result = spawnSync('git', ['ls-files', '--others', '--exclude-standard', '--', ...roots], {
    cwd: opsRoot,
    encoding: 'utf8',
  });
  if (result.status !== 0) {
    throw new Error(result.stderr || result.stdout || 'failed to list untracked backend files');
  }
  return result.stdout
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter(Boolean);
}

function buildExpectedOpsSource(relativePath, baseSource, opsSource, baseModuleName, opsModuleName) {
  const rewrittenBaseSource = rewriteBackendModuleReferences(baseSource, baseModuleName, opsModuleName);
  if (!backendMergedJsonPaths.has(relativePath) || !opsSource) {
    return rewrittenBaseSource;
  }

  return mergeBuiltinLocaleResources(rewrittenBaseSource, opsSource);
}

function collectSharedBaseFiles(baseBackendRoot) {
  const files = [];
  for (const entry of sharedBackendEntries) {
    const absolutePath = path.join(baseBackendRoot, entry);
    if (!fs.existsSync(absolutePath)) {
      continue;
    }
    files.push(...collectFiles(baseBackendRoot, absolutePath));
  }
  return files.sort((a, b) => a.localeCompare(b));
}

function main() {
  const { releaseLock, baseBackendRoot } = resolveCachedReleaseBackendRoot();
  const baseModuleName = releaseLock.baseGoModule || detectBackendModuleNameFromTree(baseBackendRoot);
  const opsModuleName = readGoModuleName(opsRoot);

  const missingFiles = [];
  const diffFiles = [];
  const untrackedFiles = [];

  for (const relativePath of collectSharedBaseFiles(baseBackendRoot)) {
    if (isBackendOverlayPath(relativePath)) {
      continue;
    }

    const baseFilePath = path.join(baseBackendRoot, relativePath);
    const opsFilePath = path.join(opsBackendRoot, relativePath);

    if (!fs.existsSync(opsFilePath)) {
      missingFiles.push(relativePath);
      continue;
    }

    const baseSource = readFile(baseFilePath);
    const opsSource = readFile(opsFilePath);

    const expectedOpsSource = buildExpectedOpsSource(relativePath, baseSource, opsSource, baseModuleName, opsModuleName);
    if (normalizeLineEndings(expectedOpsSource) !== normalizeLineEndings(opsSource)) {
      diffFiles.push(relativePath);
    }
  }

  for (const relativePath of listGitUntrackedPaths(backendWorktreeRoots)) {
    if (isBackendOverlayPath(relativePath)) {
      continue;
    }
    if (relativePath.startsWith('backend/modules/business/')) {
      continue;
    }
    untrackedFiles.push(relativePath);
  }

  if (missingFiles.length === 0 && diffFiles.length === 0 && untrackedFiles.length === 0) {
    console.log('OK shared backend is aligned with pantheon-base');
    return;
  }

  console.error('pantheon-ops shared backend drift detected');
  for (const relativePath of missingFiles) {
    console.error(`MISSING ${relativePath}`);
  }
  for (const relativePath of diffFiles) {
    console.error(`DIFF ${relativePath}`);
  }
  for (const relativePath of untrackedFiles) {
    console.error(`UNTRACKED ${relativePath}`);
  }
  process.exit(1);
}

main();

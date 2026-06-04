import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';
import { fileURLToPath } from 'node:url';
import {
  backendMergedJsonPaths,
  collectFiles,
  isBackendOverlayPath,
  mergeBuiltinLocaleResources,
  normalizeLineEndings,
  readGoModuleName,
  rewriteBackendModuleReferences,
  sharedBackendEntries,
} from './foundation-release/shared-foundation-rules.mjs';

const currentFilePath = fileURLToPath(import.meta.url);
const opsRoot = path.resolve(path.dirname(currentFilePath), '..');
const workspaceRoot = path.resolve(opsRoot, '..');
const baseRepoRoot = process.env.PANTHEON_BASE_REPO_ROOT
  ? path.resolve(process.env.PANTHEON_BASE_REPO_ROOT)
  : path.join(workspaceRoot, 'pantheon-base');
const baseBackendRoot = path.join(baseRepoRoot, 'backend');
const opsBackendRoot = path.join(opsRoot, 'backend');

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
  if (!fs.existsSync(baseBackendRoot)) {
    console.error(`pantheon-base backend root not found: ${baseBackendRoot}`);
    process.exit(1);
  }

  const baseModuleName = readGoModuleName(baseRepoRoot);
  const opsModuleName = readGoModuleName(opsRoot);

  const missingFiles = [];
  const diffFiles = [];

  for (const relativePath of collectSharedBaseFiles()) {
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

  if (missingFiles.length === 0 && diffFiles.length === 0) {
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
  process.exit(1);
}

main();

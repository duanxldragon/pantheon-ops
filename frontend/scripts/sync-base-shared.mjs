import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';
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
const workspaceRoot = path.resolve(opsFrontendRoot, '..', '..');
const baseRepoRoot = process.env.PANTHEON_BASE_REPO_ROOT
  ? path.resolve(process.env.PANTHEON_BASE_REPO_ROOT)
  : path.join(workspaceRoot, 'pantheon-base');
const baseSrcRoot = path.join(baseRepoRoot, 'frontend', 'src');

// In CI, pantheon-base may not be checked out as a sibling directory.
if (!fs.existsSync(baseSrcRoot)) {
  console.warn('[sync-base-shared] pantheon-base not found at', baseSrcRoot);
  console.warn('[sync-base-shared] skipping base sync check (running in CI without base repo)');
  process.exit(0);
}

const opsSrcRoot = path.join(opsFrontendRoot, 'src');

const checkMode = process.argv.includes('--check');

function readFile(filePath) {
  return fs.readFileSync(filePath, 'utf8');
}

function collectSharedBaseFiles() {
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

function collectSharedOpsOnlyFiles() {
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
  const sharedFiles = collectSharedBaseFiles();
  const changedFiles = [];
  const driftFiles = [];
  const missingFiles = [];

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

  const opsOnlyFiles = collectSharedOpsOnlyFiles();

  if (checkMode) {
    if (missingFiles.length === 0 && driftFiles.length === 0 && opsOnlyFiles.length === 0) {
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

import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';
import { fileURLToPath } from 'node:url';

const currentFilePath = fileURLToPath(import.meta.url);
const opsRoot = path.resolve(path.dirname(currentFilePath), '..');
const workspaceRoot = path.resolve(opsRoot, '..');
const baseBackendRoot = path.join(workspaceRoot, 'pantheon-base', 'backend');
const opsBackendRoot = path.join(workspaceRoot, 'pantheon-ops', 'backend');

const sharedEntries = ['cmd', 'internal', 'modules', 'pkg'];

const excludedPaths = new Set([
  'internal/scaffold/workspace.go',
  'internal/scaffold/workspace_test.go',
  'modules/business/business.go',
  'modules/business/retired_modules.go',
  'modules/platform/health.go',
  'modules/system/dynamicmodule/dynamic_module_service_test.go',
  'modules/system/generator/generator_service_test.go',
  'modules/system/iam/menu/component_registry.go',
]);

function toRepoPath(filePath) {
  return filePath.split(path.sep).join('/');
}

function readFile(filePath) {
  return fs.readFileSync(filePath, 'utf8');
}

function normalizeBackendSource(source) {
  return source
    .replaceAll('\r\n', '\n')
    .replaceAll('pantheon-platform/backend', '__BACKEND_MODULE__')
    .replaceAll('pantheon-base/backend', '__BACKEND_MODULE__')
    .replaceAll('pantheon-ops/backend', '__BACKEND_MODULE__')
    .replaceAll('module pantheon-platform', 'module __ROOT_MODULE__')
    .replaceAll('module pantheon-ops', 'module __ROOT_MODULE__');
}

function collectFiles(rootPath, currentPath = rootPath, bucket = []) {
  const entries = fs.readdirSync(currentPath, { withFileTypes: true });
  for (const entry of entries) {
    const nextPath = path.join(currentPath, entry.name);
    if (entry.isDirectory()) {
      collectFiles(rootPath, nextPath, bucket);
      continue;
    }
    bucket.push(toRepoPath(path.relative(rootPath, nextPath)));
  }
  return bucket;
}

function collectSharedBaseFiles() {
  const files = [];
  for (const entry of sharedEntries) {
    const absolutePath = path.join(baseBackendRoot, entry);
    if (!fs.existsSync(absolutePath)) {
      continue;
    }
    files.push(...collectFiles(baseBackendRoot, absolutePath));
  }
  return files.sort();
}

function main() {
  const missingFiles = [];
  const diffFiles = [];

  for (const relativePath of collectSharedBaseFiles()) {
    if (excludedPaths.has(relativePath)) {
      continue;
    }

    const baseFilePath = path.join(baseBackendRoot, relativePath);
    const opsFilePath = path.join(opsBackendRoot, relativePath);

    if (!fs.existsSync(opsFilePath)) {
      missingFiles.push(relativePath);
      continue;
    }

    const baseSource = normalizeBackendSource(readFile(baseFilePath));
    const opsSource = normalizeBackendSource(readFile(opsFilePath));

    if (baseSource !== opsSource) {
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

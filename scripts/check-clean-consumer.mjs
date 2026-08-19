import { spawnSync } from 'node:child_process';
import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';
import { fileURLToPath } from 'node:url';

import {
  frontendOverlayPaths,
  readFoundationLock,
  toRepoPath,
} from './foundation-release/shared-foundation-rules.mjs';

const currentFilePath = fileURLToPath(import.meta.url);
const defaultOpsRoot = path.resolve(path.dirname(currentFilePath), '..');

const consumerOwnedFrontendPrefixes = Object.freeze([
  'frontend/src/i18n/',
  'frontend/src/modules/generated/',
]);

function isPathWithin(candidate, entry) {
  return candidate === entry || candidate.startsWith(`${entry}/`);
}

export function classifyFrontendSource(filePath, lock) {
  const normalizedPath = toRepoPath(filePath);
  if (normalizedPath.startsWith('frontend/src/modules/business/')) {
    return 'business';
  }
  if (consumerOwnedFrontendPrefixes.some((prefix) => normalizedPath.startsWith(prefix))) {
    return 'generated-overlay';
  }

  const sourceRelativePath = normalizedPath.replace(/^frontend\/src\//u, '');
  if (frontendOverlayPaths.has(sourceRelativePath)) {
    return 'explicit-overlay';
  }

  const sharedEntries = lock.sharedPaths?.frontend ?? [];
  if (sharedEntries.some((entry) => isPathWithin(normalizedPath, toRepoPath(entry)))) {
    return 'foundation';
  }
  return 'unauthorized-residual';
}

function trackedFrontendSources(opsRoot) {
  const result = spawnSync('git', ['ls-files', 'frontend/src'], {
    cwd: opsRoot,
    encoding: 'utf8',
  });
  if (result.status !== 0) {
    throw new Error(result.stderr?.trim() || 'git ls-files frontend/src failed');
  }
  return result.stdout.split(/\r?\n/u).map((line) => line.trim()).filter(Boolean);
}

export function findUnauthorizedFrontendResiduals(opsRoot = defaultOpsRoot) {
  const lock = readFoundationLock(opsRoot);
  return trackedFrontendSources(opsRoot)
    .filter((filePath) => fs.existsSync(path.join(opsRoot, filePath)))
    .filter((filePath) => classifyFrontendSource(filePath, lock) === 'unauthorized-residual')
    .sort((left, right) => (left < right ? -1 : left > right ? 1 : 0));
}

function main() {
  const residuals = findUnauthorizedFrontendResiduals();
  if (residuals.length === 0) {
    console.log('OK clean consumer ownership: no unauthorized frontend source residuals');
    return;
  }
  console.error('pantheon-ops clean consumer ownership check failed');
  for (const residual of residuals) {
    console.error(`UNAUTHORIZED_RESIDUAL ${residual}`);
  }
  process.exitCode = 1;
}

if (process.argv[1] && path.resolve(process.argv[1]) === currentFilePath) {
  main();
}

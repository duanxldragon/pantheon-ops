import crypto from 'node:crypto';
import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';
import { spawnSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';

// Compares every file under the lock file's sharedPaths between this consumer
// tree and a pantheon-base checkout, by sha256. Any drift not explicitly
// declared in .foundation/allowed-drift.json fails the check.
//
// Usage:
//   node scripts/business-overlay/check-shared-drift.mjs --base ../pantheon-base
//   node scripts/business-overlay/check-shared-drift.mjs --base ../pantheon-base --group backend
//   node scripts/business-overlay/check-shared-drift.mjs --base ../pantheon-base --lock .foundation/foundation-release.lock.json

const defaultRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..', '..');

const readArg = (name) => {
  const index = process.argv.indexOf(name);
  return index >= 0 ? path.resolve(process.cwd(), process.argv[index + 1]) : undefined;
};

const root = readArg('--root') ?? defaultRoot;
const base = readArg('--base');
const lockArg = readArg('--lock');
const groupArgIndex = process.argv.indexOf('--group');
const group = groupArgIndex >= 0 ? process.argv[groupArgIndex + 1] : 'all';

// The canonical lock lives at the repo root (foundation-release.lock.json, the
// one rebuild-from-base.mjs also reads). Fall back to the .foundation copy for
// locked releases that carry sharedPaths there.
const defaultLock = path.join(root, 'foundation-release.lock.json');
const fallbackLock = path.join(root, '.foundation', 'foundation-release.lock.json');
const lockPath = lockArg ?? (fs.existsSync(defaultLock) ? defaultLock : fallbackLock);

if (!base || !fs.existsSync(base)) {
  console.error('FATAL --base <path-to-pantheon-base-checkout> is required and must exist');
  process.exit(2);
}

const lock = JSON.parse(fs.readFileSync(lockPath, 'utf8'));
const sharedPaths = lock.sharedPaths ?? {};

if (Object.keys(sharedPaths).length === 0) {
  console.error(`FATAL lock ${path.relative(root, lockPath)} declares no sharedPaths; nothing to compare`);
  process.exit(2);
}

const allowlistPath = path.join(root, '.foundation', 'allowed-drift.json');
const allowedDrift = fs.existsSync(allowlistPath)
  ? JSON.parse(fs.readFileSync(allowlistPath, 'utf8'))
  : {};

const sha256File = (file) => crypto.createHash('sha256').update(fs.readFileSync(file)).digest('hex');

// Compare against base HEAD, not the base working tree, so unrelated
// uncommitted changes in the base checkout never mask as consumer drift.
// Falls back to the working-tree file for paths not tracked in HEAD yet.
const gitShowHead = (baseRoot, relative) => {
  const result = spawnSync('git', ['-C', baseRoot, 'show', `HEAD:${relative}`], {
    encoding: 'buffer',
    maxBuffer: 64 * 1024 * 1024,
  });
  return result.status === 0 ? result.stdout : null;
};
const baseHashes = new Map();
const hashBaseFile = (baseRoot, relative) => {
  const key = relative.replaceAll('\\', '/');
  if (baseHashes.has(key)) return baseHashes.get(key);
  let hash;
  const head = gitShowHead(baseRoot, key);
  if (head !== null) hash = crypto.createHash('sha256').update(head).digest('hex');
  else if (fs.existsSync(path.join(baseRoot, key))) hash = sha256File(path.join(baseRoot, key));
  baseHashes.set(key, hash);
  return hash;
};

// Business overlay paths are owned by this repo; they are not shared with base
// even when they live under a shared parent directory in sharedPaths.
const businessPaths = (() => {
  const manifestPath = path.join(root, 'business-overlay.json');
  if (!fs.existsSync(manifestPath)) return [];
  return JSON.parse(fs.readFileSync(manifestPath, 'utf8')).businessPaths ?? [];
})();
const isBusinessPath = (relative) =>
  businessPaths.some((declared) => relative === declared || relative.startsWith(`${declared}/`));

const collectFiles = (absoluteRoot, relativePath, output) => {
  const absolute = path.join(absoluteRoot, relativePath);
  if (!fs.existsSync(absolute)) return;
  const stat = fs.statSync(absolute);
  if (stat.isFile()) {
    output.add(relativePath.replaceAll('\\', '/'));
    return;
  }
  if (!stat.isDirectory()) return;
  for (const entry of fs.readdirSync(absolute, { withFileTypes: true })) {
    if (entry.name === 'node_modules' || entry.name === '.git' || entry.name === 'dist') continue;
    collectFiles(absoluteRoot, path.join(relativePath, entry.name), output);
  }
};

const groups = group === 'all' ? Object.keys(sharedPaths) : [group];
const opsFiles = new Set();
for (const g of groups) {
  for (const sharedPath of sharedPaths[g] ?? []) {
    collectFiles(root, sharedPath, opsFiles);
  }
}

const modified = [];
const missingInBase = [];
const matchedAllowlist = new Set();
let comparedFiles = 0;
let skippedBusinessFiles = 0;

for (const relative of [...opsFiles].sort()) {
  if (isBusinessPath(relative)) {
    skippedBusinessFiles += 1;
    continue;
  }
  comparedFiles += 1;
  const baseHash = hashBaseFile(base, relative);
  if (baseHash === undefined) {
    missingInBase.push(relative);
    continue;
  }
  if (sha256File(path.join(root, relative)) !== baseHash) {
    modified.push(relative);
  }
}

const unexpectedModified = [];
const unexpectedMissing = [];
for (const file of modified) {
  if (Object.hasOwn(allowedDrift, file)) matchedAllowlist.add(file);
  else unexpectedModified.push(file);
}
for (const file of missingInBase) {
  if (Object.hasOwn(allowedDrift, file)) matchedAllowlist.add(file);
  else unexpectedMissing.push(file);
}
const staleAllowlist = Object.keys(allowedDrift)
  .filter((file) => !file.startsWith('$'))
  .filter((file) => !matchedAllowlist.has(file));

for (const file of unexpectedModified) console.error(`DRIFT_MODIFIED ${file}`);
for (const file of unexpectedMissing) console.error(`DRIFT_MISSING_IN_BASE ${file}`);
for (const file of staleAllowlist) console.error(`STALE_ALLOWLIST ${file} (no longer drifted; remove from .foundation/allowed-drift.json)`);
for (const file of matchedAllowlist) console.log(`ALLOWED_DRIFT ${file} — ${allowedDrift[file]}`);

const failed = unexpectedModified.length + unexpectedMissing.length + staleAllowlist.length;
if (failed > 0) {
  process.exitCode = 1;
  console.error(`FAIL shared drift check: ${unexpectedModified.length} modified, ${unexpectedMissing.length} missing in base, ${staleAllowlist.length} stale allowlist (${comparedFiles} shared files compared, ${skippedBusinessFiles} business-overlay files skipped, base=${base})`);
} else {
  console.log(`OK shared drift check: ${comparedFiles} shared files identical to base, ${matchedAllowlist.size} allowlisted drift, ${skippedBusinessFiles} business-overlay files skipped (base=${base})`);
}

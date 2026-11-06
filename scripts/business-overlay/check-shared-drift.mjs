import crypto from 'node:crypto';
import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';
import { fileURLToPath } from 'node:url';
import { assertSafeManifest, planBusinessMigrations } from './rebuild-from-base.mjs';
import { resolveBaseFromLock } from './foundation-release.mjs';

// Compare consumer bytes to the verified, locked official GitHub snapshot.
// Usage: npm run check:shared-drift -- --root .tmp/clean-base-overlay

const defaultRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..', '..');

export function checkSharedDrift({ root, snapshot, group = 'all' }) {
  const base = snapshot.baseRoot;
  const lock = snapshot.manifest;
  const sharedPaths = lock.sharedPaths ?? {};

  if (Object.keys(sharedPaths).length === 0) {
    console.error(`FATAL lock foundation-release.lock.json declares no sharedPaths; nothing to compare`);
    return 2;
  }

  const allowlistPath = path.join(root, '.foundation', 'allowed-drift.json');
  const allowedDrift = fs.existsSync(allowlistPath)
    ? JSON.parse(fs.readFileSync(allowlistPath, 'utf8'))
    : {};

  const sha256File = (file) => crypto.createHash('sha256').update(fs.readFileSync(file)).digest('hex');

  // Snapshot bytes are authoritative; never let Git discover an enclosing Ops
  // repository and substitute its HEAD for downloaded Base files.
  const hashBaseFile = (baseRoot, relative) => {
    const absolute = path.join(baseRoot, relative);
    return fs.existsSync(absolute) ? sha256File(absolute) : undefined;
  };

  // Business overlay paths are owned by this repo; they are not shared with base
  // even when they live under a shared parent directory in sharedPaths.
  const manifestPath = path.join(root, 'business-overlay.json');
  const manifest = fs.existsSync(manifestPath)
    ? JSON.parse(fs.readFileSync(manifestPath, 'utf8'))
    : { schemaVersion: 1 };
  assertSafeManifest(manifest);
  const businessPaths = manifest.businessPaths ?? [];
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

  // Only the planner's exact migration destinations are derived business assets.
  // Validate canonical bytes before skipping them; generatedFiles is not authority.
  const migrationPlans = planBusinessMigrations(manifest);
  const derivedMigrations = new Set();
  if (migrationPlans.length > 0) {
    const baseMigrationFiles = new Set();
    const migrationDirectory = 'backend/pkg/database/migrations';
    collectFiles(base, migrationDirectory, baseMigrationFiles);
    const baseVersions = new Set([...baseMigrationFiles]
      .map((file) => /^(\d+)_.*\.(up|down)\.sql$/u.exec(path.posix.basename(file))?.[1])
      .filter(Boolean).map(Number));
    for (const { source, target, version } of migrationPlans) {
      if (baseVersions.has(Number(version))) throw new Error('Business migration version conflicts with Base: ' + version);
      for (const [kind, file] of [['source', source], ['target', target]]) {
        const absolute = path.join(root, file);
        if (!fs.existsSync(absolute) || !fs.lstatSync(absolute).isFile()) {
          throw new Error('Business migration ' + kind + ' must be a regular file: ' + file);
        }
      }
      if (!fs.readFileSync(path.join(root, source)).equals(fs.readFileSync(path.join(root, target)))) {
        throw new Error('Business migration target differs from source: ' + target + ' != ' + source);
      }
      derivedMigrations.add(target);
    }
  }

  if (group !== 'all' && !Object.hasOwn(sharedPaths, group)) throw new Error('Unknown shared path group: ' + group);
  const groups = group === 'all' ? Object.keys(sharedPaths) : [group];
  const opsFiles = new Set();
  for (const g of groups) {
    for (const sharedPath of sharedPaths[g] ?? []) {
      collectFiles(root, sharedPath, opsFiles);
      collectFiles(base, sharedPath, opsFiles);
    }
  }

  const modified = [];
  const missingInBase = [];
  const matchedAllowlist = new Set();
  let comparedFiles = 0;
  let skippedBusinessFiles = 0;

  for (const relative of [...opsFiles].sort()) {
    if (derivedMigrations.has(relative)) continue;
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
    if (!fs.existsSync(path.join(root, relative)) || sha256File(path.join(root, relative)) !== baseHash) {
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

    console.error(`FAIL shared drift check: ${unexpectedModified.length} modified, ${unexpectedMissing.length} missing in base, ${staleAllowlist.length} stale allowlist (${comparedFiles} shared files compared, ${skippedBusinessFiles} business-overlay files skipped, ${derivedMigrations.size} derived business migrations verified, base=${base})`);
  } else {
    console.log(`OK shared drift check: ${comparedFiles} shared files identical to base, ${matchedAllowlist.size} allowlisted drift, ${skippedBusinessFiles} business-overlay files skipped, ${derivedMigrations.size} derived business migrations verified (base=${base})`);
  }

  return failed > 0 ? 1 : 0;
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  try {
    if (process.argv.includes('--base') || process.argv.includes('--lock')) throw new Error('Local Base or alternative lock arguments are not supported; use the official GitHub release lock');
    const argument = (name) => { const index = process.argv.indexOf(name); return index < 0 ? undefined : process.argv[index + 1]; };
    const root = path.resolve(argument('--root') ?? defaultRoot);
    process.exitCode = checkSharedDrift({ root, snapshot: resolveBaseFromLock(root), group: argument('--group') ?? 'all' });
  } catch (error) {
    console.error(error.message);
    process.exitCode = 1;
  }
}

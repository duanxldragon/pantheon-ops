import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import process from 'node:process';
import { spawnSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';
import {
  backendMergedJsonPaths,
  collectFiles,
  detectBackendModuleNameFromTree,
  ensureDir,
  isBackendBusinessPath,
  isBackendOverlayPath,
  isFrontendBusinessPath,
  isFrontendOverlayPath,
  mergeBuiltinLocaleResources,
  readFoundationLock,
  readGoModuleName,
  rewriteFrontendBaseSource,
  rewriteBackendModuleReferences,
  toRelocatedFrontendPath,
} from './shared-foundation-rules.mjs';

const DEFAULT_OPS_ROOT = process.cwd();

function parseArgs(argv) {
  const options = {
    opsRoot: DEFAULT_OPS_ROOT,
    applySharedBackend: false,
    applySharedFrontend: false,
    updateInheritanceDocs: false,
    check: false,
    dryRun: false,
    rollbackOnError: false,
    allowReleaseLineJump: false,
    skipGoValidation: false,
  };

  for (let index = 0; index < argv.length; index += 1) {
    const arg = argv[index];
    const value = argv[index + 1];

    if (arg === '--ops-root') {
      if (!value) throw new Error('--ops-root requires a path');
      options.opsRoot = path.resolve(value);
      index += 1;
    } else if (arg === '--manifest') {
      if (!value) throw new Error('--manifest requires a path');
      options.manifestPath = path.resolve(value);
      index += 1;
    } else if (arg === '--bundle') {
      if (!value) throw new Error('--bundle requires a path');
      options.bundleRoot = path.resolve(value);
      index += 1;
    } else if (arg === '--apply-shared-backend') {
      options.applySharedBackend = true;
    } else if (arg === '--apply-shared-frontend') {
      options.applySharedFrontend = true;
    } else if (arg === '--update-inheritance-docs') {
      options.updateInheritanceDocs = true;
    } else if (arg === '--check') {
      options.check = true;
    } else if (arg === '--dry-run') {
      options.dryRun = true;
    } else if (arg === '--rollback-on-error') {
      options.rollbackOnError = true;
    } else if (arg === '--allow-release-line-jump') {
      options.allowReleaseLineJump = true;
    } else if (arg === '--skip-go-validation') {
      options.skipGoValidation = true;
    } else if (arg === '--help' || arg === '-h') {
      options.help = true;
    } else {
      throw new Error(`Unknown argument: ${arg}`);
    }
  }

  return options;
}

function validateOptions(options) {
  if (options.help) {
    return;
  }
  if (!options.manifestPath) {
    throw new Error('manifest is required');
  }
  if (!options.bundleRoot) {
    throw new Error('bundle is required');
  }
  if (
    (options.applySharedBackend || options.applySharedFrontend || options.updateInheritanceDocs)
    && !options.rollbackOnError
  ) {
    throw new Error('--rollback-on-error is required whenever the foundation consumer modifies files');
  }
}

function readManifest(manifestPath) {
  if (!fs.existsSync(manifestPath)) {
    throw new Error(`manifest not found: ${manifestPath}`);
  }

  const manifest = JSON.parse(fs.readFileSync(manifestPath, 'utf8'));
  if (manifest.sourceRepo !== 'pantheon-base') {
    throw new Error('manifest sourceRepo must be pantheon-base');
  }
  if (manifest.consumerMode !== 'foundation-release-consumer') {
    throw new Error('manifest consumerMode must be foundation-release-consumer');
  }
  return manifest;
}

function replaceLine(content, patterns, nextLine) {
  for (const pattern of patterns) {
    if (pattern.test(content)) {
      return content.replace(pattern, nextLine);
    }
  }
  return `${content.trimEnd()}\n${nextLine}\n`;
}

function updateInheritanceDoc(filePath, manifest, language) {
  let content = fs.readFileSync(filePath, 'utf8');

  if (language === 'zh') {
    content = replaceLine(content, [/^- Base branch：.*$/m, /^- Base release line：.*$/m], `- Base release line：当前跟随 \`${manifest.releaseLine}\``);
    content = replaceLine(content, [/^- Base version：.*$/m], `- Base version：当前锁定到 \`${manifest.releaseVersion}\`（\`${manifest.baseCommit}\`）`);
    content = replaceLine(content, [/^- Inheritance mode：.*$/m], '- Inheritance mode：`foundation-release-consumer`');
  } else {
    content = replaceLine(content, [/^- Base branch:.*$/m, /^- Base release line:.*$/m], `- Base release line: \`${manifest.releaseLine}\``);
    content = replaceLine(content, [/^- Base version:.*$/m], `- Base version: \`${manifest.releaseVersion}\` (\`${manifest.baseCommit}\`)`);
    content = replaceLine(content, [/^- Inheritance mode:.*$/m], '- Inheritance mode: `foundation-release-consumer`');
  }

  fs.writeFileSync(filePath, content, 'utf8');
}

function updateFoundationLock(opsRoot, manifest) {
  const lockPath = path.join(opsRoot, 'foundation-release.lock.json');
  const currentLock = readFoundationLock(opsRoot);
  const nextLock = {
    ...currentLock,
    releaseLine: manifest.releaseLine,
    releaseVersion: manifest.releaseVersion,
    baseCommit: manifest.baseCommit,
    consumerMode: manifest.consumerMode,
    lockedAt: new Date().toISOString(),
    lockedBy: process.env.USER || process.env.USERNAME || 'unknown',
  };

  if (manifest.sharedPaths) {
    nextLock.sharedPaths = {
      ...currentLock.sharedPaths,
      ...manifest.sharedPaths,
    };
  }
  nextLock.releaseArtifact = {
    ...currentLock.releaseArtifact,
    ...manifest.releaseArtifact,
    localPath: `.foundation/releases/${manifest.releaseVersion}`,
  };

  fs.writeFileSync(lockPath, `${JSON.stringify(nextLock, null, 2)}\n`, 'utf8');
}

function normalizeLineEndings(source) {
  return source.replaceAll('\r\n', '\n');
}

function readUtf8(filePath) {
  return fs.readFileSync(filePath, 'utf8');
}

function writeUtf8(filePath, content) {
  ensureDir(filePath);
  fs.writeFileSync(filePath, content, 'utf8');
}

function parseReleaseLine(releaseLine, fieldName) {
  const match = /^release\/(\d+)\.(\d+)$/u.exec(releaseLine ?? '');
  if (!match) {
    throw new Error(`${fieldName} must use release/<major>.<minor> format; received ${releaseLine ?? 'missing'}`);
  }
  return { major: Number(match[1]), minor: Number(match[2]) };
}

function compareReleaseLines(left, right) {
  const leftVersion = parseReleaseLine(left, 'release line');
  const rightVersion = parseReleaseLine(right, 'release line');
  if (leftVersion.major !== rightVersion.major) {
    return leftVersion.major - rightVersion.major;
  }
  return leftVersion.minor - rightVersion.minor;
}

function validateConsumerCompatibility(opsRoot, manifest, allowReleaseLineJump) {
  const minimumCurrentRelease = manifest.consumerCompatibility?.['pantheon-ops']?.minimumCurrentRelease;
  if (!minimumCurrentRelease) {
    return null;
  }

  const currentLock = readFoundationLock(opsRoot);
  if (compareReleaseLines(currentLock.releaseLine, minimumCurrentRelease) >= 0) {
    return null;
  }

  if (allowReleaseLineJump && compareReleaseLines(manifest.releaseLine, currentLock.releaseLine) > 0) {
    return [
      `Explicit release-line jump accepted: ${currentLock.releaseLine} -> ${manifest.releaseLine}`,
      `Target requires minimum current release ${minimumCurrentRelease}`,
    ].join('; ');
  }

  throw new Error(
    [
      `foundation release ${manifest.releaseVersion} requires current release ${minimumCurrentRelease} or newer`,
      `but pantheon-ops is locked to ${currentLock.releaseLine}`,
      'Review the release transition, run a dry-run, then pass --allow-release-line-jump only for an intentional forward release-line migration.',
    ].join('; '),
  );
}

function ensureCleanWorktree(opsRoot) {
  const statusResult = spawnSync('git', ['status', '--porcelain', '--untracked-files=all'], {
    cwd: opsRoot,
    encoding: 'utf8',
  });
  if (statusResult.status !== 0) {
    return;
  }
  if (statusResult.stdout.trim()) {
    throw new Error('foundation apply requires a clean git worktree; commit or stash existing changes first');
  }
}

function createRollbackState() {
  const files = new Map();
  const directories = new Map();
  const snapshotRoot = fs.mkdtempSync(path.join(os.tmpdir(), 'pantheon-foundation-rollback-'));

  function captureFile(filePath) {
    if (files.has(filePath)) {
      return;
    }
    files.set(filePath, fs.existsSync(filePath) ? fs.readFileSync(filePath) : null);
  }

  function captureDirectory(directoryPath) {
    if (directories.has(directoryPath)) {
      return;
    }
    const backupPath = path.join(snapshotRoot, `directory-${directories.size}`);
    const exists = fs.existsSync(directoryPath);
    if (exists) {
      fs.cpSync(directoryPath, backupPath, { recursive: true });
    }
    directories.set(directoryPath, { exists, backupPath });
  }

  function restore() {
    for (const [filePath, original] of files) {
      if (original === null) {
        fs.rmSync(filePath, { force: true });
      } else {
        ensureDir(filePath);
        fs.writeFileSync(filePath, original);
      }
    }
    for (const [directoryPath, snapshot] of directories) {
      fs.rmSync(directoryPath, { recursive: true, force: true });
      if (snapshot.exists) {
        fs.cpSync(snapshot.backupPath, directoryPath, { recursive: true });
      }
    }
  }

  function cleanup() {
    fs.rmSync(snapshotRoot, { recursive: true, force: true });
  }

  return { captureFile, captureDirectory, restore, cleanup };
}

function resolveBackendModuleNames(bundleRoot, opsRoot, manifest) {
  const sharedBackendRoot = path.join(resolveBundleRoot(bundleRoot), 'shared-backend', 'backend');
  // Base owns backend/go.mod, so its imports start at the declared module name.
  // Ops keeps go.mod at the repository root and the Go packages below backend/.
  // Preserve the legacy layout when the Ops module already includes /backend.
  const opsModule = readGoModuleName(opsRoot);
  const opsModuleName = opsModule.endsWith('/backend') ? opsModule : `${opsModule}/backend`;
  const baseModuleName = manifest.baseGoModule
    || manifest.baseModule
    || detectBackendModuleNameFromTree(sharedBackendRoot);
  return { baseModuleName, opsModuleName };
}

function resolveBundleRoot(bundleRoot) {
  const nestedBundleRoot = path.join(bundleRoot, 'bundle');
  if (fs.existsSync(nestedBundleRoot)) {
    return nestedBundleRoot;
  }
  return bundleRoot;
}

function resolveReleaseRoot(bundleRoot) {
  if (fs.existsSync(path.join(bundleRoot, 'manifest.json')) && fs.existsSync(path.join(bundleRoot, 'bundle'))) {
    return bundleRoot;
  }
  const parentRoot = path.dirname(bundleRoot);
  if (path.basename(bundleRoot) === 'bundle' && fs.existsSync(path.join(parentRoot, 'manifest.json'))) {
    return parentRoot;
  }
  return null;
}

function installConsumedReleaseArtifact(bundleRoot, opsRoot, manifest, rollbackState) {
  const releaseRoot = resolveReleaseRoot(bundleRoot);
  if (!releaseRoot) {
    return;
  }

  const targetRoot = path.join(opsRoot, '.foundation', 'releases', manifest.releaseVersion);
  if (path.resolve(releaseRoot) === path.resolve(targetRoot)) {
    return;
  }
  rollbackState.captureDirectory(targetRoot);
  fs.rmSync(targetRoot, { recursive: true, force: true });
  fs.mkdirSync(path.dirname(targetRoot), { recursive: true });
  fs.cpSync(releaseRoot, targetRoot, { recursive: true });
}

function diffLines(a, b) {
  const aLines = a.split('\n');
  const bLines = b.split('\n');
  const result = [];
  const maxLines = Math.max(aLines.length, bLines.length);
  for (let i = 0; i < maxLines; i += 1) {
    const aLine = aLines[i] ?? null;
    const bLine = bLines[i] ?? null;
    if (aLine === bLine) {
      result.push(`  ${String(i + 1).padStart(3)}  ${aLine}`);
    } else {
      if (aLine !== null) result.push(`- ${String(i + 1).padStart(3)} ${aLine}`);
      if (bLine !== null) result.push(`+ ${String(i + 1).padStart(3)} ${bLine}`);
    }
  }
  return result;
}

function computeBackendChange(relativePath, sourceRoot, targetRoot, baseModuleName, opsModuleName) {
  const sourcePath = path.join(sourceRoot, relativePath);
  const targetPath = path.join(targetRoot, relativePath);
  const source = readUtf8(sourcePath);
  let nextSource = rewriteBackendModuleReferences(source, baseModuleName, opsModuleName);
  if (backendMergedJsonPaths.has(relativePath) && fs.existsSync(targetPath)) {
    nextSource = mergeBuiltinLocaleResources(nextSource, readUtf8(targetPath));
  }
  const targetExists = fs.existsSync(targetPath);
  const targetContent = targetExists ? readUtf8(targetPath) : null;
  const normalizedNext = normalizeLineEndings(nextSource);
  const normalizedTarget = targetExists ? normalizeLineEndings(targetContent) : null;
  if (!targetExists || normalizedNext !== normalizedTarget) {
    return {
      action: targetExists ? 'REWRITE' : 'CREATE',
      path: relativePath,
      targetPath,
      newContent: nextSource,
      oldContent: targetContent,
    };
  }
  return null;
}

function computeFrontendChange(relativePath, sourceRoot, targetRoot) {
  const targetRelativePath = toRelocatedFrontendPath(relativePath);
  const sourcePath = path.join(sourceRoot, relativePath);
  const targetPath = path.join(targetRoot, targetRelativePath);
  const nextSource = rewriteFrontendBaseSource(readUtf8(sourcePath), relativePath, targetRelativePath);
  const targetExists = fs.existsSync(targetPath);
  const targetContent = targetExists ? readUtf8(targetPath) : null;
  const normalizedNext = normalizeLineEndings(nextSource);
  const normalizedTarget = targetExists ? normalizeLineEndings(targetContent) : null;
  if (!targetExists || normalizedNext !== normalizedTarget) {
    return {
      action: targetExists ? 'REWRITE' : 'CREATE',
      path: targetRelativePath,
      targetPath,
      newContent: nextSource,
      oldContent: targetContent,
    };
  }
  return null;
}

function applySharedBackendBundle(bundleRoot, opsRoot, manifest, dryRun = false, rollbackState = null) {
  const sourceRoot = path.join(resolveBundleRoot(bundleRoot), 'shared-backend', 'backend');
  if (!fs.existsSync(sourceRoot)) {
    return { skipped: 0, applied: 0, dryRun };
  }

  const targetRoot = path.join(opsRoot, 'backend');
  const { baseModuleName, opsModuleName } = resolveBackendModuleNames(bundleRoot, opsRoot, manifest);
  const changes = [];

  for (const relativePath of collectFiles(sourceRoot)) {
    if (isBackendOverlayPath(relativePath) || isBackendBusinessPath(relativePath)) {
      continue;
    }

    const change = computeBackendChange(relativePath, sourceRoot, targetRoot, baseModuleName, opsModuleName);
    if (!change) {
      continue;
    }

    if (dryRun) {
      changes.push(change);
    } else {
      rollbackState?.captureFile(change.targetPath);
      writeUtf8(change.targetPath, change.newContent);
    }
  }

  if (dryRun) {
    return { skipped: 0, applied: changes.length, changes, dryRun };
  }
  return { skipped: 0, applied: collectFiles(sourceRoot).length, dryRun };
}

function applySharedFrontendBundle(bundleRoot, opsRoot, dryRun = false, rollbackState = null) {
  const sourceRoot = path.join(resolveBundleRoot(bundleRoot), 'shared-frontend', 'frontend', 'src');
  if (!fs.existsSync(sourceRoot)) {
    return { skipped: 0, applied: 0, dryRun };
  }

  const targetRoot = path.join(opsRoot, 'frontend', 'src');
  const changes = [];

  for (const relativePath of collectFiles(sourceRoot)) {
    const targetRelativePath = toRelocatedFrontendPath(relativePath);
    if (isFrontendOverlayPath(targetRelativePath) || isFrontendBusinessPath(targetRelativePath)) {
      continue;
    }
    const change = computeFrontendChange(relativePath, sourceRoot, targetRoot);
    if (!change) {
      continue;
    }

    if (dryRun) {
      changes.push(change);
    } else {
      rollbackState?.captureFile(change.targetPath);
      writeUtf8(change.targetPath, change.newContent);
    }
  }

  if (dryRun) {
    return { skipped: 0, applied: changes.length, changes, dryRun };
  }
  return { skipped: 0, applied: collectFiles(sourceRoot).length, dryRun };
}

function runCheckScript(opsRoot, scriptName) {
  const scriptPath = path.join(opsRoot, 'scripts', scriptName);
  if (!fs.existsSync(scriptPath)) {
    throw new Error(`required check script is missing: ${scriptName}`);
  }

  const result = spawnSync(process.execPath, [scriptPath], {
    cwd: opsRoot,
    encoding: 'utf8',
  });

  if (result.status !== 0) {
    throw new Error(result.stderr || result.stdout || `${scriptName} failed`);
  }

  return result.stdout.trim();
}

function runNodeScript(opsRoot, scriptRelativePath) {
  const scriptPath = path.join(opsRoot, scriptRelativePath);
  if (!fs.existsSync(scriptPath)) {
    throw new Error(`required check script is missing: ${scriptRelativePath}`);
  }

  const result = spawnSync(process.execPath, [scriptPath, '--check'], {
    cwd: opsRoot,
    encoding: 'utf8',
  });

  if (result.status !== 0) {
    throw new Error(result.stderr || result.stdout || `${scriptRelativePath} failed`);
  }

  return result.stdout.trim();
}

function validateGoBuild(opsRoot) {
  const backendRoot = path.join(opsRoot, 'backend');

  // go vet is sufficient to catch syntax errors and import resolution problems
  // without requiring the full transitive dependency graph.
  const vetResult = spawnSync('go', ['vet', './...'], {
    cwd: backendRoot,
    encoding: 'utf8',
    env: { ...process.env, CGO_ENABLED: '0' },
  });

  if (vetResult.status === 0) {
    return;
  }

  throw new Error(`go vet failed after module rewrites:\n${vetResult.stderr || vetResult.stdout}`);
}

export function consumeFoundationRelease(options) {
  validateOptions(options);

  const manifest = readManifest(options.manifestPath);
  const compatibilitySummary = validateConsumerCompatibility(
    options.opsRoot,
    manifest,
    options.allowReleaseLineJump,
  );
  const summary = [
    `Target foundation release: ${manifest.releaseVersion}`,
    `Release line: ${manifest.releaseLine}`,
  ];
  if (compatibilitySummary) {
    summary.push(compatibilitySummary);
  }

  if (options.dryRun) {
    summary.push('DRY RUN — no files will be modified');
    const backendDryRun = applySharedBackendBundle(options.bundleRoot, options.opsRoot, manifest, true);
    const frontendDryRun = applySharedFrontendBundle(options.bundleRoot, options.opsRoot, true);
    const allChanges = [
      ...(backendDryRun.changes || []),
      ...(frontendDryRun.changes || []),
    ];
    if (allChanges.length === 0) {
      console.log('No changes needed — shared paths are already aligned.');
    } else {
      console.log(`Foundation upgrade dry-run: ${allChanges.length} file(s) would change\n`);
      for (const change of allChanges) {
        console.log(`  ${change.action} ${change.path}`);
        if (change.oldContent !== null) {
          const lines = diffLines(change.oldContent, change.newContent);
          const previewLines = lines.slice(0, 12);
          for (const line of previewLines) {
            console.log(line);
          }
          if (lines.length > 12) {
            console.log(`  ... (${lines.length - 12} more lines)`);
          }
        }
        console.log();
      }
    }
    return { manifest, summary, dryRun: true, changes: allChanges };
  }

  const writesFiles = options.applySharedBackend || options.applySharedFrontend || options.updateInheritanceDocs;
  if (writesFiles) {
    ensureCleanWorktree(options.opsRoot);
  }
  const rollbackState = createRollbackState();

  try {
    if (options.updateInheritanceDocs) {
      const zhDocPath = path.join(options.opsRoot, 'docs', 'PROJECT_INHERITANCE.md');
      const enDocPath = path.join(options.opsRoot, 'docs', 'PROJECT_INHERITANCE.en.md');
      const lockPath = path.join(options.opsRoot, 'foundation-release.lock.json');
      rollbackState.captureFile(zhDocPath);
      rollbackState.captureFile(enDocPath);
      rollbackState.captureFile(lockPath);
      installConsumedReleaseArtifact(options.bundleRoot, options.opsRoot, manifest, rollbackState);
      updateInheritanceDoc(zhDocPath, manifest, 'zh');
      updateInheritanceDoc(enDocPath, manifest, 'en');
      updateFoundationLock(options.opsRoot, manifest);
      summary.push('Updated inheritance docs');
      summary.push('Updated foundation-release.lock.json');
    }

    if (options.applySharedBackend) {
      const backendResult = applySharedBackendBundle(options.bundleRoot, options.opsRoot, manifest, false, rollbackState);
      summary.push(`Applied shared-backend bundle (${backendResult.applied} files)`);

      if (!options.skipGoValidation) {
        summary.push('Running go vet to validate backend module rewrites...');
        validateGoBuild(options.opsRoot);
        summary.push('go vet passed');
      } else {
        summary.push('Skipped go validation (--skip-go-validation)');
      }
    }

    if (options.applySharedFrontend) {
      const frontendResult = applySharedFrontendBundle(options.bundleRoot, options.opsRoot, false, rollbackState);
      summary.push(`Applied shared-frontend bundle (${frontendResult.applied} files)`);
    }

    if (options.check) {
      summary.push('Running check-inheritance-contract.mjs');
      runCheckScript(options.opsRoot, 'check-inheritance-contract.mjs');
      summary.push('Running check-base-backend-sync.mjs');
      runCheckScript(options.opsRoot, 'check-base-backend-sync.mjs');
      summary.push('Running frontend/scripts/sync-base-shared.mjs --check');
      runNodeScript(options.opsRoot, path.join('frontend', 'scripts', 'sync-base-shared.mjs'));
      summary.push('Running frontend/scripts/check-menu-contract.mjs --check');
      runNodeScript(options.opsRoot, path.join('frontend', 'scripts', 'check-menu-contract.mjs'));
    }
  } catch (error) {
    rollbackState.restore();
    console.error(`Error during apply: ${error.message}`);
    console.error('Rolled back files changed by this foundation apply.');
    throw error;
  } finally {
    rollbackState.cleanup();
  }

  return { manifest, summary };
}

function printHelp() {
  console.log(`Usage:
  node scripts/foundation-release/consume-foundation-release.mjs --manifest <path> --bundle <path> [options]

Options:
  --ops-root <path>
  --apply-shared-backend
  --apply-shared-frontend
  --update-inheritance-docs
  --check
  --dry-run           preview what would change without modifying files
  --rollback-on-error required for write operations; restores touched files on failure
  --allow-release-line-jump explicitly acknowledge an intentional forward release-line migration`);
}

function main() {
  let options;

  try {
    options = parseArgs(process.argv.slice(2));
    if (options.help) {
      printHelp();
      return 0;
    }

    const result = consumeFoundationRelease(options);
    console.log(result.summary.join('\n'));
    if (result.dryRun) {
      return 0;
    }
    return 0;
  } catch (error) {
    console.error(error.message);
    return 1;
  }
}

if (process.argv[1] && fileURLToPath(import.meta.url) === path.resolve(process.argv[1])) {
  process.exitCode = main();
}

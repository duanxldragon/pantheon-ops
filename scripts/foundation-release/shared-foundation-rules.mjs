import fs from 'node:fs';
import { spawnSync } from 'node:child_process';
import path from 'node:path';

export const sharedBackendEntries = ['cmd', 'internal', 'modules', 'pkg'];

export const sharedFrontendEntries = [
  'components',
  'core',
  'store',
  'modules/auth',
  'modules/dashboard',
  'modules/system',
  'index.css',
];

export const backendOverlayPaths = new Set([
  'internal/scaffold/workspace.go',
  'internal/scaffold/workspace_test.go',
  'modules/business/business.go',
  'modules/business/generated_registry.go',
  'modules/business/retired_modules.go',
  'modules/platform/health.go',
  'modules/system/seed.go',
  'modules/system/seed_test.go',
  'modules/lowcode/dynamicmodule/dynamic_module_service_test.go',
  'modules/lowcode/generator/generator_service_test.go',
  'modules/system/iam/menu/component_registry.go',
  'modules/system/iam/menu/generated_component_registry.go',
]);

export const backendOverlayDirPrefixes = ['cmd/server/uploads/'];

export const frontendOverlayPaths = new Set([
  'core/router/componentRegistry.ts',
  'core/router/generatedComponentRegistry.ts',
  'core/router/modules.ts',
  'modules/lowcode/generator/backend-generator.ts',
]);

export const frontendRelocatedPathPrefixes = new Map([
  ['modules/system/dict/', 'modules/system/config/dict/'],
  ['modules/system/setting/', 'modules/system/config/setting/'],
  ['modules/system/menu/', 'modules/system/iam/menu/'],
  ['modules/system/permission/', 'modules/system/iam/permission/'],
  ['modules/system/role/', 'modules/system/iam/role/'],
  ['modules/system/user/', 'modules/system/iam/user/'],
  ['modules/system/dept/', 'modules/system/org/dept/'],
  ['modules/system/post/', 'modules/system/org/post/'],
]);

export const frontendRelocatedComponentPrefixes = new Map([
  ['system/dict/', 'system/config/dict/'],
  ['system/setting/', 'system/config/setting/'],
  ['system/menu/', 'system/iam/menu/'],
  ['system/permission/', 'system/iam/permission/'],
  ['system/role/', 'system/iam/role/'],
  ['system/user/', 'system/iam/user/'],
  ['system/dept/', 'system/org/dept/'],
  ['system/post/', 'system/org/post/'],
]);

export const allowedBackendOpsOnlyPaths = new Set([
  'modules/system/i18n/builtin_locale_resources_test.go',
  'pkg/contracts/permission_policies.go',
  'pkg/contracts/permission_policies_test.go',
]);

export const allowedFrontendOpsOnlyPaths = new Set([]);

export const backendMergedJsonPaths = new Set([
  'modules/system/i18n/builtin_locale_resources.json',
]);

const FOUNDATION_RELEASE_ENV = 'PANTHEON_FOUNDATION_RELEASE_ROOT';

export function toRepoPath(filePath) {
  return filePath.split(path.sep).join('/');
}

export function collectFiles(rootPath, currentPath = rootPath, bucket = []) {
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

export function ensureDir(filePath) {
  fs.mkdirSync(path.dirname(filePath), { recursive: true });
}

export function normalizeLineEndings(source) {
  return source.replaceAll('\r\n', '\n');
}

export function readFoundationLock(opsRoot) {
  const lockPath = path.join(opsRoot, 'foundation-release.lock.json');
  if (!fs.existsSync(lockPath)) {
    throw new Error(`foundation release lock not found: ${lockPath}`);
  }

  const lock = JSON.parse(fs.readFileSync(lockPath, 'utf8'));
  if (lock.consumerMode !== 'foundation-release-consumer') {
    throw new Error(`unsupported foundation consumer mode: ${lock.consumerMode ?? 'missing'}`);
  }
  if (!lock.baseCommit) {
    throw new Error(`foundation release lock is missing baseCommit: ${lockPath}`);
  }
  if (!lock.releaseVersion) {
    throw new Error(`foundation release lock is missing releaseVersion: ${lockPath}`);
  }
  return lock;
}

export function sharedBackendEntriesFromLock(lock) {
  const entries = lock.sharedPaths?.backend;
  if (!Array.isArray(entries) || entries.length === 0) {
    return [...sharedBackendEntries];
  }

  return entries
    .filter((entry) => typeof entry === 'string' && entry.startsWith('backend/'))
    .map((entry) => stripTreePrefix(entry, 'backend'));
}

export function sharedFrontendEntriesFromLock(lock) {
  const entries = lock.sharedPaths?.frontend;
  if (!Array.isArray(entries) || entries.length === 0) {
    return [...sharedFrontendEntries];
  }

  return entries
    .filter((entry) => typeof entry === 'string' && entry.startsWith('frontend/src/'))
    .map((entry) => stripTreePrefix(entry, 'frontend/src'));
}

export function resolveBaseRepoRoot(opsRoot, lock = readFoundationLock(opsRoot)) {
  if (process.env.PANTHEON_BASE_REPO_ROOT) {
    return path.resolve(process.env.PANTHEON_BASE_REPO_ROOT);
  }

  return path.resolve(opsRoot, lock.baseRepo ?? '../pantheon-base');
}

function isFoundationReleaseRoot(candidatePath) {
  return fs.existsSync(path.join(candidatePath, 'manifest.json'))
    && fs.existsSync(path.join(candidatePath, 'bundle'));
}

export function foundationReleaseCandidates(opsRoot, lock = readFoundationLock(opsRoot)) {
  const candidates = [];
  if (process.env[FOUNDATION_RELEASE_ENV]) {
    candidates.push(path.resolve(process.env[FOUNDATION_RELEASE_ENV]));
  }
  if (lock.releaseArtifact?.localPath) {
    candidates.push(path.resolve(opsRoot, lock.releaseArtifact.localPath));
  }
  candidates.push(path.resolve(opsRoot, '.foundation', 'releases', lock.releaseVersion));
  candidates.push(path.resolve(opsRoot, 'dist', 'foundation-releases', lock.releaseVersion));

  const baseRepoRoot = resolveBaseRepoRoot(opsRoot, lock);
  candidates.push(path.resolve(baseRepoRoot, 'dist', 'foundation-releases', lock.releaseVersion));

  return [...new Set(candidates)];
}

export function resolveFoundationReleaseRoot(opsRoot, lock = readFoundationLock(opsRoot)) {
  for (const candidatePath of foundationReleaseCandidates(opsRoot, lock)) {
    if (isFoundationReleaseRoot(candidatePath)) {
      return candidatePath;
    }
  }

  throw new Error(
    [
      `foundation release artifact not found for ${lock.releaseVersion}`,
      `Set ${FOUNDATION_RELEASE_ENV} to a directory containing manifest.json and bundle/,`,
      'or download/build the locked pantheon-base release artifact before running base-sync checks.',
      'Use --workspace-head only for explicit local preview checks.',
    ].join(' '),
  );
}

export function readFoundationReleaseManifest(releaseRoot, lock) {
  const manifestPath = path.join(releaseRoot, 'manifest.json');
  const manifest = JSON.parse(fs.readFileSync(manifestPath, 'utf8'));
  if (manifest.sourceRepo !== 'pantheon-base') {
    throw new Error(`foundation release manifest sourceRepo must be pantheon-base: ${manifestPath}`);
  }
  if (manifest.consumerMode !== 'foundation-release-consumer') {
    throw new Error(`foundation release manifest has unsupported consumerMode: ${manifestPath}`);
  }
  if (manifest.releaseVersion !== lock.releaseVersion) {
    throw new Error(
      `foundation release version mismatch: lock=${lock.releaseVersion} manifest=${manifest.releaseVersion}`,
    );
  }
  if (manifest.baseCommit !== lock.baseCommit) {
    throw new Error(`foundation release commit mismatch: lock=${lock.baseCommit} manifest=${manifest.baseCommit}`);
  }
  return manifest;
}

export function resolveFoundationReleasePaths(opsRoot, lock = readFoundationLock(opsRoot)) {
  const releaseRoot = resolveFoundationReleaseRoot(opsRoot, lock);
  const manifest = readFoundationReleaseManifest(releaseRoot, lock);
  return {
    releaseRoot,
    manifest,
    bundleRoot: path.join(releaseRoot, 'bundle'),
    sharedBackendRoot: path.join(releaseRoot, 'bundle', 'shared-backend', 'backend'),
    sharedFrontendRoot: path.join(releaseRoot, 'bundle', 'shared-frontend', 'frontend', 'src'),
  };
}

export function isBackendOverlayPath(relativePath) {
  return backendOverlayPaths.has(relativePath)
    || backendOverlayDirPrefixes.some((prefix) => relativePath.startsWith(prefix));
}

export function isFrontendOverlayPath(relativePath) {
  return frontendOverlayPaths.has(relativePath);
}

export function toRelocatedFrontendPath(relativePath) {
  for (const [fromPrefix, toPrefix] of frontendRelocatedPathPrefixes.entries()) {
    if (relativePath.startsWith(fromPrefix)) {
      return `${toPrefix}${relativePath.slice(fromPrefix.length)}`;
    }
  }
  return relativePath;
}

export function toOriginalFrontendPath(relativePath) {
  for (const [fromPrefix, toPrefix] of frontendRelocatedPathPrefixes.entries()) {
    if (relativePath.startsWith(toPrefix)) {
      return `${fromPrefix}${relativePath.slice(toPrefix.length)}`;
    }
  }
  return relativePath;
}

function toRelativeImportSpecifier(fromRelativePath, toRelativePath) {
  const fromDir = path.posix.dirname(toRepoPath(fromRelativePath));
  const nextSpecifier = path.posix.relative(fromDir, toRepoPath(toRelativePath));
  if (nextSpecifier.startsWith('../')) {
    return nextSpecifier;
  }
  return `./${nextSpecifier}`;
}

function rewriteRelativeImportSpecifier(specifier, originalRelativePath, relocatedRelativePath) {
  if (!specifier.startsWith('./') && !specifier.startsWith('../')) {
    return specifier;
  }

  const originalDir = path.posix.dirname(toRepoPath(originalRelativePath));
  const originalTarget = path.posix.normalize(path.posix.join(originalDir, specifier));
  const relocatedTarget = toRelocatedFrontendPath(originalTarget);
  const nextSpecifier = toRelativeImportSpecifier(relocatedRelativePath, relocatedTarget);
  return nextSpecifier === '.' ? './' : nextSpecifier;
}

export function rewriteFrontendBaseSource(baseSource, originalRelativePath, relocatedRelativePath) {
  let nextSource = baseSource.replace(
    /\b(from\s*|import\s*\(\s*|import\s+)(['"])(\.{1,2}\/[^'"]+)\2/gu,
    (match, prefix, quote, specifier) =>
      `${prefix}${quote}${rewriteRelativeImportSpecifier(
        specifier,
        originalRelativePath,
        relocatedRelativePath,
      )}${quote}`,
  );
  nextSource = nextSource.replace(
    /(\bcomponent(?:Key)?\s*:\s*['"])([^'"]+)(['"])/gu,
    (match, prefix, componentKey, suffix) => {
      for (const [fromPrefix, toPrefix] of frontendRelocatedComponentPrefixes.entries()) {
        if (componentKey.startsWith(fromPrefix)) {
          return `${prefix}${toPrefix}${componentKey.slice(fromPrefix.length)}${suffix}`;
        }
      }
      return match;
    },
  );
  return nextSource;
}

export function readGoModuleName(repoRoot) {
  const goModPath = path.join(repoRoot, 'go.mod');
  if (!fs.existsSync(goModPath)) {
    throw new Error(`go.mod not found: ${goModPath}`);
  }

  const goModSource = fs.readFileSync(goModPath, 'utf8');
  const match = goModSource.match(/^module\s+(\S+)\s*$/m);
  if (!match) {
    throw new Error(`failed to read Go module name from ${goModPath}`);
  }
  return match[1];
}

export function detectBackendModuleNameFromTree(rootPath) {
  if (!fs.existsSync(rootPath)) {
    throw new Error(`backend tree not found: ${rootPath}`);
  }

  for (const relativePath of collectFiles(rootPath)) {
    if (!relativePath.endsWith('.go')) {
      continue;
    }
    const source = fs.readFileSync(path.join(rootPath, relativePath), 'utf8');
    const match = source.match(/"([^"\s]+)\/backend(?:\/[^"\s]*)?"/);
    if (match) {
      return match[1];
    }
  }

  throw new Error(`failed to infer backend module name from ${rootPath}`);
}

export function rewriteBackendModuleReferences(source, fromModuleName, toModuleName) {
  return source.replaceAll(`${fromModuleName}/backend`, `${toModuleName}/backend`);
}

export function mergeBuiltinLocaleResources(baseSource, opsSource) {
  const nextLocales = JSON.parse(baseSource);
  const currentLocales = JSON.parse(opsSource);

  for (const [locale, currentPack] of Object.entries(currentLocales)) {
    if (!currentPack || typeof currentPack !== 'object' || Array.isArray(currentPack)) {
      continue;
    }

    if (!nextLocales[locale] || typeof nextLocales[locale] !== 'object' || Array.isArray(nextLocales[locale])) {
      nextLocales[locale] = {};
    }
    const nextPack = nextLocales[locale];

    for (const [key, value] of Object.entries(currentPack)) {
      if (!key.startsWith('business.')) {
        continue;
      }
      nextPack[key] = value;
    }
  }

  return `${JSON.stringify(nextLocales, null, 2)}\n`;
}

export function listFilesFromGitCommit(repoRoot, commit, relativePath) {
  if (commit === 'HEAD' && !fs.existsSync(path.join(repoRoot, '.git'))) {
    const absolutePath = path.join(repoRoot, relativePath);
    if (!fs.existsSync(absolutePath)) {
      return [];
    }
    const stats = fs.statSync(absolutePath);
    if (!stats.isDirectory()) {
      return [toRepoPath(relativePath)];
    }
    return collectFiles(repoRoot, absolutePath);
  }

  const result = spawnGit(repoRoot, ['ls-tree', '-r', '--name-only', commit, '--', relativePath]);
  if (result.status !== 0) {
    throw new Error(
      result.stderr?.trim()
        || `failed to list ${relativePath} from ${commit} in ${repoRoot}`,
    );
  }

  return result.stdout
    .split(/\r?\n/u)
    .map((line) => line.trim())
    .filter(Boolean)
    .map((line) => toRepoPath(line));
}

export function readFileFromGitCommit(repoRoot, commit, relativePath) {
  if (commit === 'HEAD' && !fs.existsSync(path.join(repoRoot, '.git'))) {
    return fs.readFileSync(path.join(repoRoot, relativePath), 'utf8');
  }

  const result = spawnGit(repoRoot, ['show', `${commit}:${relativePath}`]);
  if (result.status !== 0) {
    throw new Error(
      result.stderr?.trim()
        || `failed to read ${relativePath} from ${commit} in ${repoRoot}`,
    );
  }
  return result.stdout;
}

export function stripTreePrefix(filePath, treePrefix) {
  const normalizedFilePath = toRepoPath(filePath);
  const normalizedTreePrefix = toRepoPath(treePrefix).replace(/\/+$/u, '');
  const prefix = `${normalizedTreePrefix}/`;
  if (!normalizedFilePath.startsWith(prefix)) {
    return normalizedFilePath;
  }
  return normalizedFilePath.slice(prefix.length);
}

function spawnGit(repoRoot, args) {
  return spawnSync('git', ['-C', repoRoot, ...args], {
    encoding: 'utf8',
  });
}

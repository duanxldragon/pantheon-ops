import crypto from 'node:crypto';
import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';
import { spawnSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';

const currentFile = fileURLToPath(import.meta.url);
const defaultOpsRoot = path.resolve(path.dirname(currentFile), '..', '..');

function repoPath(value) {
  return value.replaceAll('\\', '/').replace(/^\.\//u, '').replace(/\/$/u, '');
}

function isWithin(candidate, parent) {
  return candidate === parent || candidate.startsWith(`${parent}/`);
}

function runGit(root, args) {
  const result = spawnSync('git', args, { cwd: root, encoding: 'utf8' });
  if (result.status !== 0) {
    throw new Error(result.stderr.trim() || `git ${args.join(' ')} failed`);
  }
  return result.stdout.trim();
}

function trackedFiles(root) {
  return runGit(root, ['ls-files', '--cached', '--others', '--exclude-standard'])
    .split(/\r?\n/u)
    .filter(Boolean)
    .map(repoPath);
}

function snapshotFiles(root) {
  const results = [];
  const visit = (directory) => {
    for (const entry of fs.readdirSync(directory, { withFileTypes: true })) {
      if (entry.name === '.git') continue;
      const absolute = path.join(directory, entry.name);
      if (entry.isDirectory()) visit(absolute);
      else if (entry.isFile()) results.push(repoPath(path.relative(root, absolute)));
    }
  };
  visit(root);
  return results.sort();
}

function sha256FileHex(filePath) {
  return crypto.createHash('sha256').update(fs.readFileSync(filePath)).digest('hex');
}

function readLock(opsRoot) {
  const lockPath = path.join(opsRoot, 'foundation-release.lock.json');
  if (!fs.existsSync(lockPath)) return null;
  return JSON.parse(fs.readFileSync(lockPath, 'utf8'));
}

// Ops consumes Base only from the published GitHub release, never from a sibling
// ../pantheon-base working tree. When the snapshot is not already cached under
// releaseArtifact.localPath, download it from the locked GitHub release and rely
// on the lock's repoSnapshot.sha256 for integrity.
function downloadRepoSnapshot(lock, releaseDir) {
  const repo = lock.releaseArtifact?.githubRepo;
  const tag = lock.releaseVersion;
  const assetName = lock.repoSnapshot?.assetName ?? 'repo.tar';
  if (!repo || !tag) {
    throw new Error(
      'Lock is missing releaseArtifact.githubRepo or releaseVersion; ' +
      'cannot download the Base snapshot from GitHub.',
    );
  }
  fs.mkdirSync(releaseDir, { recursive: true });
  const result = spawnSync(
    'gh',
    ['release', 'download', tag, '--repo', repo, '--pattern', assetName, '--dir', releaseDir],
    { encoding: 'utf8' },
  );
  if (result.status !== 0) {
    throw new Error(
      result.stderr.trim() || `failed to download ${assetName} from GitHub release ${repo}@${tag}`,
    );
  }
  return path.join(releaseDir, assetName);
}

function resolveBaseFromLock(opsRoot) {
  const lock = readLock(opsRoot);
  if (!lock?.repoSnapshot || !lock?.releaseArtifact?.localPath) return null;
  const releaseDir = path.resolve(opsRoot, lock.releaseArtifact.localPath);
  const repoDir = path.join(releaseDir, 'repo');
  let repoTar = path.join(releaseDir, lock.repoSnapshot.assetName ?? 'repo.tar');
  if (!fs.existsSync(repoTar)) {
    repoTar = downloadRepoSnapshot(lock, releaseDir);
  }
  if (!fs.existsSync(repoDir)) {
    fs.mkdirSync(repoDir, { recursive: true });
    // GNU tar treats a drive-letter path (`D:\...`) as the `host:file` remote-archive
    // syntax. Run with relative paths under `cwd` to avoid the colon entirely, so this
    // works on both GNU tar (Git Bash) and BSD tar (Windows built-in).
    const result = spawnSync('tar', ['-xf', path.basename(repoTar), '-C', path.basename(repoDir)], {
      encoding: 'utf8',
      cwd: releaseDir,
    });
    if (result.status !== 0) {
      throw new Error(result.stderr.trim() || `failed to extract ${repoTar}`);
    }
  }
  if (lock.repoSnapshot.sha256) {
    const actual = sha256FileHex(repoTar);
    if (actual !== lock.repoSnapshot.sha256) {
      throw new Error(`repo.tar sha256 mismatch: expected ${lock.repoSnapshot.sha256}, got ${actual}`);
    }
  }
  if (lock.repoSnapshot.baseCommit && lock.repoSnapshot.baseCommit !== lock.baseCommit) {
    throw new Error(`repoSnapshot.baseCommit ${lock.repoSnapshot.baseCommit} != lock.baseCommit ${lock.baseCommit}`);
  }
  return { baseRoot: repoDir, baseCommit: lock.baseCommit };
}

function copyFile(sourceRoot, targetRoot, relativePath) {
  const source = path.join(sourceRoot, relativePath);
  const target = path.join(targetRoot, relativePath);
  fs.mkdirSync(path.dirname(target), { recursive: true });
  fs.copyFileSync(source, target);
}

function copyDeclaredPath(sourceRoot, targetRoot, declaredPath, sourceTracked) {
  const normalized = repoPath(declaredPath);
  const files = sourceTracked.filter((file) => isWithin(file, normalized));
  if (files.length === 0 && !fs.existsSync(path.join(sourceRoot, normalized))) {
    throw new Error(`Declared overlay path does not exist: ${normalized}`);
  }
  for (const file of files) copyFile(sourceRoot, targetRoot, file);
  return files;
}

// Business overlay 不得覆盖 Base 的 hook/源文件（如 backend/modules/business/{business.go,
// retired_modules.go, generated_registry.go}）。若 businessPaths 声明了与 Base 同名且内容
// 不同的文件，说明把 Base 的 hook 层当作业务文件重写了 —— 改为用独立 overlay 文件注入
// （见 retired_modules_overlay.go 的 init() 追加模式）。repositoryOverlayPaths 的覆盖（如
// AGENTS.md/CLAUDE.md）是刻意的仓库级替换，不受此约束。
function assertBusinessPathsDoNotOverwriteBase(opsRoot, baseRoot, baseTracked, businessPaths) {
  const baseFiles = new Set(baseTracked);
  const conflicts = [];
  for (const declaredPath of businessPaths) {
    const normalized = repoPath(declaredPath);
    const source = path.join(opsRoot, normalized);
    if (!fs.existsSync(source)) continue;
    const files = [];
    const stat = fs.statSync(source);
    const visit = (directory) => {
      for (const entry of fs.readdirSync(directory, { withFileTypes: true })) {
        if (entry.name === '.git') continue;
        const absolute = path.join(directory, entry.name);
        if (entry.isDirectory()) visit(absolute);
        else files.push(repoPath(path.relative(opsRoot, absolute)));
      }
    };
    if (stat.isDirectory()) visit(source);
    else files.push(normalized);
    for (const file of files) {
      if (!baseFiles.has(file)) continue;
      const baseContent = fs.readFileSync(path.join(baseRoot, file), 'utf8');
      const opsContent = fs.readFileSync(path.join(opsRoot, file), 'utf8');
      if (baseContent !== opsContent) conflicts.push(file);
    }
  }
  if (conflicts.length > 0) {
    throw new Error(
      `Business overlay overwrites Base files: ${conflicts.join(', ')}. ` +
      'Remove them from businessPaths and inject data via a dedicated overlay file instead.',
    );
  }
}

function assertSafeManifest(manifest) {
  if (manifest.schemaVersion !== 1) throw new Error('Unsupported business overlay schema');
  const productRoots = ['backend', 'frontend/src', 'frontend/tests/smoke', 'database', 'schema'];
  const allowedBusinessRoots = [
    'backend/modules/business',
    'frontend/src/modules/business',
    'frontend/tests/smoke/business',
  ];
  for (const declaredPath of manifest.businessPaths ?? []) {
    const normalized = repoPath(declaredPath);
    if (productRoots.some((root) => isWithin(normalized, root)) &&
        !allowedBusinessRoots.some((root) => isWithin(normalized, root))) {
      throw new Error(`Business overlay cannot own generic product path: ${normalized}`);
    }
  }
  for (const declaredPath of manifest.repositoryOverlayPaths ?? []) {
    const normalized = repoPath(declaredPath);
    if (productRoots.some((root) => isWithin(normalized, root))) {
      throw new Error(`Repository overlay cannot own product path: ${normalized}`);
    }
  }
}

function pruneBaseRepositoryOverlay(targetRoot, manifest) {
  for (const entry of [...(manifest.replaceRoots ?? []), ...(manifest.baseExcludedPaths ?? [])]) {
    const normalized = repoPath(entry);
    if (!normalized || normalized === '.' || normalized.startsWith('../')) {
      throw new Error(`Unsafe Base exclusion path: ${entry}`);
    }
    fs.rmSync(path.join(targetRoot, normalized), { recursive: true, force: true });
  }
}

function rewriteBusinessGoImports(targetRoot, sourceModule, targetModule) {
  const backendRoot = path.join(targetRoot, 'backend', 'modules', 'business');
  if (!fs.existsSync(backendRoot)) return [];
  const changed = [];
  const visit = (directory) => {
    for (const entry of fs.readdirSync(directory, { withFileTypes: true })) {
      const absolute = path.join(directory, entry.name);
      if (entry.isDirectory()) visit(absolute);
      else if (entry.name.endsWith('.go')) {
        const before = fs.readFileSync(absolute, 'utf8');
        const after = before.replaceAll(`${sourceModule}/`, `${targetModule}/`);
        if (after !== before) {
          fs.writeFileSync(absolute, after, 'utf8');
          changed.push(repoPath(path.relative(targetRoot, absolute)));
        }
      }
    }
  };
  visit(backendRoot);
  return changed.sort();
}

function generateRegistries(targetRoot, manifest) {
  const imports = manifest.backendModules
    .map((entry) => `\t${entry.alias} "${entry.importPath}"`)
    .join('\n');
  const calls = manifest.backendModules.map((entry) => `\t${entry.alias}.${entry.init}(r, db)`).join('\n');
  const backend = `package business\n\nimport (\n${imports}\n\n\t"github.com/gin-gonic/gin"\n\t"gorm.io/gorm"\n)\n\nfunc initOverlayBusinessModules(r *gin.RouterGroup, db *gorm.DB) {\n${calls}\n}\n`;

  const frontendImports = manifest.frontendModules
    .map((entry) => `import { ${entry.export} } from '${entry.importPath}';`)
    .join('\n');
  const frontendEntries = manifest.frontendModules.map((entry) => `  ${entry.export},`).join('\n');
  const frontendModules = `import type { ModuleConfig } from '../core/router/types';\n${frontendImports}\n\nexport const overlayBusinessModules: ModuleConfig[] = [\n${frontendEntries}\n];\n`;

  const componentEntries = manifest.components
    .map((entry) => `  '${entry.key}': defineRegistryEntry(() => import('${entry.importPath}')),`)
    .join('\n');
  const components = `import { lazy, type LazyExoticComponent, type ComponentType } from 'react';\n\ntype ComponentLoader = () => Promise<{ default: ComponentType }>;\n\ninterface RegistryEntry {\n  component: LazyExoticComponent<ComponentType>;\n  preload: ComponentLoader;\n}\n\nfunction defineRegistryEntry(loader: ComponentLoader): RegistryEntry {\n  return { component: lazy(loader), preload: loader };\n}\n\nexport const businessOverlayComponentRegistry = {\n${componentEntries}\n} satisfies Record<string, RegistryEntry>;\n`;

  const menuEntries = manifest.components.map((entry) => `\t"${entry.key}": {},`).join('\n');
  const menu = `package iam\n\nvar businessOverlayMenuComponentKeys = map[string]struct{}{\n${menuEntries}\n}\n`;

  const outputs = new Map([
    ['backend/modules/business/business_overlay_registry.go', backend],
    ['frontend/src/modules/businessOverlay.ts', frontendModules],
    ['frontend/src/core/router/businessOverlayComponentRegistry.ts', components],
    ['backend/modules/system/iam/menu/business_overlay_component_registry.go', menu],
  ]);
  for (const [relativePath, content] of outputs) {
    const target = path.join(targetRoot, relativePath);
    fs.mkdirSync(path.dirname(target), { recursive: true });
    fs.writeFileSync(target, content, 'utf8');
  }
  return [...outputs.keys()];
}

function wireBusinessRegistries(targetRoot) {
  const patches = [
    {
      path: 'backend/modules/business/business.go',
      anchor: '\tInitGeneratedBusinessModules(r, db)',
      replacement: '\tinitOverlayBusinessModules(r, db)\n\tInitGeneratedBusinessModules(r, db)',
    },
    {
      path: 'frontend/src/core/router/modules.ts',
      anchor: "import { generatedBusinessModules } from '../../modules/generated/business';",
      replacement: "import { generatedBusinessModules } from '../../modules/generated/business';\nimport { overlayBusinessModules } from '../../modules/businessOverlay';",
    },
    {
      path: 'frontend/src/core/router/modules.ts',
      anchor: 'export const businessModules: ModuleConfig[] = [...generatedBusinessModules];',
      replacement: 'export const businessModules: ModuleConfig[] = [\n  ...overlayBusinessModules,\n  ...generatedBusinessModules,\n];',
    },
    {
      path: 'frontend/src/core/router/componentRegistry.ts',
      anchor: "import { generatedComponentRegistry } from './generatedComponentRegistry';",
      replacement: "import { generatedComponentRegistry } from './generatedComponentRegistry';\nimport { businessOverlayComponentRegistry } from './businessOverlayComponentRegistry';",
    },
    {
      path: 'frontend/src/core/router/componentRegistry.ts',
      anchor: '  ...staticComponentRegistry,\n  ...generatedComponentRegistry,',
      replacement: '  ...staticComponentRegistry,\n  ...businessOverlayComponentRegistry,\n  ...generatedComponentRegistry,',
    },
    {
      path: 'backend/modules/system/iam/menu/component_registry.go',
      anchor: 'var registeredMenuComponentKeys = mergeMenuComponentKeys(staticRegisteredMenuComponentKeys, generatedMenuComponentKeys)',
      replacement: 'var registeredMenuComponentKeys = mergeMenuComponentKeys(\n\tstaticRegisteredMenuComponentKeys,\n\tbusinessOverlayMenuComponentKeys,\n\tgeneratedMenuComponentKeys,\n)',
    },
  ];
  const changed = new Set();
  for (const patch of patches) {
    const target = path.join(targetRoot, patch.path);
    const content = fs.readFileSync(target, 'utf8');
    const matches = content.split(patch.anchor).length - 1;
    if (matches !== 1) throw new Error(`Base assembly contract changed: ${patch.path}`);
    fs.writeFileSync(target, content.replace(patch.anchor, patch.replacement), 'utf8');
    changed.add(patch.path);
  }
  return [...changed];
}

function generateBusinessFallbacks(targetRoot) {
  const businessRoot = path.join(targetRoot, 'frontend', 'src', 'modules', 'business');
  const outputRoot = path.join(targetRoot, 'frontend', 'src', 'i18n', 'resources', 'business');
  const locales = ['zh-CN', 'en-US', 'ja-JP', 'ko-KR', 'fr-FR'];
  fs.mkdirSync(outputRoot, { recursive: true });
  for (const locale of locales) {
    const resources = {};
    if (fs.existsSync(businessRoot)) {
      const visit = (directory) => {
        for (const entry of fs.readdirSync(directory, { withFileTypes: true })) {
          const absolute = path.join(directory, entry.name);
          if (entry.isDirectory()) visit(absolute);
          else if (entry.name === `${locale}.json`) Object.assign(resources, JSON.parse(fs.readFileSync(absolute, 'utf8')));
        }
      };
      visit(businessRoot);
    }
    const identifier = `generated${locale.replace(/[^A-Za-z0-9]/gu, '')}Fallback`;
    fs.writeFileSync(
      path.join(outputRoot, `${locale}.ts`),
      `const ${identifier} = ${JSON.stringify(resources, null, 2)};\n\nexport default ${identifier};\n`,
      'utf8',
    );
  }
}

function wireBusinessFallbacks(targetRoot) {
  const outputs = [];
  for (const locale of ['zh-CN', 'en-US']) {
    const target = path.join(targetRoot, 'frontend', 'src', 'i18n', 'resources', `${locale}.ts`);
    let content = fs.readFileSync(target, 'utf8');
    const objectAnchor = /const\s+[A-Za-z0-9_$]+Fallback\s*=\s*\{/u;
    if (!objectAnchor.test(content)) throw new Error(`Base ${locale} fallback contract changed`);
    content = `import businessFallback from './business/${locale}';\n${content}`;
    content = content.replace(objectAnchor, (match) => `${match}\n  ...businessFallback,`);
    fs.writeFileSync(target, content, 'utf8');
    outputs.push(`frontend/src/i18n/resources/${locale}.ts`);
  }
  return outputs;
}

function mergePackages(baseRoot, opsRoot, targetRoot, manifest) {
  const basePackage = JSON.parse(fs.readFileSync(path.join(baseRoot, 'package.json'), 'utf8'));
  basePackage.scripts = {
    ...basePackage.scripts,
    'rebuild:from-base': 'node scripts/business-overlay/rebuild-from-base.mjs',
    'check:business-overlay': 'node scripts/business-overlay/check-business-overlay.mjs',
    'test:business-overlay': 'node --test tests/scripts/business-overlay/*.test.mjs',
  };
  for (const scriptName of Object.keys(basePackage.scripts)) {
    if (scriptName.startsWith('release:foundation') || scriptName === 'test:foundation-release') {
      delete basePackage.scripts[scriptName];
    }
  }
  fs.writeFileSync(path.join(targetRoot, 'package.json'), `${JSON.stringify(basePackage, null, 2)}\n`, 'utf8');

  const baseFrontend = JSON.parse(fs.readFileSync(path.join(baseRoot, 'frontend', 'package.json'), 'utf8'));
  const opsFrontend = JSON.parse(fs.readFileSync(path.join(opsRoot, 'frontend', 'package.json'), 'utf8'));
  for (const scriptName of manifest.businessSmokeScripts) {
    if (!opsFrontend.scripts[scriptName]) throw new Error(`Missing business smoke script: ${scriptName}`);
    baseFrontend.scripts[scriptName] = opsFrontend.scripts[scriptName];
  }
  fs.writeFileSync(path.join(targetRoot, 'frontend', 'package.json'), `${JSON.stringify(baseFrontend, null, 2)}\n`, 'utf8');
}

function wireBusinessSmokeCoverage(opsRoot, targetRoot, manifest) {
  const opsFrontend = JSON.parse(fs.readFileSync(path.join(opsRoot, 'frontend', 'package.json'), 'utf8'));
  const rows = [];
  for (const scriptName of manifest.businessSmokeScripts) {
    const command = String(opsFrontend.scripts?.[scriptName] || '');
    const specs = command
      .split(/\s+/u)
      .filter((token) => token.startsWith('tests/smoke/business/') && token.endsWith('.spec.ts'))
      .map((token) => token.replace('tests/smoke/', ''));
    if (specs.length > 0) rows.push(`- \`${scriptName}\` -> ${specs.map((spec) => `\`${spec}\``).join(', ')}`);
  }
  const readme = path.join(targetRoot, 'frontend', 'tests', 'smoke', 'README.md');
  const marker = '## Ops Business Overlay';
  let content = fs.readFileSync(readme, 'utf8').trimEnd();
  const markerIndex = content.indexOf(marker);
  if (markerIndex >= 0) content = content.slice(0, markerIndex).trimEnd();
  if (rows.length > 0) content += `\n\n${marker}\n\n${rows.join('\n')}\n`;
  else content += '\n';
  fs.writeFileSync(readme, content, 'utf8');
  return ['frontend/tests/smoke/README.md'];
}

function protectBusinessOverlayFromGeneratedCleanup(targetRoot, manifest) {
  const cleanupPath = path.join(targetRoot, 'frontend', 'scripts', 'cleanup-generated-modules.mjs');
  let content = fs.readFileSync(cleanupPath, 'utf8');
  const injection = [
    '',
    "const BUSINESS_OVERLAY_REPORT = path.join(repoRoot, '.business-overlay-report.json');",
    'const BUSINESS_OVERLAY_OWNED_FILES = new Set(',
    '  fs.existsSync(BUSINESS_OVERLAY_REPORT)',
    "    ? JSON.parse(fs.readFileSync(BUSINESS_OVERLAY_REPORT, 'utf8')).ownedFiles.map((entry) => entry.path)",
    '    : [],',
    ');',
    'function hasBusinessOverlayOwnedDescendant(targetPath) {',
    "  const relative = path.relative(repoRoot, targetPath).replaceAll('\\\\', '/');",
    '  const prefix = `${relative}/`;',
    '  return BUSINESS_OVERLAY_OWNED_FILES.has(relative)',
    '    || Array.from(BUSINESS_OVERLAY_OWNED_FILES).some((entry) => entry.startsWith(prefix));',
    '}',
    '',
  ].join('\n');
  const anchor = 'const I18N_LOCALES =';
  if (!content.includes('BUSINESS_OVERLAY_OWNED_FILES')) {
    content = content.replace(anchor, `${injection}\n${anchor}`);
  }
  content = content.replaceAll(
    "if (hasTrackedDescendant(targetPath, repoBase, trackedFiles)) {",
    "if (hasBusinessOverlayOwnedDescendant(targetPath) || hasTrackedDescendant(targetPath, repoBase, trackedFiles)) {",
  );
  fs.writeFileSync(cleanupPath, content, 'utf8');
  return ['frontend/scripts/cleanup-generated-modules.mjs'];
}

function mergeGoDependencies(opsRoot, targetRoot, manifest) {
  const dependencies = Object.entries(manifest.goDependencies ?? {});
  if (dependencies.length === 0) return [];
  const goMod = path.join(targetRoot, 'backend', 'go.mod');
  for (const [modulePath, version] of dependencies) {
    const result = spawnSync('go', ['mod', 'edit', `-require=${modulePath}@${version}`], {
      cwd: path.dirname(goMod),
      encoding: 'utf8',
    });
    if (result.status !== 0) throw new Error(result.stderr.trim() || `Cannot add ${modulePath}@${version}`);
  }
  const sourceSum = fs.readFileSync(path.join(opsRoot, 'backend', 'go.sum'), 'utf8').split(/\r?\n/u);
  const targetSumPath = path.join(targetRoot, 'backend', 'go.sum');
  const targetLines = fs.readFileSync(targetSumPath, 'utf8').split(/\r?\n/u).filter(Boolean);
  for (const [modulePath, version] of dependencies) {
    const prefix = `${modulePath} ${version}`;
    const matches = sourceSum.filter((line) => line.startsWith(prefix));
    if (matches.length < 2) throw new Error(`Missing go.sum entries for ${modulePath}@${version}`);
    targetLines.push(...matches);
  }
  fs.writeFileSync(targetSumPath, `${[...new Set(targetLines)].sort().join('\n')}\n`, 'utf8');
  return ['backend/go.mod', 'backend/go.sum'];
}

function fileHash(filePath) {
  return crypto.createHash('sha256').update(fs.readFileSync(filePath)).digest('hex');
}

function writeReport(targetRoot, baseCommit, copiedBusiness, copiedRepository, rewritten, generated) {
  const owned = [...new Set([...copiedBusiness, ...copiedRepository, ...generated])].sort();
  const report = {
    schemaVersion: 1,
    baseCommit,
    ownedFiles: owned.map((file) => ({ path: file, sha256: fileHash(path.join(targetRoot, file)) })),
    rewrittenGoImports: rewritten,
    generatedFiles: generated.sort(),
  };
  const output = path.join(targetRoot, '.business-overlay-report.json');
  fs.writeFileSync(output, `${JSON.stringify(report, null, 2)}\n`, 'utf8');
  return report;
}

export function rebuildFromBase({ opsRoot = defaultOpsRoot, baseRoot, targetRoot, manifestPath } = {}) {
  const resolvedManifest = manifestPath ?? path.join(opsRoot, 'business-overlay.json');
  const manifest = JSON.parse(fs.readFileSync(resolvedManifest, 'utf8'));
  assertSafeManifest(manifest);

  let resolvedBase;
  let lockedBaseCommit = null;
  if (baseRoot) {
    resolvedBase = path.resolve(opsRoot, baseRoot);
  } else {
    const fromLock = resolveBaseFromLock(opsRoot);
    if (!fromLock) {
      throw new Error(
        'No locked Base snapshot found (foundation-release.lock.json). Ops consumes Base only from ' +
        'the published GitHub release, not a local ../pantheon-base working tree.',
      );
    }
    resolvedBase = fromLock.baseRoot;
    lockedBaseCommit = fromLock.baseCommit;
  }

  const resolvedTarget = path.resolve(opsRoot, targetRoot ?? '.tmp/business-overlay-rebuild');
  if (resolvedTarget === path.resolve(opsRoot) || resolvedTarget === resolvedBase) {
    throw new Error('Refusing to rebuild over a source repository');
  }
  fs.rmSync(resolvedTarget, { recursive: true, force: true });
  fs.mkdirSync(resolvedTarget, { recursive: true });

  const baseIsGit = fs.existsSync(path.join(resolvedBase, '.git'));
  const baseTracked = baseIsGit ? trackedFiles(resolvedBase) : snapshotFiles(resolvedBase);
  const baseCommit = lockedBaseCommit ?? runGit(resolvedBase, ['rev-parse', 'HEAD']);
  for (const file of baseTracked) copyFile(resolvedBase, resolvedTarget, file);
  pruneBaseRepositoryOverlay(resolvedTarget, manifest);
  assertBusinessPathsDoNotOverwriteBase(opsRoot, resolvedBase, baseTracked, manifest.businessPaths ?? []);
  const opsTracked = trackedFiles(opsRoot);
  const copiedBusiness = manifest.businessPaths.flatMap((entry) => copyDeclaredPath(opsRoot, resolvedTarget, entry, opsTracked));
  const copiedRepository = manifest.repositoryOverlayPaths.flatMap((entry) => copyDeclaredPath(opsRoot, resolvedTarget, entry, opsTracked));
  copyFile(opsRoot, resolvedTarget, 'business-overlay.json');
  copiedRepository.push('business-overlay.json');

  const rewritten = rewriteBusinessGoImports(resolvedTarget, manifest.sourceModule, `${manifest.base.module}`);
  const generated = generateRegistries(resolvedTarget, manifest);
  generated.push(...wireBusinessRegistries(resolvedTarget));
  generated.push(...mergeGoDependencies(opsRoot, resolvedTarget, manifest));
  generateBusinessFallbacks(resolvedTarget);
  generated.push(...wireBusinessFallbacks(resolvedTarget));
  generated.push(...['frontend/src/i18n/resources/business/zh-CN.ts', 'frontend/src/i18n/resources/business/en-US.ts', 'frontend/src/i18n/resources/business/ja-JP.ts', 'frontend/src/i18n/resources/business/ko-KR.ts', 'frontend/src/i18n/resources/business/fr-FR.ts']);
  mergePackages(resolvedBase, opsRoot, resolvedTarget, manifest);
  generated.push('package.json', 'frontend/package.json');
  generated.push(...wireBusinessSmokeCoverage(opsRoot, resolvedTarget, manifest));
  generated.push(...protectBusinessOverlayFromGeneratedCleanup(resolvedTarget, manifest));
  const report = writeReport(resolvedTarget, baseCommit, copiedBusiness, copiedRepository, rewritten, generated);
  return { targetRoot: resolvedTarget, report };
}

function argument(name) {
  const index = process.argv.indexOf(name);
  return index >= 0 ? process.argv[index + 1] : undefined;
}

if (process.argv[1] && path.resolve(process.argv[1]) === currentFile) {
  const result = rebuildFromBase({ baseRoot: argument('--base'), targetRoot: argument('--target'), manifestPath: argument('--manifest') });
  console.log(`OK rebuilt clean Ops tree at ${result.targetRoot}`);
  console.log(`Base commit: ${result.report.baseCommit}`);
  console.log(`Declared/generated Ops files: ${result.report.ownedFiles.length}`);
}

import crypto from 'node:crypto';
import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';
import { spawnSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';
import { resolveBaseFromLock, validateChecksum } from './foundation-release.mjs';
import { assertRelativeInput, assertRealPath, captureSelectedInputs, METADATA_LIMITS, frozenInputReader } from './foundation-candidate.mjs';
export const validateReleaseBundleChecksum = validateChecksum;

const currentFile = fileURLToPath(import.meta.url);
const defaultOpsRoot = path.resolve(path.dirname(currentFile), '..', '..');
const registryIdentifier = /^[A-Za-z_][A-Za-z0-9_]*$/u;

function repoPath(value) {
  return value.replaceAll('\\', '/').replace(/^\.\//u, '').replace(/\/$/u, '');
}

function isWithin(candidate, parent) {
  return candidate === parent || candidate.startsWith(`${parent}/`);
}

export function parseOverlayGitFiles(bytes) {
  const output = new TextDecoder('utf-8', { fatal: true }).decode(bytes);
  if (output && !output.endsWith('\0')) throw new Error('Incomplete Ops Git selection');
  return output.split('\0').filter(Boolean).map(record => {
    const staged = /^([0-7]{6}) ([a-f0-9]{40,64}) ([0-3])\t/u.exec(record);
    if (staged && (staged[1] === '160000' || staged[3] !== '0'))
      throw new Error('Unsupported Ops gitlink/submodule or unresolved index record');
    const raw = staged ? record.slice(staged[0].length) : record;
    const directory = !staged && raw.endsWith('/');
    const file = assertRelativeInput(directory ? raw.slice(0, -1) : raw);
    return { path: file, index: staged ? staged[0].trimEnd() : null, ...(directory ? { directory: true } : {}) };
  });
}

// One NUL-safe selection for freeze, copy, collision checks and source replay.
// --cached retains ignored-but-tracked files; stage metadata pins index changes.
export function selectOverlayInputs(root, manifest) {
  const result = spawnSync('git', ['ls-files', '--stage', '--cached', '--others', '--exclude-standard', '-z'],
    { cwd: root, maxBuffer: 64 * 1024 * 1024 });
  if (result.error) throw result.error;
  if (result.status !== 0) throw new Error('Cannot freeze declared Ops Git file selection');
  const declarations = [...new Set([...manifest.businessPaths, ...manifest.repositoryOverlayPaths,
    ...(manifest.businessMigrations ?? [])].map(assertRelativeInput))].sort();
  const implicit = ['business-overlay.json', 'foundation-release.lock.json', 'package.json', 'frontend/package.json',
    ...(Object.keys(manifest.goDependencies ?? {}).length ? ['backend/go.sum'] : []),
    ...(manifest.businessMigrations ?? [])];
  const selected = entry =>
    declarations.some(root => isWithin(entry.path, root)) || implicit.includes(entry.path) ||
    (entry.directory && declarations.some(root => isWithin(root, entry.path))) ||
    /^frontend\/src\/modules\/business\/.*\/(?:zh-CN|en-US|ja-JP|ko-KR|fr-FR)\.json$/u.test(entry.path);
  const entries = [];
  const deletedEntries = [];
  const expand = entry => {
    if (!selected(entry)) return;
    if (!entry.directory) {
      try {
        assertRealPath(path.join(root, entry.path));
      } catch (error) {
        // The index still lists an intentionally deleted worktree leaf. Keep
        // its index/deletion identity for frozen-selection checks, but never
        // copy or read it. Link/type/permission failures are not deletions.
        if (!entry.index || error.code !== 'ENOENT') throw error;
        deletedEntries.push({ ...entry, deleted: true });
        if (entries.length + deletedEntries.length > METADATA_LIMITS.files)
          throw new Error('Declared metadata input count exceeds limit');
        return;
      }
      entries.push(entry);
      if (entries.length + deletedEntries.length > METADATA_LIMITS.files) throw new Error('Declared metadata input count exceeds limit');
      return;
    }
    // Embedded repositories can be retained metadata, never implicit product
    // ownership. Enumerate their own Git-selected leaves (including ignored
    // tracked files), rather than recursively copying their .git or caches.
    if (!['.harness', '.foundation'].some(parent => isWithin(entry.path, parent)) ||
        !declarations.some(parent => isWithin(entry.path, parent) || isWithin(parent, entry.path)))
      throw new Error('Unsupported untracked product/foreign repository directory: ' + entry.path);
    const directory = path.join(root, entry.path);
    assertRealPath(directory, { directory: true });
    assertRealPath(path.join(directory, '.git'), { directory: true });
    const nested = spawnSync('git', ['ls-files', '--stage', '--cached', '--others', '--exclude-standard', '-z'],
      { cwd: directory, maxBuffer: 64 * 1024 * 1024 });
    if (nested.error) throw nested.error;
    if (nested.status !== 0) throw new Error('Cannot select nested metadata repository: ' + entry.path);
    for (const leaf of parseOverlayGitFiles(nested.stdout))
      expand({ ...leaf, path: assertRelativeInput(`${entry.path}/${leaf.path}`) });
  };
  for (const entry of parseOverlayGitFiles(result.stdout)) expand(entry);
  const gitFiles = [...new Set(entries.filter(entry => declarations.some(root => isWithin(entry.path, root)))
    .map(entry => entry.path))].sort();
  const files = new Set([...entries.map(entry => entry.path), ...implicit]);
  // Selector files outside declared roots still affect the chosen product inputs.
  // Pin only their bytes/identity, never directory mtimes of ignored caches.
  for (const relative of [...declarations, ...files]) {
    for (let parent = path.posix.dirname(relative); ; parent = path.posix.dirname(parent)) {
      const ignore = parent === '.' ? '.gitignore' : `${parent}/.gitignore`;
      if (fs.existsSync(path.join(root, ignore))) files.add(ignore);
      if (parent === '.') break;
    }
  }
  return Object.freeze({ declarations: Object.freeze(declarations), gitFiles: Object.freeze(gitFiles),
    files: Object.freeze([...files].sort()),
    gitEntries: Object.freeze([...entries, ...deletedEntries].map(entry => Object.freeze(entry)).sort((a, b) =>
      a.path < b.path ? -1 : a.path > b.path ? 1 : String(a.index).localeCompare(String(b.index)))) });
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

function copyFile(sourceRoot, targetRoot, relativePath, io = fs) {
  const source = path.join(sourceRoot, relativePath);
  const target = path.join(targetRoot, relativePath);
  io.mkdirSync(path.dirname(target), { recursive: true });
  io.copyFileSync(source, target);
}

function copyDeclaredPath(sourceRoot, targetRoot, declaredPath, sourceTracked, io = fs) {
  const normalized = repoPath(declaredPath);
  const files = sourceTracked.filter((file) => isWithin(file, normalized));
  if (files.length === 0 && !io.existsSync(path.join(sourceRoot, normalized))) {
    throw new Error(`Declared overlay path does not exist: ${normalized}`);
  }
  for (const file of files) copyFile(sourceRoot, targetRoot, file, io);
  return files;
}

// Business overlay 不得覆盖 Base 的 hook/源文件（如 backend/modules/business/{business.go,
// retired_modules.go, generated_registry.go}）。若 businessPaths 声明了与 Base 同名且内容
// 不同的文件，说明把 Base 的 hook 层当作业务文件重写了 —— 改为用独立 overlay 文件注入
// （见 retired_modules_overlay.go 的 init() 追加模式）。repositoryOverlayPaths 的覆盖（如
// AGENTS.md/CLAUDE.md）是刻意的仓库级替换，不受此约束。
function assertBusinessPathsDoNotOverwriteBase(opsRoot, baseRoot, baseTracked, businessPaths, sourceFiles) {
  const baseFiles = new Set(baseTracked);
  const conflicts = [];
  for (const declaredPath of businessPaths) {
    const normalized = repoPath(declaredPath);
    const files = sourceFiles.filter(file => isWithin(file, normalized));
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

// Module-owned SQL may join the shared migration chain only through this
// fixed destination. Generic businessPaths still cannot own backend/pkg.
export function planBusinessMigrations(manifest) {
  const sources = manifest.businessMigrations ?? [];
  if (!Array.isArray(sources)) throw new Error('businessMigrations must be an array');
  const versions = new Map();
  const plans = sources.map((source) => {
    const match = typeof source === 'string' && /^backend\/modules\/business\/(?:[a-z0-9_]+\/)+migrations\/(\d{6})_([a-z0-9_]+)\.(up|down)\.sql$/u.exec(source);
    if (!match) throw new Error(`Invalid business migration source: ${source}`);
    if (!(manifest.businessPaths ?? []).some((owned) => isWithin(source, repoPath(owned)))) {
      throw new Error(`Business migration source is not declared in businessPaths: ${source}`);
    }
    const [, version, name, direction] = match;
    const pair = versions.get(version) ?? { name, directions: new Set() };
    if (pair.name !== name || pair.directions.has(direction)) {
      throw new Error(`Duplicate business migration version or direction: ${version}`);
    }
    pair.directions.add(direction);
    versions.set(version, pair);
    return { source, version, target: `backend/pkg/database/migrations/${path.posix.basename(source)}` };
  });
  for (const [version, pair] of versions) {
    if (pair.directions.size !== 2) throw new Error(`Business migration requires up/down pair: ${version}`);
  }
  return plans;
}

function assembleBusinessMigrations(opsRoot, targetRoot, manifest, io = fs) {
  const plans = planBusinessMigrations(manifest);
  const targetDirectory = path.join(targetRoot, 'backend/pkg/database/migrations');
  const baseVersions = new Set((io.existsSync(targetDirectory) ? io.readdirSync(targetDirectory) : [])
    .map((name) => /^(\d+)_.*\.(up|down)\.sql$/u.exec(name)?.[1]).filter(Boolean).map(Number));
  // Validate every input and collision before copying either half of a pair.
  for (const plan of plans) {
    if (baseVersions.has(Number(plan.version))) throw new Error(`Business migration version conflicts with Base: ${plan.version}`);
    const source = path.join(opsRoot, plan.source);
    if (!io.existsSync(source) || !io.lstatSync(source).isFile()) throw new Error(`Business migration source must be a regular file: ${plan.source}`);
  }
  for (const plan of plans) {
    io.mkdirSync(path.dirname(path.join(targetRoot, plan.target)), { recursive: true });
    io.copyFileSync(path.join(opsRoot, plan.source), path.join(targetRoot, plan.target));
  }
  return plans.map((plan) => plan.target);
}

export function assertSafeManifest(manifest) {
  planBusinessMigrations(manifest);
  if (manifest.schemaVersion !== 1) throw new Error('Unsupported business overlay schema');
  validateBackendModuleEntries(manifest);
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

function validateBackendModuleEntries(manifest) {
  if (manifest.backendModules === undefined) return;
  if (!Array.isArray(manifest.backendModules)) throw new Error('backendModules must be an array');
  for (const entry of manifest.backendModules) {
    if (!entry || typeof entry !== 'object' || Array.isArray(entry))
      throw new Error('Business registry module must be an object');
    if (Object.hasOwn(entry, 'preflight') &&
        (typeof entry.preflight !== 'string' || !registryIdentifier.test(entry.preflight)))
      throw new Error('Business registry preflight must be a nonempty identifier');
    if (typeof entry.importPath !== 'string' || typeof entry.alias !== 'string' || typeof entry.init !== 'string' ||
        !/^pantheon-base\/modules\/business\/[a-z0-9_/]+$/u.test(entry.importPath) ||
        !registryIdentifier.test(entry.alias) || !registryIdentifier.test(entry.init))
      throw new Error('Business registry import must have declared business ownership');
    const relative = 'backend/' + entry.importPath.replace(/^pantheon-base\//u, '');
    if (!(manifest.businessPaths ?? []).some(owned => isWithin(relative, repoPath(owned))))
      throw new Error('Business registry import must have declared business ownership');
  }
}

function pruneBaseRepositoryOverlay(targetRoot, manifest, io = fs) {
  for (const entry of [...(manifest.replaceRoots ?? []), ...(manifest.baseExcludedPaths ?? [])]) {
    const normalized = repoPath(entry);
    if (!normalized || normalized === '.' || normalized.startsWith('../')) {
      throw new Error(`Unsafe Base exclusion path: ${entry}`);
    }
    io.rmSync(path.join(targetRoot, normalized), { recursive: true, force: true });
  }
}

function rewriteBusinessGoImports(targetRoot, sourceModule, targetModule, io = fs) {
  const backendRoot = path.join(targetRoot, 'backend', 'modules', 'business');
  if (!fs.existsSync(backendRoot)) return [];
  const changed = [];
  const visit = (directory) => {
    for (const entry of fs.readdirSync(directory, { withFileTypes: true })) {
      const absolute = path.join(directory, entry.name);
      if (entry.isDirectory()) visit(absolute);
      else if (entry.name.endsWith('.go')) {
        const before = fs.readFileSync(absolute, 'utf8');
        // Business sources from the pre-foundation migration may still carry the
        // historical local module prefix. Normalize both forms at the assembly
        // seam so the rebuilt tree always compiles against the locked Base module.
        const after = before
          .replaceAll(`${sourceModule}/`, `${targetModule}/`)
          .replaceAll('"pantheon-base/', `"${targetModule}/`);
        if (after !== before) {
          io.writeFileSync(absolute, after, 'utf8');
          changed.push(repoPath(path.relative(targetRoot, absolute)));
        }
      }
    }
  };
  visit(backendRoot);
  return changed.sort();
}

function readGoModule(targetRoot, io = fs) {
  const goMod = path.join(targetRoot, 'backend', 'go.mod');
  const content = io.readFileSync(goMod, 'utf8');
  const match = content.match(/^module\s+([^\s]+)\s*$/mu);
  if (!match) throw new Error(`Cannot determine Go module from ${goMod}`);
  return match[1];
}

// Overlay registry files are injected into an existing Base package directory, so
// they must declare that directory's package name. Base owns the package (it
// renamed the iam/menu package from `iam` to `menu`), so read the name from the
// copied Base tree instead of hardcoding one that silently stops matching after
// a Base rename.
function readHostGoPackage(targetRoot, relativeDirectory, io = fs) {
  const directory = path.join(targetRoot, relativeDirectory);
  if (!fs.existsSync(directory)) return null;
  const candidates = fs
    .readdirSync(directory)
    .filter((name) => name.endsWith('.go') && !name.endsWith('_test.go'))
    .sort();
  for (const name of candidates) {
    const source = io.readFileSync(path.join(directory, name), 'utf8');
    const match = source.match(/^package\s+([A-Za-z_][A-Za-z0-9_]*)/mu);
    if (match) return match[1];
  }
  return null;
}

function generateRegistries(targetRoot, manifest, targetModule, io = fs) {
  const imports = manifest.backendModules
    .map((entry) => `\t${entry.alias} "${entry.importPath.replace(/^pantheon-base\//u, `${targetModule}/`)}"`)
    .join('\n');
  const calls = manifest.backendModules.map((entry) => `\t${entry.alias}.${entry.init}(r, db)`).join('\n');
  const preflights = manifest.backendModules.filter(entry => Object.hasOwn(entry, 'preflight'));
  const preflightCalls = preflights.map(entry =>
    `\tif err := ${entry.alias}.${entry.preflight}(db); err != nil {\n\t\treturn err\n\t}`).join('\n');
  const preflightRegistration = preflights.length ?
    `func init() {\n\tif err := RegisterBusinessInitializationPreflight(preflightOverlayBusinessModules); err != nil {\n\t\tpanic(err)\n\t}\n}\n\nfunc preflightOverlayBusinessModules(db *gorm.DB) error {\n${preflightCalls}\n\treturn nil\n}\n\n` : '';
  const backend = `package business\n\nimport (\n${imports}\n\n\t"github.com/gin-gonic/gin"\n\t"gorm.io/gorm"\n)\n\n${preflightRegistration}func initOverlayBusinessModules(r *gin.RouterGroup, db *gorm.DB) {\n${calls}\n}\n`;

  const frontendImports = manifest.frontendModules
    .map((entry) => `import { ${entry.export} } from '${entry.importPath}';`)
    .join('\n');
  const frontendEntries = manifest.frontendModules.map((entry) => `  ${entry.export},`).join('\n');
  const frontendModules = `import type { ModuleConfig } from '../core/router/types';\n${frontendImports}\n\nexport const overlayBusinessModules: ModuleConfig[] = [\n${frontendEntries}\n];\n`;

  const componentEntries = manifest.components
    .map((entry) => `  '${entry.key}': defineRegistryEntry(() => import('${entry.importPath}')),`)
    .join('\n');
  const components = `import { lazy, type LazyExoticComponent, type ComponentType } from 'react';\n\ntype ComponentLoader = () => Promise<{ default: ComponentType }>;\n\ninterface RegistryEntry {\n  component: LazyExoticComponent<ComponentType>;\n  preload: ComponentLoader;\n}\n\nfunction defineRegistryEntry(loader: ComponentLoader): RegistryEntry {\n  return { component: lazy(loader), preload: loader };\n}\n\nexport const businessOverlayComponentRegistry = {\n${componentEntries}\n} satisfies Record<string, RegistryEntry>;\n`;

  // Align the `{}` values so the emitted map literal is gofmt-clean on first write.
  const menuKeyPrefixes = manifest.components.map((entry) => `"${entry.key}":`);
  const maxMenuKeyPrefixLen = Math.max(...menuKeyPrefixes.map((prefix) => prefix.length));
  const menuEntries = manifest.components
    .map((entry) => {
      const prefix = `"${entry.key}":`;
      return `\t${prefix}${' '.repeat(maxMenuKeyPrefixLen - prefix.length + 1)}{},`;
    })
    .join('\n');
  const menuDirectory = 'backend/modules/system/iam/menu';
  const menuPackage = readHostGoPackage(targetRoot, menuDirectory, io) ?? 'menu';
  const menu = `package ${menuPackage}\n\nvar businessOverlayMenuComponentKeys = map[string]struct{}{\n${menuEntries}\n}\n`;

  const outputs = new Map([
    ['backend/modules/business/business_overlay_registry.go', backend],
    ['frontend/src/modules/businessOverlay.ts', frontendModules],
    ['frontend/src/core/router/businessOverlayComponentRegistry.ts', components],
    [`${menuDirectory}/business_overlay_component_registry.go`, menu],
  ]);
  for (const [relativePath, content] of outputs) {
    const target = path.join(targetRoot, relativePath);
    io.mkdirSync(path.dirname(target), { recursive: true });
    io.writeFileSync(target, content, 'utf8');
  }
  return [...outputs.keys()];
}

function wireBusinessRegistries(targetRoot, io = fs) {
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
    const content = io.readFileSync(target, 'utf8');
    const matches = content.split(patch.anchor).length - 1;
    if (matches !== 1) throw new Error(`Base assembly contract changed: ${patch.path}`);
    io.writeFileSync(target, content.replace(patch.anchor, patch.replacement), 'utf8');
    changed.add(patch.path);
  }
  return [...changed];
}

function businessLocalePlan(baseRoot, baseFiles, opsRoot, opsFiles, manifest) {
  const leaves = new Map();
  const exclusions = [...(manifest.replaceRoots ?? []), ...(manifest.baseExcludedPaths ?? [])].map(repoPath);
  const locale = file => /^frontend\/src\/modules\/business\/.*\/(?:zh-CN|en-US|ja-JP|ko-KR|fr-FR)\.json$/u.test(file);
  for (const file of baseFiles) {
    if (locale(file) && !exclusions.some(entry => isWithin(file, entry))) leaves.set(file, baseRoot);
  }
  for (const file of opsFiles) {
    if (locale(file) && (manifest.businessPaths ?? []).some(entry => isWithin(file, repoPath(entry)))) leaves.set(file, opsRoot);
  }
  // One codepoint-ordered leaf plan defines insertion order and overrides:
  // copied Ops replaces the same Base leaf; later paths win duplicate keys.
  return [...leaves].sort(([a], [b]) => a < b ? -1 : a > b ? 1 : 0)
    .map(([relative, root]) => ({ relative, root }));
}

function generateBusinessFallbacks(targetRoot, io, leafPlan) {
  const outputRoot = path.join(targetRoot, 'frontend', 'src', 'i18n', 'resources', 'business');
  const locales = ['zh-CN', 'en-US', 'ja-JP', 'ko-KR', 'fr-FR'];
  io.mkdirSync(outputRoot, { recursive: true });
  for (const locale of locales) {
    const resources = {};
    for (const { relative, root } of leafPlan.filter(entry => path.posix.basename(entry.relative) === `${locale}.json`))
      Object.assign(resources, JSON.parse(io.readFileSync(path.join(root, relative), 'utf8')));
    const identifier = `generated${locale.replace(/[^A-Za-z0-9]/gu, '')}Fallback`;
    io.writeFileSync(
      path.join(outputRoot, `${locale}.ts`),
      `const ${identifier} = ${JSON.stringify(resources, null, 2)};\n\nexport default ${identifier};\n`,
      'utf8',
    );
  }
}

function wireBusinessFallbacks(targetRoot, io = fs) {
  const outputs = [];
  for (const locale of ['zh-CN', 'en-US']) {
    const target = path.join(targetRoot, 'frontend', 'src', 'i18n', 'resources', `${locale}.ts`);
    let content = io.readFileSync(target, 'utf8');
    const objectAnchor = /const\s+[A-Za-z0-9_$]+Fallback\s*=\s*\{/u;
    if (!objectAnchor.test(content)) throw new Error(`Base ${locale} fallback contract changed`);
    content = `import businessFallback from './business/${locale}';\n${content}`;
    content = content.replace(objectAnchor, (match) => `${match}\n  ...businessFallback,`);
    io.writeFileSync(target, content, 'utf8');
    outputs.push(`frontend/src/i18n/resources/${locale}.ts`);
  }
  return outputs;
}

function mergePackages(baseRoot, opsRoot, targetRoot, manifest, io = fs) {
  const basePackage = JSON.parse(io.readFileSync(path.join(baseRoot, 'package.json'), 'utf8'));
  const opsPackage = JSON.parse(io.readFileSync(path.join(opsRoot, 'package.json'), 'utf8'));
  basePackage.scripts = {
    ...basePackage.scripts,
    'fetch:base-release': 'node scripts/business-overlay/foundation-release.mjs',
    'rebuild:from-base': 'node scripts/business-overlay/rebuild-from-base.mjs',
    'check:business-overlay': 'node scripts/business-overlay/check-business-overlay.mjs',
    'check:shared-drift': 'node scripts/business-overlay/check-shared-drift.mjs',
    'test:business-overlay': 'node --test tests/scripts/business-overlay/*.test.mjs',
    // Ops-owned root scripts (repositoryOverlayPaths) that Base does not ship.
    'check:encoding': 'node scripts/check-encoding.mjs --strict',
  };
  // Ops-only governance scripts: base may not ship them, but their script
  // files are ops repository overlays, so the entries must survive the merge.
  const opsOnlyScripts = [
    'check:business-boundaries',
    'test:business-boundaries',
    'check:business-foundation',
    'check:gorm-sql-literal',
    'test:gorm-sql-literal',
  ];
  for (const scriptName of opsOnlyScripts) {
    if (opsPackage.scripts?.[scriptName] && !basePackage.scripts[scriptName]) {
      basePackage.scripts[scriptName] = opsPackage.scripts[scriptName];
    }
  }
  for (const scriptName of Object.keys(basePackage.scripts)) {
    if (scriptName.startsWith('release:foundation') || scriptName === 'test:foundation-release') {
      delete basePackage.scripts[scriptName];
    }
  }
  io.writeFileSync(path.join(targetRoot, 'package.json'), `${JSON.stringify(basePackage, null, 2)}\n`, 'utf8');

  const baseFrontend = JSON.parse(io.readFileSync(path.join(baseRoot, 'frontend', 'package.json'), 'utf8'));
  const opsFrontend = JSON.parse(io.readFileSync(path.join(opsRoot, 'frontend', 'package.json'), 'utf8'));
  for (const scriptName of manifest.businessSmokeScripts) {
    if (!opsFrontend.scripts[scriptName]) throw new Error(`Missing business smoke script: ${scriptName}`);
    baseFrontend.scripts[scriptName] = opsFrontend.scripts[scriptName];
  }
  // Ops-only frontend dependencies (e.g. the local file: tarball binding the
  // Base UI package) must survive the Base package.json overwrite, otherwise
  // the rebuilt tree cannot resolve '@duanxldragon/pantheon-base-ui' imports.
  for (const [depName, depSpec] of Object.entries(opsFrontend.dependencies ?? {})) {
    if (!baseFrontend.dependencies?.[depName]) {
      baseFrontend.dependencies ??= {};
      baseFrontend.dependencies[depName] = depSpec;
    }
  }
  // The rebuilt tree is an assembled whole (Base framework + business overlay),
  // so the Base package name may self-reference via npm package.json
  // "exports" self-reference; keep the ops declaration if Base lacks it.
  if (!baseFrontend.exports && opsFrontend.exports) {
    baseFrontend.exports = opsFrontend.exports;
  }
  io.writeFileSync(path.join(targetRoot, 'frontend', 'package.json'), `${JSON.stringify(baseFrontend, null, 2)}\n`, 'utf8');
}

function wireBusinessSmokeCoverage(opsRoot, targetRoot, manifest, io = fs) {
  const opsFrontend = JSON.parse(io.readFileSync(path.join(opsRoot, 'frontend', 'package.json'), 'utf8'));
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
  let content = io.readFileSync(readme, 'utf8').trimEnd();
  const markerIndex = content.indexOf(marker);
  if (markerIndex >= 0) content = content.slice(0, markerIndex).trimEnd();
  if (rows.length > 0) content += `\n\n${marker}\n\n${rows.join('\n')}\n`;
  else content += '\n';
  io.writeFileSync(readme, content, 'utf8');
  return ['frontend/tests/smoke/README.md'];
}

function protectBusinessOverlayFromGeneratedCleanup(targetRoot, manifest, io = fs) {
  const cleanupPath = path.join(targetRoot, 'frontend', 'scripts', 'cleanup-generated-modules.mjs');
  let content = io.readFileSync(cleanupPath, 'utf8');
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
  io.writeFileSync(cleanupPath, content, 'utf8');
  return ['frontend/scripts/cleanup-generated-modules.mjs'];
}

function mergeGoDependencies(opsRoot, targetRoot, manifest, io = fs) {
  const dependencies = Object.entries(manifest.goDependencies ?? {});
  if (dependencies.length === 0) return [];
  const goMod = path.join(targetRoot, 'backend', 'go.mod');
  for (const [modulePath, version] of dependencies) {
    const args = ['mod', 'edit', `-require=${modulePath}@${version}`];
    const options = {
      cwd: path.dirname(goMod),
      encoding: 'utf8',
    };
    const result = io.runGo ? io.runGo(args, options) : spawnSync('go', args, options);
    if (result.error) throw result.error;
    if (result.status !== 0) throw new Error(result.stderr?.trim() || `Cannot add ${modulePath}@${version}`);
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
  io.writeFileSync(targetSumPath, `${[...new Set(targetLines)].sort().join('\n')}\n`, 'utf8');
  return ['backend/go.mod', 'backend/go.sum'];
}

export function normalizeGoDependencies(targetRoot, io = fs) {
  const args = ['list', '-buildvcs=false', '-mod=mod', '-deps', '-test', './...'];
  const options = {
    cwd: path.join(targetRoot, 'backend'),
    encoding: 'utf8',
    stdio: ['ignore', 'ignore', 'pipe'],
  };
  const result = io.runGo ? io.runGo(args, options) : spawnSync('go', args, options);
  if (result.error) throw result.error;
  if (result.status !== 0) {
    throw new Error('Cannot normalize consumer Go dependencies: ' +
      (result.stderr?.trim() || 'go list failed'));
  }
  return ['backend/go.mod', 'backend/go.sum'];
}

function fileHash(filePath) {
  return crypto.createHash('sha256').update(fs.readFileSync(filePath)).digest('hex');
}

function writeReport(targetRoot, baseCommit, copiedBusiness, copiedRepository, rewritten, generated, io = fs) {
  const owned = [...new Set([...copiedBusiness, ...copiedRepository, ...generated])].sort();
  const report = {
    schemaVersion: 1,
    baseCommit,
    ownedFiles: owned.map((file) => ({ path: file, sha256: fileHash(path.join(targetRoot, file)) })),
    rewrittenGoImports: rewritten,
    generatedFiles: generated.sort(),
  };
  const output = path.join(targetRoot, '.business-overlay-report.json');
  io.writeFileSync(output, `${JSON.stringify(report, null, 2)}\n`, 'utf8');
  return report;
}

// Base publishes this packaging contract. Only excluded Base archive entries
// can leave the consumer allowlist; current consumer reviews are always checked.
export function projectConsumerReviewRegistry(baseRoot, targetRoot, manifest, baseCommit, io = fs) {
  const descriptorPath = path.join(baseRoot, 'config/foundation-consumer-artifacts.json');
  if (!io.existsSync(descriptorPath)) return [];
  const descriptor = JSON.parse(io.readFileSync(descriptorPath, 'utf8'));
  const contract = descriptor.reviewLegacyRegistry;
  if (descriptor.schemaVersion !== 1 || descriptor.sourceOwner !== 'pantheon-base' ||
      contract?.path !== 'config/review-loop-legacy-artifacts.json' ||
      contract?.artifactRoot !== '.harness/archive') {
    throw new Error('Invalid Base consumer review packaging contract');
  }
  const sourcePath = path.join(baseRoot, contract.path);
  const sourceBytes = io.readFileSync(sourcePath);
  const source = JSON.parse(sourceBytes.toString('utf8'));
  if (!Array.isArray(source.entries)) throw new Error('Base legacy review entries must be an array');
  const excluded = [...(manifest.replaceRoots ?? []), ...(manifest.baseExcludedPaths ?? [])].map(repoPath);
  const retained = [];
  const external = [];
  for (const entry of source.entries) {
    if (typeof entry !== 'string' || !entry.startsWith('.harness/archive/') ||
        entry.includes('..') || entry.includes('\\') || !entry.endsWith('/review.md')) {
      throw new Error('Base legacy entry must be a canonical archive review');
    }
    if (!io.existsSync(path.join(baseRoot, entry))) throw new Error(`Stale Base legacy entry: ${entry}`);
    if (excluded.some((root) => isWithin(entry, root))) external.push(entry);
    else {
      if (!io.existsSync(path.join(targetRoot, entry))) throw new Error(`Missing retained Base review: ${entry}`);
      retained.push(entry);
    }
  }
  if (external.length === 0) return [];
  const targetPath = path.join(targetRoot, contract.path);
  const projection = {
    ...source,
    description: 'Consumer projection of the Base archive registry. Excluded Base artifacts stay recorded below; all current consumer reviews remain subject to strict review validation.',
    entries: retained,
    projection: { sourceOwner: descriptor.sourceOwner, baseCommit,
      sourceConfigSha256: crypto.createHash('sha256').update(sourceBytes).digest('hex'),
      externalBaseArtifacts: external },
  };
  io.mkdirSync(path.dirname(targetPath), { recursive: true });
  io.writeFileSync(targetPath, `${JSON.stringify(projection, null, 2)}\n`, 'utf8');
  return [contract.path];
}

export function assembleConsumerTree({ opsRoot = defaultOpsRoot, snapshot, targetRoot, manifestPath, fileSystem = fs, sourceSelection } = {}) {
  const resolvedManifest = manifestPath ?? path.join(opsRoot, 'business-overlay.json');
  const manifest = JSON.parse(fs.readFileSync(resolvedManifest, 'utf8'));
  assertSafeManifest(manifest);
  const selection = sourceSelection ?? selectOverlayInputs(opsRoot, manifest);

  const resolvedBase = snapshot?.baseRoot;
  if (!resolvedBase || !snapshot.baseCommit) throw new Error('Verified release snapshot is required');
  const resolvedTarget = path.resolve(opsRoot, targetRoot ?? '.tmp/business-overlay-rebuild');
  const targetContainsSource = [path.resolve(opsRoot), path.resolve(resolvedBase)].some((source) => source === resolvedTarget || source.startsWith(resolvedTarget + path.sep));
  const targetInsideBase = resolvedTarget.startsWith(path.resolve(resolvedBase) + path.sep);
  if (targetContainsSource || targetInsideBase || resolvedTarget === path.parse(resolvedTarget).root || path.basename(resolvedTarget) === '.git' || (fs.existsSync(resolvedTarget) && fs.lstatSync(resolvedTarget).isSymbolicLink())) {
    throw new Error('Refusing to rebuild over a source repository');
  }
  fileSystem.rmSync(resolvedTarget, { recursive: true, force: true });
  fileSystem.mkdirSync(resolvedTarget, { recursive: true });

  const baseTracked = snapshotFiles(resolvedBase);
  const baseCommit = snapshot.baseCommit;
  // Do not create files/directories that would immediately need destructive
  // pruning. Keep full baseTracked for collision validation below. The prune
  // still rejects unexpected target entries through the supplied guarded I/O.
  const exclusions = [...(manifest.replaceRoots ?? []), ...(manifest.baseExcludedPaths ?? [])].map(repoPath);
  for (const file of baseTracked) {
    if (!exclusions.some(entry => isWithin(file, entry)))
      copyFile(resolvedBase, resolvedTarget, file, fileSystem);
  }
  pruneBaseRepositoryOverlay(resolvedTarget, manifest, fileSystem);
  const opsTracked = selection.gitFiles;
  assertBusinessPathsDoNotOverwriteBase(opsRoot, resolvedBase, baseTracked, manifest.businessPaths ?? [], opsTracked);
  const copiedBusiness = manifest.businessPaths.flatMap((entry) => copyDeclaredPath(opsRoot, resolvedTarget, entry, opsTracked, fileSystem));
  const copiedRepository = manifest.repositoryOverlayPaths.flatMap((entry) => copyDeclaredPath(opsRoot, resolvedTarget, entry, opsTracked, fileSystem));
  copyFile(opsRoot, resolvedTarget, 'business-overlay.json', fileSystem);
  copiedRepository.push('business-overlay.json');

  const targetModule = readGoModule(resolvedTarget);
  const rewritten = rewriteBusinessGoImports(resolvedTarget, manifest.sourceModule, targetModule, fileSystem);
  const generated = generateRegistries(resolvedTarget, manifest, targetModule, fileSystem);
  generated.push(...projectConsumerReviewRegistry(resolvedBase, resolvedTarget, manifest, baseCommit, fileSystem));
  generated.push(...assembleBusinessMigrations(opsRoot, resolvedTarget, manifest, fileSystem));
  rewritten.push(...rewriteBusinessGoImports(resolvedTarget, manifest.sourceModule, targetModule, fileSystem));
  generated.push(...wireBusinessRegistries(resolvedTarget, fileSystem));
  generated.push(...mergeGoDependencies(opsRoot, resolvedTarget, manifest, fileSystem));
  generateBusinessFallbacks(resolvedTarget, fileSystem,
    businessLocalePlan(resolvedBase, baseTracked, opsRoot, opsTracked, manifest));
  generated.push(...wireBusinessFallbacks(resolvedTarget, fileSystem));
  generated.push(...['frontend/src/i18n/resources/business/zh-CN.ts', 'frontend/src/i18n/resources/business/en-US.ts', 'frontend/src/i18n/resources/business/ja-JP.ts', 'frontend/src/i18n/resources/business/ko-KR.ts', 'frontend/src/i18n/resources/business/fr-FR.ts']);
  mergePackages(resolvedBase, opsRoot, resolvedTarget, manifest, fileSystem);
  generated.push('package.json', 'frontend/package.json');
  generated.push(...wireBusinessSmokeCoverage(opsRoot, resolvedTarget, manifest, fileSystem));
  generated.push(...protectBusinessOverlayFromGeneratedCleanup(resolvedTarget, manifest, fileSystem));
  if (Object.keys(manifest.goDependencies ?? {}).length > 0) {
    generated.push(...normalizeGoDependencies(resolvedTarget, fileSystem));
  }
  fileSystem.assertTargetStable?.();
  const report = writeReport(resolvedTarget, baseCommit, copiedBusiness, copiedRepository, rewritten, generated, fileSystem);
  fileSystem.assertTargetStable?.();
  return { targetRoot: resolvedTarget, report };
}

export function rebuildFromBase({ opsRoot = defaultOpsRoot, baseRoot, targetRoot, manifestPath } = {}) {
  if (baseRoot) throw new Error('Local Base sources are not supported; fetch the official GitHub release');
  return assembleConsumerTree({ opsRoot, targetRoot, manifestPath, snapshot: resolveBaseFromLock(opsRoot) });
}

// Replay only the fixed business integration seams in memory. The report is
// evidence of assembly, never permission to exempt arbitrary generatedFiles.
export function verifyBusinessDerivedOutputs({ root, snapshot, manifest, sourceRoot }) {
  if (!Array.isArray(manifest.backendModules) || !Array.isArray(manifest.frontendModules) ||
      !Array.isArray(manifest.components) || !Array.isArray(manifest.businessSmokeScripts) ||
      !snapshot.baseCommit) return new Set();
  validateBackendModuleEntries(manifest);
  const report = JSON.parse(fs.readFileSync(path.join(root, '.business-overlay-report.json'), 'utf8'));
  if (report.schemaVersion !== 1 || report.baseCommit !== snapshot.baseCommit ||
      !Array.isArray(report.ownedFiles) || !Array.isArray(report.generatedFiles))
    throw new Error('Invalid business assembly report identity');
  const owned = new Map(report.ownedFiles.map(entry => [entry.path, entry.sha256]));
  if (owned.size !== report.ownedFiles.length || owned.get('business-overlay.json') !== fileHash(path.join(root, 'business-overlay.json')))
    throw new Error('Business assembly manifest/report hash mismatch');
  const declared = relative => (manifest.businessPaths ?? []).some(entry => isWithin(relative, repoPath(entry)));
  for (const [entries, directory] of [[manifest.frontendModules, 'frontend/src/modules'], [manifest.components, 'frontend/src/core/router']]) {
    for (const entry of entries) {
      const relative = path.posix.normalize(path.posix.join(directory, entry.importPath));
      if (!/^\.{1,2}\/[A-Za-z0-9_./-]+$/u.test(entry.importPath) ||
          !relative.startsWith('frontend/src/modules/business/') || !declared(relative) ||
          (entry.export !== undefined && !registryIdentifier.test(entry.export)) ||
          (entry.key !== undefined && !/^business\/[A-Za-z0-9_/-]+$/u.test(entry.key)))
        throw new Error('Business registry import must have declared business ownership');
    }
  }
  const base = snapshot.baseRoot;
  const sources = sourceRoot ?? root;
  const frozen = snapshot.replayInputs;
  if (sourceRoot && (!frozen || frozen.root !== sourceRoot))
    throw new Error('Canonical replay requires frozen selected source inputs');
  const baseFiles = snapshotFiles(base);
  const readOps = frozenInputReader(sources, frozen?.captured ??
    captureSelectedInputs(sources, { declarations: [], files: [...owned.keys()].sort() }));
  const readBase = frozenInputReader(base, frozen ? { files: snapshot.manifest.files } :
    captureSelectedInputs(base, { declarations: [], files: baseFiles }));
  const outputs = new Map();
  const io = {
    ...fs,
    mkdirSync() {},
    writeFileSync(file, bytes) { outputs.set(file, Buffer.from(bytes)); },
    readFileSync(file, encoding) {
      let bytes = outputs.get(file);
      if (!bytes) {
        const relative = repoPath(path.relative(sources, file));
        if (path.resolve(file).startsWith(path.resolve(base) + path.sep)) bytes = readBase(file);
        else if (path.resolve(file).startsWith(path.resolve(sources) + path.sep)) bytes = readOps(file);
        else throw new Error('Unselected canonical replay input: ' + file);
        if (!frozen && relative.startsWith('frontend/src/modules/business/') &&
            (!declared(relative) || owned.get(relative) !== crypto.createHash('sha256').update(bytes).digest('hex')))
          throw new Error('Business derivation source/report hash mismatch: ' + relative);
      }
      return encoding ? bytes.toString(encoding) : bytes;
    },
  };
  generateRegistries(base, manifest, readGoModule(base, io), io);
  wireBusinessRegistries(base, io);
  generateBusinessFallbacks(base, io, businessLocalePlan(base, baseFiles, sources,
    frozen ? frozen.selection.gitFiles : [...owned.keys()], manifest));
  wireBusinessFallbacks(base, io);
  protectBusinessOverlayFromGeneratedCleanup(base, manifest, io);
  wireBusinessSmokeCoverage(sources, base, manifest, io);
  // Candidate verification has the frozen original package inputs. The official
  // standalone CLI retains its legacy package allowlist contract.
  if (sourceRoot) mergePackages(base, sources, base, manifest, io);
  const verified = new Set();
  for (const [absolute, bytes] of outputs) {
    const relative = repoPath(path.relative(base, absolute));
    const actual = path.join(root, relative);
    if (!report.generatedFiles.includes(relative) || !fs.existsSync(actual) ||
        !fs.lstatSync(actual).isFile() || !fs.readFileSync(actual).equals(bytes))
      throw new Error('Derived business output differs from canonical assembler: ' + relative);
    if (owned.get(relative) !== fileHash(actual)) throw new Error('Business output/report hash mismatch: ' + relative);
    verified.add(relative);
  }
  return verified;
}

function argument(name) {
  const index = process.argv.indexOf(name);
  return index >= 0 ? process.argv[index + 1] : undefined;
}

if (process.argv[1] && path.resolve(process.argv[1]) === currentFile) {
  if (process.argv.includes('--base')) throw new Error('--base is no longer supported; fetch the official GitHub release');
  const result = rebuildFromBase({ targetRoot: argument('--target'), manifestPath: argument('--manifest') });
  console.log(`OK rebuilt clean Ops tree at ${result.targetRoot}`);
  console.log(`Base commit: ${result.report.baseCommit}`);
  console.log(`Declared/generated Ops files: ${result.report.ownedFiles.length}`);
}

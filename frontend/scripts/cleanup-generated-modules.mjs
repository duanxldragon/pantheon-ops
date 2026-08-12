import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';
import { execFileSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';

const repoRoot = path.resolve(import.meta.dirname, '..', '..');

const GENERATED_PATHS = {
  backendBusinessDir: path.join(repoRoot, 'backend', 'modules', 'business'),
  frontendBusinessDir: path.join(repoRoot, 'frontend', 'src', 'modules', 'business'),
  schemaBusinessDir: path.join(repoRoot, 'schema', 'generated', 'business'),
  featureLedger: path.join(repoRoot, 'schema', 'generated', 'feature-ledger.json'),
  i18nDir: path.join(repoRoot, 'frontend', 'src', 'i18n', 'resources', 'generated'),
};

const REGISTRY_FILES = {
  backendRegistry: path.join(repoRoot, 'backend', 'modules', 'business', 'generated_registry.go'),
  backendMenuRegistry: path.join(
    repoRoot,
    'backend',
    'modules',
    'system',
    'iam',
    'menu',
    'generated_component_registry.go',
  ),
  frontendBusinessRegistry: path.join(repoRoot, 'frontend', 'src', 'modules', 'generated', 'business.ts'),
  frontendComponentRegistry: path.join(repoRoot, 'frontend', 'src', 'core', 'router', 'generatedComponentRegistry.ts'),
};

const REGISTRY_TEMPLATES = {
  backendRegistry: [
    'package business',
    '',
    'import (',
    '\t"github.com/gin-gonic/gin"',
    '\t"gorm.io/gorm"',
    ')',
    '',
    'func InitGeneratedBusinessModules(r *gin.RouterGroup, db *gorm.DB) {',
    '\t// Intentionally empty: the low-code module generator rewrites this file and',
    '\t// fills in registrations once generated business modules exist.',
    '}',
    '',
  ].join('\n'),

  backendMenuRegistry: [
    'package iam',
    '',
    'var generatedMenuComponentKeys = map[string]struct{}{}',
    '',
  ].join('\n'),

  frontendBusinessRegistry: [
    "import type { ModuleConfig } from '../../core/router/types';",
    '',
    'export const generatedBusinessModules: ModuleConfig[] = [];',
    '',
  ].join('\n'),

  frontendComponentRegistry: [
    "import { type LazyExoticComponent, type ComponentType } from 'react';",
    '',
    'type ComponentLoader = () => Promise<{ default: ComponentType }>;',
    '',
    'interface RegistryEntry {',
    '  component: LazyExoticComponent<ComponentType>;',
    '  preload: ComponentLoader;',
    '}',
    '',
    'export const generatedComponentRegistry = {} satisfies Record<string, RegistryEntry>;',
    '',
  ].join('\n'),
};

const I18N_LOCALES = ['zh-CN', 'en-US', 'ko-KR', 'ja-JP', 'fr-FR'];

function i18nTemplate(variableName) {
  return [`const ${variableName} = {};`, '', `export default ${variableName};`, ''].join('\n');
}

function removeDir(dir) {
  if (fs.existsSync(dir)) {
    fs.rmSync(dir, { recursive: true, force: true });
    return true;
  }
  return false;
}

function normalizePath(targetPath) {
  return targetPath.replaceAll('\\', '/');
}

function readTrackedFiles(repoBase) {
  const output = execFileSync('git', ['-C', repoBase, 'ls-files', '-z'], {
    encoding: 'utf8',
    stdio: ['ignore', 'pipe', 'ignore'],
  });
  return new Set(output.split('\0').filter(Boolean).map(normalizePath));
}

function hasTrackedDescendant(targetPath, repoBase, trackedFiles) {
  const relative = normalizePath(path.relative(repoBase, targetPath));
  const prefix = `${relative}/`;
  return Array.from(trackedFiles).some((trackedPath) => trackedPath.startsWith(prefix));
}

function removeGeneratedSubdirs(parentDir, repoBase, trackedFiles) {
  let removed = 0;
  if (!fs.existsSync(parentDir)) {
    return removed;
  }
  for (const entry of fs.readdirSync(parentDir, { withFileTypes: true })) {
    if (entry.isDirectory()) {
      const targetPath = path.join(parentDir, entry.name);
      if (hasTrackedDescendant(targetPath, repoBase, trackedFiles)) {
        removed += removeGeneratedSubdirs(targetPath, repoBase, trackedFiles);
      } else {
        removeDir(targetPath);
        removed++;
      }
    }
  }
  return removed;
}

function relativePath(repoBase, targetPath) {
  return path.relative(repoBase, targetPath).replaceAll('\\', '/');
}

function removeFilesByGlob(dir, pattern, repoBase, trackedFiles) {
  let removed = 0;
  if (!fs.existsSync(dir)) {
    return removed;
  }
  const re = new RegExp(pattern);
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const targetPath = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      removed += removeFilesByGlob(targetPath, pattern, repoBase, trackedFiles);
    } else if (entry.isFile() && re.test(entry.name)) {
      const relative = normalizePath(path.relative(repoBase, targetPath));
      if (trackedFiles.has(relative)) {
        continue;
      }
      try {
        fs.rmSync(targetPath, { force: true });
      } catch (error) {
        if (error && typeof error === 'object' && 'code' in error && error.code === 'ENOENT') {
          continue;
        }
        throw error;
      }
      removed++;
    }
  }
  return removed;
}

function writeFile(filePath, content) {
  fs.mkdirSync(path.dirname(filePath), { recursive: true });
  if (fs.existsSync(filePath)) {
    const current = fs.readFileSync(filePath, 'utf8');
    if (current === content) {
      return;
    }
  }
  const dir = path.dirname(filePath);
  const tempPath = path.join(dir, `.${path.basename(filePath)}.${process.pid}.${Date.now()}.tmp`);
  fs.writeFileSync(tempPath, content, 'utf8');
  try {
    fs.rmSync(filePath, { force: true });
    fs.renameSync(tempPath, filePath);
  } finally {
    fs.rmSync(tempPath, { force: true });
  }
}

function hasAnyPattern(content, patterns) {
  return patterns.some((pattern) => pattern.test(content));
}

function appendDirtyIfFileMatches(dirty, filePath, message, patterns) {
  if (!fs.existsSync(filePath)) {
    return;
  }
  const content = fs.readFileSync(filePath, 'utf8');
  if (hasAnyPattern(content, patterns)) {
    dirty.push(message);
  }
}

function appendDirtyDirectories(dirty, parentDir, repoBase, label, trackedFiles) {
  if (!fs.existsSync(parentDir)) {
    return;
  }
  for (const entry of fs.readdirSync(parentDir, { withFileTypes: true })) {
    if (entry.isDirectory()) {
      const targetPath = path.join(parentDir, entry.name);
      if (hasTrackedDescendant(targetPath, repoBase, trackedFiles)) {
        appendDirtyDirectories(dirty, targetPath, repoBase, label, trackedFiles);
      } else {
        dirty.push(`${label}: ${relativePath(repoBase, targetPath)}`);
      }
    }
  }
}

function appendDirtyGeneratedSchemaFiles(dirty, schemaDir, repoBase, trackedFiles) {
  if (!fs.existsSync(schemaDir)) {
    return;
  }
  for (const entry of fs.readdirSync(schemaDir, { withFileTypes: true })) {
    const targetPath = path.join(schemaDir, entry.name);
    if (entry.isDirectory()) {
      appendDirtyGeneratedSchemaFiles(dirty, targetPath, repoBase, trackedFiles);
    } else if (entry.isFile()) {
      if (!trackedFiles.has(normalizePath(path.relative(repoBase, targetPath)))) {
        dirty.push(`generated schema file still present: ${relativePath(repoBase, targetPath)}`);
      }
    }
  }
}

export function checkDirty(
  paths = GENERATED_PATHS,
  registryFiles = REGISTRY_FILES,
  repoBase = repoRoot,
  trackedFiles = readTrackedFiles(repoBase),
) {
  const dirty = [];
  const featureLedgerRelative = normalizePath(path.relative(repoBase, paths.featureLedger));
  if (trackedFiles.has(featureLedgerRelative)) {
    const featureLedgerBaseline = readIndexBaseline(paths.featureLedger, repoBase, true);
    const featureLedgerCurrent = fs.existsSync(paths.featureLedger)
      ? fs.readFileSync(paths.featureLedger, 'utf8')
      : null;
    if (featureLedgerCurrent !== featureLedgerBaseline) {
      dirty.push('feature ledger differs from tracked baseline');
    }
  }

  appendDirtyIfFileMatches(
    dirty,
    registryFiles.backendRegistry,
    'backend generated_registry.go: has generated module imports',
    [/mdqaorder/, /mdqaorderitem/, /"pantheon-base\/modules\/business\//],
  );
  appendDirtyIfFileMatches(
    dirty,
    registryFiles.frontendBusinessRegistry,
    'frontend generated/business.ts: has generated module imports',
    [/Mdqaorder/, /Mdqaorderitem/, /from\s+['"]\.\.\/business\//, /business\/mdqa/i],
  );
  appendDirtyIfFileMatches(
    dirty,
    registryFiles.frontendComponentRegistry,
    'frontend generatedComponentRegistry.ts: has generated component entries',
    [/business\/mdqa/],
  );
  appendDirtyIfFileMatches(
    dirty,
    registryFiles.backendMenuRegistry,
    'backend generated_component_registry.go: has generated component keys',
    [/business\/mdqa/],
  );

  for (const locale of I18N_LOCALES) {
    appendDirtyIfFileMatches(
      dirty,
      path.join(paths.i18nDir, `${locale}.ts`),
      `i18n ${locale}: contains generated keys`,
      [/business\.mdqa/],
    );
  }

  for (const dir of [paths.backendBusinessDir, paths.frontendBusinessDir]) {
    appendDirtyDirectories(dirty, dir, repoBase, 'generated module dir still present', trackedFiles);
  }

  appendDirtyDirectories(
    dirty,
    paths.schemaBusinessDir,
    repoBase,
    'generated schema dir still present',
    trackedFiles,
  );
  appendDirtyGeneratedSchemaFiles(dirty, paths.schemaBusinessDir, repoBase, trackedFiles);

  return dirty;
}

function readIndexBaseline(filePath, repoBase, required = false) {
  const relative = normalizePath(path.relative(repoBase, filePath));
  try {
    return execFileSync('git', ['-C', repoBase, 'show', `:${relative}`], {
      encoding: 'utf8',
      stdio: ['ignore', 'pipe', 'ignore'],
    });
  } catch {
    if (required) {
      throw new Error(`Unable to read tracked cleanup baseline from Git index: ${relative}`);
    }
    return null;
  }
}

export function cleanup(
  paths = GENERATED_PATHS,
  registryFiles = REGISTRY_FILES,
  repoBase = repoRoot,
  trackedFiles = readTrackedFiles(repoBase),
) {
  const summary = { modules: 0, schemas: 0, registries: 0, i18n: 0 };

  const backendRemoved = removeGeneratedSubdirs(paths.backendBusinessDir, repoBase, trackedFiles);
  const frontendRemoved = removeGeneratedSubdirs(paths.frontendBusinessDir, repoBase, trackedFiles);
  const schemaRemoved = removeGeneratedSubdirs(paths.schemaBusinessDir, repoBase, trackedFiles);
  summary.modules = backendRemoved + frontendRemoved + schemaRemoved;

  summary.schemas = removeFilesByGlob(
    paths.schemaBusinessDir,
    String.raw`\.json$`,
    repoBase,
    trackedFiles,
  );

  const featureLedgerRelative = normalizePath(path.relative(repoBase, paths.featureLedger));
  const featureLedgerBaseline = readIndexBaseline(
    paths.featureLedger,
    repoBase,
    trackedFiles.has(featureLedgerRelative),
  );
  if (featureLedgerBaseline !== null) {
    writeFile(paths.featureLedger, featureLedgerBaseline);
    summary.schemas++;
  }

  for (const [key, filePath] of Object.entries(registryFiles)) {
    const relative = normalizePath(path.relative(repoBase, filePath));
    const baseline = readIndexBaseline(filePath, repoBase, trackedFiles.has(relative))
      ?? REGISTRY_TEMPLATES[key];
    if (baseline) {
      writeFile(filePath, baseline);
      summary.registries++;
    }
  }

  const i18nVarNames = {
    'zh-CN': 'generatedzhCNFallback',
    'en-US': 'generatedenUSFallback',
    'ko-KR': 'generatedkoKRFallback',
    'ja-JP': 'generatedjaJPFallback',
    'fr-FR': 'generatedfrFRFallback',
  };

  for (const locale of I18N_LOCALES) {
    const filePath = path.join(paths.i18nDir, `${locale}.ts`);
    const varName = i18nVarNames[locale];
    if (varName) {
      const relative = normalizePath(path.relative(repoBase, filePath));
      writeFile(
        filePath,
        readIndexBaseline(filePath, repoBase, trackedFiles.has(relative)) ?? i18nTemplate(varName),
      );
      summary.i18n++;
    }
  }

  console.info('[generated-modules] cleanup complete');
  console.info(JSON.stringify(summary, null, 2));
}

function main(argv = process.argv.slice(2)) {
  const mode = argv.includes('--check') ? 'check' : 'cleanup';

  if (mode === 'check') {
    const dirty = checkDirty();
    if (dirty.length > 0) {
      console.error('[generated-modules] FAIL: smoke-test generated files detected');
      for (const item of dirty) {
        console.error(`  - ${item}`);
      }
      console.error('');
      console.error('Run: node frontend/scripts/cleanup-generated-modules.mjs');
      process.exit(1);
    }
    console.info('[generated-modules] OK: no generated modules found');
    return;
  }

  cleanup();
}

const currentFilePath = fileURLToPath(import.meta.url);
if (process.argv[1] && path.resolve(process.argv[1]) === currentFilePath) {
  main();
}

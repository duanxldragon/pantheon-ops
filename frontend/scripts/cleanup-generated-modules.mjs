import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';
import { generateModuleI18n } from './generate-module-i18n.mjs';

const repoRoot = path.resolve(import.meta.dirname, '..', '..');

const GENERATED_PATHS = {
  backendBusinessDir: path.join(repoRoot, 'backend', 'modules', 'business'),
  frontendBusinessDir: path.join(repoRoot, 'frontend', 'src', 'modules', 'business'),
  schemaBusinessDir: path.join(repoRoot, 'schema', 'generated', 'business'),
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
    '}',
    '',
  ].join('\n'),

  backendMenuRegistry: [
    'package iam',
    '',
    'var generatedMenuComponentKeys = map[string]struct{}{',
    '}',
    '',
  ].join('\n'),

  frontendBusinessRegistry: [
    "import type { ModuleConfig } from '../../core/router/types';",
    '',
    'export const generatedBusinessModules: ModuleConfig[] = [',
    '];',
    '',
  ].join('\n'),

  frontendComponentRegistry: [
    "import type { ComponentType, LazyExoticComponent } from 'react';",
    '',
    'type ComponentLoader = () => Promise<{ default: ComponentType }>;',
    '',
    'interface RegistryEntry {',
    '\tcomponent: LazyExoticComponent<ComponentType>;',
    '\tpreload: ComponentLoader;',
    '}',
    '',
    'export const generatedComponentRegistry = {',
    '} satisfies Record<string, RegistryEntry>;',
    '',
  ].join('\n'),
};

const I18N_LOCALES = ['zh-CN', 'en-US', 'ko-KR', 'ja-JP', 'fr-FR'];

function removeDir(dir) {
  if (fs.existsSync(dir)) {
    fs.rmSync(dir, { recursive: true, force: true });
    return true;
  }
  return false;
}

function writeFile(filePath, content) {
  fs.mkdirSync(path.dirname(filePath), { recursive: true });
  fs.writeFileSync(filePath, content, 'utf8');
}

function checkDirty() {
  const dirty = [];

  // Check backend generated_registry.go — should only import gin and gorm
  if (fs.existsSync(REGISTRY_FILES.backendRegistry)) {
    const content = fs.readFileSync(REGISTRY_FILES.backendRegistry, 'utf8');
    const importMatch = content.match(/import\s*\(([\s\S]*?)\)/);
    if (importMatch) {
      const imports = importMatch[1];
      if (/mdqaorder|mdqaorderitem|"pantheon-platform\/backend\/modules\/business\/mdqa/.test(imports)) {
        dirty.push('backend generated_registry.go: has generated module imports');
      }
    }
  }

  // Check frontend generated/business.ts — should have empty array
  if (fs.existsSync(REGISTRY_FILES.frontendBusinessRegistry)) {
    const content = fs.readFileSync(REGISTRY_FILES.frontendBusinessRegistry, 'utf8');
    if (/Mdqaorder|Mdqaorderitem|mdqa/i.test(content)) {
      dirty.push('frontend generated/business.ts: has generated module imports');
    }
  }

  // Check frontend generatedComponentRegistry.ts — should have empty object
  if (fs.existsSync(REGISTRY_FILES.frontendComponentRegistry)) {
    const content = fs.readFileSync(REGISTRY_FILES.frontendComponentRegistry, 'utf8');
    if (/business\/mdqa/.test(content)) {
      dirty.push('frontend generatedComponentRegistry.ts: has generated component entries');
    }
  }

  // Check backend generated_component_registry.go — should have empty map
  if (fs.existsSync(REGISTRY_FILES.backendMenuRegistry)) {
    const content = fs.readFileSync(REGISTRY_FILES.backendMenuRegistry, 'utf8');
    if (/business\/mdqa/.test(content)) {
      dirty.push('backend generated_component_registry.go: has generated component keys');
    }
  }

  // Check i18n files for generated keys
  for (const locale of I18N_LOCALES) {
    const filePath = path.join(GENERATED_PATHS.i18nDir, `${locale}.ts`);
    if (fs.existsSync(filePath)) {
      const content = fs.readFileSync(filePath, 'utf8');
      if (/business\.mdqa/.test(content)) {
        dirty.push(`i18n ${locale}: contains generated keys`);
      }
    }
  }

  // Check for leftover generated module directories (by name pattern)
  for (const dir of [GENERATED_PATHS.backendBusinessDir, GENERATED_PATHS.frontendBusinessDir]) {
    if (!fs.existsSync(dir)) continue;
    for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
      if (entry.isDirectory() && /^mdqa/.test(entry.name)) {
        dirty.push(`generated module dir still present: ${path.relative(repoRoot, path.join(dir, entry.name))}`);
      }
    }
  }

  // Check for leftover schema files
  if (fs.existsSync(GENERATED_PATHS.schemaBusinessDir)) {
    for (const entry of fs.readdirSync(GENERATED_PATHS.schemaBusinessDir, { withFileTypes: true })) {
      if (entry.isFile() && /^mdqa/.test(entry.name)) {
        dirty.push(`generated schema file still present: ${path.relative(repoRoot, path.join(GENERATED_PATHS.schemaBusinessDir, entry.name))}`);
      }
    }
  }

  return dirty;
}

function collectGeneratedBusinessModules() {
  const modules = new Map();
  if (!fs.existsSync(GENERATED_PATHS.schemaBusinessDir)) {
    return modules;
  }

  const visit = (dirPath) => {
    for (const entry of fs.readdirSync(dirPath, { withFileTypes: true })) {
      const nextPath = path.join(dirPath, entry.name);
      if (entry.isDirectory()) {
        visit(nextPath);
        continue;
      }
      if (!entry.name.endsWith('.json')) {
        continue;
      }

      let schema;
      try {
        schema = JSON.parse(fs.readFileSync(nextPath, 'utf8'));
      } catch {
        continue;
      }
      const name = String(schema?.name ?? '').trim().replaceAll('\\', '/');
      const segments = name.split('/');
      if (!name || segments.some((segment) => !/^[a-z][a-z0-9_]*$/u.test(segment))) {
        continue;
      }
      modules.set(name, nextPath);
    }
  };

  visit(GENERATED_PATHS.schemaBusinessDir);
  return modules;
}

function cleanup() {
  const summary = { modules: 0, schemas: 0, registries: 0, i18n: 0 };
  const generatedModules = collectGeneratedBusinessModules();

  // 1. Remove only modules backed by generated schemas. Hand-written business
  // modules share these roots and must survive smoke cleanup.
  for (const [name, schemaPath] of generatedModules) {
    for (const modulesRoot of [
      GENERATED_PATHS.backendBusinessDir,
      GENERATED_PATHS.frontendBusinessDir,
    ]) {
      if (removeDir(path.join(modulesRoot, ...name.split('/')))) {
        summary.modules++;
      }
    }
    fs.unlinkSync(schemaPath);
    summary.schemas++;
  }

  // 2. Reset dynamic-only registry files.
  for (const [key, filePath] of Object.entries(REGISTRY_FILES)) {
    const template = REGISTRY_TEMPLATES[key];
    if (template) {
      writeFile(filePath, template);
      summary.registries++;
    }
  }

  // 3. Rebuild fallbacks from hand-written module locales after generated
  // schemas and source directories have been removed.
  const i18nResult = generateModuleI18n({ checkOnly: false });
  summary.i18n = i18nResult.changes.length;

  console.info('[generated-modules] cleanup complete');
  console.info(JSON.stringify(summary, null, 2));
}

const mode = process.argv.includes('--check') ? 'check' : 'cleanup';

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
} else {
  cleanup();
}

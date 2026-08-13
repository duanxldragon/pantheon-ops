import fs from 'node:fs';
import path from 'node:path';
import vm from 'node:vm';
import { fileURLToPath, pathToFileURL } from 'node:url';

const currentFilePath = fileURLToPath(import.meta.url);
const frontendRoot = path.resolve(path.dirname(currentFilePath), '..');
const repoRoot = path.resolve(frontendRoot, '..');
const schemaGeneratedRoot = path.join(repoRoot, 'schema', 'generated');
const generatedResourcesRoot = path.join(frontendRoot, 'src', 'i18n', 'resources', 'generated');
const LOCALES = ['zh-CN', 'en-US', 'ja-JP', 'ko-KR', 'fr-FR'];

function resolveImportedModulePath(modulePath, specifier) {
  const basePath = path.resolve(path.dirname(modulePath), specifier);
  const candidates = [basePath, `${basePath}.ts`, `${basePath}.js`, `${basePath}.mjs`];
  for (const candidate of candidates) {
    if (fs.existsSync(candidate)) {
      return candidate;
    }
  }

  throw new Error(`Unable to resolve imported module '${specifier}' from ${modulePath}`);
}

function loadResourceModule(modulePath, cache = new Map()) {
  const resolvedModulePath = path.resolve(modulePath);
  if (cache.has(resolvedModulePath)) {
    return cache.get(resolvedModulePath);
  }

  const source = fs.readFileSync(resolvedModulePath, 'utf8');
  const sanitized = source
    .replace(
      /^import\s+([A-Za-z0-9_$]+)\s+from\s+['"](.+?)['"];?\s*$/gm,
      (_, binding, specifier) => `const ${binding} = __loadResourceModule(${JSON.stringify(specifier)});`,
    )
    .replace(/export default\s+([A-Za-z0-9_$]+);?\s*$/m, 'module.exports = $1;');
  const context = {
    module: { exports: {} },
    exports: {},
    __loadResourceModule: (specifier) => loadResourceModule(resolveImportedModulePath(resolvedModulePath, specifier), cache),
  };
  vm.runInNewContext(sanitized, context, { filename: resolvedModulePath }); // NOSONAR — build-only script, controlled source
  cache.set(resolvedModulePath, context.module.exports);
  return context.module.exports;
}

function collectGeneratedSchemaKeys() {
  const allowedKeys = new Set();
  if (!fs.existsSync(schemaGeneratedRoot)) {
    return allowedKeys;
  }

  const pending = [schemaGeneratedRoot];
  while (pending.length > 0) {
    const currentDir = pending.pop();
    for (const entry of fs.readdirSync(currentDir, { withFileTypes: true })) {
      const fullPath = path.join(currentDir, entry.name);
      if (entry.isDirectory()) {
        pending.push(fullPath);
        continue;
      }
      if (!entry.isFile() || path.extname(entry.name).toLowerCase() !== '.json') {
        continue;
      }
      const raw = fs.readFileSync(fullPath, 'utf8');
      const schema = JSON.parse(raw);
      const translations = schema?.i18n?.translations;
      for (const locale of ['zh', 'en']) {
        for (const key of Object.keys(translations?.[locale] || {})) {
          if (key.trim()) {
            allowedKeys.add(key);
          }
        }
      }
    }
  }

  return allowedKeys;
}

function main() {
  const allowedKeys = collectGeneratedSchemaKeys();
  const violations = [];

  for (const locale of LOCALES) {
    const resourcePath = path.join(generatedResourcesRoot, `${locale}.ts`);
    if (!fs.existsSync(resourcePath)) {
      continue;
    }
    const resource = loadResourceModule(resourcePath);
    for (const key of Object.keys(resource)) {
      if (!allowedKeys.has(key)) {
        violations.push({ locale, key });
      }
    }
  }

  if (violations.length === 0) {
    console.log('Generated i18n scope check passed.');
    return;
  }

  console.error('Generated i18n files contain keys that do not belong to schema/generated.');
  console.error('Move these keys into base or override resources instead of frontend/src/i18n/resources/generated/.');
  violations.slice(0, 50).forEach(({ locale, key }) => {
    console.error(`  - [${locale}] ${key}`);
  });
  if (violations.length > 50) {
    console.error(`  ... ${violations.length - 50} more`);
  }
  process.exitCode = 1;
}

export { loadResourceModule };

if (process.argv[1] && import.meta.url === pathToFileURL(path.resolve(process.argv[1])).href) {
  main();
}

import path from 'node:path';
import fs from 'node:fs';
import vm from 'node:vm';
import { fileURLToPath } from 'node:url';

const currentFilePath = fileURLToPath(import.meta.url);
const frontendRoot = path.resolve(path.dirname(currentFilePath), '..');
const resourcesRoot = path.join(frontendRoot, 'src', 'i18n', 'resources');
const generatedResourcesRoot = path.join(resourcesRoot, 'generated');
const outputPath = path.resolve(frontendRoot, '..', 'backend', 'modules', 'system', 'i18n', 'builtin_locale_resources.json');

const LOCALES = ['zh-CN', 'en-US', 'ja-JP', 'ko-KR', 'fr-FR'];

function loadResourceModule(modulePath, cache = new Map()) {
  const resolvedPath = path.resolve(modulePath);
  if (cache.has(resolvedPath)) {
    return cache.get(resolvedPath);
  }

  const source = fs.readFileSync(resolvedPath, 'utf8');
  const importMatches = [...source.matchAll(/import\s+([A-Za-z0-9_$]+)\s+from\s+['"](.+?)['"];?/g)];
  const importedBindings = {};

  for (const [, localName, specifier] of importMatches) {
    const nextPath = path.resolve(path.dirname(resolvedPath), `${specifier}.ts`);
    importedBindings[localName] = loadResourceModule(nextPath, cache);
  }

  const sanitized = source
    .replace(/import\s+[A-Za-z0-9_$]+\s+from\s+['"].+?['"];?\s*/g, '')
    .replace(/export default\s+([A-Za-z0-9_$]+);?\s*$/m, 'module.exports = $1;');

  const context = {
    module: { exports: {} },
    exports: {},
    ...importedBindings,
  };

  vm.runInNewContext(sanitized, context, { filename: resolvedPath });
  cache.set(resolvedPath, context.module.exports);
  return context.module.exports;
}

function loadLocale(locale) {
  const base = loadResourceModule(path.join(resourcesRoot, `${locale}.ts`));
  const generatedPath = path.join(generatedResourcesRoot, `${locale}.ts`);
  const generated = fs.existsSync(generatedPath) ? loadResourceModule(generatedPath) : {};
  return {
    ...base,
    ...generated,
  };
}

const snapshot = Object.fromEntries(
  LOCALES.map((locale) => [locale, loadLocale(locale)]),
);

const serialized = `${JSON.stringify(snapshot, null, 2)}\n`;

if (process.argv.includes('--stdout')) {
  process.stdout.write(serialized);
} else {
  const tempOutputPath = `${outputPath}.tmp`;
  fs.writeFileSync(tempOutputPath, serialized, 'utf8');
  fs.renameSync(tempOutputPath, outputPath);
  console.log(`Exported built-in i18n snapshot to ${outputPath}`);
}

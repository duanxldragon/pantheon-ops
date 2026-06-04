import fs from 'node:fs';
import { pathToFileURL } from 'node:url';
import path from 'node:path';
import process from 'node:process';
import { fileURLToPath } from 'node:url';
import { loadResourceModule } from './lib/load-resource-module.mjs';

const currentFilePath = fileURLToPath(import.meta.url);
const frontendRoot = path.resolve(path.dirname(currentFilePath), '..');
const modulesRoot = path.join(frontendRoot, 'src', 'modules');
const generatedResourcesRoot = path.join(frontendRoot, 'src', 'i18n', 'resources', 'generated');
const locales = ['zh-CN', 'en-US', 'ja-JP', 'ko-KR', 'fr-FR'];
const checkOnly = process.argv.includes('--check');

function walkLocaleFiles(dirPath, bucket = []) {
  for (const entry of fs.readdirSync(dirPath, { withFileTypes: true })) {
    const nextPath = path.join(dirPath, entry.name);
    if (entry.isDirectory()) {
      walkLocaleFiles(nextPath, bucket);
      continue;
    }
    if (entry.name.endsWith('.json') && path.basename(path.dirname(nextPath)) === 'locales') {
      bucket.push(nextPath);
    }
  }
  return bucket;
}

function readLocaleJson(filePath) {
  try {
    return JSON.parse(fs.readFileSync(filePath, 'utf8'));
  } catch (error) {
    throw new Error(`Invalid locale JSON ${path.relative(frontendRoot, filePath)}: ${error.message}`);
  }
}

function parseGeneratedResource(filePath) {
  if (!fs.existsSync(filePath)) {
    return {};
  }
  try {
    return loadResourceModule(filePath);
  } catch (error) {
    throw new Error(`Invalid generated locale resource ${path.relative(frontendRoot, filePath)}: ${error.message}`);
  }
}

function loadExistingGeneratedResources(rootDir = generatedResourcesRoot, activeLocales = locales) {
  return Object.fromEntries(
    activeLocales.map((locale) => [
      locale,
      parseGeneratedResource(path.join(rootDir, `${locale}.ts`)),
    ]),
  );
}

export function loadModuleLocales(options = {}) {
  const activeLocales = options.locales ?? locales;
  const activeModulesRoot = options.modulesRoot ?? modulesRoot;
  const activeGeneratedRoot = options.generatedResourcesRoot ?? generatedResourcesRoot;
  const resources = Object.fromEntries(activeLocales.map((locale) => [locale, {}]));
  const existingResources = loadExistingGeneratedResources(activeGeneratedRoot, activeLocales);
  const files = walkLocaleFiles(activeModulesRoot).sort((left, right) => left.localeCompare(right));
  const allKeys = new Set();

  for (const filePath of files) {
    const locale = path.basename(filePath, '.json');
    if (!activeLocales.includes(locale)) {
      continue;
    }
    const payload = readLocaleJson(filePath);
    for (const [key, value] of Object.entries(payload)) {
      if (typeof value !== 'string') {
        throw new Error(
          `Locale value must be string: ${path.relative(activeModulesRoot, filePath)} -> ${key}`,
        );
      }
      resources[locale][key] = value;
      allKeys.add(key);
    }
  }

  const englishFallback = resources['en-US'];
  const chineseFallback = resources['zh-CN'];
  for (const key of allKeys) {
    for (const locale of activeLocales) {
      if (Object.prototype.hasOwnProperty.call(resources[locale], key)) {
        continue;
      }
      const existingValue = existingResources[locale]?.[key];
      if (typeof existingValue === 'string' && existingValue.trim() !== '') {
        resources[locale][key] = existingValue;
        continue;
      }
      const englishValue = englishFallback?.[key];
      if (typeof englishValue === 'string') {
        resources[locale][key] = englishValue;
        continue;
      }
      const chineseValue = chineseFallback?.[key];
      if (typeof chineseValue === 'string') {
        resources[locale][key] = chineseValue;
      }
    }
  }

  return resources;
}

function variableNameForLocale(locale) {
  return `generated${locale.replace(/[^A-Za-z0-9]/g, '')}Fallback`;
}

function serializeResource(locale, resource) {
  const variableName = variableNameForLocale(locale);
  const entries = Object.entries(resource).sort(([left], [right]) => left.localeCompare(right));
  const body = entries.length
    ? entries
        .map(([key, value]) => `  ${JSON.stringify(key)}: ${JSON.stringify(value)},`)
        .join('\n')
    : '';
  return `const ${variableName} = {\n${body}${body ? '\n' : ''}};\n\nexport default ${variableName};\n`;
}

export function generateModuleI18n(options = {}) {
  const activeLocales = options.locales ?? locales;
  const activeGeneratedRoot = options.generatedResourcesRoot ?? generatedResourcesRoot;
  const resources = loadModuleLocales(options);
  fs.mkdirSync(activeGeneratedRoot, { recursive: true });
  const changes = [];

  for (const locale of activeLocales) {
    const nextContent = serializeResource(locale, resources[locale]);
    const outputPath = path.join(activeGeneratedRoot, `${locale}.ts`);
    const currentContent = fs.existsSync(outputPath) ? fs.readFileSync(outputPath, 'utf8') : '';

    if (currentContent !== nextContent) {
      changes.push(path.relative(frontendRoot, outputPath));
      if (!(options.checkOnly ?? checkOnly)) {
        fs.writeFileSync(outputPath, nextContent, 'utf8');
      }
    }
  }

  if ((options.checkOnly ?? checkOnly) && changes.length > 0) {
    console.error('Generated i18n resources are out of date:');
    changes.forEach((item) => console.error(`  ${item}`));
    console.error('Run `npm run i18n:generate-module` from frontend.');
    process.exit(1);
  }

  const totalKeys = Object.values(resources).reduce((sum, item) => sum + Object.keys(item).length, 0);
  console.log(`Generated module i18n resources for ${activeLocales.length} locales, ${totalKeys} keys.`);
  return { resources, changes, totalKeys };
}

function main() {
  generateModuleI18n();
}

if (import.meta.url === pathToFileURL(process.argv[1] || '').href) {
  main();
}

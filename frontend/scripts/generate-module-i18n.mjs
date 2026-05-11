import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';
import { fileURLToPath } from 'node:url';

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

function loadModuleLocales() {
  const resources = Object.fromEntries(locales.map((locale) => [locale, {}]));
  const files = walkLocaleFiles(modulesRoot).sort((left, right) => left.localeCompare(right));

  for (const filePath of files) {
    const locale = path.basename(filePath, '.json');
    if (!locales.includes(locale)) {
      continue;
    }
    const payload = readLocaleJson(filePath);
    for (const [key, value] of Object.entries(payload)) {
      if (typeof value !== 'string') {
        throw new Error(`Locale value must be string: ${path.relative(frontendRoot, filePath)} -> ${key}`);
      }
      resources[locale][key] = value;
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

function main() {
  const resources = loadModuleLocales();
  fs.mkdirSync(generatedResourcesRoot, { recursive: true });
  const changes = [];

  for (const locale of locales) {
    const nextContent = serializeResource(locale, resources[locale]);
    const outputPath = path.join(generatedResourcesRoot, `${locale}.ts`);
    const currentContent = fs.existsSync(outputPath) ? fs.readFileSync(outputPath, 'utf8') : '';

    if (currentContent !== nextContent) {
      changes.push(path.relative(frontendRoot, outputPath));
      if (!checkOnly) {
        fs.writeFileSync(outputPath, nextContent, 'utf8');
      }
    }
  }

  if (checkOnly && changes.length > 0) {
    console.error('Generated i18n resources are out of date:');
    changes.forEach((item) => console.error(`  ${item}`));
    console.error('Run `npm run i18n:generate-module` from frontend.');
    process.exit(1);
  }

  const totalKeys = Object.values(resources).reduce((sum, item) => sum + Object.keys(item).length, 0);
  console.log(`Generated module i18n resources for ${locales.length} locales, ${totalKeys} keys.`);
}

main();

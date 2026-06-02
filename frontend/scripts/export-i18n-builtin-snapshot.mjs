import path from 'node:path';
import fs from 'node:fs';
import { fileURLToPath } from 'node:url';
import { loadResourceModule } from './lib/load-resource-module.mjs';

const currentFilePath = fileURLToPath(import.meta.url);
const frontendRoot = path.resolve(path.dirname(currentFilePath), '..');
const resourcesRoot = path.join(frontendRoot, 'src', 'i18n', 'resources');
const generatedResourcesRoot = path.join(resourcesRoot, 'generated');
const outputPath = path.resolve(frontendRoot, '..', 'backend', 'modules', 'system', 'i18n', 'builtin_locale_resources.json');

const LOCALES = ['zh-CN', 'en-US', 'ja-JP', 'ko-KR', 'fr-FR'];

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

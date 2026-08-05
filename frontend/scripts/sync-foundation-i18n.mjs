import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';
import { fileURLToPath, pathToFileURL } from 'node:url';
import { resolveFoundationReleasePaths } from '../../scripts/foundation-release/shared-foundation-rules.mjs';
import { loadResourceModule } from './lib/load-resource-module.mjs';

const currentFilePath = fileURLToPath(import.meta.url);
const scriptsRoot = path.dirname(currentFilePath);
const frontendRoot = path.resolve(scriptsRoot, '..');
const opsRoot = path.resolve(frontendRoot, '..');
const sourceRoot = path.join(frontendRoot, 'src');
const fallbackResourcesRoot = path.join(sourceRoot, 'i18n', 'resources');
const foundationResourcesRoot = path.join(fallbackResourcesRoot, 'foundation');
const supportedLocales = ['zh-CN', 'en-US', 'ja-JP', 'ko-KR', 'fr-FR'];
const ignoredDirectories = new Set(['node_modules', 'dist', 'coverage']);

function walkSourceFiles(rootPath, bucket = []) {
  for (const entry of fs.readdirSync(rootPath, { withFileTypes: true })) {
    if (ignoredDirectories.has(entry.name)) {
      continue;
    }
    const nextPath = path.join(rootPath, entry.name);
    if (entry.isDirectory()) {
      walkSourceFiles(nextPath, bucket);
      continue;
    }
    if (['.ts', '.tsx'].includes(path.extname(entry.name))) {
      bucket.push(nextPath);
    }
  }
  return bucket;
}

export function extractStaticTranslationKeys(rootPath, resourcesRoot) {
  const keys = new Set();
  const patterns = [
    /\bt\s*\(\s*'([^'`]+?)'/g,
    /\bt\s*\(\s*"([^"`]+?)"/g,
    /\bi18n\.t\s*\(\s*'([^'`]+?)'/g,
    /\bi18n\.t\s*\(\s*"([^"`]+?)"/g,
  ];

  for (const filePath of walkSourceFiles(rootPath)) {
    if (filePath.startsWith(`${resourcesRoot}${path.sep}`)) {
      continue;
    }
    const source = fs.readFileSync(filePath, 'utf8');
    for (const pattern of patterns) {
      for (const match of source.matchAll(pattern)) {
        if (!match[1].includes('${')) {
          keys.add(match[1]);
        }
      }
    }
  }
  return keys;
}

function readJsonResource(filePath) {
  try {
    const resource = JSON.parse(fs.readFileSync(filePath, 'utf8'));
    if (!resource || typeof resource !== 'object' || Array.isArray(resource)) {
      throw new Error('expected an object');
    }
    return resource;
  } catch (error) {
    throw new Error(`Invalid foundation locale resource ${filePath}: ${error.message}`);
  }
}

function loadFallbackResource(resourcesRoot, locale) {
  const filePath = path.join(resourcesRoot, `${locale}.ts`);
  return fs.existsSync(filePath) ? loadResourceModule(filePath) : {};
}

export function buildFoundationFallbackResources({
  sourceRoot: activeSourceRoot,
  fallbackResourcesRoot: activeResourcesRoot,
  builtinResourcePath,
  locales = supportedLocales,
}) {
  const builtinResources = readJsonResource(builtinResourcePath);
  const fallbackResources = Object.fromEntries(
    locales.map((locale) => [locale, loadFallbackResource(activeResourcesRoot, locale)]),
  );
  const requiredKeys = extractStaticTranslationKeys(activeSourceRoot, activeResourcesRoot);
  const resources = Object.fromEntries(locales.map((locale) => [locale, {}]));

  for (const key of requiredKeys) {
    if (Object.hasOwn(fallbackResources['zh-CN'] ?? {}, key)) {
      continue;
    }
    for (const locale of locales) {
      const value = builtinResources[locale]?.[key];
      if (typeof value === 'string' && value.trim() !== '') {
        resources[locale][key] = value;
      }
    }
  }

  return Object.fromEntries(
    locales.map((locale) => [
      locale,
      Object.fromEntries(Object.entries(resources[locale]).sort(([left], [right]) => left.localeCompare(right))),
    ]),
  );
}

export function syncFoundationI18n({
  sourceRoot: activeSourceRoot = sourceRoot,
  fallbackResourcesRoot: activeResourcesRoot = fallbackResourcesRoot,
  foundationResourcesRoot: activeFoundationRoot = foundationResourcesRoot,
  builtinResourcePath,
  locales = supportedLocales,
  checkOnly = false,
}) {
  const resources = buildFoundationFallbackResources({
    sourceRoot: activeSourceRoot,
    fallbackResourcesRoot: activeResourcesRoot,
    builtinResourcePath,
    locales,
  });
  const changes = [];

  for (const locale of locales) {
    const targetPath = path.join(activeFoundationRoot, `${locale}.json`);
    const nextContent = `${JSON.stringify(resources[locale], null, 2)}\n`;
    const currentContent = fs.existsSync(targetPath) ? fs.readFileSync(targetPath, 'utf8') : '';
    if (currentContent === nextContent) {
      continue;
    }
    changes.push(targetPath);
    if (!checkOnly) {
      fs.mkdirSync(path.dirname(targetPath), { recursive: true });
      fs.writeFileSync(targetPath, nextContent, 'utf8');
    }
  }

  if (checkOnly && changes.length > 0) {
    throw new Error(
      `Foundation i18n fallback resources are out of date:\n${changes
        .map((filePath) => `  ${path.relative(frontendRoot, filePath)}`)
        .join('\n')}`,
    );
  }

  return { resources, changes };
}

function main() {
  const releasePaths = resolveFoundationReleasePaths(opsRoot);
  const builtinResourcePath = path.join(
    releasePaths.sharedBackendRoot,
    'modules',
    'system',
    'i18n',
    'builtin_locale_resources.json',
  );
  const checkOnly = process.argv.includes('--check');
  const result = syncFoundationI18n({ builtinResourcePath, checkOnly });
  const keyCount = Object.values(result.resources).reduce((sum, resource) => sum + Object.keys(resource).length, 0);
  console.log(
    checkOnly
      ? `OK foundation i18n fallback is aligned (${keyCount} locale entries)`
      : `Synced foundation i18n fallback (${keyCount} locale entries)`,
  );
}

if (import.meta.url === pathToFileURL(process.argv[1] || '').href) {
  main();
}

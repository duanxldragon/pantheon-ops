import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';
import { fileURLToPath } from 'node:url';
import { loadResourceModule } from './lib/load-resource-module.mjs';

const currentFilePath = fileURLToPath(import.meta.url);
const frontendRoot = path.resolve(path.dirname(currentFilePath), '..');
const sourceRoot = path.join(frontendRoot, 'src');
const resourcesRoot = path.join(sourceRoot, 'i18n', 'resources');
const generatedResourcesRoot = path.join(resourcesRoot, 'generated');

const baseLocale = 'zh-CN';
const targetExtensions = new Set(['.ts', '.tsx']);
const ignoredDirs = new Set(['node_modules', 'dist', 'coverage']);

function walkFiles(dirPath, bucket = []) {
  for (const entry of fs.readdirSync(dirPath, { withFileTypes: true })) {
    if (ignoredDirs.has(entry.name)) {
      continue;
    }
    const nextPath = path.join(dirPath, entry.name);
    if (entry.isDirectory()) {
      walkFiles(nextPath, bucket);
      continue;
    }
    if (targetExtensions.has(path.extname(entry.name))) {
      bucket.push(nextPath);
    }
  }
  return bucket;
}

function loadFinalBaseResource() {
  const base = loadResourceModule(path.join(resourcesRoot, `${baseLocale}.ts`));
  const generatedPath = path.join(generatedResourcesRoot, `${baseLocale}.ts`);
  const generated = fs.existsSync(generatedPath) ? loadResourceModule(generatedPath) : {};
  return {
    ...base,
    ...generated,
  };
}

function extractStaticTranslationKeys(source) {
  const keys = new Set();
  const patterns = [
    /\bt\s*\(\s*'([^'`]+?)'/g,
    /\bt\s*\(\s*"([^"`]+?)"/g,
    /\bi18n\.t\s*\(\s*'([^'`]+?)'/g,
    /\bi18n\.t\s*\(\s*"([^"`]+?)"/g,
  ];

  for (const pattern of patterns) {
    for (const match of source.matchAll(pattern)) {
      keys.add(match[1]);
    }
  }
  return keys;
}

function main() {
  const resources = loadFinalBaseResource();
  const resourceKeys = new Set(Object.keys(resources));
  const missing = [];

  for (const filePath of walkFiles(sourceRoot)) {
    if (filePath.includes(`${path.sep}i18n${path.sep}resources${path.sep}`)) {
      continue;
    }
    const source = fs.readFileSync(filePath, 'utf8');
    for (const key of extractStaticTranslationKeys(source)) {
      if (key.includes('${')) {
        continue;
      }
      if (!resourceKeys.has(key)) {
        missing.push({
          filePath: path.relative(frontendRoot, filePath),
          key,
        });
      }
    }
  }

  if (missing.length > 0) {
    console.error(`Missing i18n fallback keys: ${missing.length}`);
    missing.slice(0, 80).forEach((item) => {
      console.error(`  ${item.filePath}: ${item.key}`);
    });
    process.exit(1);
  }

  console.log(`i18n static key check passed (${resourceKeys.size} fallback keys).`);
}

main();

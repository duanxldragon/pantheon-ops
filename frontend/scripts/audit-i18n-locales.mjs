import path from 'node:path';
import process from 'node:process';
import fs from 'node:fs';
import { fileURLToPath } from 'node:url';
import { loadResourceModule } from './lib/load-resource-module.mjs';

const currentFilePath = fileURLToPath(import.meta.url);
const frontendRoot = path.resolve(path.dirname(currentFilePath), '..');
const resourcesRoot = path.join(frontendRoot, 'src', 'i18n', 'resources');
const generatedResourcesRoot = path.join(resourcesRoot, 'generated');

const LOCALES = ['zh-CN', 'en-US', 'ja-JP', 'ko-KR', 'fr-FR'];
const BASE_LOCALE = 'zh-CN';
const REFERENCE_LOCALE = 'en-US';

const SAME_AS_ENGLISH_ALLOWLIST = [
  /^app\.language\./,
  /^app\.name$/,
  /^app\.footer$/,
  /^auth\.login\.visualBadge$/,
  /^business\.cmdb\.title$/,
  /^cmdb\.menu\.root$/,
  /^common\.importErrorFileName$/,
  /^system\.audit\.costTimeValue$/,
  /^app\.command\.section\.menu$/,
  /^app\.preference\.navigation$/,
  /^dashboard\.menus$/,
  /^common\.total$/,
  /^i18n\.hero\.modules$/,
  /^i18n\.module$/,
  /^i18n\.rename\.report\.module$/,
  /^i18n\.stats\.modules$/,
  /^system\.dict\.module$/,
  /^system\.menu\.cache$/,
  /^system\.menu\.module$/,
  /^system\.menu\.type$/,
  /^system\.menu\.type\.menu$/,
  /^system\.menu\.visible$/,
  /^system\.menu\.icon\.menu$/,
  /^system\.permission\.workbench\.navCount$/,
  /^system\.setting\.item\.upload\.s3_/,
  /^system\.setting\.item\.upload\.s3_endpoint$/,
  /^system\.setting\.item\.upload\.s3_bucket$/,
  /^system\.setting\.item\.upload\.s3_region$/,
  /^system\.setting\.item\.upload\.s3_access_key_id$/,
  /^system\.setting\.item\.upload\.s3_secret_access_key$/,
  /^system\.setting\.item\.site\.logo$/,
  /^system\.setting\.item\.upload\.public_base_url$/,
  /^generator\./,
  /^theme\./,
];

function shouldIgnoreSameAsEnglish(key, value) {
  if (!value || typeof value !== 'string') {
    return true;
  }
  return SAME_AS_ENGLISH_ALLOWLIST.some((pattern) => pattern.test(key));
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

function findEmptyKeys(resource) {
  return Object.entries(resource)
    .filter(([, value]) => typeof value !== 'string' || value.trim() === '')
    .map(([key]) => key)
    .sort((a, b) => a.localeCompare(b));
}

function findSameAsEnglishKeys(locale, resource, englishResource) {
  if (locale === REFERENCE_LOCALE) {
    return [];
  }
  return Object.entries(resource)
    .filter(([key, value]) => {
      if (typeof value !== 'string') {
        return false;
      }
      if (value !== englishResource[key]) {
        return false;
      }
      return !shouldIgnoreSameAsEnglish(key, value);
    })
    .map(([key]) => key)
    .sort((a, b) => a.localeCompare(b));
}

async function main() {
  const resources = Object.fromEntries(LOCALES.map((locale) => [locale, loadLocale(locale)]));

  const baseKeys = new Set(Object.keys(resources[BASE_LOCALE]));
  const englishResource = resources[REFERENCE_LOCALE];
  let hasIssue = false;

  console.log(`I18N locale audit based on ${BASE_LOCALE}, reference ${REFERENCE_LOCALE}\n`);

  for (const locale of LOCALES) {
    const resource = resources[locale];
    const keys = new Set(Object.keys(resource));
    const missingKeys = [...baseKeys].filter((key) => !keys.has(key)).sort((a, b) => a.localeCompare(b));
    const extraKeys = [...keys].filter((key) => !baseKeys.has(key)).sort((a, b) => a.localeCompare(b));
    const emptyKeys = findEmptyKeys(resource);
    const sameAsEnglishKeys = findSameAsEnglishKeys(locale, resource, englishResource);

    console.log(`[${locale}] keys=${keys.size} missing=${missingKeys.length} extra=${extraKeys.length} empty=${emptyKeys.length} sameAsEn=${sameAsEnglishKeys.length}`);

    if (missingKeys.length > 0) {
      hasIssue = true;
      console.log('  missing:');
      missingKeys.slice(0, 50).forEach((key) => console.log(`    - ${key}`));
    }
    if (extraKeys.length > 0) {
      console.log('  extra:');
      extraKeys.slice(0, 50).forEach((key) => console.log(`    - ${key}`));
    }
    if (emptyKeys.length > 0) {
      hasIssue = true;
      console.log('  empty:');
      emptyKeys.slice(0, 50).forEach((key) => console.log(`    - ${key}`));
    }
    if (sameAsEnglishKeys.length > 0) {
      console.log('  same-as-en sample:');
      sameAsEnglishKeys.slice(0, 50).forEach((key) => console.log(`    - ${key}`));
    }
  }

  if (hasIssue) {
    process.exitCode = 1;
  }
}

await main();

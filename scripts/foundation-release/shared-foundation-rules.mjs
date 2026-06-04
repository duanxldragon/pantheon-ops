import fs from 'node:fs';
import path from 'node:path';

export const sharedBackendEntries = ['cmd', 'internal', 'modules', 'pkg'];

export const sharedFrontendEntries = [
  'components',
  'core',
  'modules/auth',
  'modules/dashboard',
  'modules/system',
  'index.css',
];

export const backendOverlayPaths = new Set([
  'internal/scaffold/workspace.go',
  'internal/scaffold/workspace_test.go',
  'modules/business/business.go',
  'modules/business/generated_registry.go',
  'modules/business/retired_modules.go',
  'modules/platform/health.go',
  'modules/system/dynamicmodule/dynamic_module_service_test.go',
  'modules/system/generator/generator_service_test.go',
  'modules/system/iam/menu/component_registry.go',
  'modules/system/iam/menu/generated_component_registry.go',
]);

export const backendOverlayDirPrefixes = ['cmd/server/uploads/'];

export const frontendOverlayPaths = new Set([
  'core/router/componentRegistry.ts',
  'core/router/generatedComponentRegistry.ts',
  'core/router/modules.ts',
  'modules/system/generator/backend-generator.ts',
]);

export const allowedFrontendOpsOnlyPaths = new Set([]);

export const backendMergedJsonPaths = new Set([
  'modules/system/i18n/builtin_locale_resources.json',
]);

export function toRepoPath(filePath) {
  return filePath.split(path.sep).join('/');
}

export function collectFiles(rootPath, currentPath = rootPath, bucket = []) {
  const entries = fs.readdirSync(currentPath, { withFileTypes: true });
  for (const entry of entries) {
    const nextPath = path.join(currentPath, entry.name);
    if (entry.isDirectory()) {
      collectFiles(rootPath, nextPath, bucket);
      continue;
    }
    bucket.push(toRepoPath(path.relative(rootPath, nextPath)));
  }
  return bucket;
}

export function ensureDir(filePath) {
  fs.mkdirSync(path.dirname(filePath), { recursive: true });
}

export function normalizeLineEndings(source) {
  return source.replaceAll('\r\n', '\n');
}

export function isBackendOverlayPath(relativePath) {
  return backendOverlayPaths.has(relativePath)
    || backendOverlayDirPrefixes.some((prefix) => relativePath.startsWith(prefix));
}

export function isFrontendOverlayPath(relativePath) {
  return frontendOverlayPaths.has(relativePath);
}

export function readGoModuleName(repoRoot) {
  const goModPath = path.join(repoRoot, 'go.mod');
  if (!fs.existsSync(goModPath)) {
    throw new Error(`go.mod not found: ${goModPath}`);
  }

  const goModSource = fs.readFileSync(goModPath, 'utf8');
  const match = goModSource.match(/^module\s+(\S+)\s*$/m);
  if (!match) {
    throw new Error(`failed to read Go module name from ${goModPath}`);
  }
  return match[1];
}

export function detectBackendModuleNameFromTree(rootPath) {
  if (!fs.existsSync(rootPath)) {
    throw new Error(`backend tree not found: ${rootPath}`);
  }

  for (const relativePath of collectFiles(rootPath)) {
    if (!relativePath.endsWith('.go')) {
      continue;
    }
    const source = fs.readFileSync(path.join(rootPath, relativePath), 'utf8');
    const match = source.match(/"([^"\s]+)\/backend(?:\/[^"\s]*)?"/);
    if (match) {
      return match[1];
    }
  }

  throw new Error(`failed to infer backend module name from ${rootPath}`);
}

export function rewriteBackendModuleReferences(source, fromModuleName, toModuleName) {
  return source.replaceAll(`${fromModuleName}/backend`, `${toModuleName}/backend`);
}

export function mergeBuiltinLocaleResources(baseSource, opsSource) {
  const nextLocales = JSON.parse(baseSource);
  const currentLocales = JSON.parse(opsSource);

  for (const [locale, currentPack] of Object.entries(currentLocales)) {
    if (!currentPack || typeof currentPack !== 'object' || Array.isArray(currentPack)) {
      continue;
    }

    if (!nextLocales[locale] || typeof nextLocales[locale] !== 'object' || Array.isArray(nextLocales[locale])) {
      nextLocales[locale] = {};
    }
    const nextPack = nextLocales[locale];

    for (const [key, value] of Object.entries(currentPack)) {
      if (!key.startsWith('business.')) {
        continue;
      }
      nextPack[key] = value;
    }
  }

  return `${JSON.stringify(nextLocales, null, 2)}\n`;
}

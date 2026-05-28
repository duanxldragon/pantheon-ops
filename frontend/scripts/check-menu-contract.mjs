import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';
import vm from 'node:vm';
import { fileURLToPath } from 'node:url';

const currentFilePath = fileURLToPath(import.meta.url);
const workspaceRoot = path.resolve(path.dirname(currentFilePath), '..', '..');
const frontendModulesRoot = path.join(workspaceRoot, 'frontend', 'src', 'modules');
const backendModulesRoot = path.join(workspaceRoot, 'backend', 'modules');
const frontendRegistryFiles = [
  path.join(workspaceRoot, 'frontend', 'src', 'core', 'router', 'componentRegistry.ts'),
  path.join(workspaceRoot, 'frontend', 'src', 'core', 'router', 'generatedComponentRegistry.ts'),
];
const backendRegistryFiles = [
  path.join(workspaceRoot, 'backend', 'modules', 'system', 'iam', 'menu', 'component_registry.go'),
  path.join(workspaceRoot, 'backend', 'modules', 'system', 'iam', 'menu', 'generated_component_registry.go'),
];
const frontendI18nFiles = new Map([
  ['zh-CN', path.join(workspaceRoot, 'frontend', 'src', 'i18n', 'resources', 'zh-CN.ts')],
  ['en-US', path.join(workspaceRoot, 'frontend', 'src', 'i18n', 'resources', 'en-US.ts')],
]);
const frontendGeneratedI18nRoot = path.join(workspaceRoot, 'frontend', 'src', 'i18n', 'resources', 'generated');

function readFile(filePath) {
  return fs.readFileSync(filePath, 'utf8');
}

function loadResourceModule(modulePath, cache = new Map()) {
  const resolvedPath = path.resolve(modulePath);
  if (cache.has(resolvedPath)) {
    return cache.get(resolvedPath);
  }

  const source = readFile(resolvedPath);
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

function walkFiles(dirPath, matcher, bucket = []) {
  for (const entry of fs.readdirSync(dirPath, { withFileTypes: true })) {
    const nextPath = path.join(dirPath, entry.name);
    if (entry.isDirectory()) {
      walkFiles(nextPath, matcher, bucket);
      continue;
    }
    if (matcher(nextPath)) {
      bucket.push(nextPath);
    }
  }
  return bucket;
}

function findMatchingBlock(source, startChar, endChar, startIndex) {
  let depth = 0;
  let inSingleQuote = false;
  let inDoubleQuote = false;
  let escaped = false;

  for (let index = startIndex; index < source.length; index += 1) {
    const char = source[index];
    if (escaped) {
      escaped = false;
      continue;
    }
    if (char === '\\') {
      escaped = true;
      continue;
    }
    if (!inDoubleQuote && char === '\'') {
      inSingleQuote = !inSingleQuote;
      continue;
    }
    if (!inSingleQuote && char === '"') {
      inDoubleQuote = !inDoubleQuote;
      continue;
    }
    if (inSingleQuote || inDoubleQuote) {
      continue;
    }
    if (char === startChar) {
      depth += 1;
    } else if (char === endChar) {
      depth -= 1;
      if (depth === 0) {
        return source.slice(startIndex, index + 1);
      }
    }
  }

  return null;
}

function extractArrayBody(source, propertyName) {
  const propertyIndex = source.indexOf(`${propertyName}:`);
  if (propertyIndex < 0) {
    return '';
  }
  const arrayStart = source.indexOf('[', propertyIndex);
  if (arrayStart < 0) {
    return '';
  }
  const block = findMatchingBlock(source, '[', ']', arrayStart);
  return block ? block.slice(1, -1) : '';
}

function extractObjectBlocks(arrayBody) {
  const blocks = [];
  for (let index = 0; index < arrayBody.length; index += 1) {
    if (arrayBody[index] !== '{') {
      continue;
    }
    const block = findMatchingBlock(arrayBody, '{', '}', index);
    if (!block) {
      continue;
    }
    blocks.push(block.slice(1, -1));
    index += block.length - 1;
  }
  return blocks;
}

function extractField(block, fieldName) {
  const patterns = [
    new RegExp(String.raw`\b${fieldName}\b\s*:\s*'([^']*)'`),
    new RegExp(String.raw`\b${fieldName}\b\s*:\s*"([^"]*)"`),
    new RegExp(String.raw`\b${fieldName}\b\s*:\s*([0-9]+)`),
  ];
  for (const pattern of patterns) {
    const match = block.match(pattern);
    if (match) {
      return match[1];
    }
  }
  return '';
}

function extractStringArray(source, propertyName) {
  const arrayBody = extractArrayBody(source, propertyName);
  if (!arrayBody) {
    return [];
  }
  return [...arrayBody.matchAll(/['"]([^'"]+)['"]/g)].map((match) => match[1]);
}

function normalizeRoutePath(routePath) {
  if (!routePath) {
    return '';
  }
  return routePath.startsWith('/') ? routePath : `/${routePath}`;
}

function parseFrontendModules() {
  const files = walkFiles(frontendModulesRoot, (filePath) => filePath.endsWith(`${path.sep}index.ts`));
  const modules = [];

  for (const filePath of files) {
    const source = readFile(filePath);
    const scope = extractField(source, 'scope');
    const moduleName = extractField(source, 'name');
    if (!scope || !moduleName) {
      continue;
    }

    const routeBlocks = extractObjectBlocks(extractArrayBody(source, 'routes'));
    const menuBlocks = extractObjectBlocks(extractArrayBody(source, 'menus'));

    modules.push({
      filePath,
      scope,
      name: moduleName,
      permissions: extractStringArray(source, 'permissions'),
      i18nNamespaces: extractStringArray(source, 'i18nNamespaces'),
      routes: routeBlocks.map((block) => ({
        path: extractField(block, 'path'),
        routeName: extractField(block, 'routeName'),
        titleKey: extractField(block, 'titleKey'),
        pagePermission: extractField(block, 'pagePermission'),
        activeMenu: extractField(block, 'activeMenu'),
        componentKey: extractField(block, 'componentKey'),
      })),
      menus: menuBlocks.map((block) => ({
        path: extractField(block, 'path'),
        routeName: extractField(block, 'routeName'),
        titleKey: extractField(block, 'titleKey'),
        module: extractField(block, 'module'),
      })),
    });
  }

  return modules;
}

function parseBackendSeeds() {
  const backendSeedFiles = [
    path.join(workspaceRoot, 'backend', 'modules', 'system', 'seed.go'),
    ...walkFiles(backendModulesRoot, (filePath) => path.basename(filePath) === 'module.go'),
    ...walkFiles(backendModulesRoot, (filePath) => path.basename(filePath).endsWith('_seed.go')),
  ];
  const seeds = [];

  for (const filePath of backendSeedFiles) {
    if (!fs.existsSync(filePath)) {
      continue;
    }
    const source = readFile(filePath);
    const matches = source.matchAll(/\{([^{}]*\bKey:\s*"[^"]+"[^{}]*)\}/gms);
    for (const match of matches) {
      const block = match[1];
      const type = extractField(block, 'Type');
      const item = {
        filePath,
        key: extractField(block, 'Key'),
        path: extractField(block, 'Path'),
        titleKey: extractField(block, 'TitleKey'),
        component: extractField(block, 'Component'),
        pagePerm: extractField(block, 'PagePerm'),
        perms: extractField(block, 'Perms'),
        routeName: extractField(block, 'RouteName'),
        module: extractField(block, 'Module'),
        type,
      };
      if (item.path || item.pagePerm || item.perms) {
        seeds.push(item);
      }
    }
  }

  return seeds;
}

function parseRegistryKeys() {
  const keys = new Set();
  for (const filePath of frontendRegistryFiles) {
    if (!fs.existsSync(filePath)) {
      continue;
    }
    const source = readFile(filePath);
    for (const match of source.matchAll(/'([^']+)':\s*(?:lazy|defineRegistryEntry)/g)) {
      keys.add(match[1]);
    }
  }
  return keys;
}

function parseBackendRegistryKeys() {
  const keys = new Set();
  for (const filePath of backendRegistryFiles) {
    if (!fs.existsSync(filePath)) {
      continue;
    }
    const source = readFile(filePath);
    for (const match of source.matchAll(/"([^"]+)":\s*\{\}/g)) {
      keys.add(match[1]);
    }
  }
  return keys;
}

function parseFallbackTranslations() {
  const translations = new Map();

  for (const locale of ['zh-CN', 'en-US']) {
    const filePath = frontendI18nFiles.get(locale);
    if (!filePath || !fs.existsSync(filePath)) {
      translations.set(locale, new Set());
      continue;
    }
    const base = loadResourceModule(filePath);
    const generatedPath = path.join(frontendGeneratedI18nRoot, `${locale}.ts`);
    const generated = fs.existsSync(generatedPath) ? loadResourceModule(generatedPath) : {};
    const moduleLocaleFiles = walkFiles(
      frontendModulesRoot,
      (modulePath) => modulePath.endsWith(`${path.sep}locales${path.sep}${locale}.json`),
    );
    const moduleLocaleKeys = moduleLocaleFiles.flatMap((modulePath) => Object.keys(JSON.parse(readFile(modulePath))));
    const keys = new Set([...Object.keys(base), ...Object.keys(generated), ...moduleLocaleKeys]);
    translations.set(locale, keys);
  }

  return translations;
}

function extractMapField(block, fieldName) {
  const pattern = new RegExp(String.raw`"${fieldName}"\s*:\s*"([^"]*)"`);
  const match = block.match(pattern);
  return match ? match[1] : '';
}

function extractStructBlocksWithFields(source, fieldNames) {
  const blocks = [];
  for (const match of source.matchAll(/\{([^{}]*)\}/gms)) {
    const block = match[1];
    if (
      fieldNames.every((fieldName) => new RegExp(String.raw`\b${fieldName}\b\s*:`).test(block)) &&
      extractField(block, 'Locale') &&
      extractField(block, 'Group') &&
      extractField(block, 'Key')
    ) {
      blocks.push(block);
    }
  }
  return blocks;
}

function extractMapBlocksWithFields(source, fieldNames) {
  const blocks = [];
  for (const match of source.matchAll(/\{([^{}]*)\}/gms)) {
    const block = match[1];
    if (
      fieldNames.every((fieldName) => new RegExp(String.raw`"${fieldName}"\s*:`).test(block)) &&
      extractMapField(block, 'locale') &&
      extractMapField(block, 'group_name') &&
      extractMapField(block, 'key')
    ) {
      blocks.push(block);
    }
  }
  return blocks;
}

function parseBackendMenuI18nSeeds() {
  const files = walkFiles(backendModulesRoot, (filePath) => filePath.endsWith('.go'));
  const translations = new Map([
    ['zh-CN', new Set()],
    ['en-US', new Set()],
  ]);

  for (const filePath of files) {
    const source = readFile(filePath);

    for (const block of extractStructBlocksWithFields(source, ['Locale', 'Group', 'Key'])) {
      const locale = extractField(block, 'Locale');
      const group = extractField(block, 'Group');
      const key = extractField(block, 'Key');
      if (translations.has(locale) && group === 'menu' && key) {
        translations.get(locale).add(key);
      }
    }

    for (const block of extractMapBlocksWithFields(source, ['locale', 'group_name', 'key'])) {
      const locale = extractMapField(block, 'locale');
      const group = extractMapField(block, 'group_name');
      const key = extractMapField(block, 'key');
      if (translations.has(locale) && group === 'menu' && key) {
        translations.get(locale).add(key);
      }
    }
  }

  return translations;
}

function parseBackendScopedI18nSeeds(groupNames) {
  const files = walkFiles(backendModulesRoot, (filePath) => filePath.endsWith('.go'));
  const translations = new Map([
    ['zh-CN', new Set()],
    ['en-US', new Set()],
  ]);
  const groups = new Set(groupNames);

  for (const filePath of files) {
    const source = readFile(filePath);

    for (const block of extractStructBlocksWithFields(source, ['Locale', 'Group', 'Key'])) {
      const locale = extractField(block, 'Locale');
      const group = extractField(block, 'Group');
      const key = extractField(block, 'Key');
      if (translations.has(locale) && groups.has(group) && key) {
        translations.get(locale).add(key);
      }
    }

    for (const block of extractMapBlocksWithFields(source, ['locale', 'group_name', 'key'])) {
      const locale = extractMapField(block, 'locale');
      const group = extractMapField(block, 'group_name');
      const key = extractMapField(block, 'key');
      if (translations.has(locale) && groups.has(group) && key) {
        translations.get(locale).add(key);
      }
    }
  }

  return translations;
}

function collectContracts() {
  const frontendModules = parseFrontendModules();
  const backendSeeds = parseBackendSeeds();
  const registryKeys = parseRegistryKeys();
  const backendRegistryKeys = parseBackendRegistryKeys();
  const fallbackTranslations = parseFallbackTranslations();
  const backendMenuI18nSeeds = parseBackendMenuI18nSeeds();
  const backendPageI18nSeeds = parseBackendScopedI18nSeeds(['page']);
  const backendPermissionI18nSeeds = parseBackendScopedI18nSeeds(['permission']);

  const routeMap = new Map();
  const menuMap = new Map();
  const routes = [];
  const menus = [];
  const permissions = [];

  for (const module of frontendModules) {
    for (const permission of module.permissions) {
      permissions.push({
        key: permission,
        scope: module.scope,
        moduleName: module.name,
        filePath: module.filePath,
      });
    }
    for (const route of module.routes) {
      const item = {
        ...route,
        path: normalizeRoutePath(route.path),
        scope: module.scope,
        moduleName: module.name,
        filePath: module.filePath,
      };
      routes.push(item);
      routeMap.set(item.path, item);
    }
    for (const menu of module.menus) {
      const item = {
        ...menu,
        scope: module.scope,
        moduleName: module.name,
        filePath: module.filePath,
      };
      menus.push(item);
      menuMap.set(menu.path, item);
    }
  }

  return { frontendModules, backendSeeds, registryKeys, backendRegistryKeys, fallbackTranslations, backendMenuI18nSeeds, backendPageI18nSeeds, backendPermissionI18nSeeds, routes, menus, permissions, routeMap, menuMap };
}

function main() {
  const { frontendModules, backendSeeds, registryKeys, backendRegistryKeys, fallbackTranslations, backendMenuI18nSeeds, backendPageI18nSeeds, backendPermissionI18nSeeds, routes, menus, permissions, routeMap, menuMap } = collectContracts();
  const errors = [];
  const backendPageSeeds = backendSeeds.filter((item) => item.path && item.type === 'C');
  const backendActionSeeds = backendSeeds.filter((item) => item.perms && item.type === 'F');
  const backendPermissionKeys = new Set(backendSeeds.flatMap((item) => [item.pagePerm, item.perms]).filter(Boolean));
  const frontendPermissionKeys = new Set(permissions.map((item) => item.key));

  function reportDuplicates(items, keySelector, label) {
    const seen = new Map();
    for (const item of items) {
      const key = keySelector(item);
      if (!key) {
        continue;
      }
      const existing = seen.get(key);
      if (existing) {
        errors.push(`${label} 重复: ${key} (${existing.filePath} 与 ${item.filePath})`);
        continue;
      }
      seen.set(key, item);
    }
  }

  reportDuplicates(routes, (item) => item.path, '路由 path');
  reportDuplicates(menus, (item) => item.path, '菜单 path');
  reportDuplicates(routes, (item) => item.routeName, '路由 routeName');
  reportDuplicates(menus, (item) => item.routeName, '菜单 routeName');
  reportDuplicates(permissions, (item) => item.key, '权限 key');

  for (const module of frontendModules) {
    if (!['platform', 'system', 'business'].includes(module.scope)) {
      errors.push(`非法模块 scope: ${module.scope} (${module.filePath})`);
    }
    if (module.i18nNamespaces.length === 0) {
      errors.push(`模块缺少 i18nNamespaces: ${module.name} (${module.filePath})`);
    }
  }

  for (const key of registryKeys) {
    if (!backendRegistryKeys.has(key)) {
      errors.push(`后端组件白名单缺少 key: ${key}`);
    }
  }

  for (const key of backendRegistryKeys) {
    if (!registryKeys.has(key)) {
      errors.push(`前端组件注册表缺少 key: ${key}`);
    }
  }

  for (const [routePath, route] of routeMap.entries()) {
    if (!route.routeName) {
      errors.push(`路由缺少 routeName: ${routePath} (${route.filePath})`);
    }
    if (!route.titleKey) {
      errors.push(`路由缺少 titleKey: ${routePath} (${route.filePath})`);
    }
    if (!route.componentKey) {
      errors.push(`路由缺少 componentKey: ${routePath} (${route.filePath})`);
      continue;
    }
    if (!registryKeys.has(route.componentKey)) {
      errors.push(`未注册 component key: ${route.componentKey} (${route.filePath} -> ${routePath})`);
    }
    if (route.pagePermission && !frontendPermissionKeys.has(route.pagePermission)) {
      errors.push(`路由 pagePermission 未声明到 permissions: ${route.pagePermission} (${route.filePath} -> ${routePath})`);
    }
    const isManagedMenuRoute = menuMap.has(routePath);
    for (const locale of ['zh-CN', 'en-US']) {
      if (!fallbackTranslations.get(locale)?.has(route.titleKey)) {
        errors.push(`fallback 语言包缺少页面 key: ${route.titleKey} (${locale}, ${routePath})`);
      }
      const expectedBackendPageSet = isManagedMenuRoute ? backendMenuI18nSeeds : backendPageI18nSeeds;
      if (!expectedBackendPageSet.get(locale)?.has(route.titleKey)) {
        errors.push(`后端 i18n seed 缺少页面 key: ${route.titleKey} (${locale}, ${routePath})`);
      }
    }
  }

  for (const [menuPath, menu] of menuMap.entries()) {
    if (!menu.routeName) {
      errors.push(`菜单缺少 routeName: ${menuPath} (${menu.filePath})`);
    }
    if (!menu.titleKey) {
      errors.push(`菜单缺少 titleKey: ${menuPath} (${menu.filePath})`);
    }
    if (!menu.module) {
      errors.push(`菜单缺少 module: ${menuPath} (${menu.filePath})`);
    }
    if (menu.scope === 'platform' && menu.module !== 'platform') {
      errors.push(`platform 菜单 module 必须为 platform: ${menuPath} (${menu.filePath})`);
    }
    if (menu.scope === 'system' && !menu.module.startsWith('system.')) {
      errors.push(`system 菜单 module 必须以 system. 开头: ${menuPath} (${menu.filePath})`);
    }
    if (menu.scope === 'business' && !menu.module.startsWith('business.')) {
      errors.push(`business 菜单 module 必须以 business. 开头: ${menuPath} (${menu.filePath})`);
    }

    const route = routeMap.get(menuPath);
    if (!route) {
      errors.push(`菜单缺少对应路由 manifest: ${menuPath} (${menu.filePath})`);
      continue;
    }
    const backendSeed = backendPageSeeds.find((item) => item.path === menuPath);
    if (!backendSeed) {
      errors.push(`后端缺少菜单 seed: ${menuPath} (${menu.filePath})`);
      continue;
    }

    if (menu.titleKey !== backendSeed.titleKey) {
      errors.push(`titleKey 不一致: ${menuPath} frontend=${menu.titleKey} backend=${backendSeed.titleKey}`);
    }
    if (menu.module !== backendSeed.module) {
      errors.push(`module 不一致: ${menuPath} frontend=${menu.module} backend=${backendSeed.module}`);
    }
    if (menu.routeName !== backendSeed.routeName) {
      errors.push(`routeName 不一致: ${menuPath} frontend=${menu.routeName || '<empty>'} backend=${backendSeed.routeName || '<empty>'}`);
    }
    if (route.pagePermission !== backendSeed.pagePerm) {
      errors.push(`pagePermission 不一致: ${menuPath} frontend=${route.pagePermission || '<empty>'} backend=${backendSeed.pagePerm || '<empty>'}`);
    }
    if (route.componentKey !== backendSeed.component) {
      errors.push(`component key 不一致: ${menuPath} frontend=${route.componentKey || '<empty>'} backend=${backendSeed.component || '<empty>'}`);
    }
    for (const locale of ['zh-CN', 'en-US']) {
      if (!fallbackTranslations.get(locale)?.has(menu.titleKey)) {
        errors.push(`fallback 语言包缺少菜单 key: ${menu.titleKey} (${locale}, ${menuPath})`);
      }
      if (!backendMenuI18nSeeds.get(locale)?.has(menu.titleKey)) {
        errors.push(`后端 i18n seed 缺少菜单 key: ${menu.titleKey} (${locale}, ${menuPath})`);
      }
    }
  }

  for (const seed of backendPageSeeds) {
    if (!menuMap.has(seed.path)) {
      errors.push(`前端缺少菜单 manifest: ${seed.path} (${seed.filePath})`);
      continue;
    }
    if (!routeMap.has(seed.path)) {
      errors.push(`前端缺少页面路由: ${seed.path} (${seed.filePath})`);
    }
  }

  for (const permissionKey of backendPermissionKeys) {
    if (!frontendPermissionKeys.has(permissionKey)) {
      errors.push(`后端 seed 权限未声明到 frontend permissions: ${permissionKey}`);
    }
  }

  for (const seed of backendActionSeeds) {
    if (!seed.titleKey) {
      errors.push(`后端权限 seed 缺少 titleKey: ${seed.perms} (${seed.filePath})`);
      continue;
    }
    for (const locale of ['zh-CN', 'en-US']) {
      if (!fallbackTranslations.get(locale)?.has(seed.titleKey)) {
        errors.push(`fallback 语言包缺少权限 key: ${seed.titleKey} (${locale}, ${seed.perms})`);
      }
      if (!backendPermissionI18nSeeds.get(locale)?.has(seed.titleKey)) {
        errors.push(`后端 i18n seed 缺少权限 key: ${seed.titleKey} (${locale}, ${seed.perms})`);
      }
    }
  }

  if (errors.length > 0) {
    console.error('❌ manifest / menu seed 一致性检查失败');
    for (const error of errors) {
      console.error(`- ${error}`);
    }
    process.exit(1);
  }

  console.log(`✅ manifest / menu seed 一致性检查通过（${menuMap.size} 个菜单，${routeMap.size} 条路由，${frontendPermissionKeys.size} 个权限，${registryKeys.size} 个组件键）`);
}

main();

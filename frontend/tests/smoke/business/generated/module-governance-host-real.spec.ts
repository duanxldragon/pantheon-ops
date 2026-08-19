import fs from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { expect, test, type APIRequestContext } from '@playwright/test';
import { ModuleExporter } from '../../../../src/modules/lowcode/generator/exporter';
import type { GenerateAndRegisterResp, GeneratorTablePreview } from '../../../../src/modules/lowcode/generator/api';
import type { ModuleField, ModuleSchema, PageActionKey } from '../../../../src/modules/lowcode/generator/schema';
import {
  buildAuditActionKey,
  buildFieldHelpTextKey,
  buildFieldLabelKey,
  buildFieldPlaceholderKey,
  buildModuleNamespace,
  buildPermissionTitleKey,
  buildTitleKey,
  generateDefaultMenus,
  generateDefaultPermissions,
  inferModelName,
} from '../../../../src/modules/lowcode/generator/schema';
import {
  adminCredentials,
  apiBaseUrl,
  apiRequestHeaders,
  getApiOperationToken,
  installClientSession,
  installOperationToken,
  loginByApi,
  type BrowserLoginResult,
} from '../../helpers/auth';
import { readGoModulePath } from '../../helpers/go-module';

const currentDir = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(currentDir, '../../../../..');
const platformWorkspaceRoot = path.resolve(repoRoot, '..');
const moduleName = 'cmdb/smoke';
const moduleKey = buildModuleNamespace('business', moduleName);
const pageRoutePath = '/business/cmdb/smoke';
const backendModuleRelativePath = path.join('backend', 'modules', 'business', moduleName);
const frontendModuleRelativePath = path.join('frontend', 'src', 'modules', 'business', moduleName);
const schemaRelativePath = path.join('schema', 'generated', 'business', `${moduleName}.json`);
const backendLeafName = 'smoke';
const frontendModelName = 'CmdbSmoke';
const backendRegistryRelativePath = path.join('backend', 'modules', 'business', 'generated_registry.go');
const frontendRegistryRelativePath = path.join('frontend', 'src', 'modules', 'generated', 'business.ts');
const componentRegistryRelativePath = path.join(
  'frontend',
  'src',
  'core',
  'router',
  'generatedComponentRegistry.ts',
);
const tableName = 'biz_cmdb_host';
const appBaseUrl = process.env.PANTHEON_WEB_BASE_URL ?? 'http://127.0.0.1:5173';
const generatedArtifactRelativePaths = [
  backendModuleRelativePath,
  frontendModuleRelativePath,
  schemaRelativePath,
];

function isRuntimeGeneratedModuleReady(status: unknown) {
  return status === 1 || status === 3;
}

async function listCandidateRepoRoots() {
  const entries = await fs.readdir(platformWorkspaceRoot, { withFileTypes: true });
  const repoRoots = entries
    .filter((entry) => entry.isDirectory() && entry.name.startsWith('pantheon-'))
    .map((entry) => path.join(platformWorkspaceRoot, entry.name));
  return Array.from(new Set([repoRoot, ...repoRoots]));
}

async function hasTarget(target: string) {
  try {
    await fs.stat(target);
    return true;
  } catch {
    return false;
  }
}

async function locateGeneratedRepo(relativePaths: string[]) {
  const candidateRoots = await listCandidateRepoRoots();
  for (const candidateRoot of candidateRoots) {
    const results = await Promise.all(
      relativePaths.map((relativePath) => hasTarget(path.join(candidateRoot, relativePath))),
    );
    if (results.every(Boolean)) {
      return candidateRoot;
    }
  }
  return null;
}

async function fetchTablePreview(request: APIRequestContext, login: BrowserLoginResult): Promise<GeneratorTablePreview> {
  const response = await request.get(`${apiBaseUrl}/lowcode/generator/table-schema`, {
    headers: apiRequestHeaders(login),
    params: {
      datasourceId: 'current',
      tableName,
    },
    failOnStatusCode: false,
  });
  expect(response.ok()).toBeTruthy();
  const payload = await response.json();
  expect(payload.code).toBe(200);
  return payload.data as GeneratorTablePreview;
}

function buildI18nTranslations(fields: ModuleField[]) {
  const titleKey = buildTitleKey('business', moduleName);
  const zh: Record<string, string> = {
    [titleKey]: '主机管理',
  };
  const en: Record<string, string> = {
    [titleKey]: 'Host Management',
  };

  for (const field of fields) {
    zh[buildFieldLabelKey('business', moduleName, field.name)] = field.label;
    en[buildFieldLabelKey('business', moduleName, field.name)] = field.labelEn || field.label;
    if (field.placeholder) {
      zh[buildFieldPlaceholderKey('business', moduleName, field.name)] = field.placeholder;
      en[buildFieldPlaceholderKey('business', moduleName, field.name)] = field.placeholderEn || field.placeholder;
    }
    if (field.helpText) {
      zh[buildFieldHelpTextKey('business', moduleName, field.name)] = field.helpText;
      en[buildFieldHelpTextKey('business', moduleName, field.name)] = field.helpTextEn || field.helpText;
    }
    for (const option of field.enumOptions || []) {
      const key = `${buildModuleNamespace('business', moduleName)}.field.${field.name}.option.${option.value}`;
      zh[key] = option.label;
      en[key] = option.labelEn || option.label;
    }
  }

  const actions: Exclude<PageActionKey, 'detail'>[] = ['view', 'create', 'update', 'delete', 'export', 'import'];
  const zhActionText: Record<Exclude<PageActionKey, 'detail'>, string> = {
    view: '查看',
    create: '新增',
    update: '编辑',
    delete: '删除',
    export: '导出',
    import: '导入',
  };
  const enActionText: Record<Exclude<PageActionKey, 'detail'>, string> = {
    view: 'View',
    create: 'Create',
    update: 'Update',
    delete: 'Delete',
    export: 'Export',
    import: 'Import',
  };

  for (const action of actions) {
    zh[buildPermissionTitleKey('business', moduleName, action)] = `${zhActionText[action]}主机`;
    en[buildPermissionTitleKey('business', moduleName, action)] = `${enActionText[action]} Host`;
  }

  zh[buildAuditActionKey('business', moduleName, 'create')] = '新增主机';
  zh[buildAuditActionKey('business', moduleName, 'update')] = '编辑主机';
  zh[buildAuditActionKey('business', moduleName, 'delete')] = '删除主机';
  en[buildAuditActionKey('business', moduleName, 'create')] = 'Create Host';
  en[buildAuditActionKey('business', moduleName, 'update')] = 'Update Host';
  en[buildAuditActionKey('business', moduleName, 'delete')] = 'Delete Host';

  return { zh, en };
}

function buildSchema(preview: GeneratorTablePreview): ModuleSchema {
  const fields = preview.fields;
  const i18n = buildI18nTranslations(fields);
  const schema: ModuleSchema = {
    name: moduleName,
    displayName: '主机管理',
    displayNameEn: 'Host Management',
    description: 'CMDB Host lifecycle smoke test',
    scope: 'business',
    parentMenu: '/business/cmdb',
    templateLevel: 'enterprise',
    pageActionTemplate: 'masterData',
    pageActions: ['view', 'create', 'update', 'delete', 'export', 'import'],
    metadata: {
      boundedContext: 'CMDB',
      owner: 'Pantheon QA',
      summary: '使用平台库 biz_cmdb_host 验证低代码数据库导入闭环',
      sourceMode: 'database',
      sourceDatasourceId: 'current',
      sourceDatasourceName: '当前平台库',
      sourceTable: preview.tableName,
    },
    model: {
      tableName: preview.tableName,
      modelName: inferModelName({
        name: moduleName,
        displayName: '主机管理',
        scope: 'business',
        model: { tableName: preview.tableName, fields },
        menus: [],
        permissions: [],
        i18n: { namespace: '', translations: { zh: {}, en: {} } },
      } as ModuleSchema),
      fields,
    },
    menus: [],
    permissions: [],
    i18n: {
      namespace: buildModuleNamespace('business', moduleName),
      translations: i18n,
    },
    enableExport: true,
    enableImport: true,
    enableAudit: true,
    enableDataScope: false,
  };
  schema.menus = generateDefaultMenus(schema);
  schema.permissions = generateDefaultPermissions(schema);
  return schema;
}

async function purgeModuleIfExists(request: APIRequestContext, login: BrowserLoginResult, operationToken: string) {
  const response = await request.delete(`${apiBaseUrl}/lowcode/dynamic-modules/${moduleKey}/purge?dropTable=false&purgeSource=true`, {
    headers: {
      ...apiRequestHeaders(login),
      'X-Operation-Token': operationToken,
    },
    failOnStatusCode: false,
  });
  if ([200, 404, 500, 403].includes(response.status())) {
    return;
  }
  expect(response.ok()).toBeTruthy();
}

async function getModuleStatus(request: APIRequestContext, login: BrowserLoginResult) {
  const response = await request.get(`${apiBaseUrl}/lowcode/dynamic-modules/${moduleKey}`, {
    headers: apiRequestHeaders(login),
    failOnStatusCode: false,
  });
  return {
    status: response.status(),
    payload: await response.json(),
  };
}

async function readFileContains(target: string, fragment: string) {
  const content = await fs.readFile(target, 'utf8');
  return content.includes(fragment);
}

test('cmdb host database-import flow generates a temporary module without dropping source table', async ({ page }) => {
  const login = await loginByApi(page.request, adminCredentials);
  const operationToken = await getApiOperationToken(page.request, login);

  await purgeModuleIfExists(page.request, login, operationToken);

  const preview = await fetchTablePreview(page.request, login);
  expect(preview.tableName).toBe(tableName);
  expect(preview.suggestedName).toBe('cmdb/host');
  expect(preview.suggestedScope).toBe('business');
  expect(preview.fields.length).toBeGreaterThan(8);
  expect(preview.fields.some((field) => field.name === 'hostname')).toBeTruthy();
  expect(preview.fields.some((field) => field.name === 'ip')).toBeTruthy();
  expect(preview.fields.some((field) => field.name === 'os')).toBeTruthy();
  expect(preview.fields.some((field) => field.name === 'status')).toBeTruthy();

  const schema = buildSchema(preview);
  const hostnameField = preview.fields.find((field) => field.name === 'hostname');
  const ipField = preview.fields.find((field) => field.name === 'ip');
  const statusField = preview.fields.find((field) => field.name === 'status');
  expect(hostnameField).toBeTruthy();
  expect(ipField).toBeTruthy();
  expect(statusField).toBeTruthy();
  const exporter = new ModuleExporter(schema);
  const files = exporter.generateAll();
  expect(files.length).toBeGreaterThanOrEqual(10);
  expect(files.map((file) => file.path)).toEqual(expect.arrayContaining([
    `backend/modules/business/${moduleName}/${backendLeafName}_model.go`,
    `backend/modules/business/${moduleName}/${backendLeafName}_dto.go`,
    `backend/modules/business/${moduleName}/${backendLeafName}_service.go`,
    `backend/modules/business/${moduleName}/${backendLeafName}_handler.go`,
    `backend/modules/business/${moduleName}/module.go`,
    `frontend/src/modules/business/${moduleName}/index.ts`,
    `frontend/src/modules/business/${moduleName}/api.ts`,
    `frontend/src/modules/business/${moduleName}/${frontendModelName}List.tsx`,
    `frontend/src/modules/business/${moduleName}/${frontendModelName}Form.tsx`,
    `frontend/src/modules/business/${moduleName}/${frontendModelName}Detail.tsx`,
  ]));

  const generateResponse = await page.request.post(`${apiBaseUrl}/lowcode/dynamic-modules/generate`, {
    headers: {
      ...apiRequestHeaders(login),
      'X-Operation-Token': operationToken,
    },
    data: {
      schema,
      files,
      overwrite: false,
    },
    failOnStatusCode: false,
  });
  expect(generateResponse.ok()).toBeTruthy();
  const generatePayload = await generateResponse.json();
  expect(generatePayload.code).toBe(200);

  const result = generatePayload.data as GenerateAndRegisterResp;
  expect(result.module.name).toBe(moduleKey);
  expect(isRuntimeGeneratedModuleReady(result.module.status)).toBe(true);
  expect(result.module.tableName).toBe(tableName);
  expect(result.summary.routePath).toBe(pageRoutePath);
  expect(result.summary.parentMenuSource).toBe('explicit');
  expect(result.requiresRestart).toBe(true);
  expect(result.requiresFrontendBuild).toBe(true);

  await expect.poll(async () => {
    const status = (await getModuleStatus(page.request, login)).payload.data?.status;
    return isRuntimeGeneratedModuleReady(status);
  }).toBe(true);

  await expect.poll(async () => locateGeneratedRepo(generatedArtifactRelativePaths), {
    timeout: 30000,
  }).toBeTruthy();
  const generatedRepoRoot = await locateGeneratedRepo(generatedArtifactRelativePaths);
  expect(generatedRepoRoot).toBeTruthy();

  const backendRegistry = path.join(generatedRepoRoot!, backendRegistryRelativePath);
  const frontendRegistry = path.join(generatedRepoRoot!, frontendRegistryRelativePath);
  const componentRegistry = path.join(generatedRepoRoot!, componentRegistryRelativePath);
  const backendModuleImport = `${await readGoModulePath(generatedRepoRoot!)}/modules/business/${moduleName}`;

  // Registry imports the module by Go import path, not the directory path.
  await expect.poll(async () => readFileContains(backendRegistry, backendModuleImport)).toBe(true);
  await expect.poll(async () => readFileContains(frontendRegistry, `../business/${moduleName}`)).toBe(true);
  await expect.poll(async () => readFileContains(componentRegistry, `business/${moduleName}/${frontendModelName}List`)).toBe(true);

  await page.goto(`${appBaseUrl}/login`, { waitUntil: 'domcontentloaded' });
  await installClientSession(page, login);
  await installOperationToken(page, login.accessToken);
  await page.route(`**/api/v1/business/${moduleName}/list**`, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        code: 200,
        data: {
          items: [
            {
              id: 1,
              hostname: 'qa-host-001.internal',
              ip: '10.20.30.41',
              os: 'Ubuntu 24.04',
              status: 'active',
              sshPort: 22,
              osVersion: '24.04',
              owner: 'QA Team',
              remark: 'database-import smoke fixture',
              createdAt: '2026-05-23T11:00:00+08:00',
              updatedAt: '2026-05-23T11:00:00+08:00',
            },
          ],
          total: 1,
          page: 1,
          pageSize: 20,
        },
      }),
    });
  });
  await page.goto(`${appBaseUrl}${pageRoutePath}`, { waitUntil: 'networkidle' });
  await expect(page.getByText('主机管理', { exact: true }).first()).toBeVisible();
  const table = page.locator('.arco-table').first();
  await expect(table).toBeVisible();
  await expect(table.getByText(hostnameField!.label, { exact: true })).toBeVisible();
  await expect(table.getByText(ipField!.label, { exact: true })).toBeVisible();
  await expect(table.getByText(statusField!.label, { exact: true })).toBeVisible();
  await expect(page.getByText('qa-host-001.internal', { exact: true })).toBeVisible();
  await expect(page.getByText('10.20.30.41', { exact: true })).toBeVisible();

  const purgeResponse = await page.request.delete(`${apiBaseUrl}/lowcode/dynamic-modules/${moduleKey}/purge?dropTable=false&purgeSource=true`, {
    headers: {
      ...apiRequestHeaders(login),
      'X-Operation-Token': operationToken,
    },
  });
  expect(purgeResponse.ok()).toBeTruthy();
  const purgePayload = await purgeResponse.json();
  expect(purgePayload.code).toBe(200);

  await expect.poll(async () => {
    const response = await getModuleStatus(page.request, login);
    return response.payload.code;
  }).not.toBe(200);
  await expect.poll(async () => locateGeneratedRepo(generatedArtifactRelativePaths), {
    timeout: 30000,
  }).toBeFalsy();
  await expect.poll(async () => readFileContains(backendRegistry, backendModuleImport)).toBe(false);
  await expect.poll(async () => readFileContains(frontendRegistry, `../business/${moduleName}`)).toBe(false);
  await expect.poll(async () => readFileContains(componentRegistry, `business/${moduleName}/${frontendModelName}List`)).toBe(false);

  const tableCheck = await page.request.get(`${apiBaseUrl}/lowcode/generator/table-schema`, {
    headers: apiRequestHeaders(login),
    params: {
      datasourceId: 'current',
      tableName,
    },
  });
  expect(tableCheck.ok()).toBeTruthy();
  const tablePayload = await tableCheck.json();
  expect(tablePayload.code).toBe(200);
  expect(tablePayload.data?.tableName).toBe(tableName);
});

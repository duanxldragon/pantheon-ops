import fs from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { expect, test, type APIRequestContext } from '@playwright/test';
import { ModuleExporter } from '../../../../src/modules/system/generator/exporter';
import type { GenerateAndRegisterResp, GeneratorTablePreview } from '../../../../src/modules/system/generator/api';
import type { ModuleField, ModuleSchema, PageActionKey } from '../../../../src/modules/system/generator/schema';
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
} from '../../../../src/modules/system/generator/schema';
import {
  adminCredentials,
  apiBaseUrl,
  apiRequestHeaders,
  getApiOperationToken,
  loginByApi,
  type BrowserLoginResult,
} from '../../helpers/auth';

const currentDir = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(currentDir, '../../../../..');
const platformWorkspaceRoot = path.resolve(repoRoot, '..');
const moduleName = 'cmdbhostqa';
const moduleKey = `business.${moduleName}`;
const routePath = `/business/${moduleName}`;
const backendModuleRelativePath = path.join('backend', 'modules', 'business', moduleName);
const frontendModuleRelativePath = path.join('frontend', 'src', 'modules', 'business', moduleName);
const schemaRelativePath = path.join('schema', 'generated', 'business', `${moduleName}.json`);
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
const generatedArtifactRelativePaths = [
  backendModuleRelativePath,
  frontendModuleRelativePath,
  schemaRelativePath,
];

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

type TablePreviewProbe =
  | { available: true; data: GeneratorTablePreview }
  | { available: false; status: number; code: number; message: string };

async function fetchTablePreview(request: APIRequestContext, login: BrowserLoginResult): Promise<TablePreviewProbe> {
  const response = await request.get(`${apiBaseUrl}/system/generator/table-schema`, {
    headers: apiRequestHeaders(login),
    params: {
      datasourceId: 'current',
      tableName,
    },
    failOnStatusCode: false,
  });
  expect(response.ok()).toBeTruthy();
  const payload = await response.json();
  if (payload.code !== 200) {
    return {
      available: false,
      status: response.status(),
      code: Number(payload.code ?? response.status()),
      message: String(payload.msg ?? payload.message ?? ''),
    };
  }
  return { available: true, data: payload.data as GeneratorTablePreview };
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
  const response = await request.delete(`${apiBaseUrl}/system/dynamic-modules/${moduleKey}/purge?dropTable=false&purgeSource=true`, {
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
  const response = await request.get(`${apiBaseUrl}/system/dynamic-modules/${moduleKey}`, {
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

  const previewProbe = await fetchTablePreview(page.request, login);
  if (!previewProbe.available) {
    test.skip(
      true,
      `requires source table ${tableName} in current datasource (status=${previewProbe.status}, code=${previewProbe.code}, message=${previewProbe.message || 'n/a'})`,
    );
    return;
  }
  const preview = previewProbe.data;
  expect(preview.tableName).toBe(tableName);
  expect(preview.suggestedName).toBe('cmdb/host');
  expect(preview.suggestedScope).toBe('business');
  expect(preview.fields.length).toBeGreaterThan(8);
  expect(preview.fields.some((field) => field.name === 'hostname')).toBeTruthy();
  expect(preview.fields.some((field) => field.name === 'ip')).toBeTruthy();
  expect(preview.fields.some((field) => field.name === 'os')).toBeTruthy();
  expect(preview.fields.some((field) => field.name === 'status')).toBeTruthy();

  const schema = buildSchema(preview);
  const exporter = new ModuleExporter(schema);
  const files = exporter.generateAll();
  expect(files.length).toBe(10);

  const generateResponse = await page.request.post(`${apiBaseUrl}/system/dynamic-modules/generate`, {
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
  expect([1, 3]).toContain(result.module.status);
  expect(result.module.tableName).toBe(tableName);
  expect(result.summary.routePath).toBe(routePath);
  expect(result.summary.parentMenuSource).toBe('explicit');

  await expect.poll(async () => {
    const status = (await getModuleStatus(page.request, login)).payload.data?.status;
    return status === 1 || status === 3;
  }).toBe(true);

  await expect.poll(async () => locateGeneratedRepo(generatedArtifactRelativePaths), {
    timeout: 30000,
  }).toBeTruthy();
  const generatedRepoRoot = await locateGeneratedRepo(generatedArtifactRelativePaths);
  expect(generatedRepoRoot).toBeTruthy();

  const backendRegistry = path.join(generatedRepoRoot!, backendRegistryRelativePath);
  const frontendRegistry = path.join(generatedRepoRoot!, frontendRegistryRelativePath);
  const componentRegistry = path.join(generatedRepoRoot!, componentRegistryRelativePath);

  await expect.poll(async () => readFileContains(backendRegistry, `backend/modules/business/${moduleName}`)).toBe(true);
  await expect.poll(async () => readFileContains(frontendRegistry, `../business/${moduleName}`)).toBe(true);
  await expect.poll(async () => readFileContains(componentRegistry, `business/${moduleName}/CmdbhostqaList`)).toBe(true);

  const purgeResponse = await page.request.delete(`${apiBaseUrl}/system/dynamic-modules/${moduleKey}/purge?dropTable=false&purgeSource=true`, {
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
  await expect.poll(async () => readFileContains(backendRegistry, `backend/modules/business/${moduleName}`)).toBe(false);
  await expect.poll(async () => readFileContains(frontendRegistry, `../business/${moduleName}`)).toBe(false);
  await expect.poll(async () => readFileContains(componentRegistry, `business/${moduleName}/CmdbhostqaList`)).toBe(false);

  const tableCheck = await page.request.get(`${apiBaseUrl}/system/generator/table-schema`, {
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

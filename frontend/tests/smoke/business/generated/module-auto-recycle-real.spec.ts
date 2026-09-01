import { expect, test, type Page } from '@playwright/test';
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

const moduleName = 'autorecycleqa';
const moduleKey = `business.${moduleName}`;
const tableName = 'biz_auto_recycle_qa';
const appBaseUrl = process.env.PANTHEON_WEB_BASE_URL ?? 'http://127.0.0.1:5174';

function buildGenerateRequest() {
  return {
    schema: {
      name: moduleName,
      templateVersion: 'v1',
      displayName: '自动回收 QA 模块',
      displayNameEn: 'Auto Recycle QA Module',
      description: 'QA module for lifecycle purge verification',
      scope: 'business',
      parentMenu: '',
      templateLevel: 'enterprise',
      pageActionTemplate: 'masterData',
      pageActions: ['view', 'create', 'update', 'delete', 'detail'],
      metadata: {
        businessContext: 'qa',
        businessContextTitle: 'QA',
        businessContextTitleEn: 'QA',
        tableRole: 'main',
        autoRecycle: true,
        boundedContext: 'qa',
        owner: 'codex',
        summary: '验证临时 QA 表自动回收',
        sourceMode: 'manual',
        sourceTable: '',
      },
      model: {
        tableName,
        modelName: 'Autorecycleqa',
        fields: [
          {
            name: 'name',
            type: 'string',
            label: '名称',
            labelEn: 'Name',
            required: true,
            searchable: true,
            sortable: true,
            visibleInList: true,
            visibleInForm: true,
            placeholder: '请输入名称',
            placeholderEn: 'Enter name',
          },
          {
            name: 'status',
            type: 'enum',
            label: '状态',
            labelEn: 'Status',
            required: true,
            searchable: true,
            sortable: true,
            visibleInList: true,
            visibleInForm: true,
            placeholder: '请选择状态',
            placeholderEn: 'Select status',
            enumOptions: [
              { value: 'draft', label: '草稿', labelEn: 'Draft' },
              { value: 'active', label: '启用', labelEn: 'Active' },
            ],
          },
        ],
      },
      dependencies: [],
      relations: [],
      menus: [
        {
          key: moduleKey,
          titleKey: `${moduleKey}.title`,
          path: `/business/${moduleName}`,
          component: `business/${moduleName}/AutorecycleqaList`,
          pagePermission: `business:${moduleName}:list`,
          type: 'C',
          icon: 'apps',
          routeName: `business-${moduleName}`,
          module: moduleKey,
        },
      ],
      permissions: ['list', 'view', 'create', 'update', 'delete', 'detail'].map((action) => ({
        key: `business:${moduleName}:${action}`,
        name: `${moduleKey}.permission.${action}`,
        type: action === 'list' ? 'menu' : 'button',
        module: moduleKey,
      })),
      i18n: {
        namespace: moduleKey,
        translations: {
          zh: {
            [`${moduleKey}.title`]: '自动回收 QA 模块',
          },
          en: {
            [`${moduleKey}.title`]: 'Auto Recycle QA Module',
          },
        },
      },
    },
    overwrite: true,
  };
}

async function purgeModuleIfExists(page: Page, login: BrowserLoginResult, operationToken: string) {
  const response = await page.request.delete(
    `${apiBaseUrl}/lowcode/dynamic-modules/${moduleKey}/purge?dropTable=false&purgeSource=true`,
    {
      headers: {
        ...apiRequestHeaders(login),
        'X-Operation-Token': operationToken,
      },
      failOnStatusCode: false,
    },
  );
  expect([200, 404, 500]).toContain(response.status());
}

async function listCurrentTables(page: Page, login: BrowserLoginResult) {
  const response = await page.request.get(`${apiBaseUrl}/lowcode/generator/tables`, {
    headers: apiRequestHeaders(login),
    params: { datasourceId: 'current' },
  });
  expect(response.ok()).toBeTruthy();
  const payload = await response.json();
  expect(payload.code).toBe(200);
  return Array.isArray(payload.data) ? payload.data as Array<{ tableName: string }> : [];
}

test('auto-recycle governance flow purges managed table through real UI and backend runtime', async ({
  page,
}) => {
  test.setTimeout(180_000);
  const login = await loginByApi(page.request, adminCredentials);
  const operationToken = await getApiOperationToken(page.request, login);
  await purgeModuleIfExists(page, login, operationToken);

  const generateResponse = await page.request.post(`${apiBaseUrl}/lowcode/dynamic-modules/generate`, {
    headers: {
      ...apiRequestHeaders(login),
      'X-Operation-Token': operationToken,
    },
    data: buildGenerateRequest(),
  });
  expect(generateResponse.ok()).toBeTruthy();
  const generatePayload = await generateResponse.json();
  expect(generatePayload.code).toBe(200);
  expect(generatePayload.data?.module?.name).toBe(moduleKey);

  await page.goto(`${appBaseUrl}/login`, { waitUntil: 'domcontentloaded' });
  await installClientSession(page, login);
  await installOperationToken(page, login.accessToken);

  try {
    await page.goto(`${appBaseUrl}/system/modules`, { waitUntil: 'domcontentloaded' });
    const row = page.locator('.arco-table-tr').filter({ hasText: moduleKey }).first();
    await expect(row).toBeVisible();

    await row.getByRole('button', { name: '彻底删除', exact: true }).click();
    const purgeDialog = page.getByRole('dialog').filter({
      has: page.getByText('彻底删除模块', { exact: true }),
    });
    await expect(purgeDialog).toBeVisible();
    await expect(
      purgeDialog.getByText(`该模块已标记为临时模块，彻底删除时会自动回收业务表 ${tableName}。`),
    ).toBeVisible();
    await expect(
      purgeDialog.getByText('临时模块的业务表会随彻底删除一起自动回收，这个动作不需要额外勾选。'),
    ).toBeVisible();
    await expect(purgeDialog.getByText('同时删除业务数据表', { exact: false })).toHaveCount(0);

    await purgeDialog.locator('.arco-checkbox').first().click();
    await expect(purgeDialog.getByRole('button', { name: '彻底删除', exact: true })).toBeEnabled();
    await purgeDialog.getByRole('button', { name: '彻底删除', exact: true }).click();
    await expect(purgeDialog).toBeHidden();
    await expect
      .poll(async () => page.locator('.arco-table-tr').filter({ hasText: moduleKey }).count())
      .toBe(0);

    await expect
      .poll(async () => {
        const response = await page.request.get(`${apiBaseUrl}/lowcode/dynamic-modules/${moduleKey}`, {
          headers: apiRequestHeaders(login),
          failOnStatusCode: false,
        });
        const payload = await response.json().catch(() => ({}));
        return Number(payload?.code ?? response.status());
      }, { timeout: 30_000, intervals: [1000, 2000, 3000] })
      .not.toBe(200);

    await expect
      .poll(async () => {
        const tables = await listCurrentTables(page, login);
        return tables.some((item) => item.tableName === tableName);
      }, { timeout: 60_000, intervals: [1000, 2000, 3000] })
      .toBe(false);
  } finally {
    await purgeModuleIfExists(page, login, operationToken);
  }
});

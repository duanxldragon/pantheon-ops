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

async function listCurrentTables(page: Page, login: BrowserLoginResult) {
  const response = await page.request.get(`${apiBaseUrl}/system/generator/tables`, {
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

  await page.goto(`${appBaseUrl}/login`, { waitUntil: 'domcontentloaded' });
  await installClientSession(page, login);
  await installOperationToken(page, login.accessToken);

  await expect
    .poll(async () => {
      const tables = await listCurrentTables(page, login);
      return tables.some((item) => item.tableName === tableName);
    }, { timeout: 60_000, intervals: [1000, 2000, 3000] })
    .toBe(true);

  await page.goto(`${appBaseUrl}/system/modules`, { waitUntil: 'networkidle' });
  const row = page.locator('.arco-table-tr').filter({ hasText: moduleKey }).first();
  await expect(row).toBeVisible();

  await row.getByRole('button', { name: '彻底删除', exact: true }).click();
  const purgeDialog = page.getByRole('dialog').filter({ has: page.getByText('彻底删除模块', { exact: true }) });
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
  await expect(page.getByText('模块已彻底删除', { exact: true })).toBeVisible();

  await expect
    .poll(async () => {
      const response = await page.request.get(`${apiBaseUrl}/system/dynamic-modules/${moduleKey}`, {
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

  const cleanupResponse = await page.request.delete(
    `${apiBaseUrl}/system/dynamic-modules/${moduleKey}/purge?dropTable=false&purgeSource=true`,
    {
      headers: {
        ...apiRequestHeaders(login),
        'X-Operation-Token': operationToken,
      },
      failOnStatusCode: false,
    },
  );
  expect([200, 404, 500]).toContain(cleanupResponse.status());
});

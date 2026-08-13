import { expect, test, type Route } from '@playwright/test';
import {
  adminCredentials,
  installClientSession,
  installOperationToken,
  loginByApi,
} from '../../helpers/auth';

const masterModuleName = 'mdqaorder';
const detailModuleName = 'mdqaorderitem';
const masterRoute = `/business/${masterModuleName}`;
const appBaseUrl = process.env.PANTHEON_WEB_BASE_URL ?? 'http://127.0.0.1:5174';

type ChildRow = {
  id: number;
  itemName: string;
  quantity: number;
  enabled: boolean;
  orderId: number;
  remark?: string;
};

async function fulfillBusinessJson(route: Route, body: unknown) {
  await route.fulfill({
    status: 200,
    contentType: 'application/json',
    body: JSON.stringify({
      code: 200,
      data: body,
    }),
  });
}

test('generated master-detail detail page supports child create/edit submit in browser QA', async ({ page }) => {
  test.setTimeout(120_000);
  const login = await loginByApi(page.request, adminCredentials);

  await page.goto(`${appBaseUrl}/login`, { waitUntil: 'domcontentloaded' });
  await installClientSession(page, login);
  await installOperationToken(page, login.accessToken);

  const createdPayloads: Array<Record<string, unknown>> = [];
  const updatedPayloads: Array<Record<string, unknown>> = [];
  const consoleEvents: string[] = [];
  const pageErrors: string[] = [];
  const routeHits = {
    masterDetail: 0,
    childList: 0,
    childCreate: 0,
    childUpdate: 0,
  };
  let childRows: ChildRow[] = [
    { id: 501, itemName: '初始明细', quantity: 2, enabled: true, orderId: 101, remark: 'seed' },
  ];

  page.on('console', (message) => {
    consoleEvents.push(`[${message.type()}] ${message.text()}`);
  });
  page.on('pageerror', (error) => {
    pageErrors.push(error.message);
  });

  await page.route(`**/api/v1/business/${masterModuleName}/101`, async (route) => {
    routeHits.masterDetail += 1;
    await fulfillBusinessJson(route, {
      id: 101,
      name: 'QA 主订单',
      status: 'active',
      createdAt: '2026-05-19T18:00:00+08:00',
      updatedAt: '2026-05-19T18:00:00+08:00',
    });
  });

  await page.route(`**/api/v1/business/${detailModuleName}/list**`, async (route) => {
    routeHits.childList += 1;
    await fulfillBusinessJson(route, {
      items: childRows,
      total: childRows.length,
      page: 1,
      pageSize: 5,
    });
  });

  await page.route(`**/api/v1/business/${detailModuleName}`, async (route) => {
    if (!route.request().url().endsWith(`/api/v1/business/${detailModuleName}`)) {
      await route.fallback();
      return;
    }
    routeHits.childCreate += 1;
    const payload = route.request().postDataJSON() as Record<string, unknown>;
    createdPayloads.push(payload);
    childRows = [
      ...childRows,
      {
        id: 502,
        itemName: String(payload.itemName || ''),
        quantity: Number(payload.quantity || 0),
        enabled: Boolean(payload.enabled),
        orderId: Number(payload.orderId || 0),
        remark: String(payload.remark || ''),
      },
    ];
    await fulfillBusinessJson(route, childRows[childRows.length - 1]);
  });

  await page.route(`**/api/v1/business/${detailModuleName}/501`, async (route) => {
    routeHits.childUpdate += 1;
    const payload = route.request().postDataJSON() as Record<string, unknown>;
    updatedPayloads.push(payload);
    childRows = childRows.map((row) =>
      row.id === 501
        ? {
            ...row,
            itemName: String(payload.itemName || row.itemName),
            quantity: Number(payload.quantity || row.quantity),
            enabled: Boolean(payload.enabled),
            orderId: Number(payload.orderId || row.orderId),
            remark: String(payload.remark || row.remark || ''),
          }
        : row,
    );
    await fulfillBusinessJson(route, childRows.find((row) => row.id === 501));
  });

  await page.goto(`${appBaseUrl}/system/generator`, { waitUntil: 'domcontentloaded' });
  await expect(page.locator('.arco-breadcrumb').getByText('模块生成器', { exact: true })).toBeVisible();

  await page.goto(`${appBaseUrl}${masterRoute}/101`, { waitUntil: 'domcontentloaded' });
  await expect
    .poll(
      async () => ({
        relatedDataVisible: await page
          .getByText('关联数据', { exact: true })
          .isVisible()
          .catch(() => false),
        routeHits,
        consoleEvents,
        pageErrors,
      }),
      { timeout: 30_000, intervals: [1000, 2000, 3000] },
    )
    .toMatchObject({
      relatedDataVisible: true,
    });

  await expect(page.getByText('关联数据', { exact: true })).toBeVisible();
  await expect(page.getByRole('button', { name: '新增子表行', exact: true })).toBeVisible();
  await expect(page.getByText('初始明细', { exact: true })).toBeVisible();

  await page.getByRole('button', { name: '新增子表行', exact: true }).click();
  const createDialog = page.getByRole('dialog', { name: '新增子表记录' });
  await expect(createDialog).toBeVisible();
  await createDialog.getByRole('textbox', { name: '明细名称', exact: true }).fill('新增明细');
  await createDialog.getByRole('spinbutton', { name: '数量', exact: true }).fill('8');
  await createDialog.getByRole('switch', { name: '启用', exact: true }).click();
  await createDialog.getByRole('textbox', { name: '备注', exact: true }).fill('create from qa');
  await createDialog.getByRole('button', { name: /创建|新增/, exact: false }).click();

  await expect.poll(() => createdPayloads.length).toBe(1);
  expect(createdPayloads[0]).toMatchObject({
    itemName: '新增明细',
    quantity: 8,
    enabled: true,
    orderId: 101,
    remark: 'create from qa',
  });
  await expect(page.getByText('新增明细', { exact: true })).toBeVisible();

  const firstEditButton = page
    .locator('.arco-table')
    .first()
    .getByRole('button', { name: '编辑子表行', exact: true })
    .first();
  await firstEditButton.click();

  const editDialog = page.getByRole('dialog', { name: '编辑子表记录' });
  await expect(editDialog).toBeVisible();
  const nameInput = editDialog.getByRole('textbox', { name: '明细名称', exact: true });
  await nameInput.fill('');
  await nameInput.fill('初始明细-已编辑');
  const quantityInput = editDialog.getByRole('spinbutton', { name: '数量', exact: true });
  await quantityInput.fill('');
  await quantityInput.fill('5');
  await editDialog.getByRole('textbox', { name: '备注', exact: true }).fill('edit from qa');
  await editDialog.getByRole('button', { name: '保存', exact: true }).click();

  await expect.poll(() => updatedPayloads.length).toBe(1);
  expect(updatedPayloads[0]).toMatchObject({
    itemName: '初始明细-已编辑',
    quantity: 5,
    enabled: true,
    orderId: 101,
    remark: 'edit from qa',
  });
  await expect(page.getByText('初始明细-已编辑', { exact: true })).toBeVisible();
});

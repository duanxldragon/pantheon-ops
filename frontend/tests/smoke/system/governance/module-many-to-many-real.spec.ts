import { expect, test, type APIRequestContext, type Route } from '@playwright/test';
import {
  adminCredentials,
  apiBaseUrl,
  getApiOperationToken,
  installClientSession,
  installOperationToken,
  loginByApi,
} from '../../helpers/auth';

const ownerModuleName = 'm2mqaasset';
const targetModuleName = 'm2mqatag';
const ownerRoute = `/business/${ownerModuleName}`;
const appBaseUrl = process.env.PANTHEON_WEB_BASE_URL ?? 'http://127.0.0.1:5174';

type RelationRow = {
  id: number;
  value: number;
  name?: string;
  status?: string;
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

async function purgeModule(
  request: APIRequestContext,
  accessToken: string,
  csrfToken: string,
  operationToken: string,
  moduleKey: string,
) {
  await request.delete(
    `${apiBaseUrl}/system/dynamic-modules/${moduleKey}/purge?dropTable=false&purgeSource=true`,
    {
      headers: {
        Authorization: `Bearer ${accessToken}`,
        'X-CSRF-Token': csrfToken,
        'X-Operation-Token': operationToken,
        Cookie: `pantheon_csrf_token=${csrfToken}`,
      },
      failOnStatusCode: false,
    },
  );
}

test('generated many-to-many detail page supports bind and unbind in browser QA', async ({
  page,
}) => {
  test.setTimeout(120_000);
  const login = await loginByApi(page.request, adminCredentials);
  const operationToken = await getApiOperationToken(page.request, login);

  try {
    await page.goto(`${appBaseUrl}/login`, { waitUntil: 'domcontentloaded' });
    await installClientSession(page, login);
    await installOperationToken(page, login.accessToken);

    const bindPayloads: Array<Record<string, unknown>> = [];
    const unbindTargets: string[] = [];
    const consoleEvents: string[] = [];
    const pageErrors: string[] = [];
    const routeHits = {
      ownerDetail: 0,
      relationList: 0,
      relationOptions: 0,
      relationBind: 0,
      relationUnbind: 0,
    };
    const optionRows = [
      { id: 901, name: '标签A', status: 'active' },
      { id: 902, name: '标签B', status: 'active' },
      { id: 903, name: '标签C', status: 'disabled' },
    ];
    let relationRows: RelationRow[] = [{ id: 901, value: 901, name: '标签A', status: 'active' }];

    page.on('console', (message) => {
      consoleEvents.push(`[${message.type()}] ${message.text()}`);
    });
    page.on('pageerror', (error) => {
      pageErrors.push(error.message);
    });

    await page.route(`**/api/v1/business/${ownerModuleName}/101`, async (route) => {
      routeHits.ownerDetail += 1;
      await fulfillBusinessJson(route, {
        id: 101,
        name: 'QA 资产',
        status: 'active',
        createdAt: '2026-05-20T10:00:00+08:00',
        updatedAt: '2026-05-20T10:00:00+08:00',
      });
    });

    await page.route(`**/api/v1/business/${ownerModuleName}/101/relations/assetTags`, async (route) => {
      routeHits.relationList += 1;
      if (route.request().method() === 'POST') {
        routeHits.relationBind += 1;
        const payload = route.request().postDataJSON() as Record<string, unknown>;
        bindPayloads.push(payload);
        const targetIds = Array.isArray(payload.targetIds)
          ? payload.targetIds.map((item) => Number(item))
          : [];
        relationRows = targetIds
          .map((id) => optionRows.find((item) => item.id === id))
          .filter((item): item is (typeof optionRows)[number] => Boolean(item))
          .map((item) => ({ id: item.id, value: item.id, name: item.name, status: item.status }));
        await fulfillBusinessJson(route, { success: true });
        return;
      }
      await fulfillBusinessJson(route, {
        items: relationRows.map((item) => ({ id: item.id, value: item.value })),
      });
    });

    await page.route(
      `**/api/v1/business/${ownerModuleName}/101/relations/assetTags/*`,
      async (route) => {
        routeHits.relationUnbind += 1;
        const targetId = route.request().url().split('/').pop() || '';
        unbindTargets.push(targetId);
        relationRows = relationRows.filter((item) => String(item.id) !== targetId);
        await fulfillBusinessJson(route, { success: true });
      },
    );

    await page.route(`**/api/v1/business/${targetModuleName}/options`, async (route) => {
      routeHits.relationOptions += 1;
      await fulfillBusinessJson(route, optionRows);
    });

    await page.goto(`${appBaseUrl}/system/generator`, { waitUntil: 'domcontentloaded' });
    await expect(page.locator('.arco-breadcrumb').getByText('模块生成器', { exact: true })).toBeVisible();

    await page.goto(`${appBaseUrl}${ownerRoute}/101`, { waitUntil: 'domcontentloaded' });
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
    await expect(page.getByRole('button', { name: '新增关联', exact: true })).toBeVisible();
    await expect(page.locator('.arco-table').first().getByText('标签A', { exact: true })).toBeVisible();

    await page.getByRole('button', { name: '新增关联', exact: true }).click();
    const bindDialog = page.getByRole('dialog', { name: '新增关联' });
    await expect(bindDialog).toBeVisible();
    await bindDialog.locator('.arco-select-view').click();
    await page.getByRole('option', { name: '标签B', exact: true }).click();
    await bindDialog.getByRole('button', { name: '保存', exact: true }).click();

    await expect.poll(() => bindPayloads.length).toBe(1);
    expect(bindPayloads[0]).toMatchObject({
      targetIds: expect.arrayContaining(['901', '902']),
    });
    await expect(page.locator('.arco-table').first().getByText('标签B', { exact: true })).toBeVisible();

    const removeButton = page
      .locator('.arco-table tr')
      .filter({ has: page.getByText('标签A', { exact: true }) })
      .getByRole('button', { name: '移除关联', exact: true })
      .first();
    await removeButton.click();

    await expect.poll(() => unbindTargets.length).toBe(1);
    expect(unbindTargets[0]).toBe('901');
    await expect(page.locator('.arco-table').first().getByText('标签A', { exact: true })).toHaveCount(0);
    await expect(page.locator('.arco-table').first().getByText('标签B', { exact: true })).toBeVisible();
  } finally {
    await purgeModule(
      page.request,
      login.accessToken,
      login.csrfToken,
      operationToken,
      `business.${ownerModuleName}`,
    );
    await purgeModule(
      page.request,
      login.accessToken,
      login.csrfToken,
      operationToken,
      `business.${targetModuleName}`,
    );
  }
});

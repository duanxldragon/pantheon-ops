import { test } from '../../fixtures/coverage';
import fs from 'node:fs/promises';
import { expect } from '@playwright/test';
import {
  apiBaseUrl,
  authHeaders,
  closeExtraBrowserContext,
  createSharedAdminLogin,
  dismissVisibleSuccessDialog,
  expectVisiblePageTitle,
  installClientSession,
  signInAsAdmin,
  verifiedHeaders,
  waitForOkApiResponse,
  withListViewOnlyViewer,
} from './system-pages-helpers';

test.beforeEach(async ({ page }) => {
  await signInAsAdmin(page);
});

test('i18n workspace keeps translation work primary and utility actions subordinate', async ({
  page,
}) => {
  await page.goto('/system/i18n', { waitUntil: 'domcontentloaded' });

  await expectVisiblePageTitle(page, '国际化管理');
  const summaryBar = page.locator('.governance-summary-bar');
  const tableCard = page.locator('.i18n-list-page__table-card');
  const batchBar = tableCard.locator('.table-batch-action-bar');
  const tableBody = tableCard.locator('.system-list__table, .arco-empty').first();

  await expect(summaryBar).toBeVisible();
  await expect(tableCard).toBeVisible();
  await expect(batchBar).toBeVisible();
  await expect(tableBody).toBeVisible();
  await expect(tableCard.getByRole('button', { name: '导出' })).toBeVisible();
  await expect(tableCard.getByRole('button', { name: '导入' })).toBeVisible();

  const tableHierarchy = await page.evaluate(() => {
    const card = document.querySelector<HTMLElement>('.i18n-list-page__table-card');
    const actionBar = document.querySelector<HTMLElement>(
      '.i18n-list-page__table-card .table-batch-action-bar',
    );
    const table = document.querySelector<HTMLElement>(
      '.i18n-list-page__table-card .system-list__table, .i18n-list-page__table-card .arco-empty',
    );
    return card && actionBar && table
      ? {
          actionBottom: actionBar.getBoundingClientRect().bottom,
          actionHeight: actionBar.getBoundingClientRect().height,
          cardTop: card.getBoundingClientRect().top,
          cardHeight: card.getBoundingClientRect().height,
          controlRegionHeight: table.getBoundingClientRect().top - card.getBoundingClientRect().top,
          tableHeight: table.getBoundingClientRect().height,
          tableTop: table.getBoundingClientRect().top,
        }
      : null;
  });
  expect(tableHierarchy).not.toBeNull();
  expect(tableHierarchy!.actionHeight).toBeGreaterThan(0);
  expect(tableHierarchy!.cardTop).toBeLessThan(tableHierarchy!.tableTop);
  expect(tableHierarchy!.cardHeight).toBeGreaterThan(tableHierarchy!.actionHeight);
  expect(tableHierarchy!.tableHeight).toBeGreaterThan(tableHierarchy!.controlRegionHeight);
  expect(tableHierarchy!.tableTop).toBeGreaterThanOrEqual(tableHierarchy!.actionBottom - 1);
});

test('i18n smoke: detail edit create and delete dialogs work', async ({ page }) => {
  const accessToken = await signInAsAdmin(page);
  const seedKey = `i18n.smoke.${Date.now()}`;
  const createResponse = await page.request.post(`${apiBaseUrl}/system/i18n`, {
    headers: await verifiedHeaders(page, accessToken),
    data: {
      module: 'system.config',
      group: 'messages',
      key: seedKey,
      locale: 'zh-CN',
      value: '初始值',
      remark: 'smoke',
    },
  });
  expect(createResponse.ok()).toBeTruthy();
  const createPayload = await createResponse.json();
  expect(createPayload.code).toBe(200);

  await page.goto('/system/i18n', { waitUntil: 'domcontentloaded' });
  const i18nToolbar = page.locator('.search-toolbar');
  await i18nToolbar.getByPlaceholder(/搜索/).fill(seedKey);
  await i18nToolbar.getByPlaceholder(/搜索/).press('Enter');

  const seededListResponse = await page.request.get(`${apiBaseUrl}/system/i18n/list`, {
    headers: authHeaders(accessToken),
    params: { key: seedKey, page: '1', pageSize: '10' },
  });
  const seededListPayload = await seededListResponse.json();
  const rawCreatedAt = seededListPayload.data.items[0]?.createdAt as string | undefined;
  const rawUpdatedAt = seededListPayload.data.items[0]?.updatedAt as string | undefined;

  const targetRow = page.getByRole('row', { name: new RegExp(seedKey) }).first();
  await expect(targetRow).toBeVisible();
  if (rawCreatedAt) {
    await expect(targetRow).not.toContainText(rawCreatedAt);
  }
  if (rawUpdatedAt) {
    await expect(targetRow).not.toContainText(rawUpdatedAt);
  }

  await targetRow.getByRole('button', { name: '详情' }).click();
  const detailDialog = page
    .getByRole('dialog')
    .filter({ has: page.getByText('翻译详情', { exact: true }) });
  await expect(detailDialog).toBeVisible();
  await expect(detailDialog.getByText(seedKey)).toBeVisible();
  const detailText = (await detailDialog.textContent()) || '';
  if (rawCreatedAt) {
    expect(detailText).not.toContain(rawCreatedAt);
  }
  if (rawUpdatedAt) {
    expect(detailText).not.toContain(rawUpdatedAt);
  }
  expect(detailText).toMatch(/\d{4}-\d{2}-\d{2}\s+\d{2}:\d{2}(?!:\d{2})/);
  await detailDialog.getByRole('button', { name: '关闭' }).click();

  await targetRow.getByRole('button', { name: '编辑' }).click();
  const editDialog = page
    .getByRole('dialog')
    .filter({ has: page.getByText('编辑翻译', { exact: true }) });
  await expect(editDialog).toBeVisible();
  const editTextarea = editDialog.locator('textarea').first();
  await editTextarea.fill('更新值');
  await editDialog.getByRole('button', { name: '确定' }).click();
  await expect(editDialog).toHaveCount(0);

  await expect
    .poll(async () => {
      const listResp = await page.request.get(`${apiBaseUrl}/system/i18n/list`, {
        headers: authHeaders(accessToken),
        params: { key: seedKey, page: '1', pageSize: '10' },
      });
      const listPayload = await listResp.json();
      return listPayload.data.items[0]?.value;
    })
    .toBe('更新值');

  await page.getByRole('button', { name: '新增' }).click();
  const createDialog = page
    .getByRole('dialog')
    .filter({ has: page.getByText('新增翻译', { exact: true }) });
  await expect(createDialog).toBeVisible();
  const createKey = `${seedKey}.created`;
  await createDialog.locator('input').nth(0).fill('system.config');
  await createDialog.locator('input').nth(1).fill('messages');
  await createDialog.locator('input').nth(2).fill(createKey);
  await createDialog.getByRole('combobox').first().click();
  await createDialog.getByRole('option', { name: 'en-US' }).click();
  await createDialog.locator('textarea').first().fill('Created Value');
  await Promise.all([
    waitForOkApiResponse(
      page,
      (response) =>
        response.url().includes('/system/i18n') && response.request().method() === 'POST',
    ),
    createDialog.getByRole('button', { name: '确定' }).click(),
  ]);
  await expect(createDialog).toHaveCount(0);

  await Promise.all([
    waitForOkApiResponse(
      page,
      (response) =>
        response.url().includes('/system/i18n/list') &&
        decodeURIComponent(response.url()).includes(`key=${createKey}`) &&
        response.request().method() === 'GET',
    ),
    (async () => {
      const toolbarKeyword = page.locator('.search-toolbar').getByPlaceholder(/搜索/);
      await toolbarKeyword.fill(createKey);
      await toolbarKeyword.press('Enter');
    })(),
  ]);

  const createdListResp = await page.request.get(`${apiBaseUrl}/system/i18n/list`, {
    headers: authHeaders(accessToken),
    params: { key: createKey, page: '1', pageSize: '10' },
  });
  const createdListPayload = await createdListResp.json();
  const createdRowId = createdListPayload.data.items[0]?.id as number | undefined;
  expect(createdListPayload.data.items[0]?.key).toBe(createKey);

  {
    const toolbarKeyword = page.locator('.search-toolbar').getByPlaceholder(/搜索/);
    await toolbarKeyword.fill(seedKey);
    await toolbarKeyword.press('Enter');
  }
  const deleteRow = page.getByRole('row', { name: new RegExp(seedKey) }).first();
  await deleteRow.getByRole('button', { name: '删除' }).click();
  const deleteConfirmPopup = page
    .locator(
      '.arco-popconfirm:visible, .arco-trigger-popup:visible, .arco-popover:visible, [role="tooltip"]:visible, [role="dialog"]:visible',
    )
    .filter({ has: page.getByRole('button', { name: '确定', exact: true }) })
    .last();
  await expect(deleteConfirmPopup).toBeVisible();
  await deleteConfirmPopup.getByRole('button', { name: '确定', exact: true }).click();

  await expect
    .poll(async () => {
      const listResp = await page.request.get(`${apiBaseUrl}/system/i18n/list`, {
        headers: authHeaders(accessToken),
        params: { key: seedKey, page: '1', pageSize: '10' },
      });
      const listPayload = await listResp.json();
      return listPayload.data.items.length;
    })
    .toBe(1);

  if (createdRowId) {
    await page.request
      .delete(`${apiBaseUrl}/system/i18n/${createdRowId}`, {
        headers: await verifiedHeaders(page, accessToken),
      })
      .catch(() => undefined);
  }
});

test('i18n smoke: import csv creates updates and downloads error file', async ({
  page,
}, testInfo) => {
  const accessToken = await signInAsAdmin(page);
  const seedBase = `i18n.import.${Date.now()}`;
  const updateKey = `${seedBase}.update`;
  const createKey = `${seedBase}.create`;

  const seedResponse = await page.request.post(`${apiBaseUrl}/system/i18n`, {
    headers: await verifiedHeaders(page, accessToken),
    data: {
      module: 'system.config',
      group: 'messages',
      key: updateKey,
      locale: 'zh-CN',
      value: '导入前旧值',
      remark: 'seed',
    },
  });
  expect(seedResponse.ok()).toBeTruthy();
  const seedPayload = await seedResponse.json();
  expect(seedPayload.code).toBe(200);

  const successCsv = [
    'module,group,key,locale,value,remark',
    `system.config,messages,${updateKey},zh-CN,导入后新值,updated by smoke`,
    `system.config,messages,${createKey},zh-CN,批量新增值,created by smoke`,
  ].join('\n');

  await page.goto('/system/i18n', { waitUntil: 'domcontentloaded' });
  const importButton = page.getByRole('button', { name: '导入', exact: true });
  await expect(importButton).toBeVisible();
  const successFileChooserPromise = page.waitForEvent('filechooser');
  await importButton.click();
  const successFileChooser = await successFileChooserPromise;
  await Promise.all([
    waitForOkApiResponse(
      page,
      (response) =>
        response.url().includes('/system/i18n/import') && response.request().method() === 'POST',
    ),
    successFileChooser.setFiles({
      name: 'system-i18n-import.csv',
      mimeType: 'text/csv',
      buffer: Buffer.from(`\uFEFF${successCsv}`, 'utf8'),
    }),
  ]);
  await dismissVisibleSuccessDialog(page);

  await expect
    .poll(async () => {
      const listResp = await page.request.get(`${apiBaseUrl}/system/i18n/list`, {
        headers: authHeaders(accessToken),
        params: { key: updateKey, page: '1', pageSize: '10' },
      });
      const listPayload = await listResp.json();
      return listPayload.data.items[0]?.value;
    })
    .toBe('导入后新值');

  await expect
    .poll(async () => {
      const listResp = await page.request.get(`${apiBaseUrl}/system/i18n/list`, {
        headers: authHeaders(accessToken),
        params: { key: createKey, page: '1', pageSize: '10' },
      });
      const listPayload = await listResp.json();
      return listPayload.data.items[0]?.value;
    })
    .toBe('批量新增值');

  const invalidKey = `${seedBase}.invalid`;
  const invalidCsv = [
    'module,group,key,locale,value,remark',
    `system.config,messages,${invalidKey},zh-CN,,missing value`,
    `system.config,messages,${invalidKey},zh-CN,重复值,duplicate row`,
  ].join('\n');

  const downloadPromise = page.waitForEvent('download');
  const invalidFileChooserPromise = page.waitForEvent('filechooser');
  await importButton.click();
  const invalidFileChooser = await invalidFileChooserPromise;
  await Promise.all([
    waitForOkApiResponse(
      page,
      (response) =>
        response.url().includes('/system/i18n/import') && response.request().method() === 'POST',
    ),
    invalidFileChooser.setFiles({
      name: 'system-i18n-import-invalid.csv',
      mimeType: 'text/csv',
      buffer: Buffer.from(`\uFEFF${invalidCsv}`, 'utf8'),
    }),
  ]);

  const errorDownload = await downloadPromise;
  expect(errorDownload.suggestedFilename()).toBe('system-i18n-import-errors.csv');
  const downloadPath = testInfo.outputPath('system-i18n-import-errors.csv');
  await errorDownload.saveAs(downloadPath);
  const errorCsv = await fs.readFile(downloadPath, 'utf8');
  expect(errorCsv).toContain('i18n.value.required');
  expect(errorCsv).toContain('import.duplicate.row.2');

  await expect(page.locator('.search-toolbar')).toBeVisible();

  const createdListResp = await page.request.get(`${apiBaseUrl}/system/i18n/list`, {
    headers: authHeaders(accessToken),
    params: { key: createKey, page: '1', pageSize: '10' },
  });
  const createdListPayload = await createdListResp.json();
  const createdRowId = createdListPayload.data.items[0]?.id as number | undefined;

  if (createdRowId) {
    await page.request
      .delete(`${apiBaseUrl}/system/i18n/${createdRowId}`, {
        headers: await verifiedHeaders(page, accessToken),
      })
      .catch(() => undefined);
  }
  await page.request
    .delete(`${apiBaseUrl}/system/i18n/${seedPayload.data.id}`, {
      headers: await verifiedHeaders(page, accessToken),
    })
    .catch(() => undefined);
});

test('i18n permission smoke: list-only role can view page but cannot mutate translations', async ({
  page,
}) => {
  await withListViewOnlyViewer(
    page,
    {
      slug: 'i18n',
      cleanupTag: 'system-pages:i18n-viewer',
      roleName: '国际化只读烟测角色',
      nickname: '国际化只读烟测用户',
      listPermission: 'system:i18n:list',
      apiPermissions: [
        ['/api/v1/system/i18n/list', 'GET'],
        ['/api/v1/system/i18n/overview', 'GET'],
      ],
      route: '/system/i18n',
      identityLabel: '国际化管理',
    },
    async (viewerPage) => {
      const headerActions = viewerPage.locator('.system-list__work-actions').first();
      await expect(headerActions.getByRole('button', { name: '新增' })).toHaveCount(0);
      await expect(headerActions.getByRole('button', { name: '刷新缓存' })).toHaveCount(0);
      await expect(headerActions.getByRole('button', { name: '刷新', exact: true })).toHaveCount(0);
      await expect(headerActions.getByRole('button', { name: '导出' })).toHaveCount(0);
      await expect(headerActions.getByRole('button', { name: '导入' })).toHaveCount(0);
    },
  );
});

test('refresh sync smoke: i18n page auto-updates across isolated contexts', async ({
  browser,
  page,
}) => {
  test.setTimeout(45000);
  const adminLogin = await createSharedAdminLogin(page);
  const accessToken = adminLogin.accessToken;
  const i18nKey = `i18n.sync.${Date.now()}`;

  const syncContext = await browser.newContext();
  const syncPage = await syncContext.newPage();

  try {
    await installClientSession(syncPage, adminLogin);
    await syncPage.goto('/system/i18n', { waitUntil: 'domcontentloaded' });
    {
      const toolbarKeyword = syncPage.locator('.search-toolbar').getByPlaceholder(/搜索/).first();
      await toolbarKeyword.fill(i18nKey);
      await toolbarKeyword.press('Enter');
    }
    await expect(syncPage.getByText(i18nKey, { exact: false })).toHaveCount(0);

    const createResponse = await page.request.post(`${apiBaseUrl}/system/i18n`, {
      headers: await verifiedHeaders(page, accessToken),
      data: {
        module: 'system.config',
        group: 'messages',
        key: i18nKey,
        locale: 'zh-CN',
        value: '跨上下文同步',
        remark: 'sync smoke',
      },
    });
    expect(createResponse.ok()).toBeTruthy();
    const createPayload = await createResponse.json();
    expect(createPayload.code).toBe(200);
    const createdId = createPayload.data.id as number;

    await expect(syncPage.getByText(i18nKey, { exact: false }).first()).toBeVisible({
      timeout: 15000,
    });

    await page.request
      .delete(`${apiBaseUrl}/system/i18n/${createdId}`, {
        headers: await verifiedHeaders(page, accessToken),
      })
      .catch(() => undefined);
  } finally {
    await closeExtraBrowserContext(syncContext);
  }
});

import fs from 'node:fs/promises';
import { expect, test, type Page, type Route } from '@playwright/test';
import { installOperationToken, signInAsAdmin } from '../../helpers/auth';

type CleanupCase = {
  name: string;
  path: string;
  cleanupButtonName: string;
  settingKey: string;
  listRoute: RegExp;
  cleanupRoute: RegExp;
  listPayload: Record<string, unknown>;
};

const cleanupCases: CleanupCase[] = [
  {
    name: 'login-log',
    path: '/system/login-log',
    cleanupButtonName: '清理日志',
    settingKey: 'audit.login_log_retention_options',
    listRoute: /\/api\/v1\/system\/login-log\/list(?:\?.*)?$/,
    cleanupRoute: /\/api\/v1\/system\/login-log\/cleanup$/,
    listPayload: {
      items: [
        {
          id: 1001,
          username: 'admin',
          ipaddr: '127.0.0.1',
          loginLocation: '本地',
          browser: 'Chrome',
          os: 'Windows',
          status: 1,
          msg: '',
          loginTime: '2026-04-29 09:00:00',
        },
      ],
      total: 1,
      page: 1,
      pageSize: 10,
    },
  },
  {
    name: 'session',
    path: '/system/session',
    cleanupButtonName: '清理历史会话',
    settingKey: 'audit.session_cleanup_retention_options',
    listRoute: /\/api\/v1\/system\/session\/list(?:\?.*)?$/,
    cleanupRoute: /\/api\/v1\/system\/session\/cleanup$/,
    listPayload: {
      items: [
        {
          sessionId: 'smoke-target-session',
          userId: 9001,
          username: 'audit_user',
          nickname: '审计用户',
          lastIp: '127.0.0.2',
          browser: 'Chrome',
          os: 'Windows',
          device: 'Desktop',
          userAgent: 'Mozilla/5.0',
          refreshExpiresAt: '2026-04-30 08:00:00',
          lastRefreshAt: '2026-04-29 08:00:00',
          lastActivityAt: '2026-04-29 08:30:00',
          revokedAt: '',
          createdAt: '2026-04-29 08:00:00',
        },
      ],
      total: 1,
      activeCount: 1,
      revokedCount: 0,
      page: 1,
      pageSize: 10,
    },
  },
  {
    name: 'operation-log',
    path: '/system/operation-log',
    cleanupButtonName: '清理日志',
    settingKey: 'audit.operation_log_retention_options',
    listRoute: /\/api\/v1\/system\/operation-log\/list(?:\?.*)?$/,
    cleanupRoute: /\/api\/v1\/system\/operation-log\/cleanup$/,
    listPayload: {
      items: [
        {
          id: 2001,
          title: 'system.user.create',
          businessType: 1,
          method: 'POST',
          operName: 'admin',
          operUrl: '/api/v1/system/user',
          operIp: '127.0.0.1',
          sourceDomain: 'iam',
          sourcePage: 'user',
          operParam: '{}',
          jsonResult: '{"code":200,"message":"ok"}',
          status: 1,
          failureCategory: '',
          errorMsg: '',
          operTime: '2026-04-29 10:00:00',
          costTime: 18,
        },
      ],
      total: 1,
      page: 1,
      pageSize: 10,
    },
  },
];

async function fulfillJson(route: Route, status: number, body: Record<string, unknown>) {
  await route.fulfill({
    status,
    contentType: 'application/json',
    body: JSON.stringify(body),
  });
}

async function mockAuditSetting(page: Page, settingKey: string) {
  await page.route(/\/api\/v1\/system\/setting\/group\/audit$/, async (route) => {
    await fulfillJson(route, 200, {
      code: 200,
      data: {
        groupKey: 'audit',
        items: [
          {
            settingKey,
            settingValue: '[1,7,30]',
          },
        ],
      },
    });
  });
}

function cleanupBar(page: Page, cleanupButtonName: string) {
  return page
    .locator('.page-panel')
    .filter({
      has: page.getByRole('button', { name: cleanupButtonName, exact: true }),
    })
    .first();
}

function rangePopup(page: Page) {
  return page.locator('body .arco-picker-range-container').last();
}

function cleanupDialog(page: Page) {
  return page.locator('.arco-modal:visible').last();
}

function dialogConfirmButton(page: Page) {
  return cleanupDialog(page)
    .locator('.arco-modal-footer')
    .getByRole('button', { name: '清理', exact: true })
    .last();
}

// Select a past start/end day inside the cleanup dialog's Arco RangePicker
// (single trigger, showTime enabled, mounted to body).
async function selectCleanupRange(page: Page) {
  const trigger = cleanupDialog(page).locator('.arco-picker').first();
  await trigger.click();

  const popup = rangePopup(page);
  await expect(popup).toBeVisible();

  const leftPanel = popup.locator('.arco-panel-date').first();
  const monthLabel = leftPanel.locator('.arco-picker-header-value');
  const currentMonthLabel = await monthLabel.textContent();
  const previousMonthButton = leftPanel
    .locator('.arco-picker-header-icon:not(.arco-picker-header-icon-hidden)')
    .last();
  await previousMonthButton.click();
  await expect(monthLabel).not.toHaveText(currentMonthLabel || '');

  // The previous month is fully in the past, so every in-view day is enabled.
  const enabledCells = leftPanel.locator('.arco-picker-cell-in-view:not(.arco-picker-cell-disabled)');
  await expect.poll(() => enabledCells.count()).toBeGreaterThanOrEqual(28);

  await enabledCells.nth(4).click();
  await enabledCells.nth(10).click();

  const confirm = popup.locator('.arco-picker-btn-confirm');
  if (await confirm.count()) {
    await confirm.first().click();
  }
  await expect(popup).toBeHidden();
}

test.describe('cleanup range governance smoke', () => {
  test.describe.configure({ timeout: 120000 });

  for (const cleanupCase of cleanupCases) {
    test(`${cleanupCase.name} cleanup dialog submits retention and range payloads`, async ({
      page,
    }) => {
      const accessToken = await signInAsAdmin(page);
      await page.goto('/dashboard', { waitUntil: 'networkidle' });
      await installOperationToken(page, accessToken);

      await mockAuditSetting(page, cleanupCase.settingKey);
      await page.route(cleanupCase.listRoute, async (route) => {
        await fulfillJson(route, 200, {
          code: 200,
          data: cleanupCase.listPayload,
        });
      });

      let capturedCleanupPayload: Record<string, unknown> | null = null;
      await page.route(cleanupCase.cleanupRoute, async (route) => {
        capturedCleanupPayload = route.request().postDataJSON() as Record<string, unknown>;
        await fulfillJson(route, 200, {
          code: 200,
          data: { clearedCount: 1 },
        });
      });

      await page.goto(cleanupCase.path, { waitUntil: 'networkidle' });

      const bar = cleanupBar(page, cleanupCase.cleanupButtonName);
      await expect(bar).toBeVisible();

      const cleanupTrigger = bar
        .locator('.table-batch-action-bar__meta')
        .getByRole('button', { name: cleanupCase.cleanupButtonName, exact: true })
        .first();

      // 1) Retention mode (default): irreversible warning is shown, submit sends retentionDays.
      await cleanupTrigger.click();
      const dialog = cleanupDialog(page);
      await expect(dialog).toBeVisible();
      await expect(dialog.locator('.arco-alert-warning')).toBeVisible();
      await dialogConfirmButton(page).click();

      await expect.poll(() => capturedCleanupPayload).not.toBeNull();
      expect(typeof capturedCleanupPayload?.retentionDays).toBe('number');
      expect(capturedCleanupPayload?.startedAt).toBeUndefined();
      await expect(page.locator('.arco-message')).toContainText(/已清理|清理成功/);

      // 2) Range mode: submit sends startedAt/endedAt instead of retentionDays.
      capturedCleanupPayload = null;
      await cleanupTrigger.click();
      await expect(cleanupDialog(page)).toBeVisible();
      // Arco Radio.Group type="button" renders the native input visually hidden,
      // so drive the visible label instead of the radio role.
      await cleanupDialog(page)
        .locator('.arco-radio-button')
        .filter({ hasText: '按时间范围' })
        .click();
      await selectCleanupRange(page);
      await dialogConfirmButton(page).click();

      await expect.poll(() => capturedCleanupPayload).not.toBeNull();
      expect(capturedCleanupPayload?.retentionDays).toBeUndefined();
      expect(String(capturedCleanupPayload?.startedAt || '')).toMatch(
        /\d{4}-\d{2}-\d{2}T\d{2}:\d{2}/,
      );
      expect(String(capturedCleanupPayload?.endedAt || '')).toMatch(/\d{4}-\d{2}-\d{2}T\d{2}:\d{2}/);
    });
  }

  test('login-log filter time range renders Arco shortcuts and horizontal panels', async ({
    page,
  }, testInfo) => {
    const loginLogCase = cleanupCases[0];
    const accessToken = await signInAsAdmin(page);
    await page.goto('/dashboard', { waitUntil: 'networkidle' });
    await installOperationToken(page, accessToken);

    await mockAuditSetting(page, loginLogCase.settingKey);
    await page.route(loginLogCase.listRoute, async (route) => {
      await fulfillJson(route, 200, {
        code: 200,
        data: loginLogCase.listPayload,
      });
    });

    await page.goto(loginLogCase.path, { waitUntil: 'networkidle' });

    // Login-log filters render inside the shared SearchToolbar inline slot and
    // use the shared TimeRangeFilter component (the page-local
    // .auth-login-log-page__time-range-* DOM was retired in 8ce624d8).
    const filterGrid = page.locator('.search-toolbar__inline');
    await expect(filterGrid).toBeVisible();

    const rangeTrigger = filterGrid.locator('.time-range-filter__trigger');
    await expect(rangeTrigger).toBeVisible();
    await expect(rangeTrigger).toContainText('时间范围');
    await rangeTrigger.click();

    const popup = rangePopup(page);
    await expect(popup).toBeVisible();

    const popupBox = await popup.boundingBox();
    expect(popupBox).not.toBeNull();
    expect(popupBox?.width ?? 0).toBeGreaterThanOrEqual(540);
    expect(popupBox?.width ?? 0).toBeLessThanOrEqual(700);

    const shell = popup.locator('.time-range-filter__shell');
    await expect(shell).toBeVisible();

    const shortcutRows = shell.locator('.time-range-filter__shortcut-row');
    await expect(shortcutRows).toHaveCount(2);
    await expect(shortcutRows.first().locator('.time-range-filter__shortcut')).toHaveCount(9);
    await expect(shortcutRows.nth(1).locator('.time-range-filter__shortcut')).toHaveCount(2);
    await expect(shell.getByRole('button', { name: '今天', exact: true })).toBeVisible();
    await expect(shell.getByRole('button', { name: '昨天', exact: true })).toBeVisible();

    const summary = shell.locator('.time-range-filter__summary');
    await expect(summary).toBeVisible();
    await expect(summary).toContainText('开始时间');
    await expect(summary).toContainText('结束时间');
    await expect(summary).toContainText('至');

    const summaryBox = await summary.boundingBox();
    expect(summaryBox).not.toBeNull();

    const rangePopupWrapper = popup.locator('.arco-picker-range-wrapper');
    await expect(rangePopupWrapper).toHaveCSS('flex-wrap', 'nowrap');

    const weekLabels = popup
      .locator('.arco-panel-date')
      .first()
      .locator('.arco-picker-week-list-item');
    await expect(weekLabels).toHaveText(['一', '二', '三', '四', '五', '六', '日']);

    // The shared popup CSS lays week list and date rows out as horizontal flex
    // (React 19 compat fix in list-page.css); column alignment is asserted
    // geometrically below.
    const weekList = popup.locator('.arco-panel-date').first().locator('.arco-picker-week-list');
    await expect(weekList).toHaveCSS('display', 'flex');

    const firstDateRow = popup
      .locator('.arco-panel-date')
      .first()
      .locator('.arco-picker-body .arco-picker-row')
      .first();
    await expect(firstDateRow).toHaveCSS('display', 'flex');

    const firstDateCells = firstDateRow.locator('.arco-picker-cell');
    await expect(firstDateCells).toHaveCount(7);
    for (let index = 0; index < 7; index += 1) {
      const weekBox = await weekLabels.nth(index).boundingBox();
      const dayBox = await firstDateCells.nth(index).boundingBox();
      expect(weekBox).not.toBeNull();
      expect(dayBox).not.toBeNull();

      const weekCenter = (weekBox?.x || 0) + (weekBox?.width || 0) / 2;
      const dayCenter = (dayBox?.x || 0) + (dayBox?.width || 0) / 2;
      expect(Math.abs(weekCenter - dayCenter)).toBeLessThanOrEqual(4);
      expect(Math.abs((weekBox?.width || 0) - (dayBox?.width || 0))).toBeLessThanOrEqual(4);
    }

    const rangePanels = popup.locator('.arco-panel-date');
    await expect(rangePanels).toHaveCount(2);
    const leftPanelBox = await rangePanels.nth(0).boundingBox();
    const rightPanelBox = await rangePanels.nth(1).boundingBox();
    expect(leftPanelBox).not.toBeNull();
    expect(rightPanelBox).not.toBeNull();
    expect(Math.abs((leftPanelBox?.y || 0) - (rightPanelBox?.y || 0))).toBeLessThanOrEqual(4);
    expect((rightPanelBox?.x || 0) - (leftPanelBox?.x || 0)).toBeGreaterThan(20);

    expect(
      (leftPanelBox?.y || 0) - ((summaryBox?.y || 0) + (summaryBox?.height || 0)),
    ).toBeGreaterThanOrEqual(0);

    await expect(popup.locator('.arco-picker-btn-confirm')).toBeVisible();
    await expect(shell.getByRole('button', { name: '取消', exact: true })).toBeVisible();
    await popup.screenshot({
      path: testInfo.outputPath('login-log-time-range-picker.png'),
    });

    await expect(popup.locator('.arco-picker-btn-select-time')).toBeVisible();
    await popup.locator('.arco-picker-btn-select-time').click();

    const timePanels = popup.locator('.arco-panel-date-timepicker');
    await expect(timePanels).toHaveCount(2);
    const leftTimeBox = await timePanels.nth(0).boundingBox();
    const rightTimeBox = await timePanels.nth(1).boundingBox();
    expect(leftTimeBox).not.toBeNull();
    expect(rightTimeBox).not.toBeNull();
    expect(Math.abs((leftTimeBox?.y || 0) - (rightTimeBox?.y || 0))).toBeLessThanOrEqual(4);
    expect((rightTimeBox?.x || 0) - (leftTimeBox?.x || 0)).toBeGreaterThan(20);

    await shell.getByRole('button', { name: '今天', exact: true }).click();
    await expect(rangeTrigger).toContainText('今天');
    await expect(popup).toBeHidden();
  });

  test('selected login-log rows export as csv without calling backend export endpoint', async ({
    page,
  }, testInfo) => {
    const accessToken = await signInAsAdmin(page);
    await page.goto('/dashboard', { waitUntil: 'networkidle' });
    await installOperationToken(page, accessToken);

    await mockAuditSetting(page, 'audit.login_log_retention_options');
    await page.route(/\/api\/v1\/system\/login-log\/list(?:\?.*)?$/, async (route) => {
      await fulfillJson(route, 200, {
        code: 200,
        data: {
          items: [
            {
              id: 1001,
              username: 'export_admin',
              ipaddr: '127.0.0.1',
              loginLocation: '本地',
              browser: 'Chrome',
              os: 'Windows',
              status: 1,
              msg: '',
              loginTime: '2026-04-29 09:00:00',
            },
            {
              id: 1002,
              username: 'export_guest',
              ipaddr: '127.0.0.2',
              loginLocation: '异地',
              browser: 'Firefox',
              os: 'Linux',
              status: 0,
              msg: 'auth.login.failed',
              loginTime: '2026-04-29 10:00:00',
            },
          ],
          total: 2,
          page: 1,
          pageSize: 10,
        },
      });
    });

    let exportRouteHit = false;
    await page.route(/\/api\/v1\/system\/login-log\/export$/, async (route) => {
      exportRouteHit = true;
      await fulfillJson(route, 500, { code: 500, message: 'should.not.call' });
    });

    await page.goto('/system/login-log', { waitUntil: 'networkidle' });
    const firstRow = page.locator('.arco-table tbody tr').first();
    await expect(firstRow).toBeVisible();
    await firstRow.locator('label.arco-checkbox').click();

    const downloadPromise = page.waitForEvent('download');
    await cleanupBar(page, '清理日志').getByRole('button', { name: '导出', exact: true }).click();
    const download = await downloadPromise;
    expect(exportRouteHit).toBe(false);
    expect(download.suggestedFilename()).toBe('system-login-log-export.csv');

    const filePath = testInfo.outputPath('selected-login-log-export.csv');
    await download.saveAs(filePath);
    const csv = await fs.readFile(filePath, 'utf8');
    expect(csv).toContain('username,ipaddr,loginLocation,browser,os,status,msg,loginTime');
    expect(csv).toContain('export_admin,127.0.0.1');
    expect(csv).not.toContain('export_guest,127.0.0.2');
  });

  test('selected operation-log rows export as csv without calling backend export endpoint', async ({
    page,
  }, testInfo) => {
    const accessToken = await signInAsAdmin(page);
    await page.goto('/dashboard', { waitUntil: 'networkidle' });
    await installOperationToken(page, accessToken);

    await mockAuditSetting(page, 'audit.operation_log_retention_options');
    await page.route(/\/api\/v1\/system\/operation-log\/list(?:\?.*)?$/, async (route) => {
      await fulfillJson(route, 200, {
        code: 200,
        data: {
          items: [
            {
              id: 2001,
              title: 'system.user.create',
              businessType: 1,
              method: 'POST',
              operName: 'export_admin',
              operUrl: '/api/v1/system/user',
              operIp: '127.0.0.1',
              sourceDomain: 'iam',
              sourcePage: 'user',
              operParam: '{}',
              jsonResult: '{"code":200,"message":"ok"}',
              status: 1,
              failureCategory: '',
              errorMsg: '',
              operTime: '2026-04-29 10:00:00',
              costTime: 18,
            },
            {
              id: 2002,
              title: 'system.setting.update',
              businessType: 2,
              method: 'PUT',
              operName: 'export_guest',
              operUrl: '/api/v1/system/setting/group/basic',
              operIp: '127.0.0.2',
              sourceDomain: 'config',
              sourcePage: 'setting',
              operParam: '{}',
              jsonResult: '{"code":400,"message":"request.failed"}',
              status: 2,
              failureCategory: 'validation',
              errorMsg: 'request.failed',
              operTime: '2026-04-29 11:00:00',
              costTime: 27,
            },
          ],
          total: 2,
          page: 1,
          pageSize: 10,
        },
      });
    });

    let exportRouteHit = false;
    await page.route(/\/api\/v1\/system\/operation-log\/export$/, async (route) => {
      exportRouteHit = true;
      await fulfillJson(route, 500, { code: 500, message: 'should.not.call' });
    });

    await page.goto('/system/operation-log', { waitUntil: 'networkidle' });
    const firstRow = page.locator('.arco-table tbody tr').first();
    await expect(firstRow).toBeVisible();
    await firstRow.locator('label.arco-checkbox').click();

    const downloadPromise = page.waitForEvent('download');
    await cleanupBar(page, '清理日志').getByRole('button', { name: '导出', exact: true }).click();
    const download = await downloadPromise;
    expect(exportRouteHit).toBe(false);
    expect(download.suggestedFilename()).toBe('system-operation-log-export.csv');

    const filePath = testInfo.outputPath('selected-operation-log-export.csv');
    await download.saveAs(filePath);
    const csv = await fs.readFile(filePath, 'utf8');
    expect(csv).toContain(
      'requestId,title,businessType,sourceDomain,sourcePage,method,operName,operUrl,operIp,status,failureCategory,errorMsg,operTime,costTime',
    );
    expect(csv).toContain('system.user.create');
    expect(csv).toContain('export_admin');
    expect(csv).not.toContain('system.setting.update');
    expect(csv).not.toContain('export_guest');
  });
});

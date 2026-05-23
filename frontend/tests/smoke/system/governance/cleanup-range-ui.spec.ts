import fs from 'node:fs/promises';
import { expect, test, type Locator, type Page, type Route } from '@playwright/test';
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
  return page.locator('.page-panel').filter({
    has: page.getByRole('button', { name: cleanupButtonName, exact: true }),
  }).first();
}

function popupConfirmButton(page: Page) {
  return page
    .locator('.arco-popconfirm:visible, .arco-trigger-popup:visible, .arco-popover:visible, [role="tooltip"]:visible, [role="dialog"]:visible')
    .last()
    .getByRole('button', { name: '确定', exact: true })
    .last();
}

test.describe('cleanup range governance smoke', () => {
  test.describe.configure({ timeout: 120000 });

  for (const cleanupCase of cleanupCases) {
    test(`${cleanupCase.name} cleanup range submits selected timestamps and keeps inline layout`, async ({ page }) => {
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

      const modeSelect = bar.locator('.arco-select').first();
      await modeSelect.click();
      await page.getByRole('option', { name: '按时间范围', exact: true }).click();

      const dateInputs = bar.locator('input[type="datetime-local"]');
      await expect(dateInputs).toHaveCount(2);

      const startInput = dateInputs.nth(0);
      const endInput = dateInputs.nth(1);
      await startInput.fill('2026-06-13T18:26');
      await endInput.fill('2026-06-14T09:45');

      await expect(startInput).toHaveValue('2026-06-13T18:26');
      await expect(endInput).toHaveValue('2026-06-14T09:45');

      const cleanupButton = bar
        .locator('.table-batch-action-bar__meta')
        .getByRole('button', { name: cleanupCase.cleanupButtonName, exact: true })
        .first();
      const startBox = await startInput.boundingBox();
      const endBox = await endInput.boundingBox();
      const buttonBox = await cleanupButton.boundingBox();

      expect(startBox).not.toBeNull();
      expect(endBox).not.toBeNull();
      expect(buttonBox).not.toBeNull();

      expect(Math.abs((startBox?.y || 0) - (endBox?.y || 0))).toBeLessThanOrEqual(4);
      expect(Math.abs((startBox?.y || 0) - (buttonBox?.y || 0))).toBeLessThanOrEqual(6);

      await cleanupButton.click();
      await popupConfirmButton(page).click();

      await expect
        .poll(() => capturedCleanupPayload)
        .not.toBeNull();

      expect(capturedCleanupPayload?.retentionDays).toBeUndefined();
      expect(String(capturedCleanupPayload?.startedAt || '')).toContain('2026-06-13T18:26');
      expect(String(capturedCleanupPayload?.endedAt || '')).toContain('2026-06-14T09:45');
      await expect(page.locator('.arco-message')).toContainText(/已清理|清理成功/);
    });
  }

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
